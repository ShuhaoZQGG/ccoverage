import SwiftUI

struct DashboardView: View {
    @ObservedObject var viewModel: DashboardViewModel
    @State private var showSettings = false

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            // Header
            HStack {
                Text("ccoverage")
                    .font(.headline)
                if viewModel.debtCount > 0 {
                    Text("\(viewModel.debtCount)")
                        .font(.caption.bold())
                        .foregroundStyle(.white)
                        .padding(.horizontal, 6)
                        .padding(.vertical, 2)
                        .background(Capsule().fill(.red))
                }
                Spacer()
                if viewModel.isLoading {
                    ProgressView()
                        .controlSize(.small)
                }
            }

            RepoSelectorView(viewModel: viewModel)

            if let repo = viewModel.currentRepo {
                if let error = repo.error {
                    Label(error, systemImage: "exclamationmark.triangle")
                        .font(.caption)
                        .foregroundStyle(.red)
                } else if repo.report != nil {
                    // Tab picker
                    Picker("", selection: $viewModel.selectedTab) {
                        ForEach(DashboardTab.allCases, id: \.self) { tab in
                            Text(tab.rawValue).tag(tab)
                        }
                    }
                    .pickerStyle(.segmented)

                    TypeFilterPicker(selection: $viewModel.selectedTypeFilter)

                    // Tab content
                    tabContent(repo)

                    debtDeltaSection()
                } else {
                    Text("No data yet. Waiting for first scan...")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            } else {
                Text("No repos configured. Open Settings to add one.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            Divider()

            // Footer
            HStack {
                if let repo = viewModel.currentRepo {
                    Button("View full report") {
                        openTerminalReport(repoPath: repo.repoPath)
                    }
                    .font(.caption)
                }
                Spacer()
                Button {
                    showSettings = true
                } label: {
                    Image(systemName: "gear")
                }
                .buttonStyle(.plain)
                .sheet(isPresented: $showSettings) {
                    SettingsView(viewModel: viewModel)
                }
                Button("Quit") {
                    NSApplication.shared.terminate(nil)
                }
                .font(.caption)
            }
        }
        .padding()
    }

    @ViewBuilder
    private func tabContent(_ repo: RepoState) -> some View {
        switch viewModel.selectedTab {
        case .all:
            if let report = repo.report {
                AllItemsView(report: report, typeFilter: viewModel.selectedTypeFilter)
            }
        case .last:
            LastSessionView(lastSession: repo.report?.lastSession, typeFilter: viewModel.selectedTypeFilter)
        case .compare:
            CompareView(viewModel: viewModel)
                .onAppear {
                    if repo.comparisonReport == nil && repo.comparisonError == nil {
                        Task { await viewModel.refreshComparison() }
                    }
                }
        }
    }

    @ViewBuilder
    private func debtDeltaSection() -> some View {
        if let delta = viewModel.weekDelta {
            HStack {
                if delta > 0 {
                    Text("+\(delta) since last week")
                        .font(.caption)
                        .foregroundStyle(.red)
                } else if delta < 0 {
                    Text("\(delta) since last week")
                        .font(.caption)
                        .foregroundStyle(.green)
                } else {
                    Text("unchanged since last week")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }
        }
    }

    private func openTerminalReport(repoPath: String) {
        let escapedPath = repoPath.replacingOccurrences(of: "\"", with: "\\\"")
        let script = "tell application \"Terminal\" to do script \"ccoverage report --repo-path \\\"\(escapedPath)\\\"\""
        if let appleScript = NSAppleScript(source: script) {
            var error: NSDictionary?
            appleScript.executeAndReturnError(&error)
        }
    }
}
