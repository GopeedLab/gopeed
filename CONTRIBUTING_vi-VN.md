# Hướng dẫn đóng góp cho Gopeed

Trước tiên, cảm ơn bạn đã quan tâm đến việc đóng góp cho Gopeed. Hướng dẫn này sẽ giúp bạn tham gia
phát triển Gopeed một cách tốt hơn.

## Nhánh phát triển

Hãy fork Gopeed, tạo nhánh làm việc từ `main` mới nhất và gửi pull request vào `main`.

## Phát triển cục bộ

Dùng Go 1.27.4 và Flutter 3.41.6 (Dart 3.11.4) như trong CI. Để phát triển trên web, hãy khởi động REST API của Go từ thư mục gốc của kho mã nguồn.

```sh
go run -tags nosqlite ./cmd/api
```

API lắng nghe tại `127.0.0.1:9999`. Mở một cửa sổ dòng lệnh khác tại thư mục gốc của kho mã nguồn và chạy ứng dụng Flutter trong Chrome.

```sh
cd ui/flutter
flutter pub get
flutter gen-l10n
flutter run -d chrome
```

Khi phát triển cho máy tính hoặc thiết bị di động, hãy biên dịch bindings Go cho nền tảng đích theo hướng dẫn trong [README](README_vi-VN.md).

## Dịch thuật

Các bản dịch Flutter là tệp ARB trong `ui/flutter/lib/l10n`. Hãy dùng `app_en.arb` làm mẫu nguồn,
sau đó sửa hoặc thêm `app_<locale>.arb` (ví dụ `app_de.arb` hoặc `app_zh_TW.arb`). Mỗi ngôn ngữ phải
có cùng các khóa thông điệp như mẫu tiếng Anh.

Chỉ dịch giá trị thông điệp; không đổi tên biến trong dấu ngoặc nhọn như `{count}` và `{name}`. Không dịch khóa
siêu dữ liệu bắt đầu bằng `@`. Chỉ commit các tệp nguồn ARB đã thay đổi; không cần tạo hoặc commit
`app_localizations*.dart`. CI của pull request sẽ tự động
kiểm tra tất cả ngôn ngữ, tạo mã bản địa hóa và xác minh việc tích hợp.

Các ví dụ sau hiển thị cả định nghĩa trong ARB và nội dung người dùng thực sự thấy sau khi ứng dụng truyền giá trị.

1. Chèn biến vào thông điệp

   Định nghĩa ARB:

   ```json
   "welcomeUser": "Xin chào, {name}"
   ```

   Kết quả hiển thị: khi `name = An`, người dùng thấy “Xin chào, An”. Có thể đổi trật tự câu nhưng phải giữ `{name}`.

2. Hiển thị nội dung khác nhau theo số lượng

   Định nghĩa ARB:

   ```json
   "fileCount": "{count, plural, =0{Không có tệp} =1{1 tệp} other{{count} tệp}}"
   ```

   Kết quả hiển thị:

   - `count = 0` → “Không có tệp”
   - `count = 1` → “1 tệp”
   - `count = 3` → “3 tệp”

   `=0`, `=1` và `other` lần lượt biểu thị số lượng bằng 0, bằng 1 và các số lượng còn lại. Chỉ dịch phần chữ
   hiển thị cho người dùng bên trong mỗi nhánh.

3. Hiển thị nội dung khác nhau theo giá trị biến

   Định nghĩa ARB:

   ```json
   "taskState": "{state, select, running{Đang chạy} paused{Đã tạm dừng} other{Không xác định}}"
   ```

   Kết quả hiển thị:

   - `state = running` → “Đang chạy”
   - `state = paused` → “Đã tạm dừng”
   - Giá trị khác → “Không xác định”

   `running`, `paused` và `other` là các giá trị do ứng dụng truyền vào nên không được dịch; chỉ dịch phần chữ
   hiển thị trong dấu ngoặc theo sau chúng.

## Kiểm tra mã nguồn

Dùng `gofmt` để định dạng các tệp Go đã sửa, rồi chạy kiểm thử Go từ thư mục gốc của kho mã nguồn.

```sh
go test -tags nosqlite ./...
```

Dùng `dart format` để định dạng các tệp Dart đã sửa. Sau khi cài các gói phụ thuộc của Flutter và sinh mã bản địa hóa theo hướng dẫn trên, hãy chạy các lệnh kiểm tra sau từ thư mục gốc của kho mã nguồn.

```sh
cd ui/flutter
flutter analyze
flutter test
```

Nếu sửa các mô hình có annotation trong `lib/api/model/`, hãy tạo lại mã tuần tự hóa JSON từ `ui/flutter`.

```sh
dart run build_runner build
```
