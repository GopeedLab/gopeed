---
name: gopeed-file-category-icons
description: Design or revise Gopeed file-classification icons as one cohesive SVG-to-Flutter icon font. Use for file types, generic and protocol-marked folders, unknown files, and extension-to-icon taxonomy; do not use for launcher, navigation, action, or status icons.
---

# Gopeed File Category Icons

Maintain Gopeed's file-classification visual language and its deterministic SVG → TTF → Flutter `IconData` pipeline.

The canonical editable materials live in `design/file-category-icons/`, outside the Flutter project. Read its `README.md` before changing any source. Do not add that directory to Flutter assets.

## Classification grammar

- A directory is always a folder outline. Add a classification or protocol mark only when one is known; otherwise leave the folder empty.
- A non-directory is always a file outline plus its classification mark.
- An unknown file is an empty file outline. Do not invent a generic interior mark.
- Transfer state is separate from classification. Never replace a file glyph with paused, failed, completed, or downloading artwork.
- Keep this skill scoped to file browsing, task assets, extension classification, and the font that renders those categories.

## Workflow

1. Read `design/file-category-icons/README.md`, then inspect its manifest, the classification mapping, real task-card size, and file-tree size. Read [references/style-guide.md](references/style-guide.md) for every visual revision.
2. Preserve the shared file or folder shell. Redraw only the classification mark unless the task explicitly revises the complete system.
3. Create 24×24 SVG sources containing only closed, filled `<path>` geometry. Convert strokes and lettering to paths; never leave `<text>`, live strokes, masks, clipping, filters, or external references.
4. Keep every decorated mark inside the documented safe area. Reject any mark that touches or visually merges with the shell at 16 or 20 px.
5. Preserve existing names and codepoints. Omit the codepoint only for a new category, then allocate it with the builder. Bump the font version for a visual revision.
6. Rebuild the complete TTF and generated Dart class, then generate the multi-size contact sheet.
7. Inspect 16, 20, 24, and 32 px in light and dark swatches. Check the actual Flutter task card and task-detail file tree when their sizes differ from the sheet.
8. Run the relevant classification tests, `flutter analyze`, and `flutter test` after integration changes.

## Tool setup

The scripts require the pinned packages in `scripts/requirements.txt`. Keep the environment local and untracked:

```bash
python3 -m venv .agent/skills/gopeed-file-category-icons/.venv
.agent/skills/gopeed-file-category-icons/.venv/bin/pip install -r .agent/skills/gopeed-file-category-icons/scripts/requirements.txt
```

Build and preview a manifest:

```bash
.agent/skills/gopeed-file-category-icons/.venv/bin/python .agent/skills/gopeed-file-category-icons/scripts/build_icon_font.py --manifest design/file-category-icons/manifest.json --allocate-codepoints
.agent/skills/gopeed-file-category-icons/.venv/bin/python .agent/skills/gopeed-file-category-icons/scripts/preview_icon_font.py --manifest design/file-category-icons/manifest.json
```

Use [assets/manifest.example.json](assets/manifest.example.json) and [assets/folder-symbol-template.svg](assets/folder-symbol-template.svg) only as structural starting points.

## Invariants

- Existing codepoints are API: never renumber or reuse one unless the user explicitly approves a breaking migration.
- `design/file-category-icons/` is the sole editable source of truth. Flutter keeps only the generated runtime TTF and Dart constants.
- Never add the design-material directory to `pubspec.yaml`; it must remain outside the packaged Flutter asset graph.
- The manifest is the source of truth. Do not hand-edit generated Dart constants or patch the binary font.
- Keep icon names lowercase `snake_case`; the generator emits lowerCamelCase Dart fields.
- File and folder shells in the same revision must share outline weight, baseline, fold/tab geometry, and optical scale.
- Prefer a recognizable classification symbol over lettering. If lettering is necessary, convert it to geometry and prove it remains legible at 16 px; PDF-like multi-letter marks need deliberate spacing rather than scaling text into the available area.
- New classification groups require both an extension/domain mapping and a glyph. Do not add a glyph that no classifier can reach.
- Rebuild the complete font after every source change so all platforms consume the same binary.
