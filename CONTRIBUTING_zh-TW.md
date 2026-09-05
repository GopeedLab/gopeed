# Gopeed 協助指南

首先感謝您願意幫助我們改進並優化該項目，這份指南將會幫助您更好的參與 Gopeed 的開發。

## 分支說明

本項目只有一個分支，即 `main` 分支，如果您想要參與 Gopeed 的開發，請先 fork 該項目，再在您自己的 fork 中進行開發，開發完成後再開啟PR，以合併至 `main` 分支。

## 離線開發

建議使用 web 端進行開發與調試，首先啟動服務，使用指令 `go run cmd/api/main.go` 啟動 ，該服務默認連接埠為 `9999`，接著以 `debug` 模式啟動前端 flutter 項目即可。

## 翻譯

Flutter 翻譯檔位於 `ui/flutter/lib/l10n`，請以 `app_en.arb` 為來源範本，修改或新增
`app_<locale>.arb`（例如 `app_de.arb` 或 `app_zh_TW.arb`）。每個語言都必須與英文範本包含相同的訊息 key。

只翻譯訊息值，不要修改 `{count}`、`{name}` 等大括號中的變數名稱。以 `@` 開頭的中繼資料 key 不需要翻譯。
提交時只需包含修改過的 ARB 來源檔，不需要產生或提交 `app_localizations*.dart`。PR 的 CI
會自動驗證所有語言、產生國際化程式碼並檢查整合結果。

以下範例同時列出 ARB 中填寫的訊息定義，以及程式傳值後使用者最終看到的內容。

1. 在訊息中插入變數

   ARB 定義：

   ```json
   "welcomeUser": "你好，{name}"
   ```

   最終顯示：當 `name = 小明` 時，顯示「你好，小明」。翻譯時可以調整整句話，但必須保留 `{name}`。

2. 依數量顯示不同訊息

   ARB 定義：

   ```json
   "fileCount": "{count, plural, =0{沒有檔案} =1{1 個檔案} other{{count} 個檔案}}"
   ```

   最終顯示：

   - `count = 0` →「沒有檔案」
   - `count = 1` →「1 個檔案」
   - `count = 3` →「3 個檔案」

   `=0`、`=1`、`other` 分別代表數量為 0、數量為 1 和其他數量。只翻譯各分支大括號內顯示給使用者的文字。

3. 依變數值顯示不同訊息

   ARB 定義：

   ```json
   "taskState": "{state, select, running{執行中} paused{已暫停} other{未知狀態}}"
   ```

   最終顯示：

   - `state = running` →「執行中」
   - `state = paused` →「已暫停」
   - 其他值 →「未知狀態」

   `running`、`paused`、`other` 是程式傳入的分支值，不能翻譯；只翻譯其後大括號內的顯示文字。

## flutter開發

每次提交PR前請務必執行 `dart format ./ui/flutter`

如果需要編輯 api/models，請打開build_runner watcher:

```
flutter pub run build_runner watch
```
