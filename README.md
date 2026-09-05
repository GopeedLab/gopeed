# [![](_docs/img/banner.svg)](https://gopeed.com)

[![Test Status](https://github.com/GopeedLab/gopeed/workflows/test/badge.svg)](https://github.com/GopeedLab/gopeed/actions?query=workflow%3Atest)
[![Codecov](https://codecov.io/gh/GopeedLab/gopeed/branch/main/graph/badge.svg)](https://codecov.io/gh/GopeedLab/gopeed)
[![Release](https://img.shields.io/github/release/GopeedLab/gopeed.svg)](https://github.com/GopeedLab/gopeed/releases)
[![Download](https://img.shields.io/github/downloads/GopeedLab/gopeed/total.svg)](https://github.com/GopeedLab/gopeed/releases)
[![Donate](https://img.shields.io/badge/%24-donate-ff69b4.svg)](https://gopeed.com/docs/donate)
[![WeChat](https://img.shields.io/badge/WeChat%20Official%20Account-07C160?logo=wechat&logoColor=white)](https://raw.githubusercontent.com/GopeedLab/gopeed/main/_docs/img/weixin.png)
[![Discord](https://img.shields.io/discord/1037992631881449472?label=Discord&logo=discord&style=social)](https://discord.gg/ZUJqJrwCGB)

<a href="https://trendshift.io/repositories/7953" target="_blank"><img src="https://trendshift.io/api/badge/repositories/7953" alt="GopeedLab%2Fgopeed | Trendshift" style="width: 250px; height: 55px;" width="250" height="55"/></a>

[English](/README.md) | [中文](/README_zh-CN.md) | [日本語](/README_ja-JP.md) | [正體中文](/README_zh-TW.md) | [Tiếng Việt](/README_vi-VN.md)

## 🚀 Introduction

Gopeed (short for **Go Speed**) is a fast, modern, free, and open-source download manager built with Go and Flutter. It supports HTTP, HTTPS, BitTorrent, magnet links, and ed2k on desktop, mobile, and the web.

Beyond core download management, Gopeed offers browser integration, JavaScript extensions, a REST API, a CLI, and a self-hosted web UI for customization and automation.

Visit ✈ [Official Website](https://gopeed.com)

![Application screenshot](_docs/img/ui-concept-en.png)

## ✨ Features

- ⚡ **High-speed downloads** — combine concurrent tasks, multi-connection HTTP transfers, and peer-to-peer BitTorrent downloads to make the most of your bandwidth.
- 🧲 **Multiple protocols** — download HTTP/HTTPS files, torrents, magnet links, and ed2k resources from a single app.
- 🌱 **Full-featured BitTorrent** — use DHT peer discovery, uTP transport, Web Seeds, selective file downloads, tracker management, peer and piece statistics, and ratio- or time-based seeding limits.
- 📋 **Flexible task management** — pause, resume, retry, run batch operations, search, filter by status, organize with categories, and recover tasks after a restart.
- 🪶 **Lightweight native experience** — the main interface is rendered natively with Flutter. No Electron. No WebView shell. Enjoy a smaller footprint, lower overhead, and responsive performance.
- 💻 **Cross-platform** — available for Windows, macOS, Linux, Android, iOS, and the web, with Docker and QNAP deployment options.
- 🎨 **Customizable appearance** — follow your system theme or choose light or dark mode, with eight accent colors.
- 📐 **Responsive interface** — task lists, navigation, settings, and detail views adapt to phones, tablets, and resizable desktop windows.
- 🗣️ **Available in 20+ languages** — including English, Simplified and Traditional Chinese, Japanese, Korean, and many more.
- 🌐 **Browser integration** — send downloads from Chrome, Edge, Firefox, and other compatible browsers directly to Gopeed.
- 🧩 **JavaScript extensions** — add support for video platforms, AI model hubs, cloud storage services, and other download sources.
- 🔌 **Automation-ready** — integrate with Gopeed through its REST API, CLI, authenticated web UI, webhooks, and post-download scripts.
- 🛠️ **Built-in essentials** — customize headers and the User-Agent, use proxies and GitHub mirrors, receive notifications, and extract archives automatically.

## ⬇️ Download

### 🧪 Gopeed 2.0.0 Beta

Gopeed 2.0.0 is currently in public beta. It introduces the new 2.0.0 experience, but some features may still be incomplete or unstable. Please try it and report any issues you encounter.

- [Download Gopeed 2.0.0 Beta 1](https://github.com/GopeedLab/gopeed/releases/tag/v2.0.0-beta.1)

Once the features and stability meet our release standards, we will publish the official Gopeed 2.0.0 release. Beta users will be able to upgrade directly to the final release, while existing stable users will not be automatically moved onto the beta channel.

### Stable release

- [Official Download](https://gopeed.com)
- [GitHub Releases](https://github.com/GopeedLab/gopeed/releases/latest)

### 🛠️ Command-line tool

Install the CLI with `go install`:

```bash
go install github.com/GopeedLab/gopeed/cmd/gopeed@latest
```

## 🔌 Browser Extension

Use the Gopeed browser extension to send downloads from Chrome, Edge, Firefox, and other compatible browsers directly to Gopeed: [GopeedLab/browser-extension](https://github.com/GopeedLab/browser-extension)

## 📱 WeChat Official Account

Follow Gopeed's official WeChat account for updates and news.

<img src="_docs/img/weixin.png" width="200" />

## 💝 Donate

If Gopeed is useful to you, please consider [supporting its development](https://gopeed.com/docs/donate). Thank you!

## 👨‍💻 Development

Gopeed consists of a Flutter front end and a Go back end. They communicate over HTTP, using Unix sockets on Unix-like systems and TCP on Windows.

> The front-end source is located in the `ui/flutter` directory.

### 🌍 Environment

1. Go 1.25+
2. Flutter 3.41+

### 📋 Clone

```bash
git clone git@github.com:GopeedLab/gopeed.git
```

### 🤝 Contributing

See [CONTRIBUTING.md](/CONTRIBUTING.md).

### 🏗️ Build

#### Desktop

Set up Flutter desktop development using the official [Flutter desktop guide](https://docs.flutter.dev/development/platform-integration/desktop), and make sure a working C toolchain is available for cgo. Then run the commands for your platform.

Commands:

- Windows

```bash
go build -tags nosqlite -ldflags="-w -s" -buildmode=c-shared -o ui/flutter/windows/libgopeed.dll github.com/GopeedLab/gopeed/bind/desktop
cd ui/flutter
flutter build windows
```

- macOS

```bash
go build -tags nosqlite -ldflags="-w -s" -buildmode=c-shared -o ui/flutter/macos/Frameworks/libgopeed.dylib github.com/GopeedLab/gopeed/bind/desktop
cd ui/flutter
flutter build macos
```

- Linux

```bash
go build -tags nosqlite -ldflags="-w -s" -buildmode=c-shared -o ui/flutter/linux/bundle/lib/libgopeed.so github.com/GopeedLab/gopeed/bind/desktop
cd ui/flutter
flutter build linux
```

#### Mobile

Mobile builds also require a working cgo toolchain. Install and initialize `gomobile`:

```bash
go install golang.org/x/mobile/cmd/gomobile@latest
go get golang.org/x/mobile/bind
gomobile init
```

Commands:

- Android

```bash
gomobile bind -tags nosqlite -ldflags="-w -s -checklinkname=0" -o ui/flutter/android/app/libs/libgopeed.aar -target=android -androidapi 21 -javapkg="com.gopeed" github.com/GopeedLab/gopeed/bind/mobile
cd ui/flutter
flutter build apk
```

- iOS

```bash
gomobile bind -tags nosqlite -ldflags="-w -s" -o ui/flutter/ios/Frameworks/Libgopeed.xcframework -target=ios github.com/GopeedLab/gopeed/bind/mobile
cd ui/flutter
flutter build ios --no-codesign
```

#### Web

Build the web app and server:

```bash
cd ui/flutter
flutter build web
cd ../../
rm -rf cmd/web/dist
cp -r ui/flutter/build/web cmd/web/dist
go build -tags nosqlite,web -ldflags="-s -w" -o bin/ github.com/GopeedLab/gopeed/cmd/web
```

## ❤️ Credits

### 👥 Contributors

<a href="https://github.com/GopeedLab/gopeed/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=GopeedLab/gopeed" />
</a>

### 🏢 JetBrains

[![goland](_docs/img/goland.svg)](https://www.jetbrains.com/?from=gopeed)

## 📄 License

[GPLv3](LICENSE)
