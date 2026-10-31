# 自主决策记录（ADR）

**最后更新**：2026-10-31

## 概述

记录开发过程中的重要决策。每条 ADR 捕捉背景、决策与影响。

## ADR 索引

| ID | 标题 | Stage | 日期 | 状态 |
|----|------|-------|------|------|
| 1 | HTTP 框架选用 Gin | 0 | 2026-10-31 | 已采纳 |
| 2 | 迁移使用 golang-migrate | 0 | 2026-10-31 | 已采纳 |
| 3 | 配置 YAML + env 覆盖 | 0 | 2026-10-31 | 已采纳 |
| 4 | 前端 html/template 服务端渲染 | 0 | 2026-10-31 | 已采纳 |
| 5 | 任务入口收敛到 Taskfile | 0 | 2026-10-31 | 已采纳 |
| 6 | IP 报告页走 CDN 缓存，TTL 分级 | 0 | 2026-10-31 | 已采纳 |
| 7 | 单机部署边界 + 状态抽象 | 0 | 2026-10-31 | 已采纳 |
| 8 | Go 版本定为 1.26.0 | 0 | 2026-10-31 | 已采纳 |
| 9 | 迁移嵌入二进制，不依赖外部 CLI | 0 | 2026-10-31 | 已采纳 |
| 10 | 静态资源缓存判定抽为纯函数 | 0 | 2026-10-31 | 已采纳 |

---

## 决策模板

```markdown
### Decision N: [标题]

**背景**：什么情况促成了这个决策

**决策**：选择了哪条路

**理由**：为什么选这条

**对后续 Stage 的影响**：（不可为空）
- [对未来的具体影响]
- [启用了什么 / 限制了什么]

**日期**：YYYY-MM-DD

**相关 Stage**：Stage N

**考虑过的替代方案**：
1. [替代方案 1]：[为什么不选]
2. [替代方案 2]：[为什么不选]
```

---

### Decision 1: HTTP 框架选用 Gin

**背景**：初稿决定用标准库 `net/http`，理由是路由只有三条。评审时判断：随着数据源 adapter 增多，中间件（超时、限流、日志、CORS、错误恢复）会逐个手写。

**决策**：用 `github.com/gin-gonic/gin`。

**理由**：gin 的路由分组、参数绑定、中间件链能省掉大量样板；这类样板写错往往比框架 bug 更难发现。

**对后续 Stage 的影响**：
- Stage 2 的 adapter 触发错误时，统一走 gin 的 `c.Error` 与恢复中间件
- Stage 4 的 handler 用 `c.HTML` 渲染模板，不自行拼 `http.ResponseWriter`
- Stage 5 的限流中间件直接接 gin 的 `Use()`

**日期**：2026-10-31

**相关 Stage**：Stage 0

**考虑过的替代方案**：
1. 标准库 `net/http`：路由增强版够用，但中间件要自建，长期维护成本高于引入 gin
2. Echo / Chi：能力接近，gin 生态与示例最多

---

### Decision 2: 迁移使用 golang-migrate，版本化 up/down

**背景**：初稿是手写 SQL + embed，无版本管理。

**决策**：`golang-migrate/migrate`，文件按 `NNNN_name.up.sql` / `.down.sql` 组织。

**理由**：数据库变更必须可回滚、可追溯。源会持续增加，`observations` 表结构大概率会演进，没有 up/down 的回滚手段等于没有安全网。

**对后续 Stage 的影响**：
- Stage 3 增加缓存相关表时必须带 down 脚本
- Stage 5 的部署流程里 `migrate up` 是启动前置步骤
- 回滚策略从"删目录"变为"`task migrate:down`"

**日期**：2026-10-31

**相关 Stage**：Stage 0

**考虑过的替代方案**：
1. 手写 SQL + embed：无法回滚、无版本记录
2. Atlas / goose：能力接近，golang-migrate 的 CLI 在容器里更好用

---

### Decision 3: 配置走 YAML 文件 + 环境变量覆盖

**背景**：初稿是纯环境变量。

**决策**：`configs/config.yaml` 为主，`IPSPOT_<SECTION>_<KEY>` 环境变量覆盖。

**理由**：Docker / k8s 里 ConfigMap 挂文件、Secret 走环境变量，两种都要顺手。纯环境变量在配置项变多后难以阅读和版本管理。

**对后续 Stage 的影响**：
- Stage 2 新增源配置时统一加在 `sources:` 段
- Stage 5 的容器化直接用同一份 YAML，不需要维护第二套配置

**日期**：2026-10-31

**相关 Stage**：Stage 0

**考虑过的替代方案**：
1. 纯环境变量：容器友好但可读性差、无默认值集中管理
2. viper：会带进十余个间接依赖，配置项不到 15 个不值得

---

### Decision 4: 前端用 html/template 服务端渲染，不引入 React/Next

