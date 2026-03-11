Output demo.gif

Set FontSize 14
Set Width 1200
Set Height 800
Set Theme "Catppuccin Mocha"

Type "# ccoverage — configuration coverage for Claude Code"
Enter
Sleep 1s

Type "ccoverage scan --repo-path . --format text"
Enter
Sleep 3s

Type "ccoverage report --repo-path . --format text"
Enter
Sleep 3s

Type "ccoverage report --status Dormant --format text"
Enter
Sleep 3s

Type "ccoverage report --status Dormant --error-on-match"
Enter
Sleep 3s
