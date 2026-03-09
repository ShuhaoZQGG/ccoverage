import Foundation

struct AppSettings: Codable {
    var watchedRepoPaths: [String]
    var ccoverageBinaryPath: String?
    var pollIntervalSeconds: Int
    var lookbackDays: Int

    init(watchedRepoPaths: [String] = [], ccoverageBinaryPath: String? = nil, pollIntervalSeconds: Int = 60, lookbackDays: Int = 30) {
        self.watchedRepoPaths = watchedRepoPaths
        self.ccoverageBinaryPath = ccoverageBinaryPath
        self.pollIntervalSeconds = pollIntervalSeconds
        self.lookbackDays = lookbackDays
    }

    private static var configDir: URL {
        FileManager.default.homeDirectoryForCurrentUser.appendingPathComponent(".ccoverage")
    }

    private static var configFile: URL {
        configDir.appendingPathComponent("menubar.json")
    }

    static func load() -> AppSettings {
        let url = configFile
        guard let data = try? Data(contentsOf: url),
              let settings = try? JSONDecoder().decode(AppSettings.self, from: data) else {
            var settings = AppSettings()
            settings.watchedRepoPaths = RepoDetector.detectRepos()
            return settings
        }
        return settings
    }

    func save() {
        let url = Self.configFile
        try? FileManager.default.createDirectory(at: Self.configDir, withIntermediateDirectories: true)
        if let data = try? JSONEncoder().encode(self) {
            try? data.write(to: url)
        }
    }
}
