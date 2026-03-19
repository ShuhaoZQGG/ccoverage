# ccoverage

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![CI](https://github.com/ShuhaoZQGG/ccoverage/actions/workflows/ci.yml/badge.svg)](https://github.com/ShuhaoZQGG/ccoverage/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/ShuhaoZQGG/ccoverage)](https://goreportcard.com/report/github.com/ShuhaoZQGG/ccoverage)

**Know what's actually working in your Claude Code setup — and what's dead weight.**

![ccoverage demo](demo.gif)

## The Problem

Claude Code configs grow organically. You add skills, MCP servers, hooks, and CLAUDE.md files as you go. Over time, some become outdated, redundant, or forgotten. Nobody knows what's actually used — until something breaks or your context window fills up with irrelevant instructions.

## What ccoverage Does

ccoverage scans your repo's Claude Code configuration and joins it against real session history to classify each item:

- **Active** — used recently and frequently
- **Underused** — seen in sessions, but below your usage threshold
- **Dormant** — configured but never appeared in any session

This is session-backed analysis, not just static file detection. It checks what you *actually use*.

## Quick Start

### Homebrew (macOS/Linux)

```sh
brew install ShuhaoZQGG/tap/ccoverage
```

### Go

```sh
go install github.com/ShuhaoZQGG/ccoverage@latest
```

### Download from Releases

Grab a prebuilt binary from [GitHub Releases](https://github.com/ShuhaoZQGG/ccoverage/releases) for Linux, macOS, or Windows (amd64/arm64).

## Usage

### `scan` — See what's configured

```sh
ccoverage scan --target . --format text
```

Lists all detected configuration items without checking session data.

### `report` — Full coverage analysis

```sh
ccoverage report --target . --lookback-days 30 --format json
```

Scans config, matches against session history, and classifies each item.

Filter by status or config type:

```sh
ccoverage report --status Dormant
ccoverage report --type MCP --format json
ccoverage report --status Dormant,Underused --format md
```

### `init` — Install the SessionEnd hook

```sh
ccoverage init --target ~/Project/MyRepo
```

Adds a `SessionEnd` hook to the repo's `.claude/settings.json` so you get a one-line coverage summary after every Claude Code session.

### `summary` — One-line summary (hook use)

```sh
ccoverage summary --target .
```

Outputs a compact summary line. Designed to run as a `SessionEnd` hook.

## Config Types Detected

| Type | What it covers |
|------|---------------|
| CLAUDE.md | All CLAUDE.md files (root, nested, `.claude/` directory) |
| Skill | Skill definitions in project settings |
| MCP | MCP server configurations |
| Hook | Lifecycle hooks (PreToolUse, PostToolUse, etc.) |
| Command | Custom slash commands |
| Plugin | Plugin configurations with component discovery |

## How It Works

1. **Scan** — Reads `.claude/settings.json`, CLAUDE.md files, and project config to build a manifest of all configuration items
2. **Match** — Locates session JSONL files and parses tool calls, content blocks, and file paths to find evidence of each config item being used
3. **Classify** — Compares usage counts against the threshold (default: 2) to assign Active, Underused, or Dormant status

## CI Integration

Use `--error-on-match` to fail your build when dormant config is detected:

```yaml
# .github/workflows/config-hygiene.yml
- name: Check for dormant config
  run: |
    go install github.com/ShuhaoZQGG/ccoverage@latest
    ccoverage report --status Dormant --error-on-match
```

Exit code 1 means matches were found. Clean up your config or adjust the filter.

## SessionEnd Hook

Run `ccoverage init` on your repo to automatically get a coverage summary after every Claude Code session:

```sh
ccoverage init --target .
```

This installs a hook that runs `ccoverage summary` at session end, showing something like:

```
ccoverage: 12 items — 8 Active, 2 Underused, 2 Dormant
```

## Output Formats

**Text** (default) — human-readable table for terminal use

```sh
ccoverage report --format text
```

**JSON** — machine-readable for scripting and CI

```sh
ccoverage report --format json
```

**Markdown** — for documentation or PR comments

```sh
ccoverage report --format md
```

## Menubar App

ccoverage includes a companion macOS menubar app that displays your latest coverage summary at a glance. It runs as a menubar-only app — it won't appear in the Dock, just in the menu bar.

### Install via DMG (recommended)

**Requirements:** macOS 14 (Sonoma) or later, and the `ccoverage` CLI already in your PATH.

1. Install the CLI first (see [Quick Start](#quick-start) above)
2. Download **CCoverage.dmg** from the [latest release](https://github.com/ShuhaoZQGG/ccoverage/releases/latest)
3. Open the DMG and drag **CCoverage** to Applications
4. Launch CCoverage from Applications

The app is signed and notarized by Apple — no Gatekeeper warnings.

### Build from Source

**Requirements:** macOS 14 (Sonoma) or later, Swift 5.9+, and the `ccoverage` CLI already in your PATH.

```sh
cd menubar
swift build                # debug build
# swift build -c release   # optimized build

# run the app
.build/arm64-apple-macosx/debug/CCoverageMenuBar
```

## License

[MIT](LICENSE)
