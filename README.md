[![Test Status](https://github.com/monkeyWie/gopeed/workflows/test/badge.svg)](https://github.com/monkeyWie/gopeed/actions?query=workflow%3Atest)
[![Codecov](https://codecov.io/gh/monkeyWie/gopeed/branch/main/graph/badge.svg)](https://codecov.io/gh/monkeyWie/gopeed)
[![Release](https://img.shields.io/github/release/monkeyWie/gopeed.svg?style=flat-square)](https://github.com/monkeyWie/gopeed/releases)
[![Discord](https://img.shields.io/discord/1037992631881449472?label=Discord&logo=discord&style=social)](https://discord.gg/ZUJqJrwCGB)

![](_docs/img/banner.png)

[English](/README.md) | [中文](/README_zh-CN.md)

## Introduction

Gopeed is a high-speed downloader developed by `Golang`+`Flutter`, which supports (HTTP, BitTorrent, Magnet) protocol downloads and supports all platforms.

## Installation

**Supported platforms**

- [x] windows
- [x] macos
- [x] linux
- [x] android
- [ ] ios
- [x] web
- [x] docker

[To Release](https://github.com/monkeyWie/gopeed/releases/latest)

> Tips: If the macos open failed, please execute the `xattr -d com.apple.quarantine /Applications/Gopeed.app` command in the terminal

### Command tool

use `go install`:

```bash
go install github.com/monkeyWie/gopeed/cmd/gopeed
```

### Docker

```bash
docker run -d liwei2633/gopeed -v /path/to/download:/download -p 9999:9999
```

When the docker container is running, you can access the web page through `http://localhost:9999`.
> Tip: Modify the download path to `/download` on the setting page to access the downloaded files on the host.

## Showcase

![](_docs/img/ui-demo.png)

## Development

This project is divided into two parts, the front end uses `flutter`, the back end uses `Golang`, and the two sides communicate through the `http` protocol. On the unix system, `unix socket` is used, and on the windows system, `tcp` protocol is used.

> The front code is located in the `ui/flutter` directory.

### Environment

1. Golang 1.19+
2. Flutter 3.0+

### Clone

```bash
git clone git@github.com:monkeyWie/gopeed.git
```

### Build

#### Desktop

First, you need to configure the environment according to the [flutter desktop official website document](https://docs.flutter.dev/development/platform-integration/desktop), and then you need to prepare the `cgo` environment, which can be searched for yourself.

command:

- windows

```bash
go build -tags nosqlite -ldflags="-w -s" -buildmode=c-shared -o ui/flutter/windows/libgopeed.dll github.com/monkeyWie/gopeed/bind/desktop
cd ui/flutter
flutter build windows
```

- macos

```bash
go build -tags nosqlite -ldflags="-w -s" -buildmode=c-shared -o ui/flutter/macos/Frameworks/libgopeed.dylib github.com/monkeyWie/gopeed/bind/desktop
cd ui/flutter
flutter build macos
```

- linux

```bash
go build -tags nosqlite -ldflags="-w -s" -buildmode=c-shared -o ui/flutter/linux/bundle/lib/libgopeed.so github.com/monkeyWie/gopeed/bind/desktop
cd ui/flutter
flutter build linux
```

#### Mobile

Same as before, you also need to prepare the `cgo` environment, and then install `gomobile`:

```bash
go install golang.org/x/mobile/cmd/gomobile@latest
gomobile init
```

command:

- android

```bash
gomobile bind -tags nosqlite -ldflags="-w -s" -o ui/flutter/android/app/libs/libgopeed.aar -target=android -androidapi 19 -javapkg=com.gopeed github.com/monkeyWie/gopeed/bind/mobile
cd ui/flutter
flutter build apk
```

#### Web

Web platform communicates directly with the backend http server, no additional environment is required.

command:

```bash
cd ui/flutter
flutter build web
cd ../../
cp -r ui/flutter/build/web cmd/web/dist
go build -tags nosqlite,web -ldflags="-s -w" -o bin/ github.com/monkeyWie/gopeed/cmd/web
````

## License

[GPLv3](LICENSE)
