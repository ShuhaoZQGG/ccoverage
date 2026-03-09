# Code Intelligence

Prefer Go LSP over Grep/Read for code navigation — it's faster, precise, and avoids reading entire files:
- `workspaceSymbol` to find where something is defined
- `findReferences` to see all usages across the codebase
- `goToDefinition` / `goToImplementation` to jump to source
- `hover` for type info without reading the file

Use Grep only when LSP isn't available or for text/pattern searches (comments, strings, config).

After writing or editing code, check LSP diagnostics and fix errors before proceeding.

# internal/ packages

## Data Flow

```
types -> scanner -> usage -> coverage -> output
```

All packages import `types`. Only `coverage` imports `usage`. No other cross-imports.

## Package Guide

### types (`types/types.go`)

All shared structs and enums. Changes here affect every other package.

- `ManifestItem` — single config item (type, name, paths, exists flag, metadata)
- `Manifest` — collection of items + repo path + scan timestamp
- `UsageEvent` — single observed activation (type, name, session ID, timestamp, cwd)
- `UsageSummary` — aggregated stats per item (activations, unique sessions, time range)
- `CoverageResult` — item + usage + status
- `CoverageReport` — full report with results array and summary counts

### scanner (`scanner/`)

`BuildManifest(repoPath)` runs 5 sub-scanners in sequence: `scanClaudeMD`, `scanSkills`, `scanMCP`, `scanHooks`, `scanCommands`. Each returns `[]ManifestItem`.

- `scanClaudeMD` walks the tree, skipping `.git`, `node_modules`, `.claude`
- `scanSkills` reads `.claude/skills/` directory
- `scanMCP` parses `.mcp.json` and `~/.claude.json` (project-scoped + global mcpServers)
- `scanHooks` parses `.claude/settings.json` hook definitions
- `scanCommands` reads `.claude/commands/` directory
- Missing files/dirs = nil slice, not error

### usage (`usage/`)

Three files, three responsibilities:

- **`locator.go`** — `LocateSessionFiles(repoPath, lookbackDays)` finds JSONL files under `~/.claude/projects/<encoded-path>/`. `encodeRepoPath` replaces `/` with `-` (no trimming, keeps leading dash). Also globs one level deep for subagent sessions.
- **`jsonl.go`** — `ParseSessionFile(path)` reads one JSONL file. Returns events, cwds, and touched dirs. Handles 3 line types:
  - `assistant`: extracts tool_use blocks (Skill, mcp__*, Agent) and file paths from Read/Edit/Write/Glob/Grep via `extractTouchedDirs`
  - `user`: extracts `<command-name>` tags for slash commands
  - `progress`: extracts hook_progress events
  - 4MB scanner buffer. Content field can be string or array — check first byte.
- **`matcher.go`** — `MatchUsage(manifest, sessionFiles)` orchestrates parsing and correlation. Returns `map["Type:Name"]*UsageSummary`. CLAUDE.md items use directory-containment matching: a CLAUDE.md is "used" if any session cwd **or touched file directory** is at or beneath its parent directory (`isDirOrDescendant`).

### coverage (`coverage/`)

- **`status.go`** — `Classify(item, usage, threshold)` is a pure function. Priority: Dormant (zero activations) > Underused (<= threshold) > Active.
- **`analyzer.go`** — `Analyze(manifest, sessionFiles, lookbackDays, threshold)` calls `usage.MatchUsage`, then `Classify` per item. Builds the final `CoverageReport`.

### output (`output/`)

Three renderers, same column structure (status, type, name, activations, sessions, last seen):

- `RenderText` — tabwriter-aligned, ANSI colors when stdout is a TTY (`isTTY()` checks `ModeCharDevice`)
- `RenderJSON` — `json.MarshalIndent`, only renderer that returns error
- `RenderMarkdown` — pipe-delimited table

## Test Patterns

- Table-driven tests with `t.Run` subtests
- `t.TempDir()` for filesystem isolation
- `testdata/sample_session.jsonl` fixture for JSONL parsing tests
- Tests live alongside their package (`*_test.go` files)
