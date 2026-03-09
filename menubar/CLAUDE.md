# CCoverageMenuBar

macOS menu bar app that displays ccoverage reports. Runs the `ccoverage` CLI periodically and on session events, shows results in a floating panel with tabs for all items, last session, and period comparison.

## Build / Run

```sh
swift build                    # debug build
swift build -c release         # release build

# Run from build output:
.build/arm64-apple-macosx/debug/CCoverageMenuBar

# Requires ccoverage binary in PATH or configured in Settings
```

SPM only, no Xcode project. Single executable target `CCoverageMenuBar`. Minimum macOS 14. No external dependencies (Foundation + AppKit + SwiftUI + Combine only).

## Architecture

MVVM with AppKit/SwiftUI hybrid and actor-based concurrency.

```
AppDelegate (lifecycle, service setup)
  ├─ StatusBarController (NSPanel + NSHostingController bridge)
  │   └─ DashboardView (SwiftUI)
  │       ├─ AllItemsView      (tab: all items by status)
  │       ├─ LastSessionView   (tab: single session snapshot)
  │       └─ CompareView       (tab: two-period diff)
  ├─ DashboardViewModel (@MainActor, ObservableObject — main orchestrator)
  │   ├─ ReportRunner (actor — shells out to ccoverage CLI)
  │   ├─ DebtTracker (UserDefaults persistence of dormancy history)
  │   └─ [RepoState] (per-repo @Published state)
  ├─ PollScheduler (Timer-based periodic refresh)
  └─ SessionWatcher (FSEventStream on ~/.claude/projects/)
```

## Data Flow

```
Refresh trigger (poll / FSEvent / manual / startup)
  → DashboardViewModel.refresh()
    → ReportRunner.runReport()
      → ShellExecutor.run("ccoverage", ["report", "--format", "json", "--last-session"])
        → Process (async)
      → JSONDecoder (ISO8601 + fractional seconds)
    → RepoState.report = CoverageReport  (triggers UI update)
    → DebtTracker.recordDebt()
```

Refresh triggers: PollScheduler (configurable interval, default 60s), SessionWatcher (FSEventStream), manual (CompareView refresh button), startup (AppDelegate).

## Package Structure

```
Sources/CCoverageMenuBar/
  App/          AppMain.swift, AppDelegate.swift
  Models/       CoverageReport, RepoState, AppSettings
  Services/     ReportRunner, PollScheduler, SessionWatcher, DebtTracker
  StatusBar/    StatusBarController, MenuBarIcon, PopoverContentView
  Views/        DashboardView, DashboardViewModel, AllItemsView,
                LastSessionView, CompareView, SettingsView,
                RepoSelectorView, StatusBadge, TypeFilterPicker, DashboardTab
  Utilities/    ShellExecutor, RepoDetector
```

## Key Conventions

- **Naming**: PascalCase types, camelCase members. Service suffixes: `*Runner`, `*Scheduler`, `*Watcher`, `*Tracker`. Views use `*View` suffix.
- **Concurrency**: `@MainActor` on all `ObservableObject` classes (DashboardViewModel, RepoState). `actor` for ReportRunner. `async`/`await` throughout, `withTaskGroup` for parallel repo refresh.
- **Error handling**: Custom error enums conforming to `LocalizedError` (ReportRunnerError, ShellError). Errors stored in `RepoState.error` and displayed inline in views.
- **Graceful degradation**: Missing binary shows "Set the path in Settings". Empty output explains "repo may have no Claude Code configuration". Decode errors include first 200 chars of output.
- **JSON coding**: snake_case keys for CLI interop. Custom date decoding handles ISO8601 with fractional seconds.
- **Persistence**: Settings in `~/.ccoverage/menubar.json` (Codable). Debt history in UserDefaults (key: `ccoverage_debt_history_<path>`, 90-day window). Report data is transient (in-memory).
- **Menubar app**: `.accessory` activation policy (no dock icon). Global event monitor for click-away dismissal.

## Gotchas

- **NSPanel configuration**: Borderless + nonactivatingPanel + floating level. `hidesOnDeactivate = false` is required or the panel disappears when app loses focus. NSVisualEffectView provides the blurred popover background with 10pt corner radius.
- **Panel positioning**: Anchored below the status bar button. Max height is min(600pt, 90% screen height). Global click monitor dismisses the panel but skips clicks when NSOpenPanel is visible (settings repo picker).
- **@MainActor is mandatory**: Both DashboardViewModel and RepoState are `@MainActor`. All `@Published` property writes must happen on main thread — use `await MainActor.run { }` when updating from actor contexts.
- **FSEventStream callback bridging**: SessionWatcher's C callback dispatches to `DispatchQueue.main.async` to bridge into Swift concurrency.
- **CLI integration**: ReportRunner invokes `ccoverage report --format json --last-session`. The binary path comes from AppSettings (user-configurable) or defaults to PATH lookup.
- **Panel resize on state change**: A Combine sink on `viewModel.objectWillChange` calls `updatePanelSize()` via `DispatchQueue.main.async` to keep the panel height in sync with content.
- **RepoDetector path decoding**: Reverses the Go-side encoding (`-Users-foo-project` → `/Users/foo/project`), same logic as `usage/locator.go`.

## Code Intelligence

Prefer Swift LSP over Grep/Read for code navigation:
- `workspaceSymbol` to find type/function definitions
- `findReferences` to see all usages
- `goToDefinition` to jump to source
- `hover` for type info without reading the file

This project has `swift-lsp` plugin installed and enabled.

Use Grep only for text/pattern searches (comments, strings, config files).

After writing or editing code, check LSP diagnostics and fix errors before proceeding.
