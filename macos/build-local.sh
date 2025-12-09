#!/bin/bash
set -e

# Build and sign Octobud macOS app + installer locally
# Usage: ./macos/build-local.sh [--skip-sign] [--skip-notarize]

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
# Use VERSION env var if set, otherwise read from VERSION file, otherwise default to "dev"
VERSION="${VERSION:-$(cat "$ROOT_DIR/VERSION" 2>/dev/null || echo "dev")}"
SKIP_SIGN=false
SKIP_NOTARIZE=true  # Default to skip notarization for local testing

# Parse arguments
for arg in "$@"; do
    case $arg in
        --skip-sign)
            SKIP_SIGN=true
            shift
            ;;
        --notarize)
            SKIP_NOTARIZE=false
            shift
            ;;
        *)
            ;;
    esac
done

echo "╔══════════════════════════════════════════════════════════════╗"
echo "║           Octobud macOS Build Script                         ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""
echo "  Version: $VERSION"
echo "  Sign: $([ "$SKIP_SIGN" = true ] && echo "SKIP" || echo "YES")"
echo "  Notarize: $([ "$SKIP_NOTARIZE" = true ] && echo "SKIP" || echo "YES")"
echo ""

# Create dist directory
DIST_DIR="$ROOT_DIR/dist"
rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

# Step 1: Build frontend
echo "▶ Step 1/7: Building frontend..."
cd "$ROOT_DIR/frontend"
npm run build
echo "  ✓ Frontend built"

# Step 2: Copy frontend to backend
echo "▶ Step 2/7: Copying frontend assets..."
mkdir -p "$ROOT_DIR/backend/web/dist"
cp -r "$ROOT_DIR/frontend/build/"* "$ROOT_DIR/backend/web/dist/"
echo "  ✓ Frontend assets copied"

# Step 3: Build Go binary
echo "▶ Step 3/7: Building Go binary..."
cd "$ROOT_DIR/backend"
CGO_ENABLED=1 go build -ldflags="-s -w -X github.com/octobud-hq/octobud/backend/internal/version.Version=$VERSION" -o "$DIST_DIR/octobud" ./cmd/octobud
echo "  ✓ Binary built: $DIST_DIR/octobud"

# Step 4: Create .app bundle
echo "▶ Step 4/7: Creating .app bundle..."
APP_DIR="$DIST_DIR/Octobud.app"
mkdir -p "$APP_DIR/Contents/MacOS"
mkdir -p "$APP_DIR/Contents/Resources"

# Copy binary
cp "$DIST_DIR/octobud" "$APP_DIR/Contents/MacOS/"

# Copy and update Info.plist
cp "$SCRIPT_DIR/Info.plist" "$APP_DIR/Contents/"
sed -i '' "s/VERSION/$VERSION/g" "$APP_DIR/Contents/Info.plist"

# Create/copy icon
if [ -f "$SCRIPT_DIR/AppIcon.icns" ]; then
    cp "$SCRIPT_DIR/AppIcon.icns" "$APP_DIR/Contents/Resources/"
else
    echo "  ⚠ No AppIcon.icns found, creating from favicon..."
    cd "$SCRIPT_DIR"
    ./create-icns.sh "$ROOT_DIR/frontend/static/favicon.png" "$APP_DIR/Contents/Resources/AppIcon.icns" 2>/dev/null || true
fi

echo "  ✓ App bundle created: $APP_DIR"

# Step 5: Sign .app (optional)
if [ "$SKIP_SIGN" = false ]; then
    echo "▶ Step 5/7: Signing .app bundle..."
    
    # Find Developer ID Application certificate
    SIGN_IDENTITY=$(security find-identity -v -p codesigning | grep "Developer ID Application" | head -1 | sed 's/.*"\(.*\)".*/\1/')
    
    if [ -z "$SIGN_IDENTITY" ]; then
        echo "  ⚠ No 'Developer ID Application' certificate found!"
        echo "    Run: security find-identity -v -p codesigning"
        echo "    Skipping signing..."
    else
        echo "  Using identity: $SIGN_IDENTITY"
        
        # Sign the binary first, then the app bundle (required for proper signing)
        codesign --force --options runtime \
            --sign "$SIGN_IDENTITY" \
            --timestamp \
            "$APP_DIR/Contents/MacOS/octobud"
        
        codesign --force --options runtime \
            --sign "$SIGN_IDENTITY" \
            --timestamp \
            "$APP_DIR"
        
        echo "  ✓ App signed"
        
        # Verify signature
        codesign --verify --deep --strict "$APP_DIR" && echo "  ✓ Signature verified"
    fi
else
    echo "▶ Step 5/7: Skipping .app signing (--skip-sign)"
fi

# Step 6: Create .pkg installer
echo "▶ Step 6/7: Creating .pkg installer..."
PKG_UNSIGNED="$DIST_DIR/Octobud-$VERSION-unsigned.pkg"
PKG_SIGNED="$DIST_DIR/Octobud-$VERSION.pkg"

# Create component package
pkgbuild \
    --root "$APP_DIR" \
    --identifier "io.octobud" \
    --version "$VERSION" \
    --install-location "/Applications/Octobud.app" \
    --scripts "$SCRIPT_DIR/scripts" \
    "$PKG_UNSIGNED"

echo "  ✓ Package created: $PKG_UNSIGNED"

# Step 7: Sign .pkg (optional)
if [ "$SKIP_SIGN" = false ]; then
    echo "▶ Step 7/7: Signing .pkg installer..."
    
    # Find Developer ID Installer certificate
    INSTALLER_IDENTITY=$(security find-identity -v | grep "Developer ID Installer" | head -1 | sed 's/.*"\(.*\)".*/\1/')
    
    if [ -z "$INSTALLER_IDENTITY" ]; then
        echo "  ⚠ No 'Developer ID Installer' certificate found!"
        echo "    Skipping pkg signing..."
        mv "$PKG_UNSIGNED" "$PKG_SIGNED"
    else
        echo "  Using identity: $INSTALLER_IDENTITY"
        productsign --sign "$INSTALLER_IDENTITY" "$PKG_UNSIGNED" "$PKG_SIGNED"
        rm "$PKG_UNSIGNED"
        echo "  ✓ Package signed: $PKG_SIGNED"
        
        # Verify signature
        pkgutil --check-signature "$PKG_SIGNED" | head -5
    fi
else
    echo "▶ Step 7/7: Skipping .pkg signing (--skip-sign)"
    mv "$PKG_UNSIGNED" "$PKG_SIGNED"
fi

# Optional: Notarize
if [ "$SKIP_SIGN" = false ] && [ "$SKIP_NOTARIZE" = false ]; then
    echo ""
    echo "▶ Notarizing package..."
    echo "  This requires App Store Connect API credentials."
    echo "  Set these environment variables:"
    echo "    APPLE_API_KEY_ID"
    echo "    APPLE_API_KEY_ISSUER" 
    echo "    APPLE_API_KEY_PATH (path to .p8 file)"
    
    if [ -n "$APPLE_API_KEY_ID" ] && [ -n "$APPLE_API_KEY_ISSUER" ] && [ -n "$APPLE_API_KEY_PATH" ]; then
        xcrun notarytool submit "$PKG_SIGNED" \
            --key "$APPLE_API_KEY_PATH" \
            --key-id "$APPLE_API_KEY_ID" \
            --issuer "$APPLE_API_KEY_ISSUER" \
            --wait
        
        echo "  Stapling notarization ticket..."
        xcrun stapler staple "$PKG_SIGNED"
        echo "  ✓ Package notarized and stapled"
    else
        echo "  ⚠ Missing notarization credentials, skipping..."
    fi
fi

echo ""
echo "╔══════════════════════════════════════════════════════════════╗"
echo "║                     Build Complete!                          ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""
echo "  Output directory: $DIST_DIR"
echo ""
ls -la "$DIST_DIR"
echo ""
echo "  To test the app:"
echo "    open $APP_DIR"
echo ""
echo "  To test the installer:"
echo "    open $PKG_SIGNED"
echo ""

