# moejs 替换 goja 的实验评估

日期：2026-11-14。目标顺序：**真实扩展先跑通 → 判断实际收益 → 再决定是否正式替换**。

## 判断

**值得继续做完整宿主移植，但目前不足以批准直接替换生产 goja。**

三个真实扩展的完整 bundle 已在 moejs 中跑通离线解析和公开网站联网解析，结果与 goja 相同。热运行 CPU 收益明显；当前 Gopeed 的事件路径每次新建引擎、编译并加载 bundle，冷启动的收益远小于热运行倍数。压缩体积基本没有显著收益。

“扩展解析可运行”已验证。“真实下载、FFmpeg 合并、YouTube SABR/PoToken、WebView 和 Go 任务生命周期完整可用”尚未验证，不能把本实验当成这些路径已经兼容。

## 版本与方法

- Gopeed 基线：`be0a395447d788419e83acdfd693e48a1a1706bf`，分支基于该版 main。
- goja：`v0.0.0-20260106131823-651366fbe6e3`，沿用项目依赖。
- moejs：`v0.0.0-20261113160646-b2c7fed9c42d`，固定上游提交；纯 Go，当前主模块的 Go 1.26.0 足够。
- Apple M1、8 GiB RAM、macOS 14.6.1、Go 1.26.0；性能测试 `GOMAXPROCS=1`、`-pgo=off`、每项 500 ms、5 次重复，报告中位数。没有套用上游的 PGO 或上游插件跑分。
- 最终性能测量单独执行，未与本次构建或联网测试并发。记录在 [toolchain.txt](results/toolchain.txt)、[bench.txt](results/bench.txt)。`ns_min/ns_max` 见 [bench-summary.json](results/bench-summary.json)。这不是多机器统计结论。
- goja 使用生产 `engine.NewEngine`；moejs 使用其 eventloop/webapi 加 Gopeed 的 polyfill/XHR。两侧都运行未经改写的发布 bundle。moejs 使用 `MutableIntrinsics: true`，保留 goja 的可写 builtin 语义。
- 冷运行包含建引擎、宿主 JS 初始化、编译/加载 bundle、一次解析及关闭；热运行复用已加载 runtime。宿主初始化能力不完全对等：goja 含完整生产注入，moejs 尚未移植 File/VM/FFmpeg 等原生桥。**冷运行数字是两个现有原型的对比，不是完成生产移植后的承诺。**
- 回放提供合成 HTTP 响应，不替换 Bilibili SDK、YouTube.js 或 Twitter SDK。YouTube session/config 固定返回 503，验证本地 session fallback 与 player JSON 解析；未覆盖成功 config、真实 player decipher 或媒体流。
- 基准中 Gopeed namespace/task/Blob 等由相同 JS mock 提供，结果以 JSON 字符串导出用于比较；不包含完整 Go 对象桥接及持久化的成本，原型的 Resource 拷出另有功能验证，未纳入时延表。
- 每次结果包含文件名、URL、labels、请求路径，两端比较相同输出。额外使用相同 JS 的 buffered Response 再测热运行，降低原生 Response/Stream 差异的影响。

## 真实扩展验证

