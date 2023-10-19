[![Test Status](https://github.com/GopeedLab/gopeed/workflows/test/badge.svg)](https://github.com/GopeedLab/gopeed/actions?query=workflow%3Atest)
[![Codecov](https://codecov.io/gh/GopeedLab/gopeed/branch/main/graph/badge.svg)](https://codecov.io/gh/GopeedLab/gopeed)
[![Release](https://img.shields.io/github/release/GopeedLab/gopeed.svg)](https://github.com/GopeedLab/gopeed/releases)
[![Download](https://img.shields.io/github/downloads/GopeedLab/gopeed/total.svg)](https://github.com/GopeedLab/gopeed/releases)
[![Donate](https://img.shields.io/badge/%24-donate-ff69b4.svg)](https://docs.gopeed.com/donate.html)
[![Discord](https://img.shields.io/discord/1037992631881449472?label=Discord&logo=discord&style=social)](https://discord.gg/ZUJqJrwCGB)

![](_docs/img/banner.png)

[![ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/R6R6IJGN6)

[English](/README.md) | [中文](/README_zh-CN.md) | [日本語](/README_ja-JP.md)

## はじめに

Gopeed (正式名 Go Speed) は `Golang` + `Flutter` によって開発された高速ダウンローダーで、(HTTP、BitTorrent、Magnet) プロトコルをサポートし、すべてのプラットフォームをサポートします。

見て下さい ✈ [公式ウェブサイト](https://gopeed.com) | 📖 [開発ドキュメント](https://docs.gopeed.com)

## インストール

**サポートするプラットフォーム**

- [x] windows
- [x] macos
- [x] linux
- [x] android
- [x] ios
- [x] web
- [x] docker

[ダウンロード](https://github.com/GopeedLab/gopeed/releases/latest)

> ヒント: macos を開くのに失敗した場合は、ターミナルで `xattr -d com.apple.quarantine /Applications/Gopeed.app` コマンドを実行してください

### コマンドツール

`go install` を使用:

```bash
go install github.com/GopeedLab/gopeed/cmd/gopeed@latest
```

### Docker

#### 直接始める

```bash
docker run -d -p 9999:9999 -v /path/to/download:/root/Downloads liwei2633/gopeed
```

#### Docker Compose の使用

```bash
docker-compose up -d
```

#### Gopeed へのアクセス

docker コンテナが起動しているときは、`http://localhost:9999` からウェブページにアクセスできます。

## 寄付

もしこのプロジェクトがお気に召しましたら、このプロジェクトの発展を支援するために[寄付](/.donate/index.md#donate)をご検討ください！

## ショーケース

![](_docs/img/ui-demo.png)

## 開発

このプロジェクトは二つの部分に分かれており、フロントエンドでは `flutter` を、バックエンドでは `Golang` を使用し、両者は `http` プロトコルで通信する。ユニックスシステムでは `unix socket` を、ウィンドウズシステムでは `tcp` プロトコルを使用します。

> フロントコードは `ui/flutter` ディレクトリにあります。

### 環境

1. Go 言語 1.19+
2. Flutter 3.10+

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

#### Web

コマンド:

```bash
cd ui/flutter
flutter build web
cd ../../
rm -rf cmd/web/dist
cp -r ui/flutter/build/web cmd/web/dist
go build -tags nosqlite,web -ldflags="-s -w" -o bin/ github.com/GopeedLab/gopeed/cmd/web
go run cmd/web/main.go
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
