import SwiftUI

struct FullReportView: View {
    let report: CoverageReport
    @Environment(\.dismiss) private var dismiss
    @State private var searchText = ""
    @State private var typeFilter: String? = nil

    private var filteredResults: [CoverageResult] {
        var results = report.results
        if let filter = typeFilter {
            results = results.filter { $0.item.type == filter }
        }
        if !searchText.isEmpty {
            results = results.filter {
                $0.item.name.localizedCaseInsensitiveContains(searchText)
            }
        }
        return results
    }

    private var filteredSummary: ReportSummary {
        let results = filteredResults
        return ReportSummary(
            totalItems: results.count,
            active: results.filter { $0.status == "Active" }.count,
            underused: results.filter { $0.status == "Underused" }.count,
            dormant: results.filter { $0.status == "Dormant" }.count
        )
    }

    private var groupedResults: [(String, [CoverageResult])] {
        let order = ["Active", "Underused", "Dormant"]
        let grouped = Dictionary(grouping: filteredResults) { $0.status }
        return order.compactMap { status in
            guard let items = grouped[status], !items.isEmpty else { return nil }
            return (status, items)
        }
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            // Header
            VStack(alignment: .leading, spacing: 8) {
                HStack {
                    Text("Full Report")
                        .font(.title3.bold())
                    Spacer()
                    Button { dismiss() } label: {
                        Image(systemName: "xmark.circle.fill")
                            .foregroundStyle(.secondary)
                    }
                    .buttonStyle(.plain)
                    .keyboardShortcut(.escape, modifiers: [])
                }

                Text(report.repoPath)
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .lineLimit(1)
                    .truncationMode(.middle)

                HStack(spacing: 12) {
                    Text("\(report.sessionsAnalyzed) sessions")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    Text("\(report.lookbackDays)d lookback")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }

                HStack(spacing: 8) {
                    StatusBadge(label: "Active", count: filteredSummary.active, color: .green)
                    StatusBadge(label: "Underused", count: filteredSummary.underused, color: .yellow)
                    StatusBadge(label: "Dormant", count: filteredSummary.dormant, color: .red)
                }

                HStack(spacing: 8) {
                    TextField("Search items...", text: $searchText)
                        .textFieldStyle(.roundedBorder)
                        .font(.caption)
                    TypeFilterPicker(selection: $typeFilter)
                }
            }
            .padding()

            Divider()

            // Item list
            List {
                ForEach(groupedResults, id: \.0) { status, items in
                    Section {
                        ForEach(items, id: \.item.name) { result in
                            resultRow(result)
                        }
                    } header: {
                        Text("\(status) (\(items.count))")
                    }
                }
            }
            .listStyle(.inset)

            Divider()

            // Footer
            HStack {
                Button("Copy as Text") {
                    copyReportAsText()
                }
                .font(.caption)
                Spacer()
                Text("\(filteredResults.count) items")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            .padding()
        }
        .frame(minWidth: 500, minHeight: 400, idealHeight: 600)
    }

    @ViewBuilder
    private func resultRow(_ result: CoverageResult) -> some View {
        VStack(alignment: .leading, spacing: 2) {
            HStack(spacing: 4) {
                Circle()
                    .fill(colorForStatus(result.status))
                    .frame(width: 6, height: 6)
                Text(result.item.name)
                    .font(.caption)
                    .lineLimit(1)
                Spacer()
                Text(result.item.displayType)
                    .font(.caption2)
                    .foregroundStyle(.secondary)
                    .padding(.horizontal, 4)
                    .padding(.vertical, 1)
                    .background(RoundedRectangle(cornerRadius: 3).fill(Color.secondary.opacity(0.15)))
                Text("\(result.usage.totalActivations) activations")
                    .font(.caption2.monospacedDigit())
                    .foregroundStyle(.secondary)
            }
            HStack(spacing: 8) {
                Text(result.item.path)
                    .font(.caption2)
                    .foregroundStyle(.tertiary)
                    .lineLimit(1)
                    .truncationMode(.middle)
                Spacer()
                Text("\(result.usage.uniqueSessions) sessions")
                    .font(.caption2)
                    .foregroundStyle(.tertiary)
                if let firstSeen = result.usage.firstSeen, firstSeen > Date(timeIntervalSince1970: 946684800) {
                    Text("first: \(firstSeen, format: .dateTime.month(.abbreviated).day())")
                        .font(.caption2)
                        .foregroundStyle(.tertiary)
                }
                if let lastSeen = result.usage.lastSeen, lastSeen > Date(timeIntervalSince1970: 946684800) {
                    Text("last: \(lastSeen, style: .relative)")
                        .font(.caption2)
                        .foregroundStyle(.tertiary)
                }
            }
            if let metadata = result.item.metadata, !metadata.isEmpty {
                Text(metadata.map { "\($0.key)=\($0.value)" }.joined(separator: ", "))
                    .font(.caption2)
                    .foregroundStyle(.quaternary)
                    .lineLimit(1)
            }
        }
    }

    private func colorForStatus(_ status: String) -> Color {
        switch status {
        case "Active": return .green
        case "Underused": return .yellow
        case "Dormant": return .red
        default: return .gray
        }
    }

    private func copyReportAsText() {
        var lines: [String] = []
        lines.append("Coverage Report: \(report.repoPath)")
        lines.append("Sessions analyzed: \(report.sessionsAnalyzed) | Lookback: \(report.lookbackDays) days")
        lines.append("Active: \(report.summary.active) | Underused: \(report.summary.underused) | Dormant: \(report.summary.dormant)")
        lines.append("")

        let dateFormatter = DateFormatter()
        dateFormatter.dateStyle = .short
        dateFormatter.timeStyle = .short

        for (status, items) in groupedResults {
            lines.append("--- \(status) (\(items.count)) ---")
            for result in items {
                var line = "  \(result.item.name) [\(result.item.displayType)]"
                line += " - \(result.usage.totalActivations) activations, \(result.usage.uniqueSessions) sessions"
                if let lastSeen = result.usage.lastSeen, lastSeen > Date(timeIntervalSince1970: 946684800) {
                    line += ", last: \(dateFormatter.string(from: lastSeen))"
                }
                lines.append(line)
                lines.append("    path: \(result.item.path)")
            }
            lines.append("")
        }

        NSPasteboard.general.clearContents()
        NSPasteboard.general.setString(lines.joined(separator: "\n"), forType: .string)
    }
}
