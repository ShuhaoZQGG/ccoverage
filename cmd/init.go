package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Install ccoverage session hook into .claude/settings.json",
	Long:  "Adds a PreToolUse hook to .claude/settings.json so that a one-line coverage summary appears automatically on session start.",
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("init: resolve executable path: %w", err)
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return fmt.Errorf("init: eval symlinks: %w", err)
	}

	absRepo, err := filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("init: resolve repo path: %w", err)
	}

	settingsPath := filepath.Join(absRepo, ".claude", "settings.json")

	// Ensure .claude/ directory exists.
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0755); err != nil {
		return fmt.Errorf("init: create .claude dir: %w", err)
	}

	// Read existing settings or start fresh.
	settings := make(map[string]interface{})
	data, err := os.ReadFile(settingsPath)
	if err == nil {
		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("init: parse settings.json: %w", err)
		}
	}

	hookCommand := fmt.Sprintf("%s summary --target %s", exePath, absRepo)

	// Build the ccoverage hook entry.
	ccoverageHook := map[string]interface{}{
		"type":    "command",
		"command": hookCommand,
	}
	ccoverageMatcher := map[string]interface{}{
		"matcher": "",
		"hooks":   []interface{}{ccoverageHook},
	}

	// Get or create hooks map.
	hooks, _ := settings["hooks"].(map[string]interface{})
	if hooks == nil {
		hooks = make(map[string]interface{})
	}

	// Remove legacy ccoverage hooks from previous event types.
	removeCcoverageHooks(hooks, "PreToolUse")
	removeCcoverageHooks(hooks, "SessionStart")
	removeCcoverageHooks(hooks, "Stop")

	// Install under SessionEnd (exit 2 + stderr shows message to user after session closes).
	endHooks, _ := hooks["SessionEnd"].([]interface{})

	// Check if a ccoverage hook already exists; update in place if so.
	found := false
	for i, entry := range endHooks {
		m, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		innerHooks, _ := m["hooks"].([]interface{})
		for _, ih := range innerHooks {
			hm, ok := ih.(map[string]interface{})
			if !ok {
				continue
			}
			cmd, _ := hm["command"].(string)
			if len(cmd) > 0 && containsCcoverage(cmd) {
				endHooks[i] = ccoverageMatcher
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		endHooks = append(endHooks, ccoverageMatcher)
	}

	hooks["SessionEnd"] = endHooks
	settings["hooks"] = hooks

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("init: marshal settings: %w", err)
	}

	if err := os.WriteFile(settingsPath, append(out, '\n'), 0644); err != nil {
		return fmt.Errorf("init: write settings.json: %w", err)
	}

	fmt.Println("Installed ccoverage session hook in .claude/settings.json")
	return nil
}

func containsCcoverage(s string) bool {
	return strings.Contains(s, "ccoverage summary") || strings.Contains(s, "ccoverage\" summary")
}

// removeCcoverageHooks removes any ccoverage hooks from the given event key.
func removeCcoverageHooks(hooks map[string]interface{}, eventKey string) {
	entries, _ := hooks[eventKey].([]interface{})
	if len(entries) == 0 {
		return
	}
	var kept []interface{}
	for _, entry := range entries {
		m, ok := entry.(map[string]interface{})
		if !ok {
			kept = append(kept, entry)
			continue
		}
		innerHooks, _ := m["hooks"].([]interface{})
		isCcoverage := false
		for _, ih := range innerHooks {
			hm, ok := ih.(map[string]interface{})
			if !ok {
				continue
			}
			cmd, _ := hm["command"].(string)
			if containsCcoverage(cmd) {
				isCcoverage = true
				break
			}
		}
		if !isCcoverage {
			kept = append(kept, entry)
		}
	}
	if len(kept) == 0 {
		delete(hooks, eventKey)
	} else {
		hooks[eventKey] = kept
	}
}
