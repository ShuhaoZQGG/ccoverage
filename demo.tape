Output demo.gif

Set FontSize 14
Set Width 1200
Set Height 800
Set Theme "Catppuccin Mocha"
Set TypingSpeed 50ms

Type "# ccoverage — configuration coverage for Claude Code"
Enter
Sleep 1.5s

# Scan: discover what's configured
Ctrl+L
Type "# [1/7] Scan: list all config items (CLAUDE.md, skills, MCP, plugins)"
Enter
Sleep 1s
Type "ccoverage scan --target . --format text"
Enter
Sleep 3s

# Report: full coverage analysis
Ctrl+L
Type "# [2/7] Report: analyze usage over the last 30 days"
Enter
Sleep 1s
Type "ccoverage report -t . -f text"
Enter
Sleep 3s

# Last session: what was used in the most recent session
Ctrl+L
Type "# [3/7] Last session: show what was active in the latest session"
Enter
Sleep 1s
Type "ccoverage report --target . --last-session"
Enter
Sleep 3s

# Filter by status
Ctrl+L
Type "# [4/7] Filter: show only dormant (unused) items"
Enter
Sleep 1s
Type "ccoverage report -s Dormant -f text"
Enter
Sleep 3s

# Init: install SessionEnd hook
Ctrl+L
Type "# [5/7] Init: install a SessionEnd hook to track coverage automatically"
Enter
Sleep 1s
Type "ccoverage init --target ."
Enter
Sleep 3s
Enter
Type "# After each Claude Code session ends, you'll see:"
Enter
Sleep 1s
Type "# hook failed: ccoverage: 16 items — 5 Active, 4 Underused, 7 Dormant"
Enter
Sleep 1.5s
Type '# Note: "hook failed:" is normal — Claude Code shows this prefix'
Enter
Type "# because the hook uses exit code 2 to display output to the user"
Enter
Sleep 3s

# CI gate
Ctrl+L
Type "# [6/7] CI gate: fail the build if unused config lingers in the repo"
Enter
Type "# Use in GitHub Actions / CI to enforce config hygiene across the team"
Enter
Sleep 1s
Type "ccoverage report -s Dormant -e"
Enter
Sleep 3s

# Help
Ctrl+L
Type "# [7/7] Help: see all commands and flags"
Enter
Sleep 1s
Type "ccoverage --help"
Enter
Sleep 3s