**背景**：需要判断是否引入前端框架。

**决策**：`html/template` 服务端渲染。如需交互，顺序是「内联 JSON + uPlot」→「htmx」→「/app 子站独立前端」。**明确否决 Astro 与岛屿架构。**

**理由**：三条硬约束——数据必须在首屏 HTML 里（SEO）、Go 单二进制不应引入第二个运行时、这个站没有客户端状态（一个输入框 + 一次提交 + 一段只读报告）。

**对后续 Stage 的影响**：
- Stage 4 的交互只能是表单提交；图表用内联 JSON + uPlot，不引入框架或构建链
- Stage 6 的静态页生成可直接复用模板，无需另建渲染管线
- 若将来有登录后页面，可单独在 `/app` 引入，不影响主站

**日期**：2026-10-31

**相关 Stage**：Stage 0 / 4

**考虑过的替代方案**：
1. Next.js（SSR）：能解决 SEO，但需要 Node 常驻进程，两个容器两套依赖，个人项目负担过重
2. Astro 岛屿架构：**已否决**。需要 Node 构建链（回到两个运行时）、静态页与报告页分属两套模板语法、报告数据被推回客户端拉取，违背「数据必须在首屏 HTML」的核心约束。此项目是「工具站 + 零交互」，不是「内容站 + 少量交互」，场景不匹配

---

### Decision 5: 任务入口统一收敛到 Taskfile

**背景**：build / test / migrate / run 分散在各处命令，易漂移。

**决策**：全部收敛到 `Taskfile.yml`，`task migrate:*` 与程序共用同一份配置读取 DSN。

**理由**：统一入口避免"本地能跑、容器里跑不通"；DSN 单点来源避免迁移和程序连到不同库。

**对后续 Stage 的影响**：
- Stage 5 的部署脚本走 `task`，不另起 Makefile 或 shell 脚本
- CI 里直接调 `task`，与本地完全一致

**日期**：2026-10-31

**相关 Stage**：Stage 0 / 5

**考虑过的替代方案**：
1. Makefile：跨平台体验差，Windows 需额外层
2. 纯 shell 脚本：无依赖声明、无并行、无变量作用域

---

### Decision 6: IP 报告页走 CDN 缓存，TTL 分级

**背景**：初稿写的是 HTML 全部 `no-cache`，理由是"数据时效性强"。但项目里还定过另一条：免费源额度是硬上限，24h 缓存命中率 > 60% 才能撑住规模。两条自相矛盾。

**决策**：HTML 也缓存，按页型分级 TTL：

| 页面 | 边缘 TTL | 浏览器 TTL |
|---|---|---|
| `/ip/<addr>` | 1 小时 + `stale-while-revalidate=86400` | 60 秒 |
| `/asn/<asn>` | 1 天 + `stale-while-revalidate=604800` | 10 分钟 |
| `/` | 1 小时 | 5 分钟 |

**理由**：底层观测数据本身就是 24 小时才刷新一次，缓存 1 小时不会带来任何可见的新鲜度损失。而边缘命中意味着 origin 不碰、源 API 不花，这是额度策略的地基。`stale-while-revalidate` 保证用户永远不等刷新。

**对后续 Stage 的影响**：
- Stage 3 的额度测算必须以"边缘命中率"为前提，而不是只看 origin 缓存
- Stage 4 的报告页**不允许出现任何个性化内容**（不读 Cookie、不按 UA 分流），否则不可缓存
- Stage 4 的"被查询 N 次"字段不能上缓存页，否则严重少计——移到 `/api/*` 或不缓存的页面
- Stage 3 必须在 origin 侧对同一 IP 做 singleflight，防止缓存击穿时并发打爆额度

**日期**：2026-10-31

**相关 Stage**：Stage 0 / 3 / 4

**考虑过的替代方案**：
1. 全站 `no-cache`：额度必然撑不住，与 H3 假设冲突
2. 长 TTL（24h）：与数据的更新周期一致，但改版或数据修正后用户会看到明显过期内容，1 小时是更稳的折中

---

### Decision 7: 明确单机部署边界，但把状态抽象为接口

**背景**：只考虑单机是否足够，是否需要现在就为多副本留出设计。

**决策**：架构目标是单实例，但三样进程内状态必须走接口抽象、配额计数必须持久化。同时把三条「迁移信号」写进验收。

**理由**：多副本现在做不起（成本、SQLite 单写者、需求未验证），但单机状态有一个危险特性——**多副本时不会报错，只会静默变差**。配额守卫各算各的会超限，缓存各存各的会让命中率除以 N。这类问题上线后很难归因，所以宁可现在花 0 成本定规矩。

**具体三条**：
1. `cache` / `quota` 走接口，单机只是一种实现
2. 配额计数落 `quota_usage` 表，不放内存（0.1ms 的代价换「重启不清零 + 天然兼容多副本」）
3. singleflight 记 `shared` 日志，作为迁移信号

