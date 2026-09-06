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
- [THE TRENDY MERMAID](https://studyquests.github.io/the-trendy-mermaid.html)
- [CATEGORY PUZZLE 7](https://iskillquest.pages.dev/category-puzzle-7.html)
- [WAR STATE IO CONQUER BATTLES](https://learnquester.github.io/war-state-io-conquer-battles.html)
- [APPLE WORM](https://quizverses-9d2f2.web.app/apple-worm.html)
- [ELLIE AND FRIENDS ART BLOOM AESTHETIC](https://quizverses.pages.dev/ellie-and-friends-art-bloom-aesthetic.html)
- [CATEGORY BATTLE](https://quizverses-9d2f2.web.app/category-battle.html)
- [LOOP GHOST](https://quizverses.pages.dev/loop-ghost.html)
- [STICK COLOR WAR](https://quizverses-9d2f2.web.app/stick-color-war.html)
- [MISSION SANTA DELIVER THE GIFTS](https://studyplaying.github.io/mission-santa-deliver-the-gifts.html)
- [CATEGORY BRAIN260](https://learnquester.pages.dev/category-brain260.html)
- [FURRY WEDDING PROPOSAL](https://quizverses.github.io/furry-wedding-proposal.html)
- [ELLIE AND FRIENDS VENICE CARNIVAL](https://quizverses-9d2f2.web.app/ellie-and-friends-venice-carnival.html)
- [FUN IQ PUZZLE](https://studyplayings.pages.dev/fun-iq-puzzle.html)
- [CATEGORY TITANIUMNETWORK](https://quizverses.pages.dev/category-titaniumnetwork.html)
- [SPACE SHOOTER SPEED TYPING CHALLENGE](https://quizverses-9d2f2.web.app/space-shooter-speed-typing-challenge.html)
- [PIN MASTER](https://studyplayings.pages.dev/pin-master.html)
- [HUGGY MIX SPRUNKI MUSIC BOX](https://learnquesters.pages.dev/huggy-mix-sprunki-music-box.html)
- [CATEGORY PUZZLE 6](https://quizverses.github.io/category-puzzle-6.html)
- [CATEGORY MOUSE](https://studyplayings.web.app/category-mouse.html)
- [SINGLE STROKE ENERGY LINE PUZZLE](https://quizverses.github.io/single-stroke-energy-line-puzzle.html)
- [CATEGORY SCHOOL UNBLOCKER](https://quizverses.pages.dev/category-school-unblocker.html)
- [CATEGORY QUIZ](https://quizverses.github.io/category-quiz.html)
- [MALL ANOMALY](https://studyplaying.github.io/mall-anomaly.html)
- [CATEGORY BATTLE524](https://quizverses.pages.dev/category-battle524.html)
- [CATEGORY FIGHTING124](https://quizverses.pages.dev/category-fighting124.html)
- [CATEGORY PUZZLE 4](https://thelearnquester.web.app/category-puzzle-4.html)
- [PIZZA PUZZLE](https://quizverses.github.io/pizza-puzzle.html)
- [CATEGORY EDUCATIONAL](https://quizverses.pages.dev/category-educational.html)
- [CATEGORY GROW99](https://quizverses.pages.dev/category-grow99.html)
- [OBBY CLIMB RACING](https://quizverses.github.io/obby-climb-racing.html)
- [ARROW COUNT MASTER](https://studyplayings.pages.dev/arrow-count-master.html)
- [DUNGEON MASTER CULT CRAFT](https://quizverses.github.io/dungeon-master-cult-craft.html)
- [ARROW SHIFT LOGIC TREE](https://studyplayings.pages.dev/arrow-shift-logic-tree.html)
- [CATEGORY WAR137](https://studyplayings.pages.dev/category-war137.html)
- [MICROPLASTICS FEEDING](https://quizverses.pages.dev/microplastics-feeding.html)
- [CAT VS GRANNY CAT SIMULATOR](https://learnquester.pages.dev/cat-vs-granny-cat-simulator.html)
- [CATEGORY MEDIEVAL15](https://quizverses.pages.dev/category-medieval15.html)
- [POPS QUEST](https://studyplaying.github.io/pops-quest.html)
- [BOUNCEPOP QUEST](https://studyplayings.pages.dev/bouncepop-quest.html)
- [SAVAGE DEFENDERS](https://studyplaying.github.io/savage-defenders.html)
- [ASMR NAIL TREATMENT](https://quizverses.github.io/asmr-nail-treatment.html)
- [SWEEPER CURLING](https://studyplayings.pages.dev/sweeper-curling.html)
- [CATEGORY CONTROLLER 3](https://quizverses.github.io/category-controller-3.html)
- [MERGE FRUIT](https://studyplaying.github.io/merge-fruit.html)
- [PUMPKING VS MUMMY](https://quizverses-9d2f2.web.app/pumpking-vs-mummy.html)
- [OBBY MASSIVE ATTACK](https://learnquesters.pages.dev/obby-massive-attack.html)
- [SUPER SLIME](https://quizverses.github.io/super-slime.html)
- [AVATAR MASTER FIX UP FACE](https://studyplayings.web.app/avatar-master-fix-up-face.html)
- [CATEGORY RELAXING223](https://studyplayings.pages.dev/category-relaxing223.html)
- [ONU LIVE](https://learnquester.github.io/onu-live.html)
- [CATEGORY DRESS UP 2](https://quizverses.github.io/category-dress-up-2.html)
- [CATEGORY SANDBOX41](https://studyplayings.web.app/category-sandbox41.html)
- [CATEGORY MOUSE1 707](https://quizverses-9d2f2.web.app/category-mouse1-707.html)
- [CATEGORY WAR137](https://quizverses.github.io/category-war137.html)
- [CATEGORY DIRT BIKE18](https://quizverses.github.io/category-dirt-bike18.html)
- [INDEX19](https://studyplayings.web.app/index19.html)
- [CATEGORY MOBILE2 095](https://quizverses.github.io/category-mobile2-095.html)
- [SPRUNKI COLORING BOOK](https://studyplayings.pages.dev/sprunki-coloring-book.html)
- [PULL THE THREAD PUZZLE](https://studyplaying.github.io/pull-the-thread-puzzle.html)
- [HEXA SORT WINTER EDITION](https://quizverses.pages.dev/hexa-sort-winter-edition.html)
- [MATCHING PUZZLE](https://studyplaying.github.io/matching-puzzle.html)
- [CATEGORY SPORTS](https://quizverses.pages.dev/category-sports.html)
- [HEADLESS JOE](https://quizverses.github.io/headless-joe.html)
- [ONLINE CAR DESTRUCTION SIMULATOR 3D](https://quizverses.pages.dev/online-car-destruction-simulator-3d.html)
- [CATEGORY SPEED158](https://studyplayings.web.app/category-speed158.html)
- [PIXEL MINI GOLF](https://thelearnquester.web.app/pixel-mini-golf.html)
- [CATEGORY FIGHTING124](https://quizverses.github.io/category-fighting124.html)
- [CYBER ARROW](https://learnquester.github.io/cyber-arrow.html)
- [SHELF SHIFT MATCH](https://quizverses.pages.dev/shelf-shift-match.html)
- [CATEGORY SIMULATION 3](https://quizverses.github.io/category-simulation-3.html)
- [GLADIATORS MERGE AND FIGHT](https://studyplayings.pages.dev/gladiators-merge-and-fight.html)
- [MERGE 2048 CAKE](https://studyplaying.github.io/merge-2048-cake.html)
- [CATEGORY MATH29](https://quizverses-9d2f2.web.app/category-math29.html)
- [CATEGORY CASUAL 4](https://learnquester.github.io/category-casual-4.html)
- [CATEGORY PUZZLE 5](https://studyplayings.pages.dev/category-puzzle-5.html)
- [TILE HEXA SORT](https://quizverses.pages.dev/tile-hexa-sort.html)
- [HALLOWEEN MAKEUP TRENDS](https://quizverses.github.io/halloween-makeup-trends.html)
- [DRAG MATCH MAZE TILE](https://thelearnquester.web.app/drag-match-maze-tile.html)
- [MADNESS SHERIFFS COMPOUND OFFICIAL](https://learnquesters.pages.dev/madness-sheriffs-compound-official.html)
- [IDOL LIVESTREAM DOLL DRESS UP](https://learnquester.github.io/idol-livestream-doll-dress-up.html)
- [OBBY DUMB OR GENIUS IQ TEST](https://quizverses.pages.dev/obby-dumb-or-genius-iq-test.html)
- [MY COTTAGECORE AESTHETIC LOOK](https://learnquesters.pages.dev/my-cottagecore-aesthetic-look.html)
- [HIDE AND SEEK HORROR ESCAPE](https://quizverses-9d2f2.web.app/hide-and-seek-horror-escape.html)
- [CATEGORY GROW GAMES](https://quizverses.github.io/category-grow-games.html)
- [MAD TRUCK](https://learnquesters.pages.dev/mad-truck.html)
- [CATEGORY TOWER DEFENSE118](https://quizverses.github.io/category-tower-defense118.html)
- [WORDS FROM WORDS](https://thelearnquester.web.app/words-from-words.html)
- [STEAMPUNK TOWER BUILDER](https://quizverses.github.io/steampunk-tower-builder.html)
- [SWORD AND SPIN](https://quizverses.github.io/sword-and-spin.html)
- [CATEGORY SOCCER](https://quizverses.github.io/category-soccer.html)
- [SPORTSBALL MERGE](https://quizverses.github.io/sportsball-merge.html)
- [GOOD SORT MASTER TRIPLE MATCH](https://quizverses-9d2f2.web.app/good-sort-master-triple-match.html)
- [CATEGORY STICKMAN175](https://studyplayings.pages.dev/category-stickman175.html)
- [CATEGORY SORTING44](https://thelearnquester.web.app/category-sorting44.html)
- [CATEGORY PUZZLE 5](https://quizverses.github.io/category-puzzle-5.html)
- [FIDGET TOYS POP IT](https://learnquester.github.io/fidget-toys-pop-it.html)
- [YUMMY TALES 4](https://studyplayings.pages.dev/yummy-tales-4.html)
- [CATEGORY FPS 2](https://studyplayings.web.app/category-fps-2.html)
- [BARREL ROLLER AMAZING RUNNER](https://quizverses.pages.dev/barrel-roller-amazing-runner.html)
- [CATEGORY CAR 3](https://quizverses.github.io/category-car-3.html)
- [ASTRAL ESCAPE](https://learnquesters.pages.dev/astral-escape.html)
- [THE FLOWERS MERGE AND SELL BOUQUETS](https://quizverses.pages.dev/the-flowers-merge-and-sell-bouquets.html)
- [NSR STREET CAR RACING](https://learnquester.pages.dev/nsr-street-car-racing.html)
- [CATEGORY SURVIVAL366](https://quizverses-9d2f2.web.app/category-survival366.html)
- [FUNNY FRUITS MERGE AND GATHER WATERMELON](https://studyquests.pages.dev/funny-fruits-merge-and-gather-watermelon.html)
- [HARD ROCK ZOMBIE TRUCK](https://learnquesters.pages.dev/hard-rock-zombie-truck.html)
- [MEDIEVAL ESCAPE](https://studyplayings.pages.dev/medieval-escape.html)
- [NEW YEAR MAKEUP TRENDS](https://studyquesthub.web.app/new-year-makeup-trends.html)
- [TROPICAL CUBES 2048](https://studyplaying.github.io/tropical-cubes-2048.html)
- [MEGA SHARK](https://quizverses.github.io/mega-shark.html)
- [SUDOKU BRAIN BLOCKS](https://studyplaying.github.io/sudoku-brain-blocks.html)
- [BUS JAM ESCAPE](https://learnquesters.pages.dev/bus-jam-escape.html)
- [SKYSCRAPER TO THE SKY](https://studyquesthub.web.app/skyscraper-to-the-sky.html)
- [LITTLE ALCHEMY](https://quizverses-9d2f2.web.app/little-alchemy.html)
- [CATEGORY SOCCER 2](https://studyplayings.pages.dev/category-soccer-2.html)
- [ROYAL PUZZLE BURST](https://thelearnquester.web.app/royal-puzzle-burst.html)
- [MAHJONG CRIMES PUZZLE STORY](https://studyplaying.github.io/mahjong-crimes-puzzle-story.html)
- [BITBALL](https://studyquesthub.web.app/bitball.html)
- [RUN N SHOOT](https://studyquests.pages.dev/run-n-shoot.html)
- [PANDA KITCHEN IDLE TYCOON](https://quizverses-9d2f2.web.app/panda-kitchen-idle-tycoon.html)
- [CUBEREALM IO](https://studyquesthub.web.app/cuberealm-io.html)
- [CATEGORY AGILITY](https://studyplaying.github.io/category-agility.html)
- [POTION MERGE WITCH](https://thelearnquester.web.app/potion-merge-witch.html)
- [RENT OUT LANDLORD TYCOON](https://studyplaying.github.io/rent-out-landlord-tycoon.html)
- [SCARY PAIRS](https://studyplayings.web.app/scary-pairs.html)
- [MINICRAFT CHEF CAKE WARS](https://thelearnquester.web.app/minicraft-chef-cake-wars.html)
- [CELEBRITY THANKSGIVING PREP](https://studyquesthub.web.app/celebrity-thanksgiving-prep.html)
- [CATEGORY SCRATCH17](https://studyplayings.web.app/category-scratch17.html)
- [PAINT MASTER](https://studyquests.pages.dev/paint-master.html)
- [CATEGORY SURVIVAL366](https://studyplayings.pages.dev/category-survival366.html)
