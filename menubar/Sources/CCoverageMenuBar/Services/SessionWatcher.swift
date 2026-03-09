import Foundation
import CoreServices

final class SessionWatcher {
    private var stream: FSEventStreamRef?
    private let onEvent: () -> Void

    init(onEvent: @escaping () -> Void) {
        self.onEvent = onEvent
        startWatching()
    }

    private func startWatching() {
        let home = FileManager.default.homeDirectoryForCurrentUser
        let watchPath = home.appendingPathComponent(".claude/projects").path as CFString
        let pathsToWatch = [watchPath] as CFArray

        var context = FSEventStreamContext()
        context.info = Unmanaged.passUnretained(self).toOpaque()

        let callback: FSEventStreamCallback = { _, info, _, _, _, _ in
            guard let info = info else { return }
            let watcher = Unmanaged<SessionWatcher>.fromOpaque(info).takeUnretainedValue()
            DispatchQueue.main.async {
                watcher.onEvent()
            }
        }

        stream = FSEventStreamCreate(
            nil,
            callback,
            &context,
            pathsToWatch,
            FSEventStreamEventId(kFSEventStreamEventIdSinceNow),
            2.0, // 2-second coalesce latency
            FSEventStreamCreateFlags(kFSEventStreamCreateFlagUseCFTypes | kFSEventStreamCreateFlagFileEvents)
        )

        if let stream = stream {
            FSEventStreamScheduleWithRunLoop(stream, CFRunLoopGetMain(), CFRunLoopMode.defaultMode.rawValue)
            FSEventStreamStart(stream)
        }
    }

    deinit {
        if let stream = stream {
            FSEventStreamStop(stream)
            FSEventStreamInvalidate(stream)
            FSEventStreamRelease(stream)
        }
    }
}
