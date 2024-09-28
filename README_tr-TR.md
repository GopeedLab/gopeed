# [![](_docs/img/banner.png)](https://gopeed.com)

[![Test Status](https://github.com/GopeedLab/gopeed/workflows/test/badge.svg)](https://github.com/GopeedLab/gopeed/actions?query=workflow%3Atest)
[![Codecov](https://codecov.io/gh/GopeedLab/gopeed/branch/main/graph/badge.svg)](https://codecov.io/gh/GopeedLab/gopeed)
[![Release](https://img.shields.io/github/release/GopeedLab/gopeed.svg)](https://github.com/GopeedLab/gopeed/releases)
[![Download](https://img.shields.io/github/downloads/GopeedLab/gopeed/total.svg)](https://github.com/GopeedLab/gopeed/releases)
[![Donate](https://img.shields.io/badge/%24-donate-ff69b4.svg)](https://docs.gopeed.com/donate.html)
[![Discord](https://img.shields.io/discord/1037992631881449472?label=Discord&logo=discord&style=social)](https://discord.gg/ZUJqJrwCGB)

<a href="https://trendshift.io/repositories/7953" target="_blank"><img src="https://trendshift.io/api/badge/repositories/7953" alt="GopeedLab%2Fgopeed | Trendshift" style="width: 250px; height: 55px;" width="250" height="55"/></a>

[![ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/R6R6IJGN6)

[English](/README.md) | [中文](/README_zh-CN.md) | [日本語](/README_ja-JP.md) | [正體中文](/README_zh-TW.md) | [Tiếng Việt](/README_vi-VN.md) | [Türkçe] (/README_tr-TR.md)

## Introduction

Gopeed (tam adıyla Go Speed), Golang ve Flutter ile geliştirilmiş yüksek hızlı bir indirme yöneticisidir ve (HTTP, BitTorrent, Magnet) protokollerini destekler. Tüm platformlarla uyumludur. Temel indirme işlevlerinin yanı sıra, Gopeed aynı zamanda API'ler aracılığıyla entegrasyon veya eklentiler geliştirilip yüklenerek daha fazla özelliğin uygulanmasını destekleyen oldukça özelleştirilebilir bir indirme yöneticisidir.

Ziyaret et ✈ [Resmi Web Site](https://gopeed.com) | 📖 [Resmi Dokümanlar](https://docs.gopeed.com)

## Download

<table>
    <thead>
        <tr>
            <th>Platform</th>
            <th>Paket Tipi</th>
            <th>İndirme Bağlantıları</th>
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

Kurulum hakkında daha detaylı bilgi almak için, [buradaki kurulum sayfasını](https://docs.gopeed.com/install.html) ziyaret edebilirsiniz.

### Komut Aracı

`go install` komutunu kullanın:

```bash
go install github.com/GopeedLab/gopeed/cmd/gopeed@latest
```

## Tarayıcı Eklentisi

Gopeed, ayrıca internet tarayıcısı üzerindeki indirmelerinizi de yönetmek için Chrome, Edge, Firefox gibi tarayıcıları destekleyen bir eklentiye de sahip. Daha fazla bilgi için şu adrese göz atabilirsiniz: [https://github.com/GopeedLab/browser-extension](https://github.com/GopeedLab/browser-extension)

## Bağış ve Destek

Eğer Gopeed'i beğendiyseniz, projenin geliştirilmesine destek olmak için [bağış yapmayı](https://docs.gopeed.com/donate.html) düşünebilirsiniz, teşekkürler. :)

## Uygulamanın Görseli

![](_docs/img/ui-demo.png)

## Geliştirme

Bu proje iki kısma ayrılmıştır: front-end kısmı flutter, backend kısmı ise Golang kullanılarak geliştirilmiştir ve iki taraf http protokolü üzerinden iletişim kurar. Unix sistemlerinde unix socket kullanılırken, Windows sistemlerinde tcp protokolü kullanılmaktadır.
> Front-end kodları `ui/flutter` dizininde yer almaktadır.

### Geliştirme Ortamı

1. Golang 1.22+
2. Flutter 3.16+

### Clone

```bash
git clone git@github.com:GopeedLab/gopeed.git
```

### Katkıda Bulunma

Projeye katkıda bulunmak için lütfen [CONTRIBUTING.md](/CONTRIBUTING_tr-TR.md) belgesine göz atın.

### Build İşlemleri

#### Masaüstü

Öncelikle, [resmi Flutter dokümanları](https://docs.flutter.dev/development/platform-integration/desktop) doğrultusunda ortamı yapılandırmanız gerekmektedir. Ardından, cgo ortamının uygun şekilde ayarlandığından emin olmalısınız. Cgo ortamını kurmak için ayrıntılı talimatlar ve ilgili kaynaklar için internette mevcut rehberlerden faydalanabilirsiniz.


komutlar:

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

#### Mobil

Daha önce olduğu gibi, `cgo` ortamını hazırlamalısınız, ardından `gomobile` kurulumunu gerçekleştirmeniz gerekecek:

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

## Emeği Geçenler

### Katkıda Bulunanlar

<a href="https://github.com/GopeedLab/gopeed/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=GopeedLab/gopeed" />
</a>

### JetBrains

[![goland](_docs/img/goland.svg)](https://www.jetbrains.com/?from=gopeed)

## Lisans

[GPLv3](Lisansı)
