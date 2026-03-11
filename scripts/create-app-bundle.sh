#!/bin/bash
set -euo pipefail

APP_NAME="CCoverageMenuBar"
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

# Copy binary — prefer release universal binary
BINARY="menubar/.build/apple/Products/Release/${APP_NAME}"
if [ ! -f "${BINARY}" ]; then
    BINARY="menubar/.build/release/${APP_NAME}"
fi
if [ ! -f "${BINARY}" ]; then
    echo "Error: Release binary not found. Run 'make menubar-release' first."
    exit 1
fi

cp "${BINARY}" "${MACOS}/${APP_NAME}"
chmod +x "${MACOS}/${APP_NAME}"

# Copy icon if present
ICON_SRC="menubar/Resources/AppIcon.icns"
if [ -f "${ICON_SRC}" ]; then
    cp "${ICON_SRC}" "${RESOURCES}/AppIcon.icns"
    ICON_ENTRY="<key>CFBundleIconFile</key>
	<string>AppIcon</string>"
else
    ICON_ENTRY=""
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
	<string>com.shuhaozhang.ccoverage.menubar</string>
	<key>CFBundleName</key>
	<string>${APP_NAME}</string>
	<key>CFBundleDisplayName</key>
	<string>CCoverage Menu Bar</string>
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
	${ICON_ENTRY}
</dict>
</plist>
PLIST

echo "Created ${APP_BUNDLE}"
