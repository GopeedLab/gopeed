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

Truy cập ✈ [Trang web chính thức](https://gopeed.com)

![Ảnh chụp màn hình ứng dụng](_docs/img/ui-concept-en.png)

## ✨ Tính năng chính

- ⚡ **Tải xuống tốc độ cao** — kết hợp nhiều tác vụ, truyền HTTP phân đoạn qua nhiều kết nối và tải BitTorrent P2P để tận dụng tối đa băng thông.
- 🧲 **Đa giao thức** — quản lý HTTP, HTTPS, BitTorrent, Magnet và ed2k trong một giao diện.
- 🌱 **Bộ công cụ BT đầy đủ** — hỗ trợ khám phá Peer qua DHT, truyền tải uTP, Web Seed, chọn tệp, quản lý Tracker, xem Peer/mảnh và kiểm soát seed theo tỷ lệ hoặc thời gian.
- 📋 **Quản lý tác vụ thực tế** — tạm dừng, tiếp tục, thử lại, thao tác hàng loạt, tìm kiếm, bộ lọc, danh mục và khôi phục khi khởi động.
- 🪶 **Trải nghiệm native gọn nhẹ** — giao diện chính được Flutter kết xuất native. Không Electron. Không vỏ bọc WebView. Kích thước nhỏ hơn, ít tốn tài nguyên hơn và phản hồi nhanh hơn.
- 💻 **Đa nền tảng** — Windows, macOS, Linux, Android, iOS, Web, Docker và QNAP.
- 🎨 **Giao diện tùy biến** — hỗ trợ chế độ theo hệ thống, sáng, tối và 8 màu nhấn.
- 📐 **Bố cục thích ứng** — danh sách tác vụ, điều hướng, cài đặt và màn hình chi tiết tự điều chỉnh cho điện thoại, máy tính bảng và cửa sổ desktop có thể thay đổi kích thước.
- 🗣️ **Hơn 20 ngôn ngữ giao diện** — gồm tiếng Việt, tiếng Anh, tiếng Trung giản thể và phồn thể, tiếng Nhật, tiếng Hàn cùng nhiều ngôn ngữ khác.
- 🌐 **Tích hợp trình duyệt** — gửi tải xuống từ Chrome, Edge, Firefox và các trình duyệt tương thích sang Gopeed.
- 🧩 **Tiện ích JavaScript** — thêm nguồn tải từ nền tảng video, kho mô hình AI, lưu trữ đám mây và nhiều dịch vụ khác.
- 🤖 **Tích hợp AI** — cung cấp giao diện MCP để kết nối với các AI Agent tương thích và tạo, kiểm tra hoặc quản lý tác vụ tải xuống bằng ngôn ngữ tự nhiên.
- 🔌 **Giao diện mở** — REST API, CLI, Web UI có xác thực, webhook và script sau khi tải.
- 🛠️ **Công cụ tích hợp** — Header/User-Agent tùy chỉnh, proxy, mirror GitHub, thông báo và tự động giải nén.

## 🤖 Tích hợp AI

Sau khi kết nối Gopeed với AI Agent, bạn có thể quản lý tải xuống bằng ngôn ngữ tự nhiên. Ví dụ, hãy nói với AI Agent:

> Hãy tải xuống phiên bản Gopeed mới nhất dành cho Windows

| Tool | Mô tả |
| --- | --- |
| `resolve_task` | Phân tích URL hoặc URI tải xuống và trả về thông tin tài nguyên cùng danh sách tệp trước khi tạo tác vụ. |
| `create_task` | Tạo và bắt đầu tác vụ từ ID tài nguyên đã phân tích hoặc từ yêu cầu tải xuống trực tiếp. |
| `list_tasks` | Liệt kê tác vụ và có thể lọc theo ID hoặc trạng thái. |
| `get_task` | Lấy yêu cầu, tài nguyên, tùy chọn và tiến độ hiện tại của một tác vụ. |
| `get_task_status` | Lấy trạng thái chạy rút gọn và tiến độ của từng tệp trong một tác vụ. |
| `get_task_stats` | Lấy số liệu thống kê theo giao thức, gồm kết nối HTTP hoặc Peer và dữ liệu seed của BitTorrent. |
| `pause_task` | Tạm dừng tác vụ. |
| `continue_task` | Tiếp tục tác vụ đã tạm dừng hoặc bị lỗi. |
| `delete_task` | Xóa tác vụ và có thể xóa cả các tệp đã tải xuống. |

## ⬇️ Tải về

### 🧪 Gopeed 2.0.0 Beta

Gopeed 2.0.0 hiện đang trong giai đoạn beta công khai, với giao diện được thiết kế lại, kiến trúc giao tiếp native kết nối trực tiếp ứng dụng desktop và di động với lõi Go qua FFI, trải nghiệm đa nền tảng nhất quán hơn, khả năng quản lý tác vụ được cải thiện, API linh hoạt hơn và khả năng tích hợp AI Agent qua MCP. Một số tính năng có thể vẫn chưa hoàn thiện hoặc chưa ổn định, vì vậy hãy dùng thử và gửi phản hồi nếu bạn gặp vấn đề.

- [Tải Gopeed 2.0.0 Beta 1](https://github.com/GopeedLab/gopeed/releases/tag/v2.0.0-beta.1)

Khi các tính năng và độ ổn định đáp ứng tiêu chuẩn phát hành, chúng tôi sẽ phát hành Gopeed 2.0.0 chính thức. Người dùng bản beta có thể nâng cấp trực tiếp lên bản chính thức, trong khi người dùng bản ổn định hiện tại sẽ không tự động được chuyển sang kênh beta.

### Bản ổn định

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
2. Flutter 3.41+

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
