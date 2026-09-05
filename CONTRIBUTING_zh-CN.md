# Gopeed 贡献指南

首先感谢您对贡献代码感兴趣，这份指南将帮助您更好的参与到 Gopeed 的开发中来。

## 分支说明

本项目只有一个主分支，即 `main` 分支，如果您想要参与到 Gopeed 的开发中来，请先 fork 本项目，然后在您的 fork 项目中进行开发，开发完成后再向本项目提交
PR，合并到 `main` 分支。

## 本地开发

建议通过 web 端进行开发调试，首先启动后端服务，通过命令行 `go run cmd/api/main.go` 启动 ，服务启动默认端口为 `9999`，然后以 `debug` 模式启动前端
flutter 项目即可运行。

## 翻译

Flutter 国际化文件位于 `ui/flutter/lib/l10n`，请以 `app_en.arb` 为源模板，修改或新增
`app_<locale>.arb`（例如 `app_de.arb` 或 `app_zh_TW.arb`）。每个语种必须与英文模板包含相同的文案 key。

只翻译文案值，不要修改 `{count}`、`{name}` 等花括号中的变量名。以 `@` 开头的元数据 key 不需要翻译。
提交时只包含修改过的 ARB 源文件，不需要生成或提交 `app_localizations*.dart`。PR 的 CI 会自动校验所有语种、
生成国际化代码并检查集成结果。

下面的例子同时给出 ARB 中填写的文案定义，以及程序传值后用户最终看到的内容。

1. 在文案中插入变量

   ARB 定义：

   ```json
   "welcomeUser": "你好，{name}"
   ```

   最终显示：当 `name = 小明` 时，显示“你好，小明”。翻译时可以调整整句话，但必须保留 `{name}`。

2. 根据数量显示不同文案

   ARB 定义：

   ```json
   "fileCount": "{count, plural, =0{没有文件} =1{1 个文件} other{{count} 个文件}}"
   ```

   最终显示：

   - `count = 0` → “没有文件”
   - `count = 1` → “1 个文件”
   - `count = 3` → “3 个文件”

   `=0`、`=1`、`other` 分别表示数量为 0、数量为 1 和其他数量。只翻译每个分支花括号内显示给用户的文字。

3. 根据变量值显示不同文案

   ARB 定义：

   ```json
   "taskState": "{state, select, running{运行中} paused{已暂停} other{未知状态}}"
   ```

   最终显示：

   - `state = running` → “运行中”
   - `state = paused` → “已暂停”
   - 其他值 → “未知状态”

   `running`、`paused`、`other` 是程序传入的分支值，不能翻译；只翻译它们后面花括号内的显示文字。

## flutter开发

每次提交前请务必`dart format ./ui/flutter`

如果要编辑api/models，请打开build_runner watcher:

```
flutter pub run build_runner watch
```
