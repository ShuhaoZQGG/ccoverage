package types

import "time"

type ConfigType string

const (
	ConfigClaudeMD ConfigType = "CLAUDE.md"
	ConfigSkill    ConfigType = "Skill"
	ConfigMCP      ConfigType = "MCP"
	ConfigHook     ConfigType = "Hook"
	ConfigCommand  ConfigType = "Command"
)

type ManifestItem struct {
	Type         ConfigType        `json:"type"`
	Name         string            `json:"name"`
	Path         string            `json:"path"`
	AbsPath      string            `json:"abs_path"`
	LastModified time.Time         `json:"last_modified"`
	Exists       bool              `json:"exists"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type Manifest struct {
	RepoPath  string         `json:"repo_path"`
	Items     []ManifestItem `json:"items"`
	ScannedAt time.Time      `json:"scanned_at"`
}

type UsageEvent struct {
	ConfigType ConfigType `json:"config_type"`
	Name       string     `json:"name"`
	SessionID  string     `json:"session_id"`
	Timestamp  time.Time  `json:"timestamp"`
	Cwd        string     `json:"cwd"`
}

type UsageSummary struct {
	TotalActivations int       `json:"total_activations"`
	UniqueSessions   int       `json:"unique_sessions"`
	FirstSeen        *time.Time `json:"first_seen,omitempty"`
	LastSeen         *time.Time `json:"last_seen,omitempty"`
}

type Status string

const (
	StatusActive    Status = "Active"
	StatusUnderused Status = "Underused"
	StatusDormant   Status = "Dormant"
	StatusOrphaned  Status = "Orphaned"
)

type CoverageResult struct {
	Item   ManifestItem `json:"item"`
	Usage  UsageSummary `json:"usage"`
	Status Status       `json:"status"`
}

type LastSessionItem struct {
	Type   ConfigType `json:"type"`
	Name   string     `json:"name"`
	Active bool       `json:"active"`
	Count  int        `json:"count"`
}

type LastSessionReport struct {
	SessionID string            `json:"session_id"`
	Timestamp time.Time         `json:"timestamp"`
	Items     []LastSessionItem `json:"items"`
}

type CoverageReport struct {
	RepoPath         string             `json:"repo_path"`
	LookbackDays     int                `json:"lookback_days"`
	SessionsAnalyzed int                `json:"sessions_analyzed"`
	Results          []CoverageResult   `json:"results"`
	Summary          ReportSummary      `json:"summary"`
	LastSession      *LastSessionReport `json:"last_session,omitempty"`
}

type ReportSummary struct {
	TotalItems int `json:"total_items"`
	Active     int `json:"active"`
	Underused  int `json:"underused"`
	Dormant    int `json:"dormant"`
	Orphaned   int `json:"orphaned"`
}
