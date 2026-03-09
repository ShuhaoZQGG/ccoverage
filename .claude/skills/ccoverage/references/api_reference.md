# ccoverage CLI Reference

## Commands

### scan
Discover all Claude Code config items in a repo (no session analysis).

```sh
ccoverage scan --repo-path /path/to/repo --format text
```

All items show as "Dormant" since no session data is analyzed.

### report
Full pipeline: scan -> locate sessions -> parse -> classify -> filter -> render.

```sh
ccoverage report --repo-path . --lookback-days 30 --threshold 2 --format text
ccoverage report --status Dormant --format json
ccoverage report --type MCP,Skill --format md
ccoverage report --status Dormant --error-on-match   # CI gate: exit 1 if matches
ccoverage report --last-session                       # per-item hit/miss for latest session
```

**Flags:**
| Flag | Default | Description |
|------|---------|-------------|
| `--repo-path` | `.` | Repository to analyze |
| `--lookback-days` | `30` | How far back to search session history |
| `--threshold` | `2` | Activations <= N = Underused; > N = Active |
| `--format` | `text` | Output: `text`, `json`, `md` |
| `--status` | (all) | Comma-separated filter: Active, Underused, Dormant |
| `--type` | (all) | Comma-separated filter: CLAUDE.md, Skill, MCP, Hook, Command, Plugin |
| `--error-on-match` | false | Exit 1 if any results remain after filtering |
| `--last-session` | false | Include per-item hit/miss for most recent session |

Filters are case-insensitive. Multiple filters are ANDed.

### summary
One-line summary for SessionEnd hooks.

```sh
ccoverage summary --repo-path /path/to/repo
```

Output examples:
- `ccoverage: 67% active | 2 items need attention | run "ccoverage report" for details`
- `ccoverage: 100% active | all config items healthy`

Writes to stderr, exits with code 2. This is the only way to display messages from SessionEnd hooks.

### init
Install a SessionEnd hook into `.claude/settings.json`.

```sh
ccoverage init --repo-path /path/to/repo
```

Creates/updates `.claude/settings.json` with a SessionEnd hook that runs `ccoverage summary`. Removes legacy hooks from older event types.

## Output Formats

### text (default)
```
STATUS      TYPE      NAME          ACTIVATIONS  SESSIONS  % SESSIONS  LAST SEEN
Active      Skill     db-migration  5            3         100.0%      2025-03-08
Underused   MCP       supabase      2            1         33.3%       2025-03-07
Dormant     CLAUDE.md CLAUDE.md     0            0         -           -

Total: 3 | Active: 1 | Underused: 1 | Dormant: 1
```

### json
```json
{
  "repo_path": "/path/to/repo",
  "lookback_days": 30,
  "sessions_analyzed": 3,
  "results": [
    {
      "item": { "type": "Skill", "name": "db-migration", "path": "..." },
      "usage": { "total_activations": 5, "unique_sessions": 3 },
      "status": "Active"
    }
  ],
  "summary": { "total_items": 3, "active": 1, "underused": 1, "dormant": 1 }
}
```

### md
Pipe-delimited markdown table suitable for PRs and docs.

## Status Classification

| Status | Condition |
|--------|-----------|
| **Dormant** | 0 activations |
| **Underused** | 1 to threshold activations |
| **Active** | > threshold activations |

## Config Types Scanned

| Type | Source |
|------|--------|
| CLAUDE.md | `CLAUDE.md` files at any directory level |
| Skill | `.claude/skills/` directory |
| MCP | `.mcp.json` + `~/.claude.json` |
| Hook | `.claude/settings.json` hooks |
| Command | `.claude/commands/` directory |
| Plugin | `.claude/plugins/` directory |

## Installation

```sh
# From source (CGO_ENABLED=0 required on macOS)
CGO_ENABLED=0 go build -o ccoverage .

# Or go install
go install github.com/shuhaozhang/ccoverage@latest
```

## Gotchas

- **macOS build**: Must use `CGO_ENABLED=0` (Go 1.21 dyld issue)
- **Session data location**: `~/.claude/projects/<encoded-path>/` where path encoding replaces `/` with `-` keeping the leading dash
- **4MB line buffer**: Session JSONL lines exceeding 4MB are silently dropped
- **Hook stdin**: Claude Code pipes JSON but may not close pipe; ccoverage uses decode-one-object pattern with 500ms timeout
