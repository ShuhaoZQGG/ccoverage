import Foundation

enum ShellError: Error, LocalizedError {
    case nonZeroExit(code: Int32, stderr: String)

    var errorDescription: String? {
        switch self {
        case .nonZeroExit(let code, let stderr):
            return "Process exited with code \(code): \(stderr)"
        }
    }
}

struct ShellExecutor {
    static func run(executablePath: String, arguments: [String]) async throws -> String {
        let process = Process()
        process.executableURL = URL(fileURLWithPath: executablePath)
        process.arguments = arguments

        let stdoutPipe = Pipe()
        let stderrPipe = Pipe()
        process.standardOutput = stdoutPipe
        process.standardError = stderrPipe

        try process.run()
        process.waitUntilExit()

        let stdoutData = stdoutPipe.fileHandleForReading.readDataToEndOfFile()
        let stderrData = stderrPipe.fileHandleForReading.readDataToEndOfFile()

        if process.terminationStatus != 0 {
            let stderrStr = String(data: stderrData, encoding: .utf8) ?? ""
            throw ShellError.nonZeroExit(code: process.terminationStatus, stderr: stderrStr)
        }

        return String(data: stdoutData, encoding: .utf8) ?? ""
    }
}
