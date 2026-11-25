# Gopeed 贡献指南

首先感谢您对贡献代码感兴趣，这份指南将帮助您更好的参与到 Gopeed 的开发中来。

## 分支说明

先 fork Gopeed，从最新的 `main` 创建开发分支，再向 `main` 提交 PR。

## 本地开发

使用与 CI 一致的 Go 1.27.4 和 Flutter 3.41.6（Dart 3.11.4）。可以先通过 Web 端调试，在仓库根目录启动 Go REST API。

```sh
go run -tags nosqlite ./cmd/api
```

API 监听 `127.0.0.1:9999`。在另一个终端中从仓库根目录运行 Flutter Web。

```sh
cd ui/flutter
flutter pub get
flutter gen-l10n
flutter run -d chrome
```

开发桌面端或移动端时，需要为目标平台编译 Go 绑定。构建步骤见 [README](README_zh-CN.md)。

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

## 代码检查

用 `gofmt` 格式化修改过的 Go 文件，再在仓库根目录运行 Go 测试。

```sh
go test -tags nosqlite ./...
```

用 `dart format` 格式化修改过的 Dart 文件。按上文安装 Flutter 依赖并生成国际化代码后，从仓库根目录运行以下检查。

```sh
cd ui/flutter
flutter analyze
flutter test
```

如果修改了 `lib/api/model/` 中带注解的模型，请在 `ui/flutter` 重新生成 JSON 序列化代码。

```sh
dart run build_runner build
```
