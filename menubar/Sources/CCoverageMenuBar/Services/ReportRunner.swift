import Foundation

private let isoFormatterWithFrac: ISO8601DateFormatter = {
    let f = ISO8601DateFormatter()
    f.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
    return f
}()

private let isoFormatter: ISO8601DateFormatter = {
    let f = ISO8601DateFormatter()
    f.formatOptions = [.withInternetDateTime]
    return f
}()

func resolveCCoverageBinaryPath(settings: AppSettings) -> String? {
    let fm = FileManager.default
    if let explicit = settings.ccoverageBinaryPath, fm.isExecutableFile(atPath: explicit) {
        return explicit
    }
    if let pathEnv = ProcessInfo.processInfo.environment["PATH"] {
        for dir in pathEnv.split(separator: ":") {
            let candidate = "\(dir)/ccoverage"
            if fm.isExecutableFile(atPath: candidate) { return candidate }
        }
    }

    var fallbacks = [
        "/opt/homebrew/bin/ccoverage",
        "/usr/local/bin/ccoverage",
        NSHomeDirectory() + "/go/bin/ccoverage",
    ]
    let env = ProcessInfo.processInfo.environment
    if let gobin = env["GOBIN"], !gobin.isEmpty {
        fallbacks.append(gobin + "/ccoverage")
    }
    if let gopath = env["GOPATH"], !gopath.isEmpty {
        fallbacks.append(gopath + "/bin/ccoverage")
    }
    for fallback in fallbacks {
        if fm.isExecutableFile(atPath: fallback) { return fallback }
    }

    if let shellResolved = resolveViaLoginShell() {
        return shellResolved
    }
    return nil
}

private func resolveViaLoginShell() -> String? {
    let process = Process()
    process.executableURL = URL(fileURLWithPath: "/bin/zsh")
    process.arguments = ["-l", "-c", "which ccoverage"]

    let pipe = Pipe()
    process.standardOutput = pipe
    process.standardError = FileHandle.nullDevice

    do {
        try process.run()
        process.waitUntilExit()
    } catch {
        return nil
    }

    guard process.terminationStatus == 0 else { return nil }
    let data = pipe.fileHandleForReading.readDataToEndOfFile()
    let path = String(data: data, encoding: .utf8)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
    guard !path.isEmpty, FileManager.default.isExecutableFile(atPath: path) else { return nil }
    return path
}

actor ReportRunner {
    func runReport(repoPath: String, settings: AppSettings) async throws -> CoverageReport {
        let binaryPath = try resolveBinaryPath(settings: settings)

        let output = try await ShellExecutor.run(
            executablePath: binaryPath,
            arguments: [
                "report",
                "--target", repoPath,
                "--format", "json",
                "--last-session",
                "--lookback-days", String(settings.lookbackDays)
            ]
        )

        return try decodeReport(from: output)
    }

    func runReport(repoPath: String, lookbackDays: Int, settings: AppSettings) async throws -> CoverageReport {
        let binaryPath = try resolveBinaryPath(settings: settings)

        let output = try await ShellExecutor.run(
            executablePath: binaryPath,
            arguments: [
                "report",
                "--target", repoPath,
                "--format", "json",
                "--last-session",
                "--lookback-days", String(lookbackDays)
            ]
        )

        return try decodeReport(from: output)
    }

    func runComparisonReport(repoPath: String, recentDays: Int, baselineDays: Int, settings: AppSettings) async throws -> ComparisonReport {
        async let recent = runReport(repoPath: repoPath, lookbackDays: recentDays, settings: settings)
        async let baseline = runReport(repoPath: repoPath, lookbackDays: baselineDays, settings: settings)
        return ComparisonBuilder.build(recent: try await recent, baseline: try await baseline, recentDays: recentDays, baselineDays: baselineDays)
    }

    private func decodeReport(from output: String) throws -> CoverageReport {
        let trimmed = output.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmed.isEmpty {
            throw ReportRunnerError.emptyOutput
        }

        guard let data = trimmed.data(using: .utf8) else {
            throw ReportRunnerError.invalidOutput
        }

        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .custom { decoder in
            let container = try decoder.singleValueContainer()
            let string = try container.decode(String.self)

            if let date = isoFormatterWithFrac.date(from: string) {
                return date
            }
            if let date = isoFormatter.date(from: string) {
                return date
            }

            throw DecodingError.dataCorruptedError(
                in: container,
                debugDescription: "Cannot decode date: \(string)"
            )
        }

        do {
            return try decoder.decode(CoverageReport.self, from: data)
        } catch let error as DecodingError {
            throw ReportRunnerError.decodeFailed(
                detail: describeDecodingError(error),
                output: String(trimmed.prefix(200))
            )
        }
    }

    private func describeDecodingError(_ error: DecodingError) -> String {
        switch error {
        case .typeMismatch(let type, let ctx):
            return "Type mismatch for \(type) at \(ctx.codingPath.map(\.stringValue).joined(separator: "."))"
        case .keyNotFound(let key, _):
            return "Missing key '\(key.stringValue)'"
        case .valueNotFound(let type, let ctx):
            return "Null value for \(type) at \(ctx.codingPath.map(\.stringValue).joined(separator: "."))"
        case .dataCorrupted(let ctx):
            return "Data corrupted at \(ctx.codingPath.map(\.stringValue).joined(separator: ".")): \(ctx.debugDescription)"
        @unknown default:
            return error.localizedDescription
        }
    }

    private func resolveBinaryPath(settings: AppSettings) throws -> String {
        guard let path = resolveCCoverageBinaryPath(settings: settings) else {
            throw ReportRunnerError.binaryNotFound
        }
        return path
    }
}

enum ReportRunnerError: Error, LocalizedError {
    case binaryNotFound
    case invalidOutput
    case emptyOutput
    case decodeFailed(detail: String, output: String)

    var errorDescription: String? {
        switch self {
        case .binaryNotFound:
            return "ccoverage binary not found. Install with:\n• brew install ShuhaoZQGG/tap/ccoverage\n• go install github.com/ShuhaoZQGG/ccoverage@latest\nOr set the path in Settings → Advanced."
        case .invalidOutput:
            return "Invalid output from ccoverage."
        case .emptyOutput:
            return "No output from ccoverage. The repo may have no Claude Code configuration."
        case .decodeFailed(let detail, let output):
            return "Failed to decode ccoverage output: \(detail)\nOutput: \(output)"
        }
    }
}
