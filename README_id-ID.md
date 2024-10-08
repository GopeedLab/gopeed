# [![](_docs/img/banner.png)](https://gopeed.com)

[![Status Tes](https://github.com/GopeedLab/gopeed/workflows/test/badge.svg)](https://github.com/GopeedLab/gopeed/actions?query=workflow%3Atest)
[![Codecov](https://codecov.io/gh/GopeedLab/gopeed/branch/main/graph/badge.svg)](https://codecov.io/gh/GopeedLab/gopeed)
[![Rilis](https://img.shields.io/github/release/GopeedLab/gopeed.svg)](https://github.com/GopeedLab/gopeed/releases)
[![Unduh](https://img.shields.io/github/downloads/GopeedLab/gopeed/total.svg)](https://github.com/GopeedLab/gopeed/releases)
[![Donasi](https://img.shields.io/badge/%24-donate-ff69b4.svg)](https://docs.gopeed.com/donate.html)
[![Discord](https://img.shields.io/discord/1037992631881449472?label=Discord&logo=discord&style=social)](https://discord.gg/ZUJqJrwCGB)

<a href="https://trendshift.io/repositories/7953" target="_blank"><img src="https://trendshift.io/api/badge/repositories/7953" alt="GopeedLab%2Fgopeed | Trendshift" style="width: 250px; height: 55px;" width="250" height="55"/></a>

[![ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/R6R6IJGN6)

[English](/README.md) | [中文](/README_zh-CN.md) | [日本語](/README_ja-JP.md) | [正體中文](/README_zh-TW.md) | [Tiếng Việt](/README_vi-VN.md) | [Indonesia](/README_id-ID.md)

## Pengantar

Gopeed (nama lengkap Go Speed), merupakan pengunduh berkecepatan tinggi yang dikembangkan menggunakan Golang + Flutter. Gopeed mendukung protokol (HTTP, BitTorrent, Magnet), dan dapat digunakan di semua platform. Selain fungsi pengunduhan dasar, Gopeed juga merupakan pengunduh yang sangat dapat disesuaikan, memungkinkan penambahan fitur melalui integrasi dengan [APIs](https://docs.gopeed.com/dev-api.html) atau pemasangan dan pengembangan [Ekstensi](https://docs.gopeed.com/dev-extension.html).

Kunjungi ✈ [Situs Web Resmi](https://gopeed.com) | 📖 [Dokumentasi Resmi](https://docs.gopeed.com)

## Unduh

<table>
    <thead>
        <tr>
            <th>Platform</th>
            <th>Jenis Paket</th>
            <th>Tautan Unduh</th>
        </tr>
    </thead>
    <tbody>
        <tr>
            <td rowspan=2>Windows</td>
            <td><code>Penginstal EXE</code></td>
            <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-windows-amd64.zip">Link</a></td>
        </tr>
        <tr>
            <td><code>ZIP Portabel</code></td>
            <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-windows-amd64-portable.zip">Link</a></td>
        </tr>
        <tr>
            <td>MacOS</td>
            <td><code>Penginstal DMG</code></td>          
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

Untuk informasi lebih lanjut tentang instalasi, silakan lihat [Instalasi](https://docs.gopeed.com/install.html)

### Perintah Terminal

Gunakan `go install`:

```bash
go install github.com/GopeedLab/gopeed/cmd/gopeed@latest
```

## Ekstensi Peramban

Gopeed juga menyediakan ekstensi peramban untuk mengambil alih unduhan peramban, mendukung peramban seperti Chrome, Edge, Firefox, dll. Silakan lihat informasi lebih lanjut di: [https://github.com/GopeedLab/browser-extension](https://github.com/GopeedLab/browser-extension)

## Donate

Jika Anda menyukai proyek ini, silakan pertimbangkan untuk [mendonasikan](https://docs.gopeed.com/donate.html) untuk mendukung pengembangan proyek ini, terima kasih!

## Mempertunjukkan

![](_docs/img/ui-demo.png)

## Pengembangan

Proyek ini dibagi menjadi dua bagian. Bagian frontend menggunakan `flutter`, bagian backend menggunakan `Golang`, dan kedua sisi berkomunikasi melalui protokol `http`. Pada sistem unix, digunakan `unix socket`, sedangkan pada sistem windows, digunakan protokol `tcp`.

> Kode frontend terletak di direktori `ui/flutter`.

### Environment

1. Golang 1.22+
2. Flutter 3.16+

### Clone

```bash
git clone git@github.com:GopeedLab/gopeed.git
```

### Kontribusi

Silakan lihat [CONTRIBUTING.md](/CONTRIBUTING.md)

### Membangun

#### Desktop

Pertama, Anda perlu mengkonfigurasi lingkungan sesuai dengan dokumentasi resmi [Flutter desktop website](https://docs.flutter.dev/development/platform-integration/desktop), Kemudian, Anda perlu memastikan lingkungan cgo telah dikonfigurasi dengan benar. Untuk petunjuk terperinci tentang pengaturan lingkungan cgo, silakan merujuk ke sumber daya yang relevan yang tersedia secara online.

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

Sama seperti sebelumnya, Anda juga perlu menyiapkan lingkungan `cgo`, dan kemudian menginstal `gomobile`:

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

## Atribusi

### Kontributor

<a href="https://github.com/GopeedLab/gopeed/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=GopeedLab/gopeed" />
</a>

### JetBrains

[![goland](_docs/img/goland.svg)](https://www.jetbrains.com/?from=gopeed)

## Lisensi

[GPLv3](LICENSE)
