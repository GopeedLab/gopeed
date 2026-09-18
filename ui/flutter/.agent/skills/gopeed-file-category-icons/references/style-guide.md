# Gopeed file-category icon style

## Composition

- Unknown file: empty file shell.
- Known file: file shell plus one classification mark.
- Generic folder: empty folder shell.
- Known or protocol folder: folder shell plus one classification mark.
- Status and actions never belong inside a classification glyph.

## Geometry

- Use a `0 0 24 24` viewBox and keep the optical drawing inside `2…22` on both axes.
- Use the current shared shell as the source of truth. The established file shell spans roughly `x=3…21` with a 1.5-unit outline; do not silently introduce a heavier one-off border.
- Keep decorated marks approximately inside `x=6.2…17.8` and `y=9.2…19`. Preserve at least 1.5 units of visible separation from the shell wherever its interior edge is closer.
- Prefer radii around 1.5–2 units, consistent terminal shapes, and simple silhouettes. Avoid details or gaps below 1.25 units.
- Center by optical mass rather than bounding-box arithmetic when the shape has a tab, arrow, or asymmetric label.
- Use non-zero winding for counters and holes; reverse inner-contour direction where a cutout is required.

## Classification marks

- Use one dominant mark, not several miniature objects.
- Prefer semantic geometry: play for video, note for audio, image silhouette for images, grid for spreadsheets, a zipper for archive/compressed files, and a cylinder for databases.
- Avoid lettering when a semantic mark is clearer. When a conventional label such as PDF is retained, build each letter as filled geometry with at least 1.25-unit stems and explicit inter-letter gaps.
- Keep an empty file truly empty. A question mark, star, or extension text changes its meaning.

## Protocol-marked folders

- Keep the folder silhouette recognizable before adding the protocol symbol.
- Place a simple filled protocol symbol in the lower folder body with clear space from every edge.
- Prefer semantic marks such as a magnet/swarm for BitTorrent or a donkey/server-network mark for ED2K. Do not default to acronym lettering.
- Let the visible folder occupy about 80–85% of the 24-unit canvas. Avoid adding generator padding on top of the SVG optical margin; double padding makes Flutter icons look undersized.
- Compare optical center and occupied area against Flutter's `folder_outlined` at the actual widget size, not only in an enlarged preview.

## Consistency checks

- Compare new icons beside at least one existing folder icon and one file-type icon.
- Inspect the generated contact sheet at 16, 20, 24, and 32 px on light and dark swatches.
- Reject icons that depend on color, contain live text/strokes, have inconsistent baseline/weight, drift off center, become ambiguous at 16 px, or leave less than the shared safe-area separation from the shell.
- Compare decorated icons against both empty shells. Interior marks must not make the shell look heavier by merging into its border.

## Flutter usage

Register the generated TTF under `flutter.fonts` using the exact `fontFamily` from the manifest. Use the generated Dart constants rather than raw codepoints. Render icon-font glyphs with `Icon`; do not use `Text` because icon sizing and semantics should follow Flutter's icon pipeline.
