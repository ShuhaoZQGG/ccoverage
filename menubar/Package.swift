// swift-tools-version: 5.9
import PackageDescription
let package = Package(
    name: "CCoverageMenuBar",
    platforms: [.macOS(.v14)],
    targets: [
        .executableTarget(name: "CCoverageMenuBar", path: "Sources/CCoverageMenuBar")
    ]
)
