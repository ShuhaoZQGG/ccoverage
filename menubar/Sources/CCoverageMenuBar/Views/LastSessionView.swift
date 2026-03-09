import SwiftUI

struct LastSessionView: View {
    let lastSession: LastSessionReport?
    var typeFilter: String? = nil

    private var filteredItems: [LastSessionItem] {
        guard let session = lastSession else { return [] }
        guard let filter = typeFilter else { return session.items }
        return session.items.filter { $0.type == filter }
    }

    var body: some View {
        if let session = lastSession {
            VStack(alignment: .leading, spacing: 4) {
                Text("Last Session")
                    .font(.subheadline.bold())
                Text(session.sessionID.prefix(12) + "...")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
                Text(session.timestamp, style: .relative)
                    .font(.caption2)
                    .foregroundStyle(.secondary)

                ScrollView {
                    VStack(alignment: .leading, spacing: 4) {
                        ForEach(filteredItems, id: \.name) { item in
                            HStack(spacing: 4) {
                                Image(systemName: item.active ? "checkmark.circle.fill" : "xmark.circle")
                                    .foregroundStyle(item.active ? .green : .red)
                                    .font(.caption2)
                                Text(item.name)
                                    .font(.caption)
                                    .lineLimit(1)
                                if item.count > 0 {
                                    Text("\(item.count)x")
                                        .font(.caption2)
                                        .foregroundStyle(.secondary)
                                }
                                Spacer()
                                Text(item.type)
                                    .font(.caption2)
                                    .foregroundStyle(.secondary)
                            }
                        }
                    }
                }
                .frame(maxHeight: 260)
            }
        } else {
            Text("No session data available.")
                .font(.caption)
                .foregroundStyle(.secondary)
        }
    }
}
