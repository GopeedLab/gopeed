#!/usr/bin/env python3
"""Build a deterministic Gopeed file-category font and Flutter IconData class from SVG paths."""

from __future__ import annotations

import argparse
import json
import re
import sys
import xml.etree.ElementTree as ET
from pathlib import Path

try:
    from fontTools.fontBuilder import FontBuilder
    from fontTools.pens.boundsPen import BoundsPen
    from fontTools.pens.cu2quPen import Cu2QuPen
    from fontTools.pens.transformPen import TransformPen
    from fontTools.pens.ttGlyphPen import TTGlyphPen
    from fontTools.svgLib.path import SVGPath
except ImportError as error:
    raise SystemExit("Missing fonttools; install scripts/requirements.txt in the skill's .venv") from error


NAME_PATTERN = re.compile(r"^[a-z][a-z0-9_]*$")
DART_NAME_PATTERN = re.compile(r"^[A-Za-z][A-Za-z0-9]*$")
PRIVATE_USE_START = 0xE000
PRIVATE_USE_END = 0xF8FF
ALLOWED_SVG_TAGS = {"svg", "g", "path"}


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--manifest", required=True, type=Path)
    parser.add_argument("--allocate-codepoints", action="store_true")
    return parser.parse_args()


def local_name(tag: str) -> str:
    return tag.rsplit("}", 1)[-1]


def parse_number(value: object, field: str) -> int:
    if isinstance(value, int):
        return value
    if isinstance(value, str):
        try:
            return int(value, 0)
        except ValueError as error:
            raise ValueError(f"{field} must be an integer or 0x-prefixed integer") from error
    raise ValueError(f"{field} must be an integer or string")


def parse_view_box(svg_text: str, source: Path) -> tuple[float, float, float, float]:
    try:
        root = ET.fromstring(svg_text)
    except ET.ParseError as error:
        raise ValueError(f"{source}: invalid XML: {error}") from error
    if local_name(root.tag) != "svg":
        raise ValueError(f"{source}: root element must be <svg>")
    values = re.split(r"[\s,]+", root.attrib.get("viewBox", "").strip())
    if len(values) != 4:
        raise ValueError(f"{source}: viewBox must contain four numbers")
    try:
        min_x, min_y, width, height = (float(value) for value in values)
    except ValueError as error:
        raise ValueError(f"{source}: viewBox contains a non-numeric value") from error
    if width <= 0 or height <= 0:
        raise ValueError(f"{source}: viewBox dimensions must be positive")
    if abs(width - height) > 1e-6:
        raise ValueError(f"{source}: icon viewBox must be square")

    path_count = 0
    for element in root.iter():
        tag = local_name(element.tag)
        if tag not in ALLOWED_SVG_TAGS:
            raise ValueError(f"{source}: unsupported <{tag}>; convert everything to filled paths")
        style = element.attrib.get("style", "").replace(" ", "").lower()
        stroke = element.attrib.get("stroke", "none").lower()
        if stroke != "none" or ("stroke:" in style and "stroke:none" not in style):
            raise ValueError(f"{source}: live strokes are not allowed; expand strokes to paths")
        if tag == "path":
            if not element.attrib.get("d", "").strip():
                raise ValueError(f"{source}: every <path> needs path data")
            if element.attrib.get("fill", "").lower() == "none" or "fill:none" in style:
                raise ValueError(f"{source}: paths must be filled outlines")
            path_count += 1
    if path_count == 0:
        raise ValueError(f"{source}: no paths found")
    return min_x, min_y, width, height


def dart_name(icon: dict[str, object]) -> str:
    explicit = icon.get("dartName")
    if explicit is not None:
        value = str(explicit)
    else:
        parts = str(icon["name"]).split("_")
        value = parts[0] + "".join(part.capitalize() for part in parts[1:])
    if not DART_NAME_PATTERN.fullmatch(value):
        raise ValueError(f"invalid Dart field name: {value}")
    return value


