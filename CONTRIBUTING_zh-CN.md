# Gopeed 贡献指南

首先感谢您对贡献代码感兴趣，这份指南将帮助您更好的参与到 Gopeed 的开发中来。

## 分支说明

本项目只有一个主分支，即 `main` 分支，如果您想要参与到 Gopeed 的开发中来，请先 fork 本项目，然后在您的 fork 项目中进行开发，开发完成后再向本项目提交
PR，合并到 `main` 分支。

## 本地开发

建议通过 web 端进行开发调试，首先启动后端服务，通过命令行 `go run cmd/api/main.go` 启动 ，服务启动默认端口为 `9999`，然后以 `debug` 模式启动前端
flutter 项目即可运行。

## 翻译
 
Gopeed 的国际化文件位于 `ui/flutter/lib/i18n/langs` 目录下，只需要在该目录下添加对应的语言文件即可。

请注意以 `en_us.dart` 为参照进行翻译。

## flutter开发

每次提交前请务必`dart format ./ui/flutter`

如果要编辑api/models，请打开build_runner watcher:

```
flutter pub run build_runner watch
```