| 扩展 | 固定发布包提交 | bundle 大小 | 回放 / 联网解析 | 已验证的范围 |
| --- | --- | ---: | --- | --- |
| Bilibili | [`b91db1e`](https://github.com/monkeyWie/gopeed-extension-bilibili/tree/b91db1ea52e975d8de9823f01a7af5f9ea83792f) | 920,556 B | 两个引擎均成功，输出相同 | 真实 SDK/WBI 请求签名、3 分 P、画质选择、资源组装 |
| YouTube | [`eeb78d1`](https://github.com/monkeyWie/gopeed-extension-youtube/tree/eeb78d168e053e9a2aad1dbaa309fc11f6992f7c) | 3,409,614 B | 两个引擎均成功，输出相同 | 完整 YouTube.js bundle 初始化、公开视频 metadata/title、下载任务描述 |
| Twitter | [`a963428`](https://github.com/monkeyWie/gopeed-extension-twitter/tree/a963428bbfdbee9510bf224b00b384e37514ce77) | 66,243 B | 两个引擎均成功，输出相同 | guest token/GraphQL SDK 路径、图片和最高码率视频筛选 |

联网输入：B 站 `BV1BJ4m1e7g8?p=1-3`、YouTube `dQw4w9WgXcQ`、Twitter `1734145541599383880`。未使用账户 Cookie。B 站实际返回了 480p fallback，两个引擎一致。联网不是固定延迟源，不用单次耗时推导速度倍数。

`onStart/onError` 在同一个实验 runtime 的模拟 task/blob/webview 宿主下跑通；B 站 CDN 切换和 YouTube 重试注册能返回相同结果。**这不包含真正调用 Blob opener 或读取/合并媒体。** WebView 只模拟 `navigator.userAgent`，FFmpeg mock 遇到真实 merge 会报错，不会伪造下载成功。

另有两个直接宿主检查：moejs 的显式 Go request 拷入、resource 拷出经过 `base.Resource.Validate()`；本地 HTTP server 的异步 JSON 和取消请求在两侧均可完成。取消后 HTTP server 的请求 context 均结束。但原生产 goja fetch 的取消 catch 路径因 `DOMException` 未定义而返回 `ReferenceError`，moejs 返回传入的取消原因；这个已有宿主差异被单独记录，不能视为错误语义已完全兼容，详见 [http.txt](results/http.txt)。

原始证据：[compatibility.txt](results/compatibility.txt)、[live.txt](results/live.txt)、[sources.json](results/sources.json)。

## 性能实测

### 热解析与冷解析

| 扩展 | 热 goja / moejs (ms) | 热速度倍数 | 冷 goja / moejs (ms) | 冷耗时下降 |
| --- | ---: | ---: | ---: | ---: |
| bilibili | 21.188 / 3.389 | 6.25× | 118.861 / 104.827 | 11.8% |
| youtube | 3.193 / 0.277 | 11.54× | 283.412 / 239.646 | 15.4% |
| twitter | 3.575 / 0.505 | 7.08× | 17.239 / 11.849 | 31.3% |

### 相同 JS buffered Response 的对照

| 扩展 | goja (ms) | moejs (ms) | 速度倍数 | 每次分配 goja / moejs (KiB) |
| --- | ---: | ---: | ---: | ---: |
| bilibili | 22.076 | 3.528 | 6.26× | 8036.0 / 1267.2 |
| youtube | 2.895 | 0.290 | 10.00× | 1071.1 / 151.3 |
| twitter | 3.450 | 0.491 | 7.03× | 1527.4 / 238.3 |

更换为相同的 JS Response 后仍有 6–10 倍差距，说明收益并不只来自 moejs 的原生 Response/Stream。但 URL/Headers 等底层实现仍不同，因此它仍是完整运行时对照，不能宣称是严格隔离的解释器指令效率。

### 空 runtime 加一次 `1+1` 的对照

| 配置 | 耗时 (µs) | 每次分配 (KiB) |
| --- | ---: | ---: |
| goja | 2.85 | 5.1 |
| moejs-mutable | 48.42 | 103.6 |
| moejs-shared | 3.86 | 5.2 |

moejs 的 mutable runtime 创建在本机反而明显更贵。shared frozen 模式能降低这部分成本，但改变 builtin 的可变性；本报告没有把这项不同合同的优化算入默认迁移收益。上述项也包含编译小脚本，并非只测构造函数。


热运行的明显收益不能直接解释成“下载快了数倍”。本次固定 JSON 较小，没有大 playlist、真实 YouTube player JS、PoToken、SABR、账号登录或 FFmpeg 媒体吞吐的测量。`B/op` 是每次调用累计分配量，**不是常驻内存、峰值 RSS 或每个 runtime 的存活内存**。

当前 [`doTrigger`](../../pkg/download/extension.go) 在匹配每个事件后调用 `newExtensionEngine` 并执行 bundle；[`engineSession`](../../pkg/download/extension_engine_session.go) 为 Blob/WebView 保活，未提供已编译 bundle 的通用缓存。因此首个事件更接近 cold 项。即使热运行快很多，也不能套用同样倍数预测首次解析。未来可以单独比较“两引擎均缓存编译产物”，避免把缓存改进归因于换引擎。

## 构建体积

`CGO_ENABLED=0`、`-trimpath -pgo=off -ldflags='-s -w'`。gzip 使用 Python 默认 level 9、`mtime=0`。gzip 只是相同压缩算法的代理，不是 Flutter installer、APK、IPA 或最终 zip 的测量。

### 独立解释器程序

| 目标 | goja (MiB) | moejs (MiB) | 未压缩减少 (MiB) | gzip 减少 (KiB) |
| --- | ---: | ---: | ---: | ---: |
| darwin/arm64 | 8.86 | 7.74 | 1.12 | 36.0 |
| linux/amd64 | 9.14 | 8.06 | 1.07 | 13.0 |
| windows/amd64 | 9.36 | 8.30 | 1.07 | 9.9 |
| android/arm64 | 9.56 | 8.44 | 1.12 | 37.3 |

### 真实 Gopeed CLI（`nosqlite`）的双引擎过渡成本

| 目标 | 原生产 CLI (MiB) | 加入 moejs/webapi (MiB) | 未压缩增加 (MiB) | gzip 增加 (MiB) |
| --- | ---: | ---: | ---: | ---: |
| darwin/arm64 | 49.63 | 56.06 | 6.43 | 2.50 |
| linux/amd64 | 51.25 | 57.92 | 6.68 | 2.69 |
| windows/amd64 | 52.42 | 59.23 | 6.81 | 2.73 |


独立解释器程序支持从命令行传入任意 JS，避免 linker 把执行功能全部裁掉；它不含完整宿主能力。真实 CLI 的第二项通过构建 overlay 额外链接 moejs、eventloop、webapi，**仍然保留生产 goja，默认扩展执行仍走 goja**。这测的是双引擎过渡阶段成本，没有伪装成完整替换的成品。

结论：独立引擎未压缩可以少约 1.1 MiB，但压缩后只省十几到几十 KiB；没有证据表明换引擎能让最终安装包大幅变小。双引擎灰度还会增大包。完全移除 goja 后的 CLI、桌面 c-shared、Android/iOS 库和 Flutter 安装包必须等完整桥接完成后再测。本次 Android 只交叉编译独立解释器，不代表 APK/FFI 已验证。

原始证据：[size.json](results/size.json)、[size-build.txt](results/size-build.txt)，构建脚本 [measure_size.py](measure_size.py)。

## 迁移阻塞与工作量

| 接入点 | 实测或代码事实 | 正式替换需要完成 |
| --- | --- | --- |
| Go struct / map 语义 | goja 对 `ctx.res`、`ctx.req.url` 的修改写回 Go；moejs `FromGo` 是 JS 快照，Go 值不变；Go 方法也不会自动映射 | 显式对象/方法包装及写回策略，覆盖 req/task/meta/extra/settings/storage，验证引用与持久化语义 |
| 数字 labels | B 站给 labels 写数字；goja 自动转成 `map[string]string`，直接 JSON 拷出会报类型错 | 原型已做 string 规范化；生产要定义完整转换和错误策略 |
| 事件循环和取消 | `goja_nodejs` 与现有 `Post`、Promise/事件超时直接绑定；moejs 有不同的 loop/API | 对齐并发归属、超时、Close、pending Promise、回调错误、后台任务生命周期，迁移现有 cancellation 测试 |
| fetch/XHR/streams | XHR JS 可复用；本次 fetch 使用 moejs webapi；本地取消请求已验证 | 保留代理、默认 UA、headers、Cookie、stream backpressure、body ownership 和 session cleanup；跑现有 Fetch WPT/流测试于新宿主 |
| Blob/FFmpeg/WebView | 生产注入层大量引用 goja `Value`、`Object`、`FunctionCall` | 移植 Blob opener/reader、FFmpeg WASM、WebView 回调与函数源码序列化，验证取消/恢复/重试 |
| File/VM/MessageError/logging | 现有 Go 方法/错误和值导出不是 moejs 的直接替代 API | 建原生函数包装，保留 `MessageError` 分类、日志格式、File 行为与子 VM 隔离 |
| 可写 builtins | 三个回放也能在 frozen 模式运行，但修改 `Array.prototype` 会失败 | 默认保持 mutable；除非扩展能力合同允许冻结，不能把共享 builtin 的启动/内存优势直接作为默认收益 |

上游 [Go values in](https://github.com/MoeForge/moejs/blob/b2c7fed9c42dd1ba8ce5ea844850cba6a1a95fb3/docs/guide.md#go-values-in) 说明快照/方法转换规则；[runtime.go](https://github.com/MoeForge/moejs/blob/b2c7fed9c42dd1ba8ce5ea844850cba6a1a95fb3/runtime.go) 说明共享冻结 builtin 和 goroutine 约束；[README](https://github.com/MoeForge/moejs/blob/b2c7fed9c42dd1ba8ce5ea844850cba6a1a95fb3/README.md#status) 仍将项目标为 alpha、API 可能变化。要固定提交，不能把上游语言测试通过数当成 Gopeed 宿主兼容证明。分发需要随带上游 [THIRD_PARTY_NOTICES](https://github.com/MoeForge/moejs/blob/b2c7fed9c42dd1ba8ce5ea844850cba6a1a95fb3/THIRD_PARTY_NOTICES)。

## 后续替换门槛

1. 在实验构建中跑完整 Downloader → extension → Go task/resource → Blob/FFmpeg/WebView 路径，实测 B 站与 YouTube 下载输出文件、取消、断点恢复、重试；不再依赖本次模拟宿主。
2. 新引擎跑现有扩展、Fetch/Stream、WebView、FFmpeg、事件取消测试；验证无 pending promise、session 泄漏及并发访问错误。
3. 在实际每事件新建 runtime 的方式下比较总耗时、CPU、峰值/常驻内存和最终安装包；增加至少一台 Windows、一台 Android/iOS 目标以及真实大型 player/playlist。
4. 再决定冷路径与实际解析体验的提升是否足以抵消桥接成本。热 CPU 已有继续投入的依据，**最终包体和完整下载性能目前没有“大幅提升”的证据**。

本 PR 仅交付可复现的调研/原型，不切换默认引擎。生产 engine 原有测试通过，记录在 [production-engine-tests.txt](results/production-engine-tests.txt)。复现命令见 [README.md](README.md)。
