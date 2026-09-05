# AGENTS.md

This document defines the implementation standards for this Flutter project.

## 1) Product Scope

- This is a cross-platform app that must support both desktop and mobile.
- Every page must use responsive layouts and behave correctly across:
  - Mobile phone sizes
  - Tablet sizes
  - Desktop window sizes, including resizable windows
- Desktop is currently the primary design target, but the architecture must remain ready for future mobile-specific variants.

## 2) Required Project Structure

Use the following structure as the baseline and do not introduce alternative top-level patterns:

```text
lib/
├── main.dart
├── app/
│   ├── app.dart
│   ├── router/
│   │   ├── app_router.dart
│   │   └── shells/
│   │       └── main_shell.dart
│   └── application/
│       └── locale_controller.dart
├── core/
│   ├── network/
│   ├── error/
│   └── utils/
├── shared/
│   └── widgets/
├── features/
│   ├── auth/
│   ├── tasks/
│   ├── extensions/
│   └── settings/
└── l10n/
    ├── app_en.arb
    ├── app_zh.arb
    └── app_zh_TW.arb
```

## 2.1) Core Tech Stack

Use the following as the baseline stack for this project:

- `flutter_riverpod`
- `go_router`
- `flutter_localizations`
- `dio`
- `window_manager`
- `tray_manager`
- `desktop_multi_window`
- `shadcn_flutter`

Guidelines:

- Do not introduce alternative frameworks for the same responsibility unless explicitly approved.
- Keep app-level state aligned with Riverpod patterns when new shared state is introduced.
- Keep navigation centralized in `lib/app/router/`.
- Keep network access built on top of Dio abstractions in `lib/core/network/`.
- Keep desktop window, tray, and multi-window logic encapsulated and platform-guarded.
- `shadcn_flutter` is the default product UI system for app surfaces and controls.

## 3) UI Design Rules

- Maintain a consistent visual language across desktop and mobile.
- Define and reuse design tokens for color, spacing, radius, typography, elevation, and motion.
- Avoid hard-coded one-off styles in page files; prefer centralized tokens and shared primitives.
- Preserve accessibility and usable contrast in both light and dark themes.
- Respect platform ergonomics:
  - Mobile: larger tap areas, reduced density, simpler layouts
  - Desktop: hover states, richer density, resize-aware layouts, window-aware composition

## 3.1) Theme Extension Rules

- Custom app colors must be exposed through Flutter `ThemeExtension`, not scattered as widget-local hard-coded values.
- Keep the app's semantic color tokens centralized in the shared theme layer under `lib/shared/theme/`.
- Widgets should read custom colors via shared theme accessors such as `AppPalette.of(context)` or `Theme.of(context).extension<...>()`, not by repeating raw hex values.
- Raw `Color(0x...)` values are allowed only inside the shared theme definition layer, platform interop code, or when a plugin API requires a direct color constant.
- When a new component needs a design-specific color, first promote it to a semantic theme token such as `taskCardHover`, `headerDivider`, or `captionButtonHover` instead of adding another local constant.
- If both `shadcn_flutter` theme data and Flutter `ThemeData` are used, keep them mapped from the same source-of-truth tokens so light and dark mode stay aligned.

## 3.2) Design Source of Truth

- When a screen or component exists under `design/`, Flutter must match its layout, spacing, hierarchy, and motion behavior before introducing local reinterpretation.
- Preserve the app's existing light and dark palette values as the canonical color source of truth even if the H5 prototype uses different colors.
- Do not copy H5 colors blindly. Rebuild structure and interaction fidelity using the Flutter theme tokens.

## 3.3) shadcn_flutter Rules

