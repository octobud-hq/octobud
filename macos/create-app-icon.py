#!/usr/bin/env python3
"""
Create a Mac app icon from baby_octo.png with a gradient background.

The gradient matches the PageHeader component:
- from-indigo-500 (#6366f1)
- via-violet-500 (#8b5cf6)  
- to-purple-500 (#a855f7)
- Gradient direction: bottom-right (to-br)
"""

import os
import sys
from pathlib import Path
import subprocess

try:
    from PIL import Image, ImageDraw
    HAS_PIL = True
except ImportError:
    HAS_PIL = False
    print("Warning: PIL/Pillow not found. Install with: pip3 install Pillow")
    print("Or use the alternative method with sips (macOS built-in tool)")
    sys.exit(1)

# Tailwind color values (RGB)
INDIGO_500 = (99, 102, 241)   # #6366f1
VIOLET_500 = (139, 92, 246)   # #8b5cf6
PURPLE_500 = (168, 85, 247)   # #a855f7

# Icon sizes required for macOS .icns
ICON_SIZES = [16, 32, 64, 128, 256, 512, 1024]


def create_gradient_background(size: int) -> Image.Image:
    """Create a gradient background from indigo -> violet -> purple (bottom-right diagonal)."""
    img = Image.new('RGB', (size, size))
    pixels = img.load()
    
    for y in range(size):
        for x in range(size):
            # Calculate distance from top-left (0,0) to bottom-right (size, size)
            # Normalize to 0.0 - 1.0
            distance = (x + y) / (2 * size)
            
            # Interpolate between the three colors
            if distance < 0.5:
                # First half: indigo -> violet
                t = distance * 2  # 0.0 -> 1.0
                r = int(INDIGO_500[0] + (VIOLET_500[0] - INDIGO_500[0]) * t)
                g = int(INDIGO_500[1] + (VIOLET_500[1] - INDIGO_500[1]) * t)
                b = int(INDIGO_500[2] + (VIOLET_500[2] - INDIGO_500[2]) * t)
            else:
                # Second half: violet -> purple
                t = (distance - 0.5) * 2  # 0.0 -> 1.0
                r = int(VIOLET_500[0] + (PURPLE_500[0] - VIOLET_500[0]) * t)
                g = int(VIOLET_500[1] + (PURPLE_500[1] - VIOLET_500[1]) * t)
                b = int(VIOLET_500[2] + (PURPLE_500[2] - VIOLET_500[2]) * t)
            
            pixels[x, y] = (r, g, b)
    
    return img


def create_icon_size(size: int, octo_image: Image.Image) -> Image.Image:
    """Create an icon of the specified size with gradient background and octo overlay."""
    # Create gradient background
    background = create_gradient_background(size)
    
    # Resize octo image to fit nicely (80% of icon size, centered)
    octo_size = int(size * 0.8)
    octo_resized = octo_image.resize((octo_size, octo_size), Image.Resampling.LANCZOS)
    
    # Calculate position to center the octo
    x_offset = (size - octo_size) // 2
    y_offset = (size - octo_size) // 2
    
    # Paste octo onto gradient background
    # Use alpha composite if octo has transparency
    if octo_resized.mode == 'RGBA':
        background = background.convert('RGBA')
        background.alpha_composite(octo_resized, (x_offset, y_offset))
        background = background.convert('RGB')
    else:
        background.paste(octo_resized, (x_offset, y_offset), octo_resized if octo_resized.mode == 'RGBA' else None)
    
    return background


def create_icns_file(iconset_dir: Path, output_path: Path):
    """Create .icns file from .iconset directory using iconutil."""
    try:
        subprocess.run(
            ['iconutil', '-c', 'icns', str(iconset_dir), '-o', str(output_path)],
            check=True,
            capture_output=True
        )
        print(f"✓ Created {output_path}")
    except subprocess.CalledProcessError as e:
        print(f"Error creating .icns file: {e.stderr.decode()}")
        sys.exit(1)
    except FileNotFoundError:
        print("Error: iconutil not found. This script requires macOS.")
        sys.exit(1)


def main():
    # Get script directory and project root
    script_dir = Path(__file__).parent
    project_root = script_dir.parent.parent
    
    # Paths
    octo_source = project_root / 'frontend' / 'static' / 'baby_octo.png'
    output_dir = script_dir
    iconset_dir = output_dir / 'AppIcon.iconset'
    icns_output = output_dir / 'AppIcon.icns'
    
    # Check if source image exists
    if not octo_source.exists():
        print(f"Error: Source image not found at {octo_source}")
        sys.exit(1)
    
    # Load octo image
    print(f"Loading {octo_source}...")
    try:
        octo_image = Image.open(octo_source)
        if octo_image.mode != 'RGBA':
            octo_image = octo_image.convert('RGBA')
        print(f"✓ Loaded image: {octo_image.size[0]}x{octo_image.size[1]}")
    except Exception as e:
        print(f"Error loading image: {e}")
        sys.exit(1)
    
    # Create iconset directory
    iconset_dir.mkdir(exist_ok=True)
    print(f"\nCreating icon sizes in {iconset_dir}...")
    
    # Generate all required icon sizes
    for size in ICON_SIZES:
        print(f"  Creating {size}x{size}...", end=' ')
        icon = create_icon_size(size, octo_image)
        
        # Save as PNG (some sizes need @2x variants)
        if size <= 128:
            # Save regular and @2x versions for smaller sizes
            icon.save(iconset_dir / f'icon_{size}x{size}.png')
            if size < 1024:
                icon_2x = create_icon_size(size * 2, octo_image)
                icon_2x.save(iconset_dir / f'icon_{size}x{size}@2x.png')
        else:
            # For larger sizes, just save the regular version
            icon.save(iconset_dir / f'icon_{size}x{size}.png')
        
        print("✓")
    
    # Create .icns file
    print(f"\nCreating {icns_output}...")
    create_icns_file(iconset_dir, icns_output)
    
    # Clean up iconset directory
    print(f"\nCleaning up {iconset_dir}...")
    import shutil
    shutil.rmtree(iconset_dir)
    
    print(f"\n✓ App icon created: {icns_output}")
    print(f"  Icon is already in macos/AppIcon.icns and will be used in builds")


if __name__ == '__main__':
    main()

