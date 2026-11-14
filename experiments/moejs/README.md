# moejs 实验评估

目标按顺序判断：**先能跑通 Gopeed 的真实扩展，再判断收益是否足够大，最后才考虑正式替换。**

这是独立 Go module。生产引擎、根目录依赖和默认构建不变。结论、实测数据及限制见 [REPORT.md](REPORT.md)。

## 复现

从本目录运行，需要 Git、Python 3 和 Go 1.26.0：

```sh
python3 fetch_sources.py
go test -v -timeout 3m
# 可选：向 Bilibili / YouTube / Twitter 等公开接口发请求；不需要账户 Cookie。
GOPEED_MOEJS_LIVE=1 go test -run TestLiveResolve -v -timeout 4m
# 单独执行，期间避免其他构建/测试负载。
GOMAXPROCS=1 go test -pgo=off -run '^$' -bench . -benchmem -benchtime=500ms -count=5 -timeout 8m > results/bench.txt
python3 summarize.py
python3 measure_size.py
```

`fetch_sources.py` 只取固定提交的原始 `dist/index.js`，校验已记录的 SHA-256，不执行 npm 安装或构建。下载文件和二进制不入库。`results/sources.json` 是来源清单。

## 实验范围

- goja 基线调用真实的 `pkg/download/engine.NewEngine`。
- moejs 原型使用其 eventloop、Web API，加载 Gopeed 现有的 polyfill 和 XHR JS；保留 mutable builtins，与 goja 的默认语义接近。
- 三个完整扩展包执行 `onResolve`，两端逐字比较资源输出和请求路径；B 站包含 SDK/WBI 签名逻辑，YouTube 包含 YouTube.js 的元数据解析。
- `replay.js` 用固定的合成 HTTP 响应隔离网络波动；没有把 SDK 函数替换成假函数。YouTube 的 session/config 请求固定返回 503，走库的本地 session fallback，再提供 player JSON。
- Blob 注册、FFmpeg 和 WebView 为模拟宿主。FFmpeg 的 mock 会拒绝真正合并；WebView 只接受 UA 查询。`onStart/onError` 只验证 mock 宿主下的回调流程，**不代表下载或 SABR/PoToken 已验证**。
- `TestMoeResolveWriteback` 演示显式的 Go request 拷入、JS resource 拷出，并验证 Gopeed `Resource.Validate()`。真实任务的可变对象/方法还需专门移植。
- `TestAsyncHTTP` 使用本地 HTTP server，验证异步 JSON 和 AbortController 请求取消。
- `BenchmarkBufferedTransport` 在两个引擎中使用同一段 JS 的 buffered Response，检查速度差是否主要来自原生 Response/Stream 实现。
- `BenchmarkExtension/*/cold` 包含建引擎、polyfill、编译和加载完整 bundle、一次 resolve 及关闭；`warm` 复用已加载的 runtime。冷启动更接近当前 `doTrigger` 路径。
- `measure_size.py` 实测独立解释器程序和真实 CLI 的双引擎共存成本；没有伪造“已经完全替换”的安装包大小。CLI 构建可能临时补充本实验 module 的依赖，最后执行 tidy。

测试只覆盖本报告所列路径。未指定 `GOPEED_MOEJS_LIVE=1` 时，扩展回放不访问外网；固定 bundle 需要事先下载。