- Use `shadcn_flutter` as the default UI component library for product-facing surfaces.
- Use the `shadcn_flutter` documentation reference at:
  - [llms-full.txt](https://sunarya-thito.github.io/shadcn_flutter/llms-full.txt)
- Prefer `import 'package:shadcn_flutter/shadcn_flutter.dart';` as the primary UI import form.
- `shadcn_flutter` should have the highest import priority in product UI files. Keep it unaliased by default, and only add aliases to secondary packages when needed to avoid symbol conflicts or to clarify source ownership.
- Prefer `ShadcnApp` / `ShadcnApp.router` at the app shell level.
- Prefer `shadcn_flutter` controls such as buttons, inputs, tabs, dividers, and overlays over Material-style product UI widgets.
- Material may remain only where required for platform/plugin interop or Flutter internals, not as the default product design system.
- Product-facing interactive widgets must not use Flutter Material components such as `TextField`, `SnackBar`, `AlertDialog`, `showDialog`, `TextButton`, `ElevatedButton`, `OutlinedButton`, `IconButton`, `Drawer`, or similar Material UI controls when an equivalent `shadcn_flutter` primitive exists.
- A `material.dart` import is acceptable only for low-level primitives such as `Icons`, `Colors`, `ChangeNotifier`, geometry, painting, focus, text editing, or platform/plugin interop. It must not be used as a shortcut to build product UI controls.
- Tooltips are a documented exception to the Material-component restriction: always use the shared `AppTooltip` from `lib/shared/widgets/app_tooltip.dart`. It intentionally wraps Flutter's Material `Tooltip` because `shadcn_flutter`'s tooltip does not inherit the app's mixed Shadcn/Material theme consistently, which can produce a transparent-looking background and oversized text.
- Never instantiate either Flutter's `Tooltip` or `shadcn_flutter`'s `Tooltip` directly in feature/page code. Keep tooltip messages localized and route all product tooltip styling and behavior through `AppTooltip`.
- When auditing UI code, distinguish between:
  - acceptable Material foundation usage
  - forbidden Material product-component usage
- If a screen already uses `shadcn_flutter`, do not mix in Material product widgets for convenience. Replace them with `shadcn_flutter` equivalents or shared wrappers built on top of `shadcn_flutter`.

## 3.4) Desktop Design Metrics

Use the following desktop reference metrics unless a specific screen requires a documented exception:

- Reference window: `1024 x 768`
- Primary rail width: `64px`
- Secondary filter sidebar width: `256px`
- Main content header height: `80px`
- Window drag header height on Windows: `30px`
- Task row height: `72px`
- Standard control radius: `4px`
- Window / overlay radius: `12px`
- Spacing rhythm: `4 / 8 / 12 / 16 / 24 / 32`

## 3.5) Motion Rules

- Motion is part of the spec, not decoration.
- Recreate the design interactions when present in `design/`, including:
  - Progress shimmer / specular sweep
  - Indeterminate progress motion
  - Animated speed gauges / meters
  - Drawer slide-in transitions
  - Hover and selection transitions
  - Batch-mode reveal transitions
- Centralize durations, curves, and motion constants whenever motion becomes shared across multiple widgets.

## 4) Responsive Layout Rules

- All screens must be responsive by default.
- Never build a page for a single fixed size only.
- Use adaptive patterns such as:
  - `LayoutBuilder`
  - breakpoints
  - `Flexible` / `Expanded`
  - desktop and mobile variants when needed
- Verify every major page on at least:
  - small mobile width
  - medium / tablet width
  - desktop width

## 5) Component Decomposition Rules

### 5.1) Split aggressively into small files

- If a child widget is reusable or has an independent UI responsibility, move it to a separate file.
- Do not keep large page files with many inline private widgets if those sections can be extracted cleanly.
- Keep widgets focused and single-purpose.

### 5.2) Two levels of component abstraction

1. App-shared, non-business components
   - Place in `lib/shared/widgets/`
   - Examples: buttons, headers, chips, cards, window chrome, reusable form controls
   - Must stay business-agnostic and design-system aligned

2. Feature or page-specific components
   - Place close to usage inside the feature module
   - Recommended paths:
     - `lib/features/<feature>/presentation/pages/`
     - `lib/features/<feature>/presentation/widgets/`

### 5.3) Separation guidance

- Shared first when the component is used by multiple features or is purely design-system UI.
- Feature-local when the component is tightly coupled to one feature's state or interaction model.
- Promote from feature-local to shared only when reuse is real, not hypothetical.

### 5.4) Dual-platform component pattern

When mobile and desktop variants differ significantly, do not keep both branches inside one large widget file.

Required pattern:

```text
lib/features/<feature>/presentation/widgets/<component>/
├── <component>.dart
├── <component>_mobile.dart
└── <component>_desktop.dart
```

Rules:

- Only import the facade file from pages or sibling widgets.
- Keep business logic outside the mobile and desktop view files when possible.
- If a shared component also needs this split, apply the same pattern under `lib/shared/widgets/<component>/`.

## 6) Desktop Window and Chrome Rules

- Do not simulate OS switching inside the app UI.
- Adapt window chrome to the actual running platform:
  - macOS: prefer native system chrome behavior
  - Windows: use custom frameless chrome
  - Linux: prefer native behavior unless custom frameless support is explicitly verified as stable
- The desktop window header / chrome must be implemented as a reusable shared component so it can be reused by:
  - the main window
  - the create-task child window
  - future child windows
- Keep desktop window infrastructure in `lib/core/window/`.
- Keep reusable visual window header components under `lib/shared/widgets/window/` or an equivalent shared path.

## 6.1) Cross-Window Capability Rules

- Follow the mandatory architecture in [`docs/window-capability-rpc.md`](docs/window-capability-rpc.md) for all communication between the main window and desktop child windows.
- The main window is the sole owner of the Gopeed runtime, Gopeed API connection, and Hive database.
- Child windows must use `AppCapabilities`; they must not initialize Gopeed, open Hive, import `lib/api/api.dart`, or access `Database.instance`.
- Add new cross-window operations through shared `RpcMethod` descriptors and the central codec/registry. Do not add feature-specific MethodChannels or duplicate host/client serialization wrappers.
- Main-to-child state updates must use centrally declared events with complete state snapshots.

## 7) Page Implementation Standard

For each page:

- Keep the page entry widget in a page file.
- Move each meaningful sub-section into dedicated widget files.
- Keep business state and side effects outside pure presentational widgets where practical.
- Keep responsive logic explicit and readable.
- Reuse shared tokens and shared window/header primitives instead of rebuilding them locally.
- Desktop pages and desktop child windows must reserve `AppDesignTokens.windowHeaderHeight` at the top of their root content surface before the first visible content starts. This padding is owned by the page/window body itself, including create-task child windows, rather than being injected by a global shell.

## 8) Naming and Organization

- Use clear, intention-revealing names.
- Favor suffixes:
  - `...Page`
  - `...Section`
  - `...Card`
  - `...Tile`
  - `...Button`
  - `...Header`
- Avoid placeholder names such as `Widget1`, `TempView`, or `CommonBox`.

## 9) Localization and Text

- All user-facing natural-language strings must be localizable. This includes widget copy, validation and error messages, empty states, dialogs, tooltips, accessibility labels, tray menus, desktop notifications, Android foreground-service notifications, and child-window copy.
- `lib/l10n/app_en.arb` is the source template. Add each new message to every `lib/l10n/app_<locale>.arb` catalog, keeping the same key set in every locale. Prefer a base locale such as `app_zh.arb`; add a region-qualified catalog such as `app_zh_TW.arb` only when it has a distinct selectable translation.
- In widgets, read messages through `context.l10n`. In non-widget or background code, resolve the configured/system locale through the shared helpers in `lib/l10n/l10n.dart`; do not introduce a separate translation lookup mechanism.
- Translate message values only, and never rename variables such as `{count}` or `{name}`. In a message like `{count, plural, =1{1 file} other{{count} files}}`, translate the user-visible branch text while keeping `count`, `plural`, branch selectors such as `=1`/`other`, and the braces valid. Preserve the corresponding ARB placeholder metadata, and do not translate metadata keys prefixed with `@`.
- Prefer parameterized messages over string concatenation when word order can vary by language. Give placeholders explicit metadata and Dart types in the English ARB when appropriate.
- Do not edit generated `app_localizations*.dart` files. Regenerate them from ARB sources with `flutter gen-l10n`.
- Language names in the locale selector should remain native autonyms. Protocol names, product names, URLs, version strings, and dynamic data do not need translation unless they form part of natural-language copy.
- Design for longer translations and right-to-left locales: avoid fixed text widths, use flexible/wrapping layouts where needed, and do not encode direction with left/right-only assumptions.
- Do not hard-code production UI copy directly in widgets. Prototype-only text must be localized before the feature is considered complete.
- Before completing localization work, run from `ui/flutter`:

```sh
dart run ../../.github/workflows/scripts/check_l10n.dart
flutter gen-l10n
flutter analyze
flutter test
```

## 10) Quality Gate

Before merging major UI work:

- Verify desktop and mobile responsive behavior.
- Verify alignment and spacing consistency with shared tokens.
- Verify widget decomposition follows shared versus feature-local rules.
- Verify no obvious style duplication remains in page files where shared components are appropriate.
- Verify desktop window chrome behavior is correct for the target platform.
- Verify screens that exist in `design/` still match the intended layout and motion behavior.
