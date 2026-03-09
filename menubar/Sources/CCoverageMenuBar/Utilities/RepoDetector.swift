import Foundation

struct RepoDetector {
    static func detectRepos() -> [String] {
        let home = FileManager.default.homeDirectoryForCurrentUser
        let projectsDir = home.appendingPathComponent(".claude/projects")
        let fm = FileManager.default

        guard let entries = try? fm.contentsOfDirectory(atPath: projectsDir.path) else {
            return []
        }

        var repos: [(path: String, modified: Date)] = []
        for entry in entries {
            // Reverse dash-encoding: leading dash kept, so -Users-foo-project -> /Users/foo/project
            let decoded = entry.replacingOccurrences(of: "-", with: "/")
            guard fm.fileExists(atPath: decoded) else { continue }

            let entryURL = projectsDir.appendingPathComponent(entry)
            if let attrs = try? fm.attributesOfItem(atPath: entryURL.path),
               let modified = attrs[.modificationDate] as? Date {
                repos.append((decoded, modified))
            } else {
                repos.append((decoded, .distantPast))
            }
        }

        repos.sort { $0.modified > $1.modified }
        return repos.map(\.path)
    }
}