def allocate_codepoints(manifest: dict[str, object], allow_allocation: bool) -> bool:
    icons = manifest.get("icons")
    if not isinstance(icons, list) or not icons:
        raise ValueError("manifest.icons must be a non-empty list")
    used = {
        parse_number(icon["codepoint"], f"icons[{index}].codepoint")
        for index, icon in enumerate(icons)
        if isinstance(icon, dict) and icon.get("codepoint") is not None
    }
    next_codepoint = parse_number(manifest.get("codepointStart", "0xE900"), "codepointStart")
    changed = False
    for index, icon in enumerate(icons):
        if not isinstance(icon, dict):
            raise ValueError(f"icons[{index}] must be an object")
        if icon.get("codepoint") is not None:
            continue
        if not allow_allocation:
            raise ValueError(f"icons[{index}] has no codepoint; rerun with --allocate-codepoints")
        while next_codepoint in used:
            next_codepoint += 1
        if next_codepoint > PRIVATE_USE_END:
            raise ValueError("no codepoints remain in the BMP private-use area")
        icon["codepoint"] = f"0x{next_codepoint:04X}"
        used.add(next_codepoint)
        next_codepoint += 1
        changed = True
    return changed


def validate_icons(manifest: dict[str, object], manifest_dir: Path) -> list[dict[str, object]]:
    icons = manifest["icons"]
    assert isinstance(icons, list)
    names: set[str] = set()
    dart_names: set[str] = set()
    codepoints: set[int] = set()
    validated: list[dict[str, object]] = []
    for index, raw_icon in enumerate(icons):
        assert isinstance(raw_icon, dict)
        name = str(raw_icon.get("name", ""))
        if not NAME_PATTERN.fullmatch(name):
            raise ValueError(f"icons[{index}].name must be lowercase snake_case")
        field_name = dart_name(raw_icon)
        codepoint = parse_number(raw_icon.get("codepoint"), f"icons[{index}].codepoint")
        if not PRIVATE_USE_START <= codepoint <= PRIVATE_USE_END:
            raise ValueError(f"{name}: codepoint must be in U+E000…U+F8FF")
        source_value = raw_icon.get("source")
        if not isinstance(source_value, str) or not source_value:
            raise ValueError(f"{name}: source must be a relative SVG path")
        source = (manifest_dir / source_value).resolve()
        if source.suffix.lower() != ".svg" or not source.is_file():
            raise ValueError(f"{name}: SVG source does not exist: {source}")
        if name in names or field_name in dart_names or codepoint in codepoints:
            raise ValueError(f"{name}: duplicate name, Dart field, or codepoint")
        names.add(name)
        dart_names.add(field_name)
        codepoints.add(codepoint)
        validated.append(
            {
                **raw_icon,
                "name": name,
                "glyphName": f"icon_{name}",
                "dartName": field_name,
                "codepoint": codepoint,
                "sourcePath": source,
            }
        )
    return sorted(validated, key=lambda icon: int(icon["codepoint"]))


