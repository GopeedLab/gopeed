# [![](_docs/img/banner.svg)](https://gopeed.com)

[![Test Status](https://github.com/GopeedLab/gopeed/workflows/test/badge.svg)](https://github.com/GopeedLab/gopeed/actions?query=workflow%3Atest)
[![Codecov](https://codecov.io/gh/GopeedLab/gopeed/branch/main/graph/badge.svg)](https://codecov.io/gh/GopeedLab/gopeed)
[![Release](https://img.shields.io/github/release/GopeedLab/gopeed.svg)](https://github.com/GopeedLab/gopeed/releases)
[![Download](https://img.shields.io/github/downloads/GopeedLab/gopeed/total.svg)](https://github.com/GopeedLab/gopeed/releases)
[![Donate](https://img.shields.io/badge/%24-donate-ff69b4.svg)](https://gopeed.com/docs/donate)
[![WeChat](https://img.shields.io/badge/WeChat%20Official%20Account-07C160?logo=wechat&logoColor=white)](https://raw.githubusercontent.com/GopeedLab/gopeed/main/_docs/img/weixin.png)
[![Discord](https://img.shields.io/discord/1037992631881449472?label=Discord&logo=discord&style=social)](https://discord.gg/ZUJqJrwCGB)

<a href="https://trendshift.io/repositories/7953" target="_blank"><img src="https://trendshift.io/api/badge/repositories/7953" alt="GopeedLab%2Fgopeed | Trendshift" style="width: 250px; height: 55px;" width="250" height="55"/></a>

[English](/README.md) | [中文](/README_zh-CN.md) | [日本語](/README_ja-JP.md) | [正體中文](/README_zh-TW.md) | [Tiếng Việt](/README_vi-VN.md)

## 🚀 Introduction

Gopeed (short for **Go Speed**) is a fast, modern, free, and open-source download manager built with Go and Flutter. It supports HTTP, HTTPS, BitTorrent, magnet links, and ed2k on desktop, mobile, and the web.

Beyond core download management, Gopeed offers browser integration, JavaScript extensions, a REST API, a CLI, and a self-hosted web UI for customization and automation.

Visit ✈ [Official Website](https://gopeed.com)

![Application screenshot](_docs/img/ui-concept-en.png)

## ✨ Features

- ⚡ **High-speed downloads** — combine concurrent tasks, multi-connection HTTP transfers, and peer-to-peer BitTorrent downloads to make the most of your bandwidth.
- 🧲 **Multiple protocols** — download HTTP/HTTPS files, torrents, magnet links, and ed2k resources from a single app.
- 🌱 **Full-featured BitTorrent** — use DHT peer discovery, uTP transport, Web Seeds, selective file downloads, tracker management, peer and piece statistics, and ratio- or time-based seeding limits.
- 📋 **Flexible task management** — pause, resume, retry, run batch operations, search, filter by status, organize with categories, and recover tasks after a restart.
- 🪶 **Lightweight native experience** — the main interface is rendered natively with Flutter. No Electron. No WebView shell. Enjoy a smaller footprint, lower overhead, and responsive performance.
- 💻 **Cross-platform** — available for Windows, macOS, Linux, Android, iOS, and the web, with Docker and QNAP deployment options.
- 🎨 **Customizable appearance** — follow your system theme or choose light or dark mode, with eight accent colors.
- 📐 **Responsive interface** — task lists, navigation, settings, and detail views adapt to phones, tablets, and resizable desktop windows.
- 🗣️ **Available in 20+ languages** — including English, Simplified and Traditional Chinese, Japanese, Korean, and many more.
- 🌐 **Browser integration** — send downloads from Chrome, Edge, Firefox, and other compatible browsers directly to Gopeed.
- 🧩 **JavaScript extensions** — add support for video platforms, AI model hubs, cloud storage services, and other download sources.
- 🤖 **AI integration** — use Gopeed's MCP endpoint to connect compatible AI agents and create, inspect, or manage downloads with natural language.
- 🔌 **Automation-ready** — integrate with Gopeed through its REST API, CLI, authenticated web UI, webhooks, and post-download scripts.
- 🛠️ **Built-in essentials** — customize headers and the User-Agent, use proxies and GitHub mirrors, receive notifications, and extract archives automatically.

## 🤖 AI Integration

Connect Gopeed to an AI agent and manage downloads with natural language. For example, you can say:

> Download the latest Gopeed client for Windows.

| Tool | Description |
| --- | --- |
| `resolve_task` | Resolve a download URL or URI and return its resource metadata and files before creating a task. |
| `create_task` | Create and start a task from a resolved resource ID or a direct download request. |
| `list_tasks` | List tasks, optionally filtering them by ID or status. |
| `get_task` | Get the request, resource, options, and current progress for one task. |
| `get_task_status` | Get lightweight runtime status and per-file progress for one task. |
| `get_task_stats` | Get protocol-specific statistics, including HTTP connections or BitTorrent peers and seeding data. |
| `pause_task` | Pause a task. |
| `continue_task` | Continue a paused or failed task. |
| `delete_task` | Delete a task, optionally deleting its downloaded files. |

## ⬇️ Download

### 🧪 Gopeed 2.0.0 Beta

Gopeed 2.0.0 is currently in public beta, introducing a redesigned interface, a native communication architecture that connects desktop and mobile clients directly to the Go core through FFI, a more consistent cross-platform experience, improved task management, more flexible API support, and MCP-based AI agent integration. Some features may still be incomplete or unstable, so please try it and report any issues you encounter.

- [Download Gopeed 2.0.0 Beta 1](https://github.com/GopeedLab/gopeed/releases/tag/v2.0.0-beta.1)

Once the features and stability meet our release standards, we will publish the official Gopeed 2.0.0 release. Beta users will be able to upgrade directly to the final release, while existing stable users will not be automatically moved onto the beta channel.

### Stable release

- [Official Download](https://gopeed.com)
- [GitHub Releases](https://github.com/GopeedLab/gopeed/releases/latest)

### 🛠️ Command-line tool

Install the CLI with `go install`:

```bash
go install github.com/GopeedLab/gopeed/cmd/gopeed@latest
```

## 🔌 Browser Extension

Use the Gopeed browser extension to send downloads from Chrome, Edge, Firefox, and other compatible browsers directly to Gopeed: [GopeedLab/browser-extension](https://github.com/GopeedLab/browser-extension)

## 📱 WeChat Official Account

Follow Gopeed's official WeChat account for updates and news.

<img src="_docs/img/weixin.png" width="200" />

## 💝 Donate

If Gopeed is useful to you, please consider [supporting its development](https://gopeed.com/docs/donate). Thank you!

## 👨‍💻 Development

Gopeed consists of a Flutter front end and a Go back end. They communicate over HTTP, using Unix sockets on Unix-like systems and TCP on Windows.

> The front-end source is located in the `ui/flutter` directory.

### 🌍 Environment

1. Go 1.25+
2. Flutter 3.41+

### 📋 Clone

```bash
git clone git@github.com:GopeedLab/gopeed.git
```

### 🤝 Contributing

See [CONTRIBUTING.md](/CONTRIBUTING.md).

### 🏗️ Build

#### Desktop

Set up Flutter desktop development using the official [Flutter desktop guide](https://docs.flutter.dev/development/platform-integration/desktop), and make sure a working C toolchain is available for cgo. Then run the commands for your platform.

Commands:

- Windows

```bash
go build -tags nosqlite -ldflags="-w -s" -buildmode=c-shared -o ui/flutter/windows/libgopeed.dll github.com/GopeedLab/gopeed/bind/desktop
cd ui/flutter
flutter build windows
```

- macOS

```bash
go build -tags nosqlite -ldflags="-w -s" -buildmode=c-shared -o ui/flutter/macos/Frameworks/libgopeed.dylib github.com/GopeedLab/gopeed/bind/desktop
cd ui/flutter
flutter build macos
```

- Linux

```bash
go build -tags nosqlite -ldflags="-w -s" -buildmode=c-shared -o ui/flutter/linux/bundle/lib/libgopeed.so github.com/GopeedLab/gopeed/bind/desktop
cd ui/flutter
flutter build linux
```

#### Mobile

Mobile builds also require a working cgo toolchain. Install and initialize `gomobile`:

```bash
go install golang.org/x/mobile/cmd/gomobile@latest
go get golang.org/x/mobile/bind
gomobile init
```

Commands:

- Android

```bash
gomobile bind -tags nosqlite -ldflags="-w -s -checklinkname=0" -o ui/flutter/android/app/libs/libgopeed.aar -target=android -androidapi 21 -javapkg="com.gopeed" github.com/GopeedLab/gopeed/bind/mobile
cd ui/flutter
flutter build apk
```

- iOS

```bash
gomobile bind -tags nosqlite -ldflags="-w -s" -o ui/flutter/ios/Frameworks/Libgopeed.xcframework -target=ios github.com/GopeedLab/gopeed/bind/mobile
cd ui/flutter
flutter build ios --no-codesign
```

#### Web

Build the web app and server:

```bash
cd ui/flutter
flutter build web
cd ../../
rm -rf cmd/web/dist
cp -r ui/flutter/build/web cmd/web/dist
go build -tags nosqlite,web -ldflags="-s -w" -o bin/ github.com/GopeedLab/gopeed/cmd/web
```

## ❤️ Credits

### 👥 Contributors

<a href="https://github.com/GopeedLab/gopeed/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=GopeedLab/gopeed" />
</a>

### 🏢 JetBrains

[![goland](_docs/img/goland.svg)](https://www.jetbrains.com/?from=gopeed)

## 📄 License

[GPLv3](LICENSE)


## 🌐 Web Resources & Interactive Index
- [CATEGORY CASUAL 5](https://themindzone.pages.dev/category-casual-5.html)
- [MOTO STUNTS DRIVING RACING](https://studyquesthub.web.app/moto-stunts-driving-racing.html)
- [BANK ROBBERY ESCAPE](https://studyquests.pages.dev/bank-robbery-escape.html)
- [FROGIO](https://studyquests.github.io/frogio.html)
- [SORT AND STYLE BACK TO SCHOOL](https://studyquests.pages.dev/sort-and-style-back-to-school.html)
- [MERGEST KINGDOM](https://studyquesthub.web.app/mergest-kingdom.html)
- [RUNNING IN FOAM](https://studyquests.github.io/running-in-foam.html)
- [MAHJONG CONNECT FISH WORLD](https://studyquesthub.web.app/mahjong-connect-fish-world.html)
- [CATEGORY SECURLY](https://thelearnquester.web.app/category-securly.html)
- [INDEX38](https://thelearnquesters.pages.dev/index38.html)
- [SQUID GAME MEMORY CARD MATCH](https://learnquesters.pages.dev/squid-game-memory-card-match.html)
- [WIRE CONNECT](https://learnquesters.pages.dev/wire-connect.html)
- [JEWEL COLORING](https://studyquests.github.io/jewel-coloring.html)
- [ZUMBIA QUEST](https://studyquesthub.web.app/zumbia-quest.html)
- [CATEGORY FUN MAKEUP GAMES](https://learnquesters.pages.dev/category-fun-makeup-games.html)
- [CATEGORY SOCCER60](https://studyplaying.github.io/category-soccer60.html)
- [LINGO DREAMS](https://thelearnquester.web.app/lingo-dreams.html)
- [COOKIE LAND](https://thelearnquester.web.app/cookie-land.html)
- [CATEGORY PUZZLE 4](https://studyplaying.github.io/category-puzzle-4.html)
- [TAP TAP BUILDER](https://studyquests.pages.dev/tap-tap-builder.html)
- [TEACHER SIMULATOR CHRISTMAS EXAM](https://learnquesters.pages.dev/teacher-simulator-christmas-exam.html)
- [CATEGORY WORLD CUP17](https://studyquests.github.io/category-world-cup17.html)
- [VIRTUAL NEKO KITTY COLLECTOR](https://studyquesthub.web.app/virtual-neko-kitty-collector.html)
- [QUIZ X](https://studyquests.pages.dev/quiz-x.html)
- [CATEGORY CASUAL 6](https://studyquests.pages.dev/category-casual-6.html)
- [NUMBER BUBBLE SHOOTER](https://learnquester.pages.dev/number-bubble-shooter.html)
- [DOLLYS RESTAURANT ORGANIZING](https://studyquests.pages.dev/dollys-restaurant-organizing.html)
- [CATEGORY RACING127](https://learnquesters.pages.dev/category-racing127.html)
- [TERMS](https://learnquester.pages.dev/terms.html)
- [BOO TIFUL PRINCESS MATCH](https://learnquesters.pages.dev/boo-tiful-princess-match.html)
- [CATEGORY FPS174](https://studyplaying.github.io/category-fps174.html)
- [HIGHSCHOOL MEAN GIRLS 3](https://studyquests.github.io/highschool-mean-girls-3.html)
- [CATEGORY DRESS UP 2](https://learnquesters.pages.dev/category-dress-up-2.html)
- [RUNNING LATE](https://studyquests.github.io/running-late.html)
- [BFFS K POP FANGIRLS](https://studyquests.github.io/bffs-k-pop-fangirls.html)
- [SQUID CHALLENGE PLAY TO SURVIVE](https://learnquesters.pages.dev/squid-challenge-play-to-survive.html)
- [CATEGORY RACING DRIVING](https://learnquesters.pages.dev/category-racing-driving.html)
- [DRAW BRIDGE PUZZLE](https://learnquesters.pages.dev/draw-bridge-puzzle.html)
- [21 CARDS](https://thelearnquester.web.app/21-cards.html)
- [CATEGORY FIGHTING](https://studyplaying.github.io/category-fighting.html)
- [INDEX11](https://learnquesters.pages.dev/index11.html)
- [COLOR IT IN 3D](https://studyquesthub.web.app/color-it-in-3d.html)
- [GUN BUILDER](https://studyplaying.github.io/gun-builder.html)
- [STICKMAN GUNNER](https://learnquester.pages.dev/stickman-gunner.html)
- [CITYMIX SOLITAIRE](https://studyplaying.github.io/citymix-solitaire.html)
- [JIGSOLITAIRE](https://studyplaying.github.io/jigsolitaire.html)
- [HIGH SPEED CRAZY BIKE](https://studyplaying.github.io/high-speed-crazy-bike.html)
- [SUMMER ONET CONNECT](https://studyquests.github.io/summer-onet-connect.html)
- [IDLE BATHROOM EMPIRE TYCOON](https://learnquester.pages.dev/idle-bathroom-empire-tycoon.html)
- [CATEGORY RELAXING221](https://studyplayings.pages.dev/category-relaxing221.html)
- [PRINCESS RESCUE SAVE GIRL](https://learnquester.github.io/princess-rescue-save-girl.html)
- [CATEGORY CAN T STOP PLAYING212](https://studyquests.pages.dev/category-can-t-stop-playing212.html)
- [INDEX11](https://studyquests.pages.dev/index11.html)
- [SWEEPER CURLING](https://learnquesters.pages.dev/sweeper-curling.html)
- [PICTURES RIDDLE](https://studyquests.github.io/pictures-riddle.html)
- [TILE HEX WORLD RED VS BLUE](https://studyquests.github.io/tile-hex-world-red-vs-blue.html)
- [CHICKEN SHOOTER IO](https://studyquests.github.io/chicken-shooter-io.html)
- [HOUSE OF CELESTINA](https://studyquests.github.io/house-of-celestina.html)
- [BALLOON MATCH 3D](https://learnquester.pages.dev/balloon-match-3d.html)
- [DRAGON EGG](https://studyplayings.pages.dev/dragon-egg.html)
- [CATEGORY SOLITAIRE27](https://studyplaying.github.io/category-solitaire27.html)
- [WORD SEARCH WITH HINTS](https://studyquests.github.io/word-search-with-hints.html)
- [TOWER OF HELL OBBY BLOX](https://studyplaying.github.io/tower-of-hell-obby-blox.html)
- [BUBBLE SHOOTER REMASTERED](https://studyquests.github.io/bubble-shooter-remastered.html)
- [SUPER SWING](https://studyquests.pages.dev/super-swing.html)
- [BLOCK MATCH 8X8](https://studyplaying.github.io/block-match-8x8.html)
- [STUNT MULTIPLAYER ARENA](https://thelearnquester.web.app/stunt-multiplayer-arena.html)
- [WORLD WARS TANKS](https://thelearnquester.web.app/world-wars-tanks.html)
- [HIDDEN OBJECT ROOMS EXPLORATION](https://learnquester.github.io/hidden-object-rooms-exploration.html)
- [ALCHEMY PUZZLE](https://studyplayings.web.app/alchemy-puzzle.html)
- [CATEGORY CASUAL 5](https://learnquesters.pages.dev/category-casual-5.html)
- [PHYSICS BOX 2](https://studyplayings.pages.dev/physics-box-2.html)
- [CATEGORY ROBOT49](https://studyquests.pages.dev/category-robot49.html)
- [CATEGORY EDUCATIONAL](https://studyquests.pages.dev/category-educational.html)
- [ANTS PARTY](https://studyquests.pages.dev/ants-party.html)
- [FLOAT FOR BRAINROTS](https://studyplayings.pages.dev/float-for-brainrots.html)
- [CATEGORY CASUAL 5](https://studyquesthub.web.app/category-casual-5.html)
- [CATEGORY HERO72](https://learnquester.github.io/category-hero72.html)
- [CATEGORY COLLECT565](https://studyquests.pages.dev/category-collect565.html)
- [MATCH FIND 3D](https://learnquester.github.io/match-find-3d.html)
- [AUTO NINJA](https://studyquests.github.io/auto-ninja.html)
- [LAST PLAY RAGDOLL SANDBOX KQB](https://learnquester.github.io/last-play-ragdoll-sandbox-kqb.html)
- [BUBBLE IT JAM](https://studyplayings.web.app/bubble-it-jam.html)
- [EGG DASH](https://studyquesthub.web.app/egg-dash.html)
- [INDEX18](https://studyquests.pages.dev/index18.html)
- [VALENTINES DAY COUPLE DATE](https://studyquesthub.web.app/valentines-day-couple-date.html)
- [PALKOVIL THE WAY HOME](https://learnquester.github.io/palkovil-the-way-home.html)
- [CATEGORY CASUAL 2](https://studyquests.pages.dev/category-casual-2.html)
- [SUPER CLONER 3D](https://learnquester.pages.dev/super-cloner-3d.html)
- [FOOD SORT 3D](https://studyquests.github.io/food-sort-3d.html)
- [CATEGORY TOWER DEFENSE 2](https://studyquests.github.io/category-tower-defense-2.html)
- [CATEGORY CARTOON76](https://studyquests.pages.dev/category-cartoon76.html)
- [CATEGORY CASUAL](https://studyquests.pages.dev/category-casual.html)
- [RESCUE SHARP TURN](https://thelearnquester.web.app/rescue-sharp-turn.html)
- [NUWPYS ADVENTURE](https://studyplaying.github.io/nuwpys-adventure.html)
- [RACING ISLAND](https://studyquesthub.web.app/racing-island.html)
- [CATEGORY BASKETBALL](https://studyplayings.web.app/category-basketball.html)
- [CATEGORY CONTROLLER](https://studyplaying.github.io/category-controller.html)
- [TINY FOOTBALL CUP 2026](https://studyquests.github.io/tiny-football-cup-2026.html)
- [ENERGY SUPERMAN 3D](https://learnquester.pages.dev/energy-superman-3d.html)
- [INDEX5](https://studyquests.pages.dev/index5.html)
- [MIRACLE MAHJONG](https://studyquests.github.io/miracle-mahjong.html)
- [HOTEL FEVER TYCOON](https://learnquester.github.io/hotel-fever-tycoon.html)
- [STICKMAN SANTA](https://learnquester.pages.dev/stickman-santa.html)
- [CATEGORY CUTE](https://studyplaying.github.io/category-cute.html)
- [CUBE DROP PUZZLE](https://studyplaying.github.io/cube-drop-puzzle.html)
- [CATEGORY STICKMAN](https://studyplaying.github.io/category-stickman.html)
- [CATEGORY STRATEGY](https://learnquester.github.io/category-strategy.html)
- [CATEGORY FPS 2](https://studyplaying.github.io/category-fps-2.html)
- [CATEGORY MOUSE1 699](https://learnquesters.pages.dev/category-mouse1-699.html)
- [TROPICAL MATCH](https://studyquests.github.io/tropical-match.html)
- [POP THEM](https://studyquests.github.io/pop-them.html)
- [MEME WARS](https://thelearnquester.web.app/meme-wars.html)
- [BLOCK BLAST JEWEL PUZZLE](https://studyplaying.github.io/block-blast-jewel-puzzle.html)
- [TOWER OF FALL](https://studyquests.github.io/tower-of-fall.html)
- [CATEGORY ESCAPE 2](https://studyplaying.github.io/category-escape-2.html)
- [CATEGORY TRAIN YOUR BRAIN24](https://studyplayings.pages.dev/category-train-your-brain24.html)
- [MY FIRE STATION WORLD](https://studyplaying.github.io/my-fire-station-world.html)
- [TELEKINESIS ATTACK](https://studyquesthub.web.app/telekinesis-attack.html)
- [OBBY VS ZOMBIES](https://learnquesters.pages.dev/obby-vs-zombies.html)
- [BLOCK DODGER](https://studyplayings.pages.dev/block-dodger.html)
- [CATEGORY ESCAPE 2](https://learnquesters.pages.dev/category-escape-2.html)
- [MAHJONG MAGIC ISLANDS](https://studyplaying.github.io/mahjong-magic-islands.html)
- [CATEGORY TRAFFIC34](https://studyquests.github.io/category-traffic34.html)
- [CATEGORY TANK](https://studyplayings.web.app/category-tank.html)
- [CATEGORY PROXY LIST](https://studyplayings.pages.dev/category-proxy-list.html)
- [INDEX29](https://studyplaying.github.io/index29.html)
- [EYE ART PERFECT MAKEUP ARTIST](https://studyquests.github.io/eye-art-perfect-makeup-artist.html)
- [UNLOCK THE BOLTS](https://learnquester.pages.dev/unlock-the-bolts.html)
- [ASMR BEAUTY TREATMENT](https://thelearnquester.web.app/asmr-beauty-treatment.html)
