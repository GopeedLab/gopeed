# CPU 与内存补充测量

日期：2026-11-14。沿用 [主报告](REPORT.md) 的引擎版本、三个固定扩展包、回放输入和 M1 / Go 1.26.0 环境。

**建议维持 goja，暂不投入完整替换。** 在当前每次事件新建引擎、重新编译加载的方式下，B 站/YouTube 的 CPU 只减少约 12% / 15%，没有一致的内存优势；连续解析时峰值 RSS 更高。Twitter 的 CPU 降幅较大，但每次只省约 5 ms。这些收益不足以抵消完整宿主桥接、兼容性和维护成本。

## 测量方式

- 每个样本使用新的独立 OS 进程，两个引擎交替先后顺序，5 轮重复，报告中位数；最终数据共 120 个进程样本。测量时未与本次其他构建或测试并发。
- `GOMAXPROCS=1 GOGC=100 GOMEMLIMIT=off`，不使用 PGO。CPU 是 `getrusage(RUSAGE_SELF)` 的 **user + system 时间**，除以完成次数；包含 GC 和编译开销，不使用瞬时 CPU 占用百分比，也不把墙钟耗时直接当 CPU 时间。
- `cold`：一次进程内先预热一次全局初始化，再连续完成 20 次“新建环境、克隆 bundle 输入字符串、编译加载、解析、关闭”。接近当前反复触发扩展事件的方式。每次使用独立 bundle 字符串，模拟生产文件读取后的输入所有权。
- `warm`：环境已加载，连续解析 200 次。仅用于确认复用环境时的 CPU 上限；当前生产事件路径没有这样复用。
- 内存组不预建引擎，从相同测量程序基线开始，分别建立并保留 1 / 8 个完整环境，每个环境执行一次解析后保持存活。8 个环境是同时保留，不是同时并行执行；不模拟实际 Blob/FFmpeg 媒体数据。
- 存活堆：`runtime.MemStats.HeapAlloc` 相对建引擎前的增量。连续 GC 并给异步清理调度机会，至少 3 轮、最多 8 轮，直到相邻值差异小于 64 KiB；不稳定则测量失败。保留源字符串并用 `runtime.KeepAlive` 保证环境仍存活，避免编译器提前释放。
- RSS：通过 `ps` 记录当前物理驻留快照，通过 `getrusage` 记录进程历史峰值；macOS 的峰值单位为字节，Linux 转换为字节。`ps`、稳定 GC 和内存快照都在计时区间外，`RUSAGE_SELF` 不包括 sampler 子进程 CPU。
- 两个引擎链接在相同测量 worker 中，RSS 含公共 Go/测试框架/代码页和 allocator 保留页，**不是完整 Gopeed 应用或最终单引擎产品的内存预测**。存活堆增量用来区分扩展保留对象和进程历史峰值。
- 全部输出按 SHA-256 比较一致，没有通过减少文件或请求处理量获得更好的数字。宿主 mock、HTTP fixture、未移植能力及真实下载尚未验证等限制与主报告相同。

## 冷路径 CPU 与连续解析峰值

| 扩展 | goja CPU (ms/次) | moejs CPU (ms/次) | CPU 减少 | goja 峰值 RSS (MiB) | moejs 峰值 RSS (MiB) |
| --- | ---: | ---: | ---: | ---: | ---: |
| bilibili | 118.962 | 104.081 | 12.5% | 154.7 | 172.3 |
| youtube | 278.476 | 237.330 | 14.8% | 283.8 | 358.9 |
| twitter | 15.555 | 10.594 | 31.9% | 79.7 | 96.8 |


峰值是连续 20 次冷解析期间**整个 worker 的进程峰值**，不是单个 runtime 的堆大小。峰值会受 Go GC/allocator 与异步清理时机影响；不能将它当作关闭后仍存活的对象。原始数据保留每轮值，汇总文件也提供 min/max。

| 扩展 | goja 累计分配 (MiB/次) | moejs 累计分配 (MiB/次) | 分配减少 |
| --- | ---: | ---: | ---: |
| bilibili | 73.69 | 60.17 | 18.3% |
| youtube | 159.91 | 146.82 | 8.2% |
| twitter | 10.96 | 7.73 | 29.5% |


