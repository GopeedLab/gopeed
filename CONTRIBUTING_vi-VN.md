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

Các tệp quốc tế hóa của Gopeed được đặt trong thư mục `ui/flutter/lib/i18n/langs`.
Bạn chỉ cần thêm tệp ngôn ngữ tương ứng trong thư mục này.

Vui lòng tham khảo `en_us.dart` để biết cách dịch thuật.

## Phát triển flutter

Đừng quên chạy `dart format ./ui/flutter` trước khi commit để giữ mã của bạn theo định dạng dart chuẩn.

Bật build_runner watcher nếu bạn muốn chỉnh sửa api/models:
