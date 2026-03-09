import AppKit
import Combine

final class AppDelegate: NSObject, NSApplicationDelegate {
    private var statusBarController: StatusBarController!
    private var pollScheduler: PollScheduler!
    private var sessionWatcher: SessionWatcher!
    private var settings: AppSettings!
    private var viewModel: DashboardViewModel!
    private var debtTracker: DebtTracker!

    func applicationDidFinishLaunching(_ notification: Notification) {
        settings = AppSettings.load()
        debtTracker = DebtTracker()
        viewModel = DashboardViewModel(settings: settings, debtTracker: debtTracker)
        statusBarController = StatusBarController(viewModel: viewModel)

        pollScheduler = PollScheduler(intervalSeconds: settings.pollIntervalSeconds) { [weak self] in
            self?.refreshAll()
        }

        sessionWatcher = SessionWatcher { [weak self] in
            self?.refreshAll()
        }

        refreshAll()
    }

    private func refreshAll() {
        Task {
            await viewModel.refresh()
            await MainActor.run {
                statusBarController.updateBadge(debtCount: viewModel.debtCount)
            }
        }
    }
}