def build_font(manifest: dict[str, object], manifest_dir: Path, icons: list[dict[str, object]]) -> Path:
    family = str(manifest.get("fontFamily", "GopeedIcons"))
    units_per_em = parse_number(manifest.get("unitsPerEm", 1024), "unitsPerEm")
    ascent = parse_number(manifest.get("ascent", 896), "ascent")
    descent = parse_number(manifest.get("descent", -128), "descent")
    padding = parse_number(manifest.get("padding", 96), "padding")
    if ascent <= 0 or descent >= 0 or ascent - descent != units_per_em:
        raise ValueError("ascent must be positive, descent negative, and their span must equal unitsPerEm")
    if padding < 0 or padding * 2 >= units_per_em:
        raise ValueError("padding must leave positive drawing space")

    glyph_order = [".notdef", *(str(icon["glyphName"]) for icon in icons)]
    glyphs = {}
    metrics = {".notdef": (units_per_em, 0)}
    empty_pen = TTGlyphPen(None)
    glyphs[".notdef"] = empty_pen.glyph()

    for icon in icons:
        source = icon["sourcePath"]
        assert isinstance(source, Path)
        svg_text = source.read_text(encoding="utf-8")
        min_x, min_y, width, height = parse_view_box(svg_text, source)
        available = units_per_em - padding * 2
        scale = min(available / width, available / height)
        glyph_width = width * scale
        glyph_height = height * scale
        left = (units_per_em - glyph_width) / 2
        bottom = descent + (units_per_em - glyph_height) / 2
        transform = (scale, 0, 0, -scale, left - scale * min_x, bottom + glyph_height + scale * min_y)
        glyph_pen = TTGlyphPen(None)
        quadratic_pen = Cu2QuPen(glyph_pen, max_err=1.0, reverse_direction=False)
        transformed_pen = TransformPen(quadratic_pen, transform)
        SVGPath.fromstring(svg_text).draw(transformed_pen)
        glyphs[str(icon["glyphName"])] = glyph_pen.glyph()

        bounds_pen = BoundsPen(None)
        SVGPath.fromstring(svg_text).draw(TransformPen(bounds_pen, transform))
        if bounds_pen.bounds is None:
            raise ValueError(f"{source}: icon produced no drawable bounds")
        metrics[str(icon["glyphName"])] = (units_per_em, round(bounds_pen.bounds[0]))

    builder = FontBuilder(units_per_em, isTTF=True)
    builder.setupGlyphOrder(glyph_order)
    builder.setupCharacterMap({int(icon["codepoint"]): str(icon["glyphName"]) for icon in icons})
    builder.setupGlyf(glyphs)
    builder.setupHorizontalMetrics(metrics)
    builder.setupHorizontalHeader(ascent=ascent, descent=descent)
    version = str(manifest.get("version", "1.0.0"))
    postscript_name = re.sub(r"[^A-Za-z0-9-]", "", family.replace(" ", "-")) or "GopeedIcons"
    builder.setupNameTable(
        {
            "familyName": family,
            "styleName": "Regular",
            "uniqueFontIdentifier": f"{family};{version}",
            "fullName": family,
            "psName": postscript_name,
            "version": f"Version {version}",
        }
    )
    builder.setupOS2(
        sTypoAscender=ascent,
        sTypoDescender=descent,
        usWinAscent=ascent,
        usWinDescent=-descent,
        usWeightClass=400,
        usWidthClass=5,
    )
    builder.setupPost()
    builder.setupMaxp()

    output = (manifest_dir / str(manifest["fontOutput"])).resolve()
    output.parent.mkdir(parents=True, exist_ok=True)
    builder.save(output)
    return output


def write_dart(manifest: dict[str, object], manifest_dir: Path, icons: list[dict[str, object]]) -> Path:
    family = str(manifest.get("fontFamily", "GopeedIcons"))
    class_name = str(manifest.get("dartClass", "GopeedIcons"))
    if not DART_NAME_PATTERN.fullmatch(class_name):
        raise ValueError("dartClass must be a valid Dart class name")
    package = manifest.get("fontPackage")
    package_argument = "" if package in (None, "") else f", fontPackage: '{package}'"
    lines = [
        "// Generated by gopeed-file-category-icons. Do not edit by hand.",
        "import 'package:flutter/widgets.dart';",
        "",
        f"abstract final class {class_name} {{",
        f"  static const String _fontFamily = '{family}';",
    ]
    for icon in icons:
        lines.append(
            f"  static const IconData {icon['dartName']} = IconData(0x{int(icon['codepoint']):04x}, fontFamily: _fontFamily{package_argument});"
        )
    lines.extend(["}", ""])
    output = (manifest_dir / str(manifest["dartOutput"])).resolve()
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text("\n".join(lines), encoding="utf-8")
    return output


def main() -> int:
    args = parse_args()
    manifest_path = args.manifest.resolve()
    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    if allocate_codepoints(manifest, args.allocate_codepoints):
        manifest_path.write_text(json.dumps(manifest, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
    manifest_dir = manifest_path.parent
    icons = validate_icons(manifest, manifest_dir)
    for required in ("fontOutput", "dartOutput"):
        if not isinstance(manifest.get(required), str) or not manifest[required]:
            raise ValueError(f"manifest.{required} must be a path")
    font_output = build_font(manifest, manifest_dir, icons)
    dart_output = write_dart(manifest, manifest_dir, icons)
    print(f"Built {len(icons)} icons")
    print(f"Font: {font_output}")
    print(f"Dart: {dart_output}")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (KeyError, OSError, ValueError, json.JSONDecodeError) as error:
        print(f"error: {error}", file=sys.stderr)
        raise SystemExit(1)
