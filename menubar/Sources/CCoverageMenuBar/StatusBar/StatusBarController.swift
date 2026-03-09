import AppKit
import Combine
import SwiftUI

final class StatusBarController {
    private let statusItem: NSStatusItem
    private let panel: NSPanel
    private let hostingController: NSHostingController<PopoverContentView>
    private var globalMonitor: Any?
    private var cancellable: AnyCancellable?
    private let panelWidth: CGFloat = 360

    init(viewModel: DashboardViewModel) {
        statusItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)

        let contentView = PopoverContentView(viewModel: viewModel)
        hostingController = NSHostingController(rootView: contentView)
        hostingController.sizingOptions = .intrinsicContentSize

        panel = NSPanel(
            contentRect: NSRect(x: 0, y: 0, width: 360, height: 100),
            styleMask: [.borderless, .nonactivatingPanel],
            backing: .buffered,
            defer: false
        )
        panel.level = .floating
        panel.isOpaque = false
        panel.backgroundColor = .clear
        panel.hasShadow = true
        panel.isMovableByWindowBackground = false
        panel.hidesOnDeactivate = false

        let visualEffect = NSVisualEffectView(frame: panel.contentView!.bounds)
        visualEffect.autoresizingMask = [.width, .height]
        visualEffect.material = .popover
        visualEffect.state = .active
        visualEffect.blendingMode = .behindWindow
        visualEffect.wantsLayer = true
        visualEffect.layer?.cornerRadius = 10
        visualEffect.layer?.masksToBounds = true

        panel.contentView?.addSubview(visualEffect)

        hostingController.view.frame = visualEffect.bounds
        hostingController.view.autoresizingMask = [.width, .height]
        hostingController.view.wantsLayer = true
        hostingController.view.layer?.backgroundColor = .clear
        visualEffect.addSubview(hostingController.view)

        if let button = statusItem.button {
            button.image = MenuBarIcon.logo()
            button.target = self
            button.action = #selector(togglePopover)
        }

        // Subscribe to viewModel changes for reactive panel resizing
        cancellable = viewModel.objectWillChange.sink { [weak self] _ in
            DispatchQueue.main.async {
                self?.updatePanelSize()
            }
        }
    }

    @objc private func togglePopover() {
        if panel.isVisible {
            hidePanel()
        } else {
            showPanel()
        }
    }

    private func showPanel() {
        guard let button = statusItem.button,
              let buttonWindow = button.window else { return }

        let buttonRect = buttonWindow.convertToScreen(button.convert(button.bounds, to: nil))

        hostingController.view.layoutSubtreeIfNeeded()
        let intrinsicHeight = hostingController.view.intrinsicContentSize.height
        let panelHeight = clampHeight(intrinsicHeight)

        panel.setContentSize(NSSize(width: panelWidth, height: panelHeight))

        let x = buttonRect.midX - panelWidth / 2
        let y = buttonRect.minY - panelHeight

        panel.setFrameOrigin(NSPoint(x: x, y: y))
        panel.orderFrontRegardless()

        globalMonitor = NSEvent.addGlobalMonitorForEvents(matching: [.leftMouseDown, .rightMouseDown]) { [weak self] event in
            if NSApp.windows.contains(where: { $0 is NSOpenPanel && $0.isVisible }) { return }
            self?.hidePanel()
        }
    }

    private func updatePanelSize() {
        guard panel.isVisible else { return }
        hostingController.view.layoutSubtreeIfNeeded()
        let intrinsicHeight = hostingController.view.intrinsicContentSize.height
        let newHeight = clampHeight(intrinsicHeight)
        let oldFrame = panel.frame
        // Keep top edge anchored (panel hangs below menu bar)
        let newY = oldFrame.maxY - newHeight
        panel.setFrame(NSRect(x: oldFrame.origin.x, y: newY, width: panelWidth, height: newHeight), display: true, animate: false)
    }

    private func clampHeight(_ height: CGFloat) -> CGFloat {
        let maxHeight = min(600, (NSScreen.main?.visibleFrame.height ?? 600) - 40)
        return min(max(height, 50), maxHeight)
    }

    private func hidePanel() {
        panel.orderOut(nil)
        if let monitor = globalMonitor {
            NSEvent.removeMonitor(monitor)
            globalMonitor = nil
        }
    }

    func updateBadge(debtCount: Int) {
        guard let button = statusItem.button else { return }
        button.image = MenuBarIcon.logo()
        if debtCount > 0 {
            button.title = " \(debtCount)"
        } else {
            button.title = ""
        }
    }
}
