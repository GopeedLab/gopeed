#!/usr/bin/env python3
"""Render a light/dark multi-size contact sheet for a generated file-category font."""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

try:
    from PIL import Image, ImageDraw, ImageFont
except ImportError as error:
    raise SystemExit("Missing Pillow; install scripts/requirements.txt in the skill's .venv") from error


def parse_number(value: object) -> int:
    return value if isinstance(value, int) else int(str(value), 0)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--manifest", required=True, type=Path)
    parser.add_argument("--sizes", default="16,20,24,32")
    return parser.parse_args()


def draw_centered(
    draw: ImageDraw.ImageDraw,
    center: tuple[float, float],
    char: str,
    font: ImageFont.FreeTypeFont,
    fill: str,
) -> None:
    bounds = draw.textbbox((0, 0), char, font=font)
    height = bounds[3] - bounds[1]
    advance = draw.textlength(char, font=font)
    draw.text(
        (center[0] - advance / 2, center[1] - height / 2 - bounds[1]),
        char,
        font=font,
        fill=fill,
    )


def main() -> int:
    args = parse_args()
    manifest_path = args.manifest.resolve()
    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    manifest_dir = manifest_path.parent
    font_path = (manifest_dir / str(manifest["fontOutput"])).resolve()
    if not font_path.is_file():
        raise ValueError(f"font does not exist; run the builder first: {font_path}")
    sizes = [int(value) for value in args.sizes.split(",") if value.strip()]
    if not sizes or any(size <= 0 for size in sizes):
        raise ValueError("sizes must contain positive integers")
    icons = sorted(manifest["icons"], key=lambda icon: parse_number(icon["codepoint"]))

    label_width = 210
    cell_width = 112
    header_height = 44
    row_height = 78
    image = Image.new(
        "RGB",
        (label_width + cell_width * len(sizes), header_height + row_height * len(icons)),
        "#F8FAFC",
    )
    draw = ImageDraw.Draw(image)
    label_font = ImageFont.load_default()
    draw.text((12, 15), "icon / codepoint", font=label_font, fill="#111827")
    for column, size in enumerate(sizes):
        x = label_width + column * cell_width
        draw.text((x + 42, 15), f"{size}px", font=label_font, fill="#111827")

    for row, icon in enumerate(icons):
        y = header_height + row * row_height
        codepoint = parse_number(icon["codepoint"])
        draw.text(
            (12, y + 29),
            f"{icon['name']}  U+{codepoint:04X}",
            font=label_font,
            fill="#111827",
        )
        for column, size in enumerate(sizes):
            x = label_width + column * cell_width
            midpoint = x + cell_width // 2
            draw.rectangle((x, y, midpoint, y + row_height), fill="#FFFFFF")
            draw.rectangle((midpoint, y, x + cell_width, y + row_height), fill="#111827")
            font = ImageFont.truetype(str(font_path), size=size)
            draw_centered(draw, (x + cell_width * 0.25, y + row_height / 2), chr(codepoint), font, "#111827")
            draw_centered(draw, (x + cell_width * 0.75, y + row_height / 2), chr(codepoint), font, "#F8FAFC")

    output_value = manifest.get("previewOutput", "file-category-icons-preview.png")
    output = (manifest_dir / str(output_value)).resolve()
    output.parent.mkdir(parents=True, exist_ok=True)
    image.save(output)
    print(f"Preview: {output}")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (KeyError, OSError, ValueError, json.JSONDecodeError) as error:
        print(f"error: {error}", file=sys.stderr)
        raise SystemExit(1)
