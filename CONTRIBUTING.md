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

The internationalization files of Gopeed are located in the `ui/flutter/assets/locales` directory.
You only need to add the corresponding language file in this directory.

Generate locales after you edit locales:

```
get generate locales 
```

Please refer to `en_us.dart` for translation.

## flutter development

Don't forget to run`dart format .`before you commit to keep your code in standard dart format

Turn on build_runner watcher if you want to edit api/models:

```
flutter pub run build_runner watch
```


get-cli commands usages:

```
 create:  
    controller:  Generate controller
    page:  Use to generate pages
    view:  Generate view
  generate:
    locales:  Generate translation file from json files
    model:  generate Class model from json
```
