import Foundation
import Combine

@MainActor
class DashboardViewModel: ObservableObject {
    @Published var repoStates: [RepoState] = []
    @Published var selectedRepoPath: String?
    @Published var isLoading = false
    @Published var selectedTab: DashboardTab = .all
    @Published var selectedTypeFilter: String? = nil
    @Published var recentDays: Int = 7
    @Published var baselineDays: Int = 14
    @Published var isLoadingComparison = false

    private(set) var settings: AppSettings
    private let reportRunner = ReportRunner()
    private let debtTracker: DebtTracker

    var currentRepo: RepoState? {
        repoStates.first { $0.repoPath == selectedRepoPath } ?? repoStates.first
    }

    var debtCount: Int {
        guard let summary = currentRepo?.report?.summary else { return 0 }
        return summary.dormant
    }

    var weekDelta: Int? {
        guard let repo = currentRepo else { return nil }
        return debtTracker.weekDelta(repoPath: repo.repoPath)
    }

    init(settings: AppSettings, debtTracker: DebtTracker) {
        self.settings = settings
        self.debtTracker = debtTracker
        syncRepoStates()
    }

    func refresh() async {
        isLoading = true
        defer { isLoading = false }

        await withTaskGroup(of: Void.self) { group in
            for state in repoStates {
                group.addTask { [settings, reportRunner, debtTracker] in
                    do {
                        let report = try await reportRunner.runReport(repoPath: state.repoPath, settings: settings)
                        let debt = report.summary.dormant
                        await MainActor.run {
                            state.report = report
                            state.error = nil
                            state.lastPoll = Date()
                        }
                        debtTracker.recordDebt(repoPath: state.repoPath, debt: debt)
                    } catch {
                        await MainActor.run {
                            state.error = error.localizedDescription
                            state.lastPoll = Date()
                        }
                    }
                }
            }
        }
    }

    func refreshComparison() async {
        guard let repo = currentRepo else { return }
        isLoadingComparison = true
        defer { isLoadingComparison = false }

        do {
            let comparison = try await reportRunner.runComparisonReport(
                repoPath: repo.repoPath,
                recentDays: recentDays,
                baselineDays: baselineDays,
                settings: settings
            )
            repo.comparisonReport = comparison
            repo.comparisonError = nil
        } catch {
            repo.comparisonError = error.localizedDescription
        }
    }

    func applySettings(_ newSettings: AppSettings) {
        settings = newSettings
        syncRepoStates()
        Task { await refresh() }
    }

    private func syncRepoStates() {
        let oldPaths = Set(repoStates.map { $0.repoPath })
        let existing = Dictionary(uniqueKeysWithValues: repoStates.map { ($0.repoPath, $0) })
        repoStates = settings.watchedRepoPaths.map { path in
            existing[path] ?? RepoState(repoPath: path)
        }
        let newPaths = settings.watchedRepoPaths.filter { !oldPaths.contains($0) }
        if let lastNew = newPaths.last {
            selectedRepoPath = lastNew
        } else if selectedRepoPath == nil || !settings.watchedRepoPaths.contains(selectedRepoPath!) {
            selectedRepoPath = settings.watchedRepoPaths.first
        }
    }
}
