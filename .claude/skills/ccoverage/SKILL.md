---
name: ccoverage
description: >
  Analyze Claude Code project configuration coverage using the ccoverage CLI.
  Scan repos for CLAUDE.md files, skills, MCP servers, hooks, commands, and
  plugins, then classify each as Active, Underused, or Dormant based on session
  history. Use when Claude needs to (1) audit a project's Claude Code
  configuration health, (2) find unused or underused config items, (3) set up
  automated coverage tracking with SessionEnd hooks, (4) create CI gates that
  fail on dormant configuration, (5) generate coverage reports in
  text/JSON/markdown, or when the user mentions "ccoverage", "config coverage",
  "dormant skills", "unused MCP servers", or "Claude Code audit".
---

# ccoverage

Analyze Claude Code project configuration coverage. Scan a repo's config items and classify usage as Active, Underused, or Dormant based on session history.

## Prerequisites

Ensure `ccoverage` is on PATH. Build from source if needed:

```sh
CGO_ENABLED=0 go build -o ccoverage .  # CGO_ENABLED=0 required on macOS
```

Or: `go install github.com/shuhaozhang/ccoverage@latest`

## Workflow

1. **Quick audit** (manifest only, no session data):
   ```sh
   ccoverage scan --repo-path /path/to/repo
   ```

2. **Full coverage report** (correlates with session history):
   ```sh
   ccoverage report --repo-path /path/to/repo --lookback-days 30 --format text
   ```

3. **Filter to actionable items**:
   ```sh
   ccoverage report --status Dormant,Underused
   ccoverage report --type MCP,Skill --status Dormant
   ```

4. **Set up automatic tracking** (installs SessionEnd hook):
   ```sh
   ccoverage init --repo-path /path/to/repo
   ```

5. **CI gate** (fail if dormant items exist):
   ```sh
   ccoverage report --status Dormant --error-on-match
   ```

## Interpreting Results

Each config item gets a status:
- **Active** (> threshold activations): Healthy, in regular use
- **Underused** (1 to threshold): Used but rarely; consider if still needed
- **Dormant** (0 activations): Never used in the lookback window; candidate for removal or investigation

Default threshold is 2; adjust with `--threshold N`.

## Acting on Results

- **Dormant CLAUDE.md**: May be scoped to a subdirectory no one works in, or instructions may be stale. Review and remove or consolidate.
- **Dormant Skills**: Skill may not trigger correctly (check description in frontmatter) or is no longer needed.
- **Dormant MCP servers**: Server may have been replaced or is misconfigured. Check `.mcp.json`.
- **Dormant Hooks**: Hook event type may have changed or hook is no longer relevant.
- **Dormant Commands**: Command may have been superseded. Check `.claude/commands/`.
- **Underused items**: May indicate the item works but isn't well-known to the team. Consider adding to onboarding docs.

## Reference

For detailed flag documentation, output format examples, config types, and gotchas, see [references/api_reference.md](references/api_reference.md).
