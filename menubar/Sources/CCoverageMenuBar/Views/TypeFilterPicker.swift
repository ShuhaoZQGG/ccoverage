import SwiftUI

struct TypeFilterPicker: View {
    @Binding var selection: String?

    private let types = ["CLAUDE.md", "Skill", "MCP", "Hook", "Command"]

    var body: some View {
        HStack(spacing: 4) {
            Text("Type:")
                .font(.caption)
                .foregroundStyle(.secondary)
            Picker("Type", selection: $selection) {
                Text("All").tag(String?.none)
                ForEach(types, id: \.self) { type in
                    Text(type).tag(Optional(type))
                }
            }
            .pickerStyle(.menu)
            .fixedSize()
        }
    }
}
