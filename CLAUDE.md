# ccoverage

Coverage analysis for Claude Code project configuration. Scans a repo's CLAUDE.md files, skills, MCP servers, hooks, and commands, then joins against session history to classify each item as Active, Underused, Dormant, or Orphaned.

## Build / Test / Run

```sh
CGO_ENABLED=0 go build -o ccoverage .   # CGO_ENABLED=0 needed on macOS (Go 1.21 dyld issue)
CGO_ENABLED=0 go test ./...             # CGO_ENABLED=0 also needed for tests on macOS

ccoverage scan   --repo-path . --format text   # manifest only, no session data
ccoverage report --repo-path . --lookback-days 30 --threshold 2 --format json
ccoverage report --status Dormant,Orphaned                     # filter by status
ccoverage report --type MCP --format json                      # filter by config type
ccoverage report --status Orphaned --error-on-match            # CI gate: exit 1 if matches
ccoverage init   --repo-path ~/Project/MyRepo                  # install SessionEnd hook
ccoverage summary --repo-path ~/Project/MyRepo                 # one-line summary (hook use)
```

Flags: `--repo-path` (default `.`), `--lookback-days` (default `30`), `--format` (`text`|`json`|`md`), `--threshold` (report only, default `2`), `--status` (comma-separated filter), `--type` (comma-separated filter), `--error-on-match` (exit 1 if results remain).

## Architecture

Five-stage pipeline, all orchestrated through `cmd/report.go`:

```
scan (scanner.BuildManifest)
  -> locate (usage.LocateSessionFiles)
    -> parse (usage.ParseSessionFile per file)
      -> classify (coverage.Analyze -> Classify per item)
        -> filter (cmd/report.go filterReport — optional --status/--type)
          -> render (output.RenderText / RenderJSON / RenderMarkdown)
```

## Commands

- **`scan`** — manifest only, no session data
- **`report`** — full pipeline with filtering (`--status`, `--type`) and CI gate (`--error-on-match`)
- **`summary`** — one-line summary for hook use. Writes to stderr + exits 2 so Claude Code `SessionEnd` hooks display it
- **`init`** — installs a `SessionEnd` hook into target repo's `.claude/settings.json`. Cleans up legacy hooks from older event types (PreToolUse, SessionStart, Stop)

## Package Dependency Graph

```
         types          (hub: all shared structs, enums, constants)
        / | \ \
  scanner usage coverage output
              \    |
               coverage imports usage
```

No other cross-imports between leaf packages. `cmd/` imports all five.

## Key Conventions

- **Error wrapping**: `fmt.Errorf("pkg: context: %w", err)` throughout
- **Graceful degradation**: missing files/dirs produce nil results, not errors. Malformed JSONL lines are skipped with `log.Printf`.
- **Manifest key format**: `"Type:Name"` (e.g., `"CLAUDE.md:CLAUDE.md"`, `"MCP:supabase"`)
- **ConfigType constants**: `ConfigClaudeMD`, `ConfigSkill`, `ConfigMCP`, `ConfigHook`, `ConfigCommand`
- **Status constants**: `StatusActive`, `StatusUnderused`, `StatusDormant`, `StatusOrphaned` (4-tier priority in that order)

## Gotchas

- **4MB JSONL buffer** (`usage/jsonl.go:19`): session lines can be huge due to tool outputs. The scanner uses a 4MB buffer; lines exceeding this are silently dropped.
- **Content field dual-type** (`usage/jsonl.go:281-296`): the `content` JSON field can be a plain string or an array of content blocks. `decodeContentBlocks` checks `raw[0]` for `[` to disambiguate.
- **Path encoding keeps leading dash** (`usage/locator.go:23`): `encodeRepoPath` replaces `/` with `-` without stripping the leading `-`, so `/Users/foo/project` becomes `-Users-foo-project`.
- **CLAUDE.md directory-containment matching** (`usage/matcher.go:110-180`): CLAUDE.md items are matched by checking if any session's cwd **or any directory touched by file-oriented tools (Read, Edit, Write, Glob, Grep)** is at or beneath the CLAUDE.md's parent directory. Touched dirs are extracted from tool_use input fields in `extractTouchedDirs` (`usage/jsonl.go`).
- **Hook output protocol**: `summary` uses exit code 2 + stderr for `SessionEnd` hooks. This is the only way to show a message to the user from SessionEnd — stdout is ignored, `systemMessage` JSON doesn't display, and exit 0 output is only visible in verbose mode. The "hook failed:" prefix is added by Claude Code automatically.
- **stdin handling in hooks**: Claude Code pipes JSON to hook stdin but may not close the pipe. Use `json.NewDecoder().Decode()` (reads one object) instead of `io.ReadAll()` (waits for EOF) to avoid hanging.
- **`os.TempDir()` on macOS**: returns `/var/folders/.../T/`, not `/tmp/`.
