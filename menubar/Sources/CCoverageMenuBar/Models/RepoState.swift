import Foundation

@MainActor
class RepoState: ObservableObject, Identifiable {
    let repoPath: String
    @Published var report: CoverageReport?
    @Published var error: String?
    @Published var lastPoll: Date?
    @Published var comparisonReport: ComparisonReport?
    @Published var comparisonError: String?

    nonisolated var id: String { repoPath }

    init(repoPath: String) {
        self.repoPath = repoPath
    }
}
