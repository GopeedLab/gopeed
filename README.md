[![Test Status](https://github.com/monkeyWie/gopeed/workflows/test/badge.svg)](https://github.com/monkeyWie/gopeed/actions?query=workflow%3Atest)
[![Codecov](https://codecov.io/gh/monkeyWie/gopeed/branch/main/graph/badge.svg)](https://codecov.io/gh/monkeyWie/gopeed)
[![Release](https://img.shields.io/github/release/monkeyWie/gopeed.svg?style=flat-square)](https://github.com/monkeyWie/gopeed/releases)

![](_docs/img/banner.png)

## 介绍

Gopeed 是一款由`Golang`+`flutter`开发的高速下载器，支持（HTTP、BitTorrent、Magnet）协议下载，并且支持全平台使用。

## 安装

[点击前往](https://github.com/monkeyWie/gopeed/releases/latest)

### 命令行工具

使用`go install`安装：

```bash
go install github.com/monkeyWie/gopeed/cmd/gopeed
```

**已支持平台**

|         | 386 | amd64 | arm64 |
| ------- | --- | ----- | ----- |
| windows | ❌  | ✅    | ❌    |
| macos   | ❌  | ✅    | ✅    |
| linux   | ❌  | ✅    | ✅    |
| android | ✅  | ✅    | ✅    |
| ios     | ❌  | ❌    | ❌    |
| web     | ✅  | ✅    | ✅    |

## 界面展示

![](_docs/img/ui-demo.png)

## 开发

本项目分为前端和后端两个部分，前端使用`flutter`，后端使用`Golang`，两边通过`http`协议进行通讯，在 unix 系统下，使用的是`unix socket`，在 windows 系统下，使用的是`tcp`协议。

> 前端代码位于`ui/flutter`目录下。

### 克隆项目

```bash
git clone git@github.com:monkeyWie/gopeed.git
```

### 环境要求

1. Golang 1.9+
2. Flutter 3.0+

### 编译

#### 桌面端

首先需要按照[flutter desktop 官网文档](https://docs.flutter.dev/development/platform-integration/desktop)进行环境配置，然后需要准备好`cgo`环境，具体可以自行搜索。

构建命令：

- macos

```bash
go build -tags nosqlite -ldflags="-w -s" -buildmode=c-shared -o bin/libgopeed.dylib github.com/monkeyWie/gopeed/bind/desktop
cd ui/flutter
flutter build macos
```

#### 移动端

同样的，首先需要把`flutter`环境配置好，具体可以参考官网文档，然后也是需要准备好`cgo`环境，接着安装`gomobile`：

```bash
go install golang.org/x/mobile/cmd/gomobile@latest
gomobile init
```

构建命令：

- android

```bash
gomobile bind -tags nosqlite -ldflags="-w -s" -o ui/flutter/android/app/libs/libgopeed.aar -target=android -androidapi 19 -javapkg=com.gopeed github.com/monkeyWie/gopeed/bind/mobile
cd ui/flutter
flutter build apk
```

#### Web 端（推荐本地调试使用）

Web 端直接与后端 http 服务通讯，不需要额外准备环境。

构建命令：

```bash
cd ui/flutter
flutter build web
cd ../../
cp -r ui/flutter/build/web cmd/web/dist
go build -tags nosqlite,web -ldflags="-s -w" -o bin/ github.com/monkeyWie/gopeed/cmd/web
```
