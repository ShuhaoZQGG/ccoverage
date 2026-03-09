import SwiftUI

struct RepoSelectorView: View {
    @ObservedObject var viewModel: DashboardViewModel

    var body: some View {
        if viewModel.repoStates.count > 1 {
            Picker("Repo", selection: $viewModel.selectedRepoPath) {
                ForEach(viewModel.repoStates) { state in
                    Text(shortenPath(state.repoPath))
                        .tag(Optional(state.repoPath))
                }
            }
            .pickerStyle(.menu)
            .labelsHidden()
        }
    }

    private func shortenPath(_ path: String) -> String {
        let components = path.split(separator: "/")
        if components.count >= 2 {
            return String(components.suffix(2).joined(separator: "/"))
        }
        return path
    }
}
