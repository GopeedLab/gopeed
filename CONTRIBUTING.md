# Gopeed contributors guide

Firstly, thank you for your interest in contributing to Gopeed. This guide will help you better
participate in the development of Gopeed.

## Branch description

This project only has one main branch, namely the `main` branch. If you want to participate in the
development of Gopeed, please fork this project first, and then develop in your fork project. After
development is completed, submit a PR to this project and merge it into the `main` branch.

## Local development

It is recommended to develop and debug through the web. First, start the backend service, and start
it by the command line `go run cmd/api/main.go`, the default port of the service is `9999`, and then
start the front-end flutter project in `debug` mode to run.

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

## flutter development

Don't forget to run`dart format ./ui/flutter`before you commit to keep your code in standard dart format

Turn on build_runner watcher if you want to edit api/models:

```
flutter pub run build_runner watch
```
