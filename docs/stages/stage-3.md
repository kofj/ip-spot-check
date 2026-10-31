# Stage 3：查询编排与缓存

## 状态
[ ] 规划中 / [ ] 进行中 / [ ] 完成

## 目标

并发编排 + 三层缓存，把外部源额度消耗压到免费层能承受的范围。

## 非目标

- 不上任务队列（见已否决方案）
- 不做跨进程分布式缓存（单机 SQLite + 内存足够）
- 不做多副本部署（见 Stage 0「单机部署边界」）
- 不做缓存一致性协议（数据本身就是最终一致）

## 核心假设

- **H3**：24 小时 origin 缓存 + 1 小时边缘缓存的组合，能把源 API 消耗压到日额度以内，且在真实流量下可观测

## 三层缓存

```
边缘 CDN（1h）
    ↓ miss
origin 内存缓存（24h）
    ↓ miss
observations 表（历史，不消耗额度）
    ↓ 无记录或已过期
外部源 API（有额度，最贵）
```

- 边缘层：由 `Cache-Control` 控制，见 ADR-6
- origin 层：进程内 LRU 或 `sync.Map` + TTL，Stage 3 用前者
- 持久层：无论是否命中，都写 `observations`，这是历史数据的来源
- 外部源：**只有前三层全部未命中才调用**

## 必须实现的三个机制

### 1. Singleflight 防击穿

同一 IP 的并发请求合并成一次外部调用。没有这个，一个被论坛贴出来的 IP 能瞬间烧掉日额度。

```go
result, err, shared := group.Do(ip.String(), func() (any, error) {
    return fetchAll(ctx, ip)
})
```

`shared` 为 true 时说明本次复用了别人的结果，这一点要记进日志——它是判断击穿频率的唯一依据。

### 2. 每源配额守卫

每个源有自己的日额度（AbuseIPDB 1000/天）。在调用前检查计数器，超限时该源直接返回 `Observation.Err = ErrQuotaExceeded`，**页面如实显示"额度耗尽"而不是静默降级**。

**计数必须持久化到数据库，不放内存。** 代价是每次查询多一次 SQLite 读写（~0.1ms），收益有三：重启不清零、多副本时天然共享、能按天做历史统计。

```go
type Guard interface {
    Acquire(ctx context.Context, source string) error  // 超限返回 ErrQuotaExceeded
    Usage(ctx context.Context, source string) (used, limit int)
}
```

接口化是为了将来换 Redis 时不动调用方。单机实现直接用 `quota_usage` 表。

### 3. 命中率埋点

这是验证 H3 的唯一方式。必须记录：

| 指标 | 用途 |
|---|---|
| 边缘命中率 | 由 Cloudflare 分析提供，不自己实现 |
| origin 缓存命中率 | 每次查询打点 |
| 源调用次数（按源分） | 决定是否需要付费源 |
| singleflight 合并次数 | 判断击穿频率 |
| **singleflight `shared` 为 true 的比例** | 多副本迁移信号：单机下应 >0 且稳定，多副本后会各进程各打一次 |

暴露在 `/metrics`（Prometheus 文本格式），便于本地也能看。

## 验收标准

- [ ] 首次查询 P95 < 3s（6 源并发）
- [ ] 缓存命中 P95 < 100ms
- [ ] 同一 IP 并发 50 请求，源只被调用 1 次
- [ ] 超配额时返回明确的"额度耗尽"，不返空值
- [ ] `/metrics` 输出四类指标
- [ ] 灰度跑 24 小时，统计出真实 origin 命中率与源消耗
- [ ] 各源消耗在免费额度内

## 单机边界（来自 Stage 0）

当前明确是单实例架构。三层缓存、配额计数、singleflight 都按单进程设计，但**接口必须是抽象层**：

| 状态 | 单机实现 | 多副本时需替换为 |
|---|---|---|
| origin 缓存 | 进程内 LRU | Redis / 边缘 KV |
| 配额计数 | SQLite `quota_usage` 表（已验证多副本兼容） | 同一张表（Postgres） |
| singleflight | `golang.org/x/sync/singleflight` | Redis 分布式锁 |

**迁移信号**：`/metrics` 中出现「同一 IP 在不同进程各触发一次拉取」，或配额使用量与实际源侧计数不符。出现即停并评估。

## 回滚方案

回落 Stage 2 版本（无缓存，直连源）。功能降级但可用，只是额度会被快速消耗。

## 已知局限

- origin 缓存是进程内的，多实例部署时会各存一份且命中率下降
- singleflight 是进程内的，多实例防击穿失效
- 配额计数已持久化，多实例可共享，是三者中唯一天然兼容多副本的部分

## 对后续 Stage 的影响

- 命中率数据决定是否需要提前购买付费源
- `/metrics` 的指标是 Stage 5 告警的依据
- 配额守卫的实现决定了 Stage 5 的运维策略（比如是否需要按源分时段调度）
