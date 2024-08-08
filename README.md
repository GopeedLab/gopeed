[![Test Status](https://github.com/GopeedLab/gopeed/workflows/test/badge.svg)](https://github.com/GopeedLab/gopeed/actions?query=workflow%3Atest)
[![Codecov](https://codecov.io/gh/GopeedLab/gopeed/branch/main/graph/badge.svg)](https://codecov.io/gh/GopeedLab/gopeed)
[![Release](https://img.shields.io/github/release/GopeedLab/gopeed.svg)](https://github.com/GopeedLab/gopeed/releases)
[![Download](https://img.shields.io/github/downloads/GopeedLab/gopeed/total.svg)](https://github.com/GopeedLab/gopeed/releases)
[![Donate](https://img.shields.io/badge/%24-donate-ff69b4.svg)](https://docs.gopeed.com/donate.html)
[![Discord](https://img.shields.io/discord/1037992631881449472?label=Discord&logo=discord&style=social)](https://discord.gg/ZUJqJrwCGB)

<a href="https://trendshift.io/repositories/7953" target="_blank"><img src="https://trendshift.io/api/badge/repositories/7953" alt="GopeedLab%2Fgopeed | Trendshift" style="width: 250px; height: 55px;" width="250" height="55"/></a>

![](_docs/img/banner.png)

[![ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/R6R6IJGN6)

[English](/README.md) | [中文](/README_zh-CN.md) | [日本語](/README_ja-JP.md) | [正體中文](/README_zh-TW.md) | [Tiếng Việt](/README_vi-VN.md)

## Introduction

Gopeed (full name Go Speed), a high-speed downloader developed by `Golang` + `Flutter`, supports (HTTP, BitTorrent, Magnet) protocol, and supports all platforms. In addition to basic download functions, Gopeed is also a highly customizable downloader that supports implementing more features through integration with [APIs](https://docs.gopeed.com/dev-api.html) or installation and development of [extensions](https://docs.gopeed.com/dev-extension.html).

Visit ✈ [Official Website](https://gopeed.com) | 📖 [Official Docs](https://docs.gopeed.com)

## Download

<table>
    <thead>
        <tr>
            <th>Platform</th>
            <th>Package Type</th>
            <th>Download Link</th>
        </tr>
    </thead>
    <tbody>
        <tr>
            <td rowspan=2>Windows</td>
            <td><code>EXE Installer</code></td>
            <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-windows-amd64.zip">Link</a></td>
        </tr>
        <tr>
            <td><code>Portable ZIP</code></td>
            <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-windows-amd64-portable.zip">Link</a></td>
        </tr>
        <tr>
            <td>MacOS</td>
            <td><code>DMG Installer</code></td>          
            <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-macos.dmg">Link</a></td>
        </tr>
        <tr>
            <td rowspan=4>Linux</td>
            <td><code>Flathub</code></td>
            <td><a href="https://flathub.org/apps/com.gopeed.Gopeed">Link</a></td>
        </tr>
        <tr>
            <td><code>SNAP</code></td>
            <td><a href="https://snapcraft.io/gopeed">Link</a></td>
        </tr>
        <tr>
            <td><code>DEB</code></td>
            <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-linux-amd64.deb">Link</a></td>
        </tr>
        <tr>
            <td><code>AppImage</code></td>
            <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-linux-x86_64.AppImage">Link</a></td>
        </tr>
        <tr>
            <td>Android</td>
            <td><code>APK</code></td>
            <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-android.apk">Link</a></td>
        </tr>
        <tr>
            <td>iOS</td>
            <td><code>IPA</code></td>
            <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-ios.ipa">Link</a></td>
        </tr>
        <tr>
            <td>Web</td>
            <td></td>
            <td><a href="https://github.com/GopeedLab/gopeed/releases/latest">Link</a></td>
        </tr>
        <tr>
            <td>Docker</td>
            <td></td>
            <td><a href="https://hub.docker.com/r/liwei2633/gopeed">Link</a></td>
        </tr>
    </tbody>
</table>

More about installation, please refer to [Installation](https://docs.gopeed.com/install.html)

### Command tool

use `go install`:

```bash
go install github.com/GopeedLab/gopeed/cmd/gopeed@latest
```

## Browser Extension

Gopeed also provides a browser extension to take over browser downloads, supporting browsers such as Chrome, Edge, Firefox, etc., please refer to: [https://github.com/GopeedLab/browser-extension](https://github.com/GopeedLab/browser-extension)

## Donate

If you like this project, please consider [donating](https://docs.gopeed.com/donate.html) to support the development of this project, thank you!

## Showcase

![](_docs/img/ui-demo.png)

## Development

This project is divided into two parts, the front end uses `flutter`, the back end uses `Golang`, and the two sides communicate through the `http` protocol. On the unix system, `unix socket` is used, and on the windows system, `tcp` protocol is used.

> The front code is located in the `ui/flutter` directory.

### Environment

1. Golang 1.21+
2. Flutter 3.16+

### Clone

```bash
git clone git@github.com:GopeedLab/gopeed.git
```

### Contributing

Please refer to [CONTRIBUTING.md](/CONTRIBUTING.md)

### Build

#### Desktop

First, you need to configure the environment according to the official [Flutter desktop website documention](https://docs.flutter.dev/development/platform-integration/desktop), then you will need to ensure the cgo environment is set up accordingly. For detailed instructions on setting up the cgo environment, please refer to relevant resources available online.

command:

- windows

```bash
go build -tags nosqlite -ldflags="-w -s" -buildmode=c-shared -o ui/flutter/windows/libgopeed.dll github.com/GopeedLab/gopeed/bind/desktop
cd ui/flutter
flutter build windows
```

- macos

```bash
go build -tags nosqlite -ldflags="-w -s" -buildmode=c-shared -o ui/flutter/macos/Frameworks/libgopeed.dylib github.com/GopeedLab/gopeed/bind/desktop
cd ui/flutter
flutter build macos
```

- linux

```bash
go build -tags nosqlite -ldflags="-w -s" -buildmode=c-shared -o ui/flutter/linux/bundle/lib/libgopeed.so github.com/GopeedLab/gopeed/bind/desktop
cd ui/flutter
flutter build linux
```

#### Mobile

Same as before, you also need to prepare the `cgo` environment, and then install `gomobile`:

```bash
go install golang.org/x/mobile/cmd/gomobile@latest
go get golang.org/x/mobile/bind
gomobile init
```

command:

- android

```bash
gomobile bind -tags nosqlite -ldflags="-w -s" -o ui/flutter/android/app/libs/libgopeed.aar -target=android -androidapi 21 -javapkg="com.gopeed" github.com/GopeedLab/gopeed/bind/mobile
cd ui/flutter
flutter build apk
```

- ios

```bash
gomobile bind -tags nosqlite -ldflags="-w -s" -o ui/flutter/ios/Frameworks/Libgopeed.xcframework -target=ios github.com/GopeedLab/gopeed/bind/mobile
cd ui/flutter
flutter build ios --no-codesign
```

#### Web

command:

```bash
cd ui/flutter
flutter build web
cd ../../
rm -rf cmd/web/dist
cp -r ui/flutter/build/web cmd/web/dist
go build -tags nosqlite,web -ldflags="-s -w" -o bin/ github.com/GopeedLab/gopeed/cmd/web
```

## Credits

### Contributors

<a href="https://github.com/GopeedLab/gopeed/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=GopeedLab/gopeed" />
</a>

### JetBrains

[![goland](_docs/img/goland.svg)](https://www.jetbrains.com/?from=gopeed)

## License

[GPLv3](LICENSE)
