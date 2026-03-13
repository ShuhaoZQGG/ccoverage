import SwiftUI

struct SettingsView: View {
    @ObservedObject var viewModel: DashboardViewModel
    @State private var editingSettings: AppSettings
    @Environment(\.dismiss) private var dismiss

    init(viewModel: DashboardViewModel) {
        self.viewModel = viewModel
        self._editingSettings = State(initialValue: viewModel.settings)
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("Settings")
                .font(.headline)

            GroupBox("Watched Repos") {
                VStack(alignment: .leading, spacing: 4) {
                    ForEach(editingSettings.watchedRepoPaths, id: \.self) { path in
                        HStack {
                            Text(path)
                                .font(.caption)
                                .lineLimit(1)
                                .truncationMode(.middle)
                            Spacer()
                            Button(role: .destructive) {
                                editingSettings.watchedRepoPaths.removeAll { $0 == path }
                            } label: {
                                Image(systemName: "minus.circle")
                            }
                            .buttonStyle(.plain)
                        }
                    }
                    Button("Add Repo...") {
                        NSApp.activate(ignoringOtherApps: true)
                        let panel = NSOpenPanel()
                        panel.canChooseDirectories = true
                        panel.canChooseFiles = false
                        panel.allowsMultipleSelection = false
                        panel.begin { response in
                            if response == .OK, let url = panel.url {
                                DispatchQueue.main.async {
                                    editingSettings.watchedRepoPaths.append(url.path)
                                }
                            }
                        }
                    }
                    .font(.caption)
                }
            }

            DisclosureGroup("Advanced") {
                VStack(alignment: .leading, spacing: 4) {
                    Text("Binary Path")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    TextField("Auto-detect", text: Binding(
                        get: { editingSettings.ccoverageBinaryPath ?? "" },
                        set: { editingSettings.ccoverageBinaryPath = $0.isEmpty ? nil : $0 }
                    ))
                    .font(.caption)
                    .textFieldStyle(.roundedBorder)
                    Text("Leave blank for auto-detection.")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }
            }
            .font(.caption)

            GroupBox("Poll Interval") {
                HStack {
                    Slider(value: Binding(
                        get: { Double(editingSettings.pollIntervalSeconds) },
                        set: { editingSettings.pollIntervalSeconds = Int($0) }
                    ), in: 10...300, step: 10)
                    Text("\(editingSettings.pollIntervalSeconds)s")
                        .font(.caption)
                        .frame(width: 40)
                }
            }

            HStack {
                Spacer()
                Button("Cancel") { dismiss() }
                Button("Save") {
                    viewModel.applySettings(editingSettings)
                    editingSettings.save()
                    dismiss()
                }
                .buttonStyle(.borderedProminent)
            }
        }
        .padding()
        .frame(width: 340)
    }
}
