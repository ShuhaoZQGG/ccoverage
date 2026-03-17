#!/bin/bash
set -euo pipefail

APP_NAME="CCoverage"
BUILD_DIR="build"
APP_BUNDLE="${BUILD_DIR}/${APP_NAME}.app"
CONTENTS="${APP_BUNDLE}/Contents"
MACOS="${CONTENTS}/MacOS"
RESOURCES="${CONTENTS}/Resources"

VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "0.1.0")
VERSION=${VERSION#v}  # strip leading v

echo "Creating ${APP_NAME}.app (version ${VERSION})..."

# Clean previous bundle
rm -rf "${APP_BUNDLE}"
mkdir -p "${MACOS}" "${RESOURCES}"

# Copy menubar binary — prefer release universal binary
MENUBAR_BINARY="menubar/.build/apple/Products/Release/CCoverageMenuBar"
if [ ! -f "${MENUBAR_BINARY}" ]; then
    MENUBAR_BINARY="menubar/.build/release/CCoverageMenuBar"
fi
if [ ! -f "${MENUBAR_BINARY}" ]; then
    echo "Error: Release binary not found. Run 'swift build -c release' in menubar/ first."
    exit 1
fi

cp "${MENUBAR_BINARY}" "${MACOS}/${APP_NAME}"
chmod +x "${MACOS}/${APP_NAME}"

# Bundle CLI binary if available
CLI_BINARY="${BUILD_DIR}/ccoverage"
if [ -f "${CLI_BINARY}" ]; then
    cp "${CLI_BINARY}" "${MACOS}/ccoverage"
    chmod +x "${MACOS}/ccoverage"
    echo "Bundled CLI binary"
else
    echo "Warning: CLI binary not found at ${CLI_BINARY}, skipping"
fi

# Copy icon
ICON_SRC="menubar/Release/AppIcon.icns"
if [ -f "${ICON_SRC}" ]; then
    cp "${ICON_SRC}" "${RESOURCES}/AppIcon.icns"
fi

# Generate Info.plist
cat > "${CONTENTS}/Info.plist" << PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleExecutable</key>
	<string>${APP_NAME}</string>
	<key>CFBundleIdentifier</key>
	<string>com.shuhaozhang.CCoverage</string>
	<key>CFBundleName</key>
	<string>${APP_NAME}</string>
	<key>CFBundleDisplayName</key>
	<string>CCoverage</string>
	<key>CFBundleVersion</key>
	<string>${VERSION}</string>
	<key>CFBundleShortVersionString</key>
	<string>${VERSION}</string>
	<key>CFBundlePackageType</key>
	<string>APPL</string>
	<key>LSMinimumSystemVersion</key>
	<string>14.0</string>
	<key>LSUIElement</key>
	<true/>
	<key>CFBundleIconFile</key>
	<string>AppIcon</string>
	<key>NSHighResolutionCapable</key>
	<true/>
</dict>
</plist>
PLIST

echo "Created ${APP_BUNDLE}"
