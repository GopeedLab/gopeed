# [![](_docs/img/banner.png)](https://gopeed.com)

[![Test Status](https://github.com/GopeedLab/gopeed/workflows/test/badge.svg)](https://github.com/GopeedLab/gopeed/actions?query=workflow%3Atest)
[![Codecov](https://codecov.io/gh/GopeedLab/gopeed/branch/main/graph/badge.svg)](https://codecov.io/gh/GopeedLab/gopeed)
[![Release](https://img.shields.io/github/release/GopeedLab/gopeed.svg)](https://github.com/GopeedLab/gopeed/releases)
[![Download](https://img.shields.io/github/downloads/GopeedLab/gopeed/total.svg)](https://github.com/GopeedLab/gopeed/releases)
[![Donate](https://img.shields.io/badge/%24-donate-ff69b4.svg)](https://docs.gopeed.com/donate.html)
[![Discord](https://img.shields.io/discord/1037992631881449472?label=Discord&logo=discord&style=social)](https://discord.gg/ZUJqJrwCGB)

<a href="https://trendshift.io/repositories/7953" target="_blank"><img src="https://trendshift.io/api/badge/repositories/7953" alt="GopeedLab%2Fgopeed | Trendshift" style="width: 250px; height: 55px;" width="250" height="55"/></a>

[![ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/R6R6IJGN6)

[English](/README.md) | [中文](/README_zh-CN.md) | [日本語](/README_ja-JP.md) | [正體中文](/README_zh-TW.md) | [Tiếng Việt](/README_vi-VN.md)

## はじめに

Gopeed (正式名 Go Speed) は `Golang` + `Flutter` によって開発された高速ダウンローダーで、(HTTP、BitTorrent、Magnet) プロトコルをサポートし、すべてのプラットフォームをサポートします。基本的なダウンロード機能に加え、[APIs](https://docs.gopeed.com/dev-api.html)との連動や[拡張機能](https://docs.gopeed.com/dev-extension.html)のインストール・開発による追加機能にも対応した、カスタマイズ性の高いダウンローダーです。

見て下さい ✈ [公式ウェブサイト](https://gopeed.com) | 📖 [開発ドキュメント](https://docs.gopeed.com)

## インストール

<table>
  <thead>
    <tr>
      <th colspan="2">Windows</th>
      <th>MacOS</th>
      <th colspan="4">Linux</th>
      <th>Android</th>
      <th>iOS</th>
      <th>Docker</th>
      <th colspan="2">Qnap</th>
      <th colspan="8">Web</th>
    </tr>
    <tr>
      <td><code>EXE</code></td>
      <td><code>Portable</code></td>
      <td><code>DMG</code></td>
      <td><code>Flathub</code></td>
      <td><code>SNAP</code></td>
      <td><code>DEB</code></td>
      <td><code>AppImage</code></td>
      <td><code>APK</code></td>
      <td><code>IPA</code></td>
      <td>-</td>
      <td colspan="2"><code>QPKG</code></td>
      <td colspan="3"><code>Windows</code></td>
      <td colspan="2"><code>MacOS</code></td>
      <td colspan="3"><code>Linux</code></td>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>amd64</td>
      <td>amd64</td>
      <td>-</td>
      <td>amd64</td>
      <td>amd64</td>
      <td>amd64</td>
      <td>amd64</td>
      <td>-</td>
      <td>-</td>
      <td>-</td>
      <td>amd64</td>
      <td>arm64</td>
      <td>amd64</td>
      <td>amd64</td>
      <td>386</td>
      <td>amd64</td>
      <td>amd64</td>
      <td>amd64</td>
      <td>amd64</td>
      <td>386</td>
    </tr>
    <tr>
      <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-windows-amd64.zip">Link</a></td>
      <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-windows-amd64-portable.zip">Link</a></td>
      <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-macos.dmg">Link</a></td>
      <td><a href="https://flathub.org/apps/com.gopeed.Gopeed">Link</a></td>
      <td><a href="https://snapcraft.io/gopeed">Link</a></td>
      <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-linux-amd64.deb">Link</a></td>
      <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-linux-x86_64.AppImage">Link</a></td>
      <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-android.apk">Link</a></td>
      <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-ios.ipa">Link</a></td>
      <td><a href="https://hub.docker.com/r/liwei2633/gopeed">Link</a></td>
      <td><a href="https://gopeed.com/api/download?tpl=gopeed-$version-qnap-amd64.qpkg">Link</a></td>
      <td><a href="https://gopeed.com/api/download?tpl=gopeed-$version-qnap-arm64.qpkg">Link</a></td>
      <td><a href="https://gopeed.com/api/download?tpl=gopeed-web-$version-windows-amd64.zip">Link</a></td>
      <td><a href="https://gopeed.com/api/download?tpl=gopeed-web-$version-windows-arm64.zip">Link</a></td>
      <td><a href="https://gopeed.com/api/download?tpl=gopeed-web-$version-windows-386.zip">Link</a></td>
      <td><a href="https://gopeed.com/api/download?tpl=gopeed-web-$version-macos-amd64.zip">Link</a></td>
      <td><a href="https://gopeed.com/api/download?tpl=gopeed-web-$version-macos-arm64.zip">Link</a></td>
      <td><a href="https://gopeed.com/api/download?tpl=gopeed-web-$version-linux-amd64.zip">Link</a></td>
      <td><a href="https://gopeed.com/api/download?tpl=gopeed-web-$version-linux-arm64.zip">Link</a></td>
      <td><a href="https://gopeed.com/api/download?tpl=gopeed-web-$version-linux-386.zip">Link</a></td>
    </tr>
  </tbody>
</table>

インストールについての詳細は、[インストール](https://docs.gopeed.com/install.html)を参照してください。

## 寄付

もしこのプロジェクトがお気に召しましたら、このプロジェクトの発展を支援するために[寄付](https://docs.gopeed.com/donate.html)をご検討ください！

## ショーケース

![](_docs/img/ui-demo.png)

## 開発

このプロジェクトは二つの部分に分かれており、フロントエンドでは `flutter` を、バックエンドでは `Golang` を使用し、両者は `http` プロトコルで通信する。ユニックスシステムでは `unix socket` を、ウィンドウズシステムでは `tcp` プロトコルを使用します。

> フロントコードは `ui/flutter` ディレクトリにあります。

### 環境

1. Go 言語 1.23+
2. Flutter 3.24+

### クローン

```bash
git clone git@github.com:GopeedLab/gopeed.git
```

### コントリビュート

[CONTRIBUTING.md](/CONTRIBUTING_ja-JP.md) をご参照ください

### ビルド

#### デスクトップ

まず、[flutter デスクトップ公式サイトドキュメント](https://docs.flutter.dev/development/platform-integration/desktop)に従って環境を設定し、自分で検索できる `cgo` 環境を用意します。

コマンド:

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

#### モバイル

先ほどと同じように、`cgo` 環境を準備し、`gomobile` をインストールする必要があります:

```bash
go install golang.org/x/mobile/cmd/gomobile@latest
go get golang.org/x/mobile/bind
gomobile init
```

コマンド:

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

#### Web

コマンド:

```bash
cd ui/flutter
flutter build web
cd ../../
rm -rf cmd/web/dist
cp -r ui/flutter/build/web cmd/web/dist
go build -tags nosqlite,web -ldflags="-s -w" -o bin/ github.com/GopeedLab/gopeed/cmd/web
```

## 感謝

### コントリビューター

<a href="https://github.com/GopeedLab/gopeed/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=GopeedLab/gopeed" />
</a>

### JetBrains

[![goland](_docs/img/goland.svg)](https://www.jetbrains.com/?from=gopeed)

## ライセンス

[GPLv3](LICENSE)
