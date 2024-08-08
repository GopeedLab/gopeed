[![Trạng thái kiểm tra](https://github.com/GopeedLab/gopeed/workflows/test/badge.svg)](https://github.com/GopeedLab/gopeed/actions?query=workflow%3Atest)
[![Codecov](https://codecov.io/gh/GopeedLab/gopeed/branch/main/graph/badge.svg)](https://codecov.io/gh/GopeedLab/gopeed)
[![Phiên bản](https://img.shields.io/github/release/GopeedLab/gopeed.svg)](https://github.com/GopeedLab/gopeed/releases)
[![Tải về](https://img.shields.io/github/downloads/GopeedLab/gopeed/total.svg)](https://github.com/GopeedLab/gopeed/releases)
[![Ủng hộ](https://img.shields.io/badge/%24-ủng%20hộ-ff69b4.svg)](https://docs.gopeed.com/donate.html)
[![Discord](https://img.shields.io/discord/1037992631881449472?label=Discord&logo=discord&style=social)](https://discord.gg/ZUJqJrwCGB)

<a href="https://trendshift.io/repositories/7953" target="_blank"><img src="https://trendshift.io/api/badge/repositories/7953" alt="GopeedLab%2Fgopeed | Trendshift" style="width: 250px; height: 55px;" width="250" height="55"/></a>

![](_docs/img/banner.png)

[![ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/R6R6IJGN6)

[English](/README.md) | [中文](/README_zh-CN.md) | [日本語](/README_ja-JP.md) | [正體中文](/README_zh-TW.md) | [Tiếng Việt](/README_vi-VN.md)

## Giới thiệu

Gopeed (tên đầy đủ Go Speed), một công cụ tải xuống tốc độ cao được phát triển bởi `Golang` + `Flutter`, hỗ trợ giao thức (HTTP, BitTorrent, Magnet) và hỗ trợ tất cả các nền tảng. Ngoài các chức năng tải xuống cơ bản, Gopeed còn là một công cụ tải xuống có thể tùy chỉnh cao cho phép triển khai thêm tính năng thông qua việc tích hợp với [APIs](https://docs.gopeed.com/dev-api.html) hoặc cài đặt và phát triển các [tiện ích mở rộng](https://docs.gopeed.com/dev-extension.html).

Truy cập ✈ [Trang web chính thức](https://gopeed.com) | 📖 [Tài liệu chính thức](https://docs.gopeed.com)

## Tải về

<table>
    <thead>
        <tr>
            <th>Nền tảng</th>
            <th>Loại gói</th>
            <th>Liên kết tải xuống</th>
        </tr>
    </thead>
    <tbody>
        <tr>
            <td rowspan=2>Windows</td>
            <td><code>Bộ cài đặt EXE</code></td>
            <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-windows-amd64.zip">Liên kết</a></td>
        </tr>
        <tr>
            <td><code>ZIP Portable</code></td>
            <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-windows-amd64-portable.zip">Liên kết</a></td>
        </tr>
        <tr>
            <td>MacOS</td>
            <td><code>Bộ cài đặt DMG</code></td>
            <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-macos.dmg">Liên kết</a></td>
        </tr>
        <tr>
            <td rowspan=4>Linux</td>
            <td><code>Flathub</code></td>
            <td><a href="https://flathub.org/apps/com.gopeed.Gopeed">Liên kết</a></td>
        </tr>
        <tr>
            <td><code>SNAP</code></td>
            <td><a href="https://snapcraft.io/gopeed">Liên kết</a></td>
        </tr>
        <tr>
            <td><code>DEB</code></td>
            <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-linux-amd64.deb">Liên kết</a></td>
        </tr>
        <tr>
            <td><code>AppImage</code></td>
            <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-linux-x86_64.AppImage">Liên kết</a></td>
        </tr>
        <tr>
            <td>Android</td>
            <td><code>APK</code></td>
            <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-android.apk">Liên kết</a></td>
        </tr>
        <tr>
            <td>iOS</td>
            <td><code>IPA</code></td>
            <td><a href="https://gopeed.com/api/download?tpl=Gopeed-$version-ios.ipa">Liên kết</a></td>
        </tr>
        <tr>
            <td>Web</td>
            <td></td>
            <td><a href="https://github.com/GopeedLab/gopeed/releases/latest">Liên kết</a></td>
        </tr>
        <tr>
            <td>Docker</td>
            <td></td>
            <td><a href="https://hub.docker.com/r/liwei2633/gopeed">Liên kết</a></td>
        </tr>
    </tbody>
</table>

Thêm thông tin về cài đặt, vui lòng tham khảo [Cài đặt](https://docs.gopeed.com/install.html)

### Command tool

Sử dụng `go install`:

```bash
go install github.com/GopeedLab/gopeed/cmd/gopeed@latest
```

## Quyên góp

Nếu bạn thích dự án này, xin vui lòng xem xét [quyên góp](https://docs.gopeed.com/donate.html) để hỗ trợ phát triển dự án này, cảm ơn bạn!

## Trưng bày

![](_docs/img/ui-demo.png)

## Development

Dự án này được chia thành hai phần, phần giao diện sử dụng `flutter`, phần backend sử dụng `Golang`, và hai phía giao tiếp thông qua giao thức `http`. Trên hệ thống unix, sử dụng `unix socket`, và trên hệ thống windows, sử dụng giao thức `tcp`.

> Mã giao diện nằm trong thư mục `ui/flutter`.

### Environment

1. Golang 1.22+
2. Flutter 3.16+

### Clone

```bash
git clone git@github.com:GopeedLab/gopeed.git
```

### Đóng góp

Vui lòng tham khảo [CONTRIBUTING_vi-VN.md](/CONTRIBUTING_vi-VN.md)

### Xây dựng

#### Desktop

Trước tiên, bạn cần cấu hình môi trường theo tài liệu chính thức của [Tài liệu trang web máy tính để bàn Flutter](https://docs.flutter.dev/development/platform-integration/desktop), sau đó bạn cần đảm bảo môi trường cgo được thiết lập đúng. Để biết hướng dẫn chi tiết về cách thiết lập môi trường cgo, vui lòng tham khảo các tài liệu tương ứng có sẵn trực tuyến.

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

Giống như trước đây, bạn cũng cần chuẩn bị môi trường `cgo` và sau đó cài đặt `gomobile`:

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

## Tín dụng

### Người đóng góp

<a href="https://github.com/GopeedLab/gopeed/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=GopeedLab/gopeed" />
</a>

### JetBrains

[![goland](_docs/img/goland.svg)](https://www.jetbrains.com/?from=gopeed)

## Giấy phép

[GPLv3](LICENSE)