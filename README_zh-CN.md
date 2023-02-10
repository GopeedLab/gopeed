[![Test Status](https://github.com/monkeyWie/gopeed/workflows/test/badge.svg)](https://github.com/monkeyWie/gopeed/actions?query=workflow%3Atest)
[![Codecov](https://codecov.io/gh/monkeyWie/gopeed/branch/main/graph/badge.svg)](https://codecov.io/gh/monkeyWie/gopeed)
[![Release](https://img.shields.io/github/release/monkeyWie/gopeed.svg?style=flat-square)](https://github.com/monkeyWie/gopeed/releases)
[![Discord](https://img.shields.io/discord/1037992631881449472?label=Discord&logo=discord&style=social)](https://discord.gg/ZUJqJrwCGB)

![](_docs/img/banner.png)

[English](/README.md) | [中文](/README_zh-CN.md)

## 介绍

Gopeed 是一款由`Golang`+`Flutter`开发的高速下载器，支持（HTTP、BitTorrent、Magnet）协议下载，并且支持全平台使用。

## 安装

**已支持平台**

- [x] windows
- [x] macos
- [x] linux
- [x] android
- [ ] ios
- [x] web
- [x] docker

[前往下载](https://github.com/monkeyWie/gopeed/releases/latest)

> 注：macos 版本运行如果提示损坏，请在终端执行 `xattr -d com.apple.quarantine /Applications/Gopeed.app` 命令

### 命令行工具

使用`go install`安装：

```bash
go install github.com/monkeyWie/gopeed/cmd/gopeed
```

### Docker

#### 直接运行

```bash
docker run -d -p 9999:9999 -v /path/to/download:/download liwei2633/gopeed
```

#### 使用 Docker Compose

```bash
docker-compose up -d
```

#### 访问服务

当 docker 容器运行时，可以通过 `http://localhost:9999` 访问 web 页面。
> 提示：在设置页面把下载路径修改为 `/download` 以便在宿主机访问下载完的文件。

## 打赏

如果觉得项目对你有帮助，请考虑[打赏](/.donate/index.md)以支持这个项目的发展，非常感谢！

## 界面展示

![](_docs/img/ui-demo.png)

## 开发

本项目分为前端和后端两个部分，前端使用`flutter`，后端使用`Golang`，两边通过`http`协议进行通讯，在 unix 系统下，使用的是`unix socket`，在 windows 系统下，使用的是`tcp`协议。

> 前端代码位于`ui/flutter`目录下。

### 环境要求

1. Golang 1.19+
2. Flutter 3.0+

### 克隆项目

```bash
git clone git@github.com:monkeyWie/gopeed.git
```

### 贡献代码

请参考[贡献指南](CONTRIBUTING_zh-CN.md)

### 编译

#### 桌面端

首先需要按照[flutter desktop 官网文档](https://docs.flutter.dev/development/platform-integration/desktop)进行环境配置，然后需要准备好`cgo`环境，具体可以自行搜索。

构建命令：

- windows

```bash
go build -tags nosqlite -ldflags="-w -s" -buildmode=c-shared -o ui/flutter/windows/libgopeed.dll github.com/monkeyWie/gopeed/bind/desktop
cd ui/flutter
flutter build windows
```

- macos

```bash
go build -tags nosqlite -ldflags="-w -s" -buildmode=c-shared -o ui/flutter/macos/Frameworks/libgopeed.dylib github.com/monkeyWie/gopeed/bind/desktop
cd ui/flutter
flutter build macos
```

- linux

```bash
go build -tags nosqlite -ldflags="-w -s" -buildmode=c-shared -o ui/flutter/linux/bundle/lib/libgopeed.so github.com/monkeyWie/gopeed/bind/desktop
cd ui/flutter
flutter build linux
```

#### 移动端

同样的也是需要准备好`cgo`环境，接着安装`gomobile`：

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

#### Web 端

构建命令：

```bash
cd ui/flutter
flutter build web
cd ../../
rm -rf cmd/web/dist
cp -r ui/flutter/build/web cmd/web/dist
go build -tags nosqlite,web -ldflags="-s -w" -o bin/ github.com/monkeyWie/gopeed/cmd/web
```

## 开源许可

基于 [GPLv3](LICENSE) 协议开源。
