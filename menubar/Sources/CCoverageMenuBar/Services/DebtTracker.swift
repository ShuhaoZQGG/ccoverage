import Foundation

struct DebtSnapshot: Codable {
    let date: Date
    let debt: Int
}

final class DebtTracker {
    private let defaults = UserDefaults.standard

    private func key(for repoPath: String) -> String {
        let encoded = repoPath.replacingOccurrences(of: "/", with: "_")
        return "ccoverage_debt_history_\(encoded)"
    }

    func recordDebt(repoPath: String, debt: Int) {
        var history = loadHistory(repoPath: repoPath)
        let today = Calendar.current.startOfDay(for: Date())

        if let idx = history.firstIndex(where: { Calendar.current.isDate($0.date, inSameDayAs: today) }) {
            history[idx] = DebtSnapshot(date: today, debt: debt)
        } else {
            history.append(DebtSnapshot(date: today, debt: debt))
        }

        // Keep last 90 days
        let cutoff = Calendar.current.date(byAdding: .day, value: -90, to: today)!
        history = history.filter { $0.date >= cutoff }

        saveHistory(repoPath: repoPath, history: history)
    }

    func weekDelta(repoPath: String) -> Int? {
        let history = loadHistory(repoPath: repoPath)
        guard let latest = history.last else { return nil }

        let targetDate = Calendar.current.date(byAdding: .day, value: -7, to: Date())!
        let closest = history
            .filter { $0.date <= targetDate }
            .min(by: { abs($0.date.timeIntervalSince(targetDate)) < abs($1.date.timeIntervalSince(targetDate)) })

        guard let baseline = closest else { return nil }
        return latest.debt - baseline.debt
    }

    private func loadHistory(repoPath: String) -> [DebtSnapshot] {
        guard let data = defaults.data(forKey: key(for: repoPath)),
              let history = try? JSONDecoder().decode([DebtSnapshot].self, from: data) else {
            return []
        }
        return history
    }

    private func saveHistory(repoPath: String, history: [DebtSnapshot]) {
        if let data = try? JSONEncoder().encode(history) {
            defaults.set(data, forKey: key(for: repoPath))
        }
    }
}
