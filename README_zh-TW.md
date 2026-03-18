# [![](_docs/img/banner.svg)](https://gopeed.com)

[![Test Status](https://github.com/GopeedLab/gopeed/workflows/test/badge.svg)](https://github.com/GopeedLab/gopeed/actions?query=workflow%3Atest)
[![Codecov](https://codecov.io/gh/GopeedLab/gopeed/branch/main/graph/badge.svg)](https://codecov.io/gh/GopeedLab/gopeed)
[![Release](https://img.shields.io/github/release/GopeedLab/gopeed.svg)](https://github.com/GopeedLab/gopeed/releases)
[![Download](https://img.shields.io/github/downloads/GopeedLab/gopeed/total.svg)](https://github.com/GopeedLab/gopeed/releases)
[![Donate](https://img.shields.io/badge/%24-donate-ff69b4.svg)](https://gopeed.com/docs/donate)
[![WeChat](https://img.shields.io/badge/%E5%BE%AE%E4%BF%A1%E5%85%AC%E4%BC%97%E5%8F%B7-07C160?logo=wechat&logoColor=white)](https://raw.githubusercontent.com/GopeedLab/gopeed/main/_docs/img/weixin.png)
[![Discord](https://img.shields.io/discord/1037992631881449472?label=Discord&logo=discord&style=social)](https://discord.gg/ZUJqJrwCGB)

<a href="https://trendshift.io/repositories/7953" target="_blank"><img src="https://trendshift.io/api/badge/repositories/7953" alt="GopeedLab%2Fgopeed | Trendshift" style="width: 250px; height: 55px;" width="250" height="55"/></a>

[![ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/R6R6IJGN6)

[English](/README.md) | [中文](/README_zh-CN.md) | [日本語](/README_ja-JP.md) | [正體中文](/README_zh-TW.md) | [Tiếng Việt](/README_vi-VN.md)

## 🚀 簡介

Gopeed（全稱 Go Speed），是一款使用`Golang`+`Flutter`編寫的高速下載軟體，支援（HTTP、BitTorrent、Magnet、ED2K）協定，同時支援所有的平台。

前往 ✈ [主頁](https://gopeed.com/zh-CN) | 📖 [文檔](https://gopeed.com/docs)

## ⬇️ 下載

<table>
  <tbody>
    <tr>
      <td rowspan="2">🪟 Windows</td>
      <td><code>EXE</code></td>
      <td>amd64</td>
      <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-windows-amd64.zip">📥</a></td>
    </tr>
    <tr>
      <td><code>Portable</code></td>
      <td>amd64</td>
      <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-windows-amd64-portable.zip">📥</a></td>
    </tr>
    <tr>
      <td rowspan="3">🍎 MacOS</td>
      <td rowspan="3"><code>DMG</code></td>
      <td>universal</td>
      <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-macos.dmg">📥</a></td>
    </tr>
    <tr>
      <td>amd64</td>
      <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-macos-amd64.dmg">📥</a></td>
    </tr>
    <tr>
      <td>arm64</td>
      <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-macos-arm64.dmg">📥</a></td>
    </tr>
    <tr>
      <td rowspan="6">🐧 Linux</td>
      <td><code>Flathub</code></td>
      <td>amd64</td>
      <td><a href="https://flathub.org/apps/com.gopeed.Gopeed">📥</a></td>
    </tr>
    <tr>
      <td><code>SNAP</code></td>
      <td>amd64</td>
      <td><a href="https://snapcraft.io/gopeed">📥</a></td>
    </tr>
    <tr>
      <td rowspan="2"><code>DEB</code></td>
      <td>amd64</td>
      <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-linux-amd64.deb">📥</a></td>
    </tr>
    <tr>
      <td>arm64</td>
      <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-linux-arm64.deb">📥</a></td>
    </tr>
    <tr>
      <td rowspan="2"><code>AppImage</code></td>
      <td>amd64</td>
      <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-linux-amd64.AppImage">📥</a></td>
    </tr>
    <tr>
      <td>arm64</td>
      <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-linux-arm64.AppImage">📥</a></td>
    </tr>
    <tr>
      <td rowspan="4">🤖 Android</td>
      <td rowspan="4"><code>APK</code></td>
      <td>universal</td>
      <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-android.apk">📥</a></td>
    </tr>
     <tr>
      <td>armeabi-v7a</td>
      <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-android-armeabi-v7a.apk">📥</a></td>
    </tr>
     <tr>
      <td>arm64-v8a</td>
      <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-android-arm64-v8a.apk">📥</a></td>
    </tr>
    <tr>
      <td>x86_64</td>
      <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-android-x86_64.apk">📥</a></td>
    </tr>
    <tr>
      <td>📱 iOS</td>
      <td><code>IPA</code></td>
      <td>universal</td>
      <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-ios.ipa">📥</a></td>
    </tr>
    <tr>
      <td>🐳 Docker</td>
      <td>-</td>
      <td>universal</td>
      <td><a href="https://hub.docker.com/r/liwei2633/gopeed">📥</a></td>
    </tr>
    <tr>
      <td rowspan="2">💾 Qnap</td>
      <td rowspan="2"><code>QPKG</code></td>
      <td>amd64</td>
      <td><a href="https://gopeed.com/api/download?tpl=gopeed-$version-qnap-amd64.qpkg">📥</a></td>
    </tr>
    <tr>
      <td>arm64</td>
      <td><a href="https://gopeed.com/api/download?tpl=gopeed-$version-qnap-arm64.qpkg">📥</a></td>
    </tr>
    <tr>
      <td rowspan="8">🌐 Web</td>
      <td rowspan="3"><code>Windows</code></td>
      <td>amd64</td>
      <td><a href="https://gopeed.com/api/download?tpl=gopeed-web-$version-windows-amd64.zip">📥</a></td>
    </tr>
    <tr>
      <td>arm64</td>
      <td><a href="https://gopeed.com/api/download?tpl=gopeed-web-$version-windows-arm64.zip">📥</a></td>
    </tr>
    <tr>
      <td>386</td>
      <td><a href="https://gopeed.com/api/download?tpl=gopeed-web-$version-windows-386.zip">📥</a></td>
    </tr>
    <tr>
      <td rowspan="2"><code>MacOS</code></td>
      <td>amd64</td>
      <td><a href="https://gopeed.com/api/download?tpl=gopeed-web-$version-macos-amd64.zip">📥</a></td>
    </tr>
    <tr>
      <td>arm64</td>
      <td><a href="https://gopeed.com/api/download?tpl=gopeed-web-$version-macos-arm64.zip">📥</a></td>
    </tr>
    <tr>
      <td rowspan="3"><code>Linux</code></td>
      <td>amd64</td>
      <td><a href="https://gopeed.com/api/download?tpl=gopeed-web-$version-linux-amd64.zip">📥</a></td>
    </tr>
    <tr>
      <td>arm64</td>
      <td><a href="https://gopeed.com/api/download?tpl=gopeed-web-$version-linux-arm64.zip">📥</a></td>
    </tr>
    <tr>
      <td>386</td>
      <td><a href="https://gopeed.com/api/download?tpl=gopeed-web-$version-linux-386.zip">📥</a></td>
    </tr>
  </tbody>
</table>

更多關於安裝的內容請參考[安裝文檔](https://gopeed.com/docs/install)

### 🛠️ 使用 CLI 安裝

使用`go install`安裝：

```bash
go install github.com/GopeedLab/gopeed/cmd/gopeed@latest
```

## 📱 微信公眾號

關注公眾號獲取項目最新動態和資訊。

<img src="_docs/img/weixin.png" width="200" />

## 💝 贊助

如果你認為該項目對你有所幫助，請考慮[贊助](https://gopeed.com/docs/donate)以支持該項目的持續發展，謝謝！

## 🖼️ 軟體介面

![](_docs/img/ui-demo.png)

## 👨‍💻 開發

該項目分為前端與後端，前端使用`flutter`編寫，後端使用`Golang`編寫，兩邊通過`http`協定進行通訊，在 unix 系統下，則使用`unix socket`，在 windows 系統下，則使用`tcp`協定。

> 前端代碼位於`ui/flutter`目錄內。

### 🌍 開發環境

1. Golang 1.24+
2. Flutter 3.38+

### 📋 克隆項目

```bash
git clone git@github.com:GopeedLab/gopeed.git
```

### 🤝 協助開發

請參考[協助指南](CONTRIBUTING_zh-TW.md)

### 🏗️ 編譯

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
gomobile bind -tags nosqlite -ldflags="-w -s -checklinkname=0" -o ui/flutter/android/app/libs/libgopeed.aar -target=android -androidapi 21 -javapkg="com.gopeed" github.com/GopeedLab/gopeed/bind/mobile
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

## ❤️ 感謝

### 貢獻者

<a href="https://github.com/GopeedLab/gopeed/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=GopeedLab/gopeed" />
</a>

### JetBrains

[![goland](_docs/img/goland.svg)](https://www.jetbrains.com/?from=gopeed)

## 軟體許可

該軟體遵循 [GPLv3](LICENSE) 。
