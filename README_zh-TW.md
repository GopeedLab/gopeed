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

## 簡介

Gopeed（全稱 Go Speed），是一款使用`Golang`+`Flutter`編寫的高速下載軟體，支援（HTTP、BitTorrent、Magnet）協定，同時支援所有的平台。

前往 ✈ [主頁](https://gopeed.com/zh-CN) | 📖 [文檔](https://docs.gopeed.com/zh/)

## 下載

<table>
    <thead>
        <tr>
            <th>運行平台</th>
            <th>安裝包類型</th>
            <th>下載鏈接</th>
        </tr>
    </thead>
    <tbody>
        <tr>
            <td rowspan=2>Windows</td>
            <td><code>EXE Installer</code></td>
            <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-windows-amd64.zip">前往</a></td>
        </tr>
        <tr>
            <td><code>Portable ZIP</code></td>
            <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-windows-amd64-portable.zip">前往</a></td>
        </tr>
        <tr>
            <td>MacOS</td>
            <td><code>DMG Installer</code></td>          
            <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-macos.dmg">前往</a></td>
        </tr>
        <tr>
            <td rowspan=4>Linux</td>
            <td><code>Flathub</code></td>
            <td><a href="https://flathub.org/apps/com.gopeed.Gopeed">前往</a></td>
        </tr>
        <tr>
            <td><code>SNAP</code></td>
            <td><a href="https://snapcraft.io/gopeed">前往</a></td>
        </tr>
        <tr>
            <td><code>DEB</code></td>
            <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-linux-amd64.deb">前往</a></td>
        </tr>
        <tr>
            <td><code>AppImage</code></td>
            <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-linux-x86_64.AppImage">前往</a></td>
        </tr>
        <tr>
            <td>Android</td>
            <td><code>APK</code></td>
            <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-android.apk">前往</a></td>
        </tr>
        <tr>
            <td>iOS</td>
            <td><code>IPA</code></td>
            <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-ios.ipa">前往</a></td>
        </tr>
        <tr>
            <td>Web</td>
            <td></td>
            <td><a href="https://github.com/GopeedLab/gopeed/releases/latest">前往</a></td>
        </tr>
        <tr>
            <td>Docker</td>
            <td></td>
            <td><a href="https://hub.docker.com/r/liwei2633/gopeed">前往</a></td>
        </tr>
    </tbody>
</table>

更多關於安裝的內容請參考[安裝文檔](https://docs.gopeed.com/zh/install.html)

### 使用CLI安裝

使用`go install`安裝：

```bash
go install github.com/GopeedLab/gopeed/cmd/gopeed@latest
```

## 贊助

如果你認為該項目對你有所幫助，請考慮[贊助](https://docs.gopeed.com/zh/donate)以支持該項目的持續發展，謝謝！

## 軟體介面

![](_docs/img/ui-demo.png)

## 開發

該項目分為前端與後端，前端使用`flutter`編寫，後端使用`Golang`編寫，兩邊通過`http`協定進行通訊，在 unix 系統下，則使用`unix socket`，在 windows 系統下，則使用`tcp`協定。

> 前端代碼位於`ui/flutter`目錄內。

### 開發環境

1. Golang 1.22+
2. Flutter 3.16+

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
