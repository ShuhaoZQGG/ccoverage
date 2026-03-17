#!/bin/bash
set -euo pipefail

APP_NAME="CCoverage"
BUILD_DIR="build"
APP_BUNDLE="${BUILD_DIR}/${APP_NAME}.app"
DMG_NAME="${APP_NAME}.dmg"
DMG_PATH="${BUILD_DIR}/${DMG_NAME}"
STAGING_DIR="${BUILD_DIR}/dmg-staging"
BG_IMAGE="menubar/Release/dmg-background.png"
ICON_SRC="menubar/Release/DMGIcon.icns"

if [ ! -d "${APP_BUNDLE}" ]; then
    echo "Error: ${APP_BUNDLE} not found. Run 'make app-bundle' first."
    exit 1
fi

echo "Creating ${DMG_NAME}..."

# Clean previous artifacts
rm -rf "${STAGING_DIR}" "${DMG_PATH}"
mkdir -p "${STAGING_DIR}"

# Stage app bundle and Applications symlink
cp -R "${APP_BUNDLE}" "${STAGING_DIR}/"
ln -s /Applications "${STAGING_DIR}/Applications"

# Use create-dmg for a polished DMG if available, otherwise fall back to hdiutil
if command -v create-dmg &>/dev/null && [ -f "${BG_IMAGE}" ]; then
    create-dmg \
        --volname "${APP_NAME}" \
        --volicon "${ICON_SRC}" \
        --background "${BG_IMAGE}" \
        --window-pos 200 120 \
        --window-size 660 400 \
        --icon-size 160 \
        --icon "${APP_NAME}.app" 180 170 \
        --hide-extension "${APP_NAME}.app" \
        --app-drop-link 480 170 \
        --no-internet-enable \
        "${DMG_PATH}" \
        "${STAGING_DIR}"
else
    if ! command -v create-dmg &>/dev/null; then
        echo "Note: create-dmg not found, using hdiutil (basic DMG layout)"
    fi
    hdiutil create \
        -volname "${APP_NAME}" \
        -srcfolder "${STAGING_DIR}" \
        -ov \
        -format UDZO \
        "${DMG_PATH}"
fi

# Clean up staging
rm -rf "${STAGING_DIR}"

echo "Created ${DMG_PATH}"
