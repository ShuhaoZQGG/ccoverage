import Foundation

struct CoverageReport: Codable {
    let repoPath: String
    let lookbackDays: Int
    let sessionsAnalyzed: Int
    let results: [CoverageResult]
    let summary: ReportSummary
    let lastSession: LastSessionReport?

    enum CodingKeys: String, CodingKey {
        case repoPath = "repo_path"
        case lookbackDays = "lookback_days"
        case sessionsAnalyzed = "sessions_analyzed"
        case results, summary
        case lastSession = "last_session"
    }
}

struct CoverageResult: Codable {
    let item: ManifestItem
    let usage: UsageSummary
    let status: String
}

struct ManifestItem: Codable {
    let type: String
    let name: String
    let path: String
    let absPath: String
    let lastModified: Date
    let metadata: [String: String]?

    var displayType: String {
        if type == "Skill", metadata?["scope"] == "root" {
            return "Skill (Global)"
        }
        return type
    }

    enum CodingKeys: String, CodingKey {
        case type, name, path
        case absPath = "abs_path"
        case lastModified = "last_modified"
        case metadata
    }
}

struct UsageSummary: Codable {
    let totalActivations: Int
    let uniqueSessions: Int
    let firstSeen: Date?
    let lastSeen: Date?

    enum CodingKeys: String, CodingKey {
        case totalActivations = "total_activations"
        case uniqueSessions = "unique_sessions"
        case firstSeen = "first_seen"
        case lastSeen = "last_seen"
    }
}

struct ReportSummary: Codable {
    let totalItems: Int
    let active: Int
    let underused: Int
    let dormant: Int

    enum CodingKeys: String, CodingKey {
        case totalItems = "total_items"
        case active, underused, dormant
    }
}

struct LastSessionReport: Codable {
    let sessionID: String
    let timestamp: Date
    let items: [LastSessionItem]

    enum CodingKeys: String, CodingKey {
        case sessionID = "session_id"
        case timestamp, items
    }
}

struct LastSessionItem: Codable {
    let type: String
    let name: String
    let active: Bool
    let count: Int

    enum CodingKeys: String, CodingKey {
        case type, name, active, count
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        type = try container.decode(String.self, forKey: .type)
        name = try container.decode(String.self, forKey: .name)
        active = try container.decode(Bool.self, forKey: .active)
        count = try container.decodeIfPresent(Int.self, forKey: .count) ?? 0
    }
}

// MARK: - Comparison

struct ComparisonReport {
    let recent: CoverageReport
    let baseline: CoverageReport
    let recentDays: Int
    let baselineDays: Int
    let items: [ComparisonItem]
    let summaryDelta: SummaryDelta
}

struct ComparisonItem: Identifiable {
    var id: String { "\(type):\(name)" }
    let name: String
    let type: String
    let recentStatus: String?
    let baselineStatus: String?
    let activationDelta: Int
    let isNew: Bool
    let isDropped: Bool

    var statusChanged: Bool {
        recentStatus != baselineStatus && !isNew && !isDropped
    }
}

struct SummaryDelta {
    let active: (old: Int, new: Int)
    let underused: (old: Int, new: Int)
    let dormant: (old: Int, new: Int)
}

enum ComparisonBuilder {
    static func build(recent: CoverageReport, baseline: CoverageReport, recentDays: Int, baselineDays: Int) -> ComparisonReport {
        let recentByKey = Dictionary(uniqueKeysWithValues: recent.results.map { ("\($0.item.type):\($0.item.name)", $0) })
        let baselineByKey = Dictionary(uniqueKeysWithValues: baseline.results.map { ("\($0.item.type):\($0.item.name)", $0) })
        let allKeys = Set(recentByKey.keys).union(baselineByKey.keys)

        let items: [ComparisonItem] = allKeys.sorted().map { key in
            let r = recentByKey[key]
            let b = baselineByKey[key]
            let name = r?.item.name ?? b?.item.name ?? key
            let type = r?.item.displayType ?? b?.item.displayType ?? ""
            let recentAct = r?.usage.totalActivations ?? 0
            let baselineAct = b?.usage.totalActivations ?? 0

            return ComparisonItem(
                name: name,
                type: type,
                recentStatus: r?.status,
                baselineStatus: b?.status,
                activationDelta: recentAct - baselineAct,
                isNew: r != nil && b == nil,
                isDropped: r == nil && b != nil
            )
        }

        let delta = SummaryDelta(
            active: (baseline.summary.active, recent.summary.active),
            underused: (baseline.summary.underused, recent.summary.underused),
            dormant: (baseline.summary.dormant, recent.summary.dormant)
        )

        return ComparisonReport(recent: recent, baseline: baseline, recentDays: recentDays, baselineDays: baselineDays, items: items, summaryDelta: delta)
    }
}
