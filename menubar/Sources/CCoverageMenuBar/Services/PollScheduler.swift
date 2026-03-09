import Foundation

final class PollScheduler {
    private var timer: Timer?
    private let onTick: () -> Void

    init(intervalSeconds: Int, onTick: @escaping () -> Void) {
        self.onTick = onTick
        scheduleTimer(interval: TimeInterval(intervalSeconds))
    }

    func updateInterval(seconds: Int) {
        timer?.invalidate()
        scheduleTimer(interval: TimeInterval(seconds))
    }

    private func scheduleTimer(interval: TimeInterval) {
        timer = Timer.scheduledTimer(withTimeInterval: interval, repeats: true) { [weak self] _ in
            self?.onTick()
        }
    }

    deinit {
        timer?.invalidate()
    }
}
