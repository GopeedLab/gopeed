# Gopeed contributors guide

Firstly, thank you for your interest in contributing to Gopeed. This guide will help you better
participate in the development of Gopeed.

## Branches

Fork Gopeed, create a branch from the latest `main`, and open a pull request targeting `main`.

## Local development

Use Go 1.27.4 and Flutter 3.41.6 (Dart 3.11.4), matching the CI toolchains. For a quick development setup, start the Go REST API from the repository root.

```sh
go run -tags nosqlite ./cmd/api
```

The API listens on `127.0.0.1:9999`. In another terminal at the repository root, run the Flutter app in Chrome.

```sh
cd ui/flutter
flutter pub get
flutter gen-l10n
flutter run -d chrome
```

Desktop and mobile development requires building the Go bindings for the target platform. See the build instructions in the [README](README.md).

## Translation

Flutter translations are ARB files in `ui/flutter/lib/l10n`. Use `app_en.arb` as the source template,
then edit or add `app_<locale>.arb` (for example, `app_de.arb` or `app_zh_TW.arb`). Every locale must
contain the same message keys as the English template.

Translate message values only, and do not rename variables inside braces such as `{count}` and `{name}`.
Do not translate metadata keys prefixed with `@`. Submit only the ARB source files you changed; do not generate or commit
`app_localizations*.dart`. The pull-request CI automatically validates the catalogs, generates the
localization code, and checks its integration.

The following examples show both the ARB definition and the text users see after the app supplies values.

1. Insert a variable into a message

   ARB definition:

   ```json
   "welcomeUser": "Hola, {name}"
   ```

   Rendered result: when `name = Ana`, users see “Hola, Ana”. You may reorder the sentence, but keep `{name}`.

2. Display different text for different quantities

   ARB definition:

   ```json
   "fileCount": "{count, plural, =0{No hay archivos} =1{1 archivo} other{{count} archivos}}"
   ```

   Rendered results:

   - `count = 0` → “No hay archivos”
   - `count = 1` → “1 archivo”
   - `count = 3` → “3 archivos”

   `=0`, `=1`, and `other` mean zero, one, and every other quantity. Translate only the user-visible text
   inside each branch.

3. Display different text for different values

   ARB definition:

   ```json
   "taskState": "{state, select, running{En curso} paused{En pausa} other{Desconocido}}"
   ```

   Rendered results:

   - `state = running` → “En curso”
   - `state = paused` → “En pausa”
   - Any other value → “Desconocido”

   `running`, `paused`, and `other` are values supplied by the app. Do not translate them; translate only
   the display text inside the following braces.

## Code checks

Format changed Go files with `gofmt` and run the Go tests from the repository root.

```sh
go test -tags nosqlite ./...
```

Format changed Dart files with `dart format`. After installing Flutter dependencies and generating localizations as shown above, run the Flutter checks from the repository root.

```sh
cd ui/flutter
flutter analyze
flutter test
```

If you change annotated models in `lib/api/model/`, regenerate their JSON serializers from `ui/flutter`.

```sh
dart run build_runner build
```
