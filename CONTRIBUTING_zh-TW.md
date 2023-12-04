# Gopeed 協助指南

首先感謝您願意幫助我們改進並優化該項目，這份指南將會幫助您更好的參與 Gopeed 的開發。

## 分支說明

本項目只有一個分支，即 `main` 分支，如果您想要參與 Gopeed 的開發，請先 fork 該項目，再在您自己的 fork 中進行開發，開發完成後再開啟PR，以合併至 `main` 分支。

## 離線開發

建議使用 web 端進行開發與調試，首先啟動服務，使用指令 `go run cmd/api/main.go` 啟動 ，該服務默認連接埠為 `9999`，接著以 `debug` 模式啟動前端 flutter 項目即可。

## 翻譯
 
Gopeed 的翻譯文件位於 `ui/flutter/lib/i18n/langs` 目錄中，只需要修改或新建翻譯文件即可。


請以 `en_us.dart` 作為參照。

## flutter開發

每次提交PR前請務必執行 `dart format ./ui/flutter`

如果需要編輯 api/models，請打開build_runner watcher:

```
flutter pub run build_runner watch
```
