#!/bin/bash
# Create macOS .icns file from a source PNG
# Usage: ./create-icns.sh <input.png> <output.icns>
#
# The input PNG should be at least 1024x1024 pixels.
# If no arguments provided, uses baby_octo.svg from frontend/static

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

INPUT="${1:-}"
OUTPUT="${2:-$SCRIPT_DIR/AppIcon.icns}"

# If no input specified, try to use baby_octo.svg
if [ -z "$INPUT" ]; then
    SVG_SOURCE="$REPO_ROOT/frontend/static/baby_octo.svg"
    if [ -f "$SVG_SOURCE" ]; then
        echo "Converting baby_octo.svg to PNG..."
        # Check if rsvg-convert is available (from librsvg)
        if command -v rsvg-convert &> /dev/null; then
            INPUT="/tmp/octobud-icon-1024.png"
            rsvg-convert -w 1024 -h 1024 "$SVG_SOURCE" -o "$INPUT"
        else
            echo "Error: rsvg-convert not found. Install with: brew install librsvg"
            echo "Or provide a 1024x1024 PNG as first argument."
            exit 1
        fi
    else
        echo "Usage: $0 <input.png> [output.icns]"
        echo "  input.png should be at least 1024x1024 pixels"
        exit 1
    fi
fi

if [ ! -f "$INPUT" ]; then
    echo "Error: Input file not found: $INPUT"
    exit 1
fi

# Create temporary iconset directory
ICONSET_DIR="/tmp/AppIcon.iconset"
rm -rf "$ICONSET_DIR"
mkdir -p "$ICONSET_DIR"

echo "Creating icon sizes..."

# Generate all required sizes
sips -z 16 16     "$INPUT" --out "$ICONSET_DIR/icon_16x16.png" > /dev/null
sips -z 32 32     "$INPUT" --out "$ICONSET_DIR/icon_16x16@2x.png" > /dev/null
sips -z 32 32     "$INPUT" --out "$ICONSET_DIR/icon_32x32.png" > /dev/null
sips -z 64 64     "$INPUT" --out "$ICONSET_DIR/icon_32x32@2x.png" > /dev/null
sips -z 128 128   "$INPUT" --out "$ICONSET_DIR/icon_128x128.png" > /dev/null
sips -z 256 256   "$INPUT" --out "$ICONSET_DIR/icon_128x128@2x.png" > /dev/null
sips -z 256 256   "$INPUT" --out "$ICONSET_DIR/icon_256x256.png" > /dev/null
sips -z 512 512   "$INPUT" --out "$ICONSET_DIR/icon_256x256@2x.png" > /dev/null
sips -z 512 512   "$INPUT" --out "$ICONSET_DIR/icon_512x512.png" > /dev/null
sips -z 1024 1024 "$INPUT" --out "$ICONSET_DIR/icon_512x512@2x.png" > /dev/null

echo "Creating .icns file..."
iconutil -c icns "$ICONSET_DIR" -o "$OUTPUT"

# Cleanup
rm -rf "$ICONSET_DIR"

echo "Created: $OUTPUT"

