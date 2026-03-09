import SwiftUI

struct CompareView: View {
    @ObservedObject var viewModel: DashboardViewModel

    private let recentOptions = [3, 7, 14, 30]
    private let baselineOptions = [7, 14, 30, 60]

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            // Period pickers
            HStack {
                Text("Recent:")
                    .font(.caption)
                Picker("", selection: $viewModel.recentDays) {
                    ForEach(recentOptions, id: \.self) { d in
                        Text("\(d)d").tag(d)
                    }
                }
                .pickerStyle(.menu)
                .frame(width: 60)

                Text("vs Baseline:")
                    .font(.caption)
                Picker("", selection: $viewModel.baselineDays) {
                    ForEach(baselineOptions, id: \.self) { d in
                        Text("\(d)d").tag(d)
                    }
                }
                .pickerStyle(.menu)
                .frame(width: 60)

                Spacer()

                Button {
                    Task { await viewModel.refreshComparison() }
                } label: {
                    Image(systemName: "arrow.triangle.2.circlepath")
                        .font(.caption)
                }
                .buttonStyle(.plain)
                .disabled(viewModel.isLoadingComparison)
            }

            if viewModel.isLoadingComparison {
                HStack {
                    Spacer()
                    ProgressView("Comparing...")
                        .controlSize(.small)
                    Spacer()
                }
            } else if let error = viewModel.currentRepo?.comparisonError {
                Label(error, systemImage: "exclamationmark.triangle")
                    .font(.caption)
                    .foregroundStyle(.red)
            } else if let comparison = viewModel.currentRepo?.comparisonReport {
                comparisonContent(comparison)
            } else {
                Text("Tap refresh to compare periods.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
        .onChange(of: viewModel.recentDays) {
            Task { await viewModel.refreshComparison() }
        }
        .onChange(of: viewModel.baselineDays) {
            Task { await viewModel.refreshComparison() }
        }
        .onChange(of: viewModel.selectedRepoPath) {
            Task { await viewModel.refreshComparison() }
        }
    }

    @ViewBuilder
    private func comparisonContent(_ comparison: ComparisonReport) -> some View {
        // Summary deltas
        VStack(alignment: .leading, spacing: 2) {
            Text("\(comparison.recentDays)d vs \(comparison.baselineDays)d")
                .font(.caption.bold())
            HStack(spacing: 10) {
                deltaLabel("Active", delta: comparison.summaryDelta.active)
                deltaLabel("Dormant", delta: comparison.summaryDelta.dormant)
            }
        }

        let typeFiltered = viewModel.selectedTypeFilter.map { f in comparison.items.filter { $0.type == f } } ?? comparison.items
        let changedItems = typeFiltered.filter { $0.isNew || $0.isDropped || $0.statusChanged }

        if changedItems.isEmpty {
            Text("No changes between periods.")
                .font(.caption)
                .foregroundStyle(.secondary)
        } else {
            ScrollView {
                VStack(alignment: .leading, spacing: 2) {
                    ForEach(changedItems) { item in
                        comparisonRow(item)
                    }
                }
            }
            .frame(maxHeight: 200)
        }
    }

    @ViewBuilder
    private func deltaLabel(_ label: String, delta: (old: Int, new: Int)) -> some View {
        let diff = delta.new - delta.old
        HStack(spacing: 2) {
            Text(label)
                .font(.caption2)
                .foregroundStyle(.secondary)
            Text("\(delta.old)")
                .font(.caption2.monospacedDigit())
            Image(systemName: "arrow.right")
                .font(.system(size: 7))
                .foregroundStyle(.secondary)
            Text("\(delta.new)")
                .font(.caption2.monospacedDigit())
            if diff != 0 {
                Text(diff > 0 ? "+\(diff)" : "\(diff)")
                    .font(.caption2.monospacedDigit().bold())
                    .foregroundStyle(diff > 0 ? (label == "Active" ? .green : .red) : (label == "Active" ? .red : .green))
            }
        }
    }

    @ViewBuilder
    private func comparisonRow(_ item: ComparisonItem) -> some View {
        HStack(spacing: 4) {
            if item.isNew {
                Text("NEW")
                    .font(.system(size: 8, weight: .bold))
                    .foregroundStyle(.white)
                    .padding(.horizontal, 4)
                    .padding(.vertical, 1)
                    .background(RoundedRectangle(cornerRadius: 3).fill(.green))
            } else if item.isDropped {
                Text("DROPPED")
                    .font(.system(size: 8, weight: .bold))
                    .foregroundStyle(.white)
                    .padding(.horizontal, 4)
                    .padding(.vertical, 1)
                    .background(RoundedRectangle(cornerRadius: 3).fill(.red))
            } else if item.statusChanged {
                HStack(spacing: 1) {
                    Text(item.baselineStatus ?? "?")
                        .font(.system(size: 8))
                    Image(systemName: "arrow.right")
                        .font(.system(size: 6))
                    Text(item.recentStatus ?? "?")
                        .font(.system(size: 8))
                }
                .foregroundStyle(.orange)
            }

            Text(item.name)
                .font(.caption)
                .lineLimit(1)
            Spacer()
            Text(item.type)
                .font(.caption2)
                .foregroundStyle(.secondary)
        }
    }
}
