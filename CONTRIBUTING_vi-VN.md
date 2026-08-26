# Hướng dẫn đóng góp cho Gopeed

Trước tiên, cảm ơn bạn đã quan tâm đến việc đóng góp cho Gopeed. Hướng dẫn này sẽ giúp bạn tham gia
phát triển Gopeed một cách tốt hơn.

## Mô tả nhánh

Dự án này chỉ có một nhánh chính duy nhất, đó là nhánh `main`. Nếu bạn muốn tham gia vào
phát triển Gopeed, hãy fork dự án này trước, sau đó phát triển trong dự án fork của bạn. Sau khi
hoàn thành phát triển, gửi một PR đến dự án này và merge vào nhánh `main`.

## Phát triển cục bộ

Đề nghị phát triển và gỡ lỗi thông qua web. Đầu tiên, khởi động dịch vụ backend bằng cách chạy
lệnh `go run cmd/api/main.go` trong dòng lệnh, cổng mặc định của dịch vụ là `9999`, sau đó
khởi động dự án flutter frontend trong chế độ `debug` để chạy.

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

## Phát triển flutter

Đừng quên chạy `dart format ./ui/flutter` trước khi commit để giữ mã của bạn theo định dạng dart chuẩn.

Bật build_runner watcher nếu bạn muốn chỉnh sửa api/models:
