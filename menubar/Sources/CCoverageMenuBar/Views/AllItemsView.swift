import SwiftUI

struct AllItemsView: View {
    let report: CoverageReport
    var typeFilter: String? = nil

    private var filteredResults: [CoverageResult] {
        guard let filter = typeFilter else { return report.results }
        return report.results.filter { $0.item.type == filter }
    }

    private var filteredSummary: ReportSummary {
        guard typeFilter != nil else { return report.summary }
        let results = filteredResults
        return ReportSummary(
            totalItems: results.count,
            active: results.filter { $0.status == "Active" }.count,
            underused: results.filter { $0.status == "Underused" }.count,
            dormant: results.filter { $0.status == "Dormant" }.count
        )
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            summarySection(filteredSummary)

            Text("\(report.sessionsAnalyzed) sessions analyzed")
                .font(.caption2)
                .foregroundStyle(.secondary)

            ScrollView {
                VStack(alignment: .leading, spacing: 2) {
                    ForEach(groupedResults, id: \.0) { status, items in
                        Text(status)
                            .font(.caption.bold())
                            .padding(.top, 4)
                        ForEach(items, id: \.item.name) { result in
                            resultRow(result)
                        }
                    }
                }
            }
            .frame(maxHeight: 260)
        }
    }

    private var groupedResults: [(String, [CoverageResult])] {
        let order = ["Active", "Underused", "Dormant"]
        let grouped = Dictionary(grouping: filteredResults) { $0.status }
        return order.compactMap { status in
            guard let items = grouped[status], !items.isEmpty else { return nil }
            return (status, items)
        }
    }

    @ViewBuilder
    private func summarySection(_ summary: ReportSummary) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("Summary (\(summary.totalItems) items)")
                .font(.subheadline.bold())
            HStack(spacing: 8) {
                StatusBadge(label: "Active", count: summary.active, color: .green)
                StatusBadge(label: "Underused", count: summary.underused, color: .yellow)
                StatusBadge(label: "Dormant", count: summary.dormant, color: .red)
            }
        }
    }

    @ViewBuilder
    private func resultRow(_ result: CoverageResult) -> some View {
        HStack(spacing: 4) {
            statusDot(result.status)
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
            Text("\(result.usage.totalActivations)")
                .font(.caption2.monospacedDigit())
                .foregroundStyle(.secondary)
            if let lastSeen = result.usage.lastSeen,
               lastSeen > Date(timeIntervalSince1970: 946684800) {
                Text(lastSeen, style: .relative)
                    .font(.caption2)
                    .foregroundStyle(.secondary)
                    .frame(width: 60, alignment: .trailing)
            }
        }
    }

    private func statusDot(_ status: String) -> some View {
        Circle()
            .fill(colorForStatus(status))
            .frame(width: 6, height: 6)
    }

    private func colorForStatus(_ status: String) -> Color {
        switch status {
        case "Active": return .green
        case "Underused": return .yellow
        case "Dormant": return .red
        default: return .gray
        }
    }
}
