# [![](_docs/img/banner.svg)](https://gopeed.com)

[![Trạng thái kiểm tra](https://github.com/GopeedLab/gopeed/workflows/test/badge.svg)](https://github.com/GopeedLab/gopeed/actions?query=workflow%3Atest)
[![Codecov](https://codecov.io/gh/GopeedLab/gopeed/branch/main/graph/badge.svg)](https://codecov.io/gh/GopeedLab/gopeed)
[![Phiên bản](https://img.shields.io/github/release/GopeedLab/gopeed.svg)](https://github.com/GopeedLab/gopeed/releases)
[![Tải về](https://img.shields.io/github/downloads/GopeedLab/gopeed/total.svg)](https://github.com/GopeedLab/gopeed/releases)
[![Ủng hộ](https://img.shields.io/badge/%24-ủng%20hộ-ff69b4.svg)](https://gopeed.com/docs/donate)
[![WeChat](https://img.shields.io/badge/WeChat%20Official%20Account-07C160?logo=wechat&logoColor=white)](https://raw.githubusercontent.com/GopeedLab/gopeed/main/_docs/img/weixin.png)
[![Discord](https://img.shields.io/discord/1037992631881449472?label=Discord&logo=discord&style=social)](https://discord.gg/ZUJqJrwCGB)

<a href="https://trendshift.io/repositories/7953" target="_blank"><img src="https://trendshift.io/api/badge/repositories/7953" alt="GopeedLab%2Fgopeed | Trendshift" style="width: 250px; height: 55px;" width="250" height="55"/></a>

[English](/README.md) | [中文](/README_zh-CN.md) | [日本語](/README_ja-JP.md) | [正體中文](/README_zh-TW.md) | [Tiếng Việt](/README_vi-VN.md)

## 🚀 Giới thiệu

Gopeed (viết tắt của **Go Speed**) là trình quản lý tải xuống nhanh, hiện đại, miễn phí và mã nguồn mở, được xây dựng bằng Go và Flutter. Gopeed hỗ trợ HTTP, HTTPS, BitTorrent, Magnet và ed2k trên máy tính, thiết bị di động và Web.

Ngoài việc quản lý tải xuống hằng ngày, Gopeed còn hỗ trợ tích hợp trình duyệt, tiện ích JavaScript, REST API, CLI và Web UI có thể tự lưu trữ để mở rộng và tự động hóa quy trình.

Truy cập ✈ [Trang web chính thức](https://gopeed.com) | 📖 [Tài liệu chính thức](https://gopeed.com/docs)

![Giao diện Gopeed trên máy tính và thiết bị di động](_docs/img/ui-concept-en.png)

## ✨ Tính năng chính

- ⚡ **Tải xuống tốc độ cao** — kết hợp nhiều tác vụ, truyền HTTP phân đoạn qua nhiều kết nối và tải BitTorrent P2P để tận dụng tối đa băng thông.
- 🧲 **Đa giao thức** — quản lý HTTP, HTTPS, BitTorrent, Magnet và ed2k trong một giao diện.
- 🌱 **Bộ công cụ BT đầy đủ** — chọn tệp, quản lý Tracker, xem Peer/mảnh và kiểm soát seed theo tỷ lệ hoặc thời gian.
- 📋 **Quản lý tác vụ thực tế** — tạm dừng, tiếp tục, thử lại, thao tác hàng loạt, tìm kiếm, bộ lọc, danh mục và khôi phục khi khởi động.
- 💻 **Đa nền tảng** — Windows, macOS, Linux, Android, iOS, Web, Docker và QNAP.
- 🎨 **Giao diện tùy biến** — hỗ trợ chế độ theo hệ thống, sáng, tối và 8 màu nhấn.
- 📐 **Bố cục thích ứng** — danh sách tác vụ, điều hướng, cài đặt và màn hình chi tiết tự điều chỉnh cho điện thoại, máy tính bảng và cửa sổ desktop có thể thay đổi kích thước.
- 🗣️ **Hơn 20 ngôn ngữ giao diện** — gồm tiếng Việt, tiếng Anh, tiếng Trung giản thể và phồn thể, tiếng Nhật, tiếng Hàn cùng nhiều ngôn ngữ khác.
- 🌐 **Tích hợp trình duyệt** — gửi tải xuống từ Chrome, Edge, Firefox và các trình duyệt tương thích sang Gopeed.
- 🧩 **Tiện ích JavaScript** — thêm nguồn tải từ nền tảng video, kho mô hình AI, lưu trữ đám mây và nhiều dịch vụ khác.
- 🔌 **Giao diện mở** — REST API, CLI, Web UI có xác thực, webhook và script sau khi tải.
- 🛠️ **Công cụ tích hợp** — Header/User-Agent tùy chỉnh, proxy, mirror GitHub, thông báo và tự động giải nén.

## ⬇️ Tải về

- [Tải xuống từ trang web chính thức](https://gopeed.com)
- [GitHub Releases](https://github.com/GopeedLab/gopeed/releases/latest)

### 🛠️ Công cụ lệnh

Sử dụng `go install`:

```bash
go install github.com/GopeedLab/gopeed/cmd/gopeed@latest
```

## 📱 WeChat Official Account

Theo dõi tài khoản chính thức để nhận các cập nhật và tin tức mới nhất.

<img src="_docs/img/weixin.png" width="200" />

## 💝 Quyên góp

Nếu bạn thích dự án này, xin vui lòng xem xét [quyên góp](https://gopeed.com/docs/donate) để hỗ trợ phát triển dự án này, cảm ơn bạn!

## 👨‍💻 Development

Dự án này được chia thành hai phần, phần giao diện sử dụng `flutter`, phần backend sử dụng `Golang`, và hai phía giao tiếp thông qua giao thức `http`. Trên hệ thống unix, sử dụng `unix socket`, và trên hệ thống windows, sử dụng giao thức `tcp`.

> Mã giao diện nằm trong thư mục `ui/flutter`.

### 🌍 Environment

1. Golang 1.25+
2. Flutter 3.38+

### 📋 Clone

```bash
git clone git@github.com:GopeedLab/gopeed.git
```

### 🤝 Đóng góp

Vui lòng tham khảo [CONTRIBUTING_vi-VN.md](/CONTRIBUTING_vi-VN.md)

### 🏗️ Xây dựng

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

command:

```bash
cd ui/flutter
flutter build web
cd ../../
rm -rf cmd/web/dist
cp -r ui/flutter/build/web cmd/web/dist
go build -tags nosqlite,web -ldflags="-s -w" -o bin/ github.com/GopeedLab/gopeed/cmd/web
```

## ❤️ Tín dụng

### Người đóng góp

<a href="https://github.com/GopeedLab/gopeed/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=GopeedLab/gopeed" />
</a>

### JetBrains

[![goland](_docs/img/goland.svg)](https://www.jetbrains.com/?from=gopeed)

## Giấy phép

[GPLv3](LICENSE)