moejs 的累计分配量确实较少，但这没有转化为更低的连续解析峰值 RSS。`B/op`、GC 后存活堆、RSS 是不同指标，不能互相替代。

## 保留环境时的内存

| 扩展 | 保留环境数 | goja 存活堆增量 (MiB) | moejs 存活堆增量 (MiB) | goja RSS 快照 (MiB) | moejs RSS 快照 (MiB) |
| --- | ---: | ---: | ---: | ---: | ---: |
| bilibili | 1 | 15.87 | 15.66 | 114.2 | 111.2 |
| bilibili | 8 | 126.72 | 120.60 | 362.0 | 302.6 |
| youtube | 1 | 32.48 | 41.22 | 203.7 | 192.2 |
| youtube | 8 | 259.64 | 320.54 | 627.8 | 680.1 |
| twitter | 1 | 2.53 | 2.63 | 75.6 | 74.7 |
| twitter | 8 | 20.06 | 19.48 | 106.9 | 93.5 |


B 站一个环境的存活堆几乎没有变化；YouTube 反而更高。多环境情况下也没有稳定、普遍的显著节省。RSS 快照包含 allocator 页面，因此与存活堆的变化比例不同。

最初单次 GC 的 YouTube 堆快照约为 32.5 → 42.2 MiB。考虑 moejs 异步清理编译缓存后，已重测驻留组，表中使用稳定 GC 的结果，不能将初始 42.2 MiB 当成最终驻留值。CPU 和冷路径峰值取自第一次进程资源测量；这些计时和峰值发生在强制 GC 之前，不受驻留测量修正影响。

关闭全部环境并等稳定 GC 后，额外存活 Go 堆回落到 goja 约 0.03–0.05 MiB、moejs 约 0.18–0.97 MiB。这包含一次性进程缓存；本实验没有据此认定泄漏。RSS 不一定同步回落，不能只看关闭后的 RSS 判断对象是否已释放。

## 热路径 CPU

| 扩展 | goja CPU (ms/次) | moejs CPU (ms/次) | CPU 减少 |
| --- | ---: | ---: | ---: |
| bilibili | 19.688 | 3.173 | 83.9% |
| youtube | 2.905 | 0.215 | 92.6% |
| twitter | 3.265 | 0.453 | 86.1% |


热路径的大幅 CPU 节省再次成立。但它要求持有并复用已加载的环境；对当前每个事件重建环境的实现，应该采用上面的 cold 结果评估迁移。

## 决策

按当前扩展和架构，CPU/内存补测没有提供足够的替换依据。尤其 B 站和 YouTube 的 CPU 节省有限，内存没有一致改善，加上主报告里压缩体积几乎不变，应先维持 goja，把本 PR 留作调研记录。

未来若出现长时间执行计算、反复复用环境的真实扩展，再重新评估；若独立研究编译缓存或环境复用，必须同时在 goja 和 moejs 上比较收益，不能把架构优化全部归给引擎替换。

## 复现和证据

```sh
cd experiments/moejs
python3 fetch_sources.py
python3 measure_resources.py > results/resources-run.txt
# 只重新汇总已保存的原始样本，不启动新测量
python3 measure_resources.py --summarize-only
# 必要时只重测驻留组，保留之前的 CPU 样本
python3 measure_resources.py --memory-only > results/resources-memory-run.txt
```

原始样本：[resources.json](results/resources.json)；汇总：[resources-summary.json](results/resources-summary.json)。首次资源运行日志：[resources-run.txt](results/resources-run.txt)；稳定 GC 后重测驻留组日志：[resources-memory-run.txt](results/resources-memory-run.txt)。首次日志中的驻留组已被后者替代。当前 worker：[resource_worker_test.go](resource_worker_test.go)；驱动：[measure_resources.py](measure_resources.py)；功能检查：[resources-tests.txt](results/resources-tests.txt)。CPU/warm 组中单次 GC 的快照保留作原始记录，内存结论只使用重新测得的 `retained` 组。
