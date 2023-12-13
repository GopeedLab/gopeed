[![Test Status](https://github.com/GopeedLab/gopeed/workflows/test/badge.svg)](https://github.com/GopeedLab/gopeed/actions?query=workflow%3Atest)
[![Codecov](https://codecov.io/gh/GopeedLab/gopeed/branch/main/graph/badge.svg)](https://codecov.io/gh/GopeedLab/gopeed)
[![Release](https://img.shields.io/github/release/GopeedLab/gopeed.svg)](https://github.com/GopeedLab/gopeed/releases)
[![Download](https://img.shields.io/github/downloads/GopeedLab/gopeed/total.svg)](https://github.com/GopeedLab/gopeed/releases)
[![Donate](https://img.shields.io/badge/%24-donate-ff69b4.svg)](https://docs.gopeed.com/donate.html)
[![Discord](https://img.shields.io/discord/1037992631881449472?label=Discord&logo=discord&style=social)](https://discord.gg/ZUJqJrwCGB)

![](_docs/img/banner.png)

[![ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/R6R6IJGN6)

[English](/README.md) | [中文](/README_zh-CN.md) | [日本語](/README_ja-JP.md) | [正體中文](/README_zh-TW.md)

## 簡介

Gopeed（全稱 Go Speed），是一款使用`Golang`+`Flutter`編寫的高速下載軟體，支援（HTTP、BitTorrent、Magnet）協定，同時支援所有的平台。

前往 ✈ [主頁](https://gopeed.com/zh-CN) | 📖 [文檔](https://docs.gopeed.com/zh/)

## 安裝

**已支援的平台**

- [x] Windows
- [x] MacOS
- [x] Linux
- [x] Android
- [x] iOS
- [x] Web
- [x] Docker

[下載](https://github.com/GopeedLab/gopeed/releases/latest)

> 註：MacOS 版運行時若提示損壞，請在終端中執行 `xattr -d com.apple.quarantine /Applications/Gopeed.app`

### 使用CLI安裝

使用`go install`安裝：

```bash
go install github.com/GopeedLab/gopeed/cmd/gopeed@latest
```

### Docker

#### 直接執行

```bash
docker run -d -p 9999:9999 -v /path/to/download:/root/Downloads -v /path/to/storage:/app/storage liwei2633/gopeed
```

[更多關於 Docker 的使用](https://docs.gopeed.com/zh/install.html#docker-%E5%AE%89%E8%A3%85)

#### 使用 Docker Compose

```bash
docker-compose up -d
```

#### 訪問服務

當 docker 容器運作時，可以通過 `http://localhost:9999` 訪問 web 頁面。

## 贊助

如果你認為該項目對你有所幫助，請考慮[贊助](https://docs.gopeed.com/zh/donate)以支持該項目的持續發展，謝謝！

## 軟體介面

![](_docs/img/ui-demo.png)

## 開發

該項目分為前端與後端，前端使用`flutter`編寫，後端使用`Golang`編寫，兩邊通過`http`協定進行通訊，在 unix 系統下，則使用`unix socket`，在 windows 系統下，則使用`tcp`協定。

> 前端代碼位於`ui/flutter`目錄內。

### 開發環境

1. Golang 1.19+
2. Flutter 3.10+

### 克隆項目

```bash
git clone git@github.com:GopeedLab/gopeed.git
```

### 協助開發

請參考[協助指南](CONTRIBUTING_zh-TW.md)

### 編譯

#### 桌面端

首先需要按照[flutter desktop 官方文檔](https://docs.flutter.dev/development/platform-integration/desktop)配置開發環境，並準備好`cgo`環境，具體方法可以自行搜索。

組建指令：

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

#### 移動設備

需要`cgo`環境，並安裝`gomobile`：

```bash
go install golang.org/x/mobile/cmd/gomobile@latest
go get golang.org/x/mobile/bind
gomobile init
```

組建指令：

- android

```bash
gomobile bind -tags nosqlite -ldflags="-w -s" -o ui/flutter/android/app/libs/libgopeed.aar -target=android -androidapi 19 -javapkg=com.gopeed github.com/GopeedLab/gopeed/bind/mobile
cd ui/flutter
flutter build apk
```

- ios

```bash
gomobile bind -tags nosqlite -ldflags="-w -s" -o ui/flutter/ios/Frameworks/Libgopeed.xcframework -target=ios github.com/GopeedLab/gopeed/bind/mobile
cd ui/flutter
flutter build ios --no-codesign
```

#### 網頁端

組建指令：

```bash
cd ui/flutter
flutter build web
cd ../../
rm -rf cmd/web/dist
cp -r ui/flutter/build/web cmd/web/dist
go build -tags nosqlite,web -ldflags="-s -w" -o bin/ github.com/GopeedLab/gopeed/cmd/web
```

## 感謝

### 貢獻者

<a href="https://github.com/GopeedLab/gopeed/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=GopeedLab/gopeed" />
</a>

### JetBrains

[![goland](_docs/img/goland.svg)](https://www.jetbrains.com/?from=gopeed)

## 軟體許可

該軟體遵循 [GPLv3](LICENSE) 。