**对后续 Stage 的影响**：
- Stage 3 的实现必须按接口写，不能直接 `sync.Map`
- Stage 3 的 `/metrics` 必须暴露 shared 比例，作为多副本迁移的观测指标
- Stage 5 若真要上多副本，替换的只是接口实现，不动调用方
- Stage 6 的历史对比不依赖缓存，不受影响

**日期**：2026-10-31

**相关 Stage**：Stage 0 / 3 / 5

**考虑过的替代方案**：
1. 直接做多副本：SQLite 需换 Postgres、缓存需换 Redis、singleflight 需换分布式锁，成本与收益严重不匹配
2. 完全不管单机限制：省事，但多副本时会静默失效且难以归因

---

### Decision 8: Go 版本定为 1.26.0

**背景**：设计稿写的是 Go 1.25。实际拉依赖时发现 `golang.org/x/sys@v0.49.0`、`modernc.org/libc`、`modernc.org/sqlite` 最新版均要求 `go >= 1.26.0`；尝试回退到兼容 1.25 的版本会把 modernc 降级到 v1.38.0，sqlite 驱动随之下探，风险高于收益。

**决策**：`go.mod` 写 `go 1.26.0` 并加 `toolchain go1.26.0`；`Taskfile.yml` 全局设 `GOTOOLCHAIN=go1.26.0`。

**理由**：本机 gvm 装的是 1.25.5，Go 的 auto toolchain 会按 `go.mod` 自动下载并使用 1.26.0。但 `go test -cover` 与 `go tool cover` 在混用两套编译器时会报 `compile: version "go1.26.0" does not match go tool version "go1.25.5"`。全局固定 toolchain 环境变量是唯一可靠解法。

**对后续 Stage 的影响**：
- 任何新增任务都必须能在 `GOTOOLCHAIN=go1.26.0` 下运行
- CI（Stage 5）需显式安装 1.26.0 或依赖 toolchain 自动下载
- 升级 Go 时需要重新验证整个依赖链的版本约束

**日期**：2026-10-31

**相关 Stage**：Stage 0

**考虑过的替代方案**：
1. 回退到兼容 Go 1.25 的依赖组合：需要把 `modernc.org/sqlite` 降到 v1.38.0，驱动能力与稳定性都下降
2. 不固定 `toolchain`，依赖 auto 行为：`-cover` 路径会失败，属于难排查的环境问题

---

### Decision 9: 迁移嵌入二进制，不依赖外部 CLI

**背景**：设计稿让 Taskfile 直接调 `migrate` CLI。实际环境中 CLI 未安装，Docker 镜像还要额外装。

**决策**：`migrate` 作为库嵌入，二进制暴露 `ipspot migrate up|down|version` 子命令。

**理由**：单二进制是部署目标的核心约束。迁移和程序读同一份配置取 DSN，不会出现「迁移连 A 库、程序连 B 库」。

**关键实现约束**：`migrate/database/sqlite` 的 `Close()` 会**关闭传入的 `*sql.DB`**。若与业务连接池共用，迁移结束后整个进程的数据库访问都会失效（报 `sql: database is closed`）。因此迁移必须用独立连接，见 `withMigrator`。

**对后续 Stage 的影响**：
- Stage 5 的 Dockerfile 不需要装 migrate 工具
- 任何在启动时自动迁移的实现，都必须在独立连接上做，用完即关
- 迁移文件的 embed 位置固定在 `internal/store/migrations/`，新增迁移必须同步到 `migrations/`（`task migrate:create` 已处理）

**日期**：2026-10-31

**相关 Stage**：Stage 0 / 5

**考虑过的替代方案**：
1. 外部 `migrate` CLI：多一个部署依赖，且本机没有
2. 手写迁移执行器：重复造轮子，且容易漏掉 dirty 状态处理

---

### Decision 10: 静态资源缓存判定抽为纯函数

**背景**：设计稿要求「带内容哈希的资源走 immutable」。集成测试里用 `httptest` 请求不存在的带 hash 文件时，`http.FileServer` 返回 404 并**清空 `Cache-Control`**，导致无法断言。

**决策**：把判定逻辑抽为 `cacheControlFor(path string) string` 纯函数，直接单测；集成测试只断言存在的资源。

**理由**：行为正确的证明不一定要经过完整 HTTP 栈。纯函数测试更稳定，也更直接表达意图。

**对后续 Stage 的影响**：
- Stage 4 引入带哈希的资产后，可以直接复用该函数
- 若将来加 CDN 特定头（如 `CDN-Cache-Control`），在此函数内扩展即可

**日期**：2026-10-31

**相关 Stage**：Stage 0 / 4

**考虑过的替代方案**：
1. 搭一个带 hash 文件的测试 fixture：可行但增加测试复杂度，且仍受 FileServer 404 行为影响
