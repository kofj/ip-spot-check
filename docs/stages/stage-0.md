# Stage 0：技术框架定型

## 状态
[x] 规划中 / [x] 进行中 / [x] 完成

## 目标

把实现层面的基础决策一次性定下来，避免 Stage 1 开始后反复返工。**Stage 0 不产出产品功能**，只产出可编译的空壳、可执行的迁移和统一的运行入口。

## 核心假设

- **H0a**：`Source` 接口的设计（原始字段不归一化、错误作为字段）能容纳 Stage 2 要接入的 6 个源，不需要二次改造
- **H0b**：`observations` 表「一行一源一时点」的结构，能同时支撑 Stage 3 的缓存 key 和 Stage 6 的历史对比，不需要加宽表

## 非目标

- 不实现任何数据源 adapter
- 不做报告页面
- 不做 CI/CD 与部署脚本
- 不引入任何"以后可能用得上"的依赖

## 技术选型

| # | 决策项 | 选择 | 理由 |
|---|---|---|---|
| 1 | 语言 | Go **1.26.0**（`go.mod` 与 `toolchain` 均固定） | 并发拉源是本来长处；单二进制；交叉编译简单。依赖链（x/sys、modernc/libc）已要求 1.26，无法回退到 1.25 |
| 2 | HTTP 框架 | **Gin** | 路由分组、参数绑定、中间件生态齐全，裸 `net/http` 手写路由在源多起来后维护成本高 |
| 3 | 数据库 | SQLite，驱动 `modernc.org/sqlite`（纯 Go，无 cgo） | 单机小数据量零运维；纯 Go 版交叉编译无障碍 |
| 4 | 迁移 | **`golang-migrate/migrate` v4.19.0（作为库）**，版本化 `NNNN_name.up.sql` / `.down.sql` | 有版本、有 up/down。**作为库嵌入二进制**，暴露 `ipspot migrate up/down/create` 子命令，不依赖外部 CLI |
| 5 | 配置 | **YAML 配置文件 + 环境变量覆盖** | 配置文件便于 Docker / k8s ConfigMap 挂载；环境变量覆盖便于容器化和密钥注入 |
| 6 | 日志 | `log/slog` | 标准库、结构化、够用 |
| 7 | 前端渲染 | **`html/template` 服务端渲染** | 见下方「为什么不用 React/Next」 |
| 8 | 交互增强 | **内联 JSON + uPlot；htmx 备选** | 不需要第二套运行时或构建链 |
| 9 | 任务入口 | **Taskfile（`task`）** | 统一 build / test / migrate / run 入口，已装 3.54 |
| 10 | 依赖预算 | 直接依赖 <= 8，每新增一个需写 ADR | 每个依赖都是三年后的维护成本 |

### 为什么不用 React / Next

这是本节最需要解释的一条。

**核心原因不是"简单"或"省事"，而是三条硬约束：**

1. **数据必须出现在首屏 HTML 里。** 项目目标里 ASN 页和事件驱动的 IP 页是 SEO 资产，爬虫看到的是 `document.body.innerHTML` 的初始内容。SPA 首屏是空 `<div id="root">`，爬虫拿到的是空气。Next.js 的 SSR 能解决，但代价是第二条。
2. **会引入第二个运行时。** 后端已经确定是 Go 单二进制。Next.js 需要 Node 常驻进程，意味着两个容器、两套依赖树、两条构建流水线、两个故障点。对一个个人 fun 项目，这是纯粹的负担。
3. **这个站没有客户端状态。** 一个输入框 + 一次提交 + 一段只读报告。没有列表异步加载、没有路由跳转、没有乐观更新。React 的价值在于管理复杂状态，这里不存在。

**如果以后确实需要交互**（历史曲线、差异对比、批量查询），按顺序考虑：

```
html/template（现状）
   → + 内联 JSON + 40KB uPlot（画曲线，无框架、无构建链）
   → + htmx（局部刷新，14KB，无需写 JS）
   → + /app 子站单独一套前端（仅在出现真正的客户端状态时）
```

**明确否决 Astro 与岛屿架构。** 它解决的是「内容站 + 少量交互」，而这个项目是「工具站 + 零交互」。引进来会出现：

- Node 构建链，等于又回到两个运行时
- 静态页在 Astro、报告页在 Go 模板，同一域名两套模板语法
- 报告页的数据被推回客户端拉取，违背「数据必须在首屏 HTML」的核心约束

**不会出现的情况**：为了做一个查询页而引入 Next.js 全家桶。这条写进红线。

### 依赖清单（初始）

| 依赖 | 版本 | 用途 |
|---|---|---|
| `github.com/gin-gonic/gin` | latest | HTTP |
| `modernc.org/sqlite` | latest | DB 驱动（纯 Go，无 CGO） |
| `github.com/golang-migrate/migrate/v4` | **固定 v4.19.0** | 迁移。v4.20+ 要求 Go >= 1.25.11，v4.19.0 只要求 1.23 |

**迁移用库不用 CLI。** `migrate` 作为库嵌入，二进制自带子命令，Docker 镜像不需要额外装工具链。驱动只用 `migrate/database/sqlite`（纯 Go），**不要用 `migrate/database/sqlite3`**——后者要求 CGO。
| `gopkg.in/yaml.v3` | latest | 配置解析 |
| `github.com/oschwald/maxminddb-golang` | latest | GeoLite2（Stage 2 加） |
| `golang.org/x/net` | latest | DNS 查询（按需） |

配置加载自己写（YAML + `os.Getenv` 覆盖），**不引入 viper**。viper 会带进十几个间接依赖，而我们的配置项不到 15 个。

## 仓库结构

```
.
├── cmd/
│   └── ipspot/main.go          # 唯一服务入口
├── internal/
│   ├── config/                 # YAML + env 覆盖
│   ├── model/                  # 数据结构，无依赖
│   ├── source/                 # 数据源 adapter，互不引用
│   ├── query/                  # 编排：并发拉源、合并、判分歧
│   ├── store/                  # DB 读写
│   ├── cache/                  # 缓存接口（Stage 3 实现）
│   └── httpapi/                # Gin 路由、handler、模板
├── migrations/
│   ├── 0001_init.up.sql
│   └── 0001_init.down.sql
├── web/
│   ├── templates/              # html/template
│   └── static/                 # 静态资源，带内容哈希
├── configs/
│   └── config.example.yaml
├── Taskfile.yml
└── go.mod
```

**边界规则**：
- `internal/source/*` 之间互不引用
- `query` 依赖 `source` 与 `store`，不反向
- `httpapi` 是唯一知道 HTTP 的包
- `model` 不依赖任何包

## 配置

`configs/config.example.yaml`：

```yaml
server:
  addr: ":8080"
  read_timeout: 10s
  write_timeout: 15s

database:
  driver: sqlite
  dsn: "file:./data/ipspot.db?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
  max_open_conns: 4

log:
  level: info
  format: json

sources:
  maxmind:
    db_path: "./data/GeoLite2-City.mmdb"
    asn_db_path: "./data/GeoLite2-ASN.mmdb"
  abuseipdb:
    api_key: ""            # 建议用环境变量 IPSPOT_SOURCES_ABUSEIPDB_API_KEY 覆盖
    daily_limit: 1000
  rdap:
    base_url: "https://rdap.org"
    timeout: 5s

cache:
  backend: memory          # memory / redis
  ttl: 24h

policy:
  show_total_score: false  # 红线，永远 false，不给配置项也行
```

**覆盖规则**：`IPSPOT_<SECTION>_<KEY>`，例如 `IPSPOT_SOURCES_ABUSEIPDB_API_KEY`、`IPSPOT_DATABASE_DSN`。这样 k8s 里 ConfigMap 挂文件、Secret 走环境变量，两种都顺手。

## 迁移

文件命名 `NNNN_name.up.sql` / `NNNN_name.down.sql`，版本四位数补零。

`0001_init.up.sql`：

```sql
CREATE TABLE observations (
    id          INTEGER PRIMARY KEY,
    ip          TEXT NOT NULL,
    source      TEXT NOT NULL,
    fields      TEXT NOT NULL,
    fetched_at  INTEGER NOT NULL,
    err         TEXT
);
CREATE INDEX idx_obs_ip_time ON observations(ip, fetched_at DESC);

CREATE TABLE queries (
    id          INTEGER PRIMARY KEY,
    ip          TEXT NOT NULL,
    queried_at  INTEGER NOT NULL
);
CREATE INDEX idx_queries_ip ON queries(ip, queried_at DESC);

CREATE TABLE quota_usage (
    source   TEXT NOT NULL,
    day      TEXT NOT NULL,          -- YYYY-MM-DD，UTC
    used     INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (source, day)
);
```

`0001_init.down.sql` 对应 `DROP TABLE`，顺序与创建相反。

只有三张表。**不做归一化宽表** —— 宽表一旦定了列，加源就要改表结构，而我们的源是会持续增加的。

`quota_usage` 从 Stage 0 就建好、Stage 3 才用：它必须落库而非放内存（见 ADR-7），提前建表避免 Stage 3 再加一次迁移。

## 前端路径与 CDN 规划

路径规划的原则是**让缓存策略能按路径前缀一刀切**。

### 三层缓存

```
浏览器/边缘 CDN  →  origin 内存/DB 缓存  →  外部源 API（有额度）
   1 小时              24 小时                 最贵的那层
```

**IP 报告必须走 CDN 缓存。** 这是免费额度能否撑住的关键——热门 IP 被反复查询是常态，边缘命中后 origin 都不碰，源 API 一次都不用花。

| 路径 | 内容 | Cache-Control |
|---|---|---|
| `/` | 首页（查询框） | `public, max-age=300, s-maxage=3600` |
| `/ip/<addr>` | IP 报告（HTML） | `public, max-age=60, s-maxage=3600, stale-while-revalidate=86400` |
| `/asn/<asn>` | ASN 聚合页（HTML） | `public, max-age=600, s-maxage=86400, stale-while-revalidate=604800` |
| `/static/<name>.<hash>.<ext>` | JS / CSS / 字体 / 图标 | `public, max-age=31536000, immutable` |
| `/api/*` | JSON 接口 | `no-store` |
| `/<addr>`（裸路径） | 纯文本 | `public, max-age=300, s-maxage=3600, stale-while-revalidate=86400` |

`stale-while-revalidate` 是关键：过期后用户仍拿到旧页面并立即返回，后台异步刷新。**用户永远不等待刷新**，而内容最多旧 1 小时——这对一份 24 小时才更新一次的数据完全不是问题。

具体约束：

- **静态资源必须带内容哈希**（`app.a3f9c2.js`），不带哈希的资源文件名不允许上线。这样能放心用一年期 immutable 缓存，改版即换名，无需 purge
- **静态资源目录固定 `/static/`**，Cloudflare 侧用路径规则直接命中，不用写复杂匹配
- **API 独立 `/api/` 前缀**，永远不参与 CDN 缓存，避免"数据被缓存住"这类最难排查的故障
- **HTML 也缓存**，但 TTL 分级：报告页 1 小时、ASN 页 1 天、首页 5 分钟
- **页面无任何个性化**：不读 Cookie、不按 UA 分流、不同身份看到同一份内容。这是能被 CDN 缓存的前提，也是一条红线
- 不用 query string 做版本（`?v=3`）：部分 CDN 和代理对 query 的缓存行为不一致

### 缓存带来的两个必须处理的问题

**1. 冷启动击穿。** 一个陌生 IP 突然被大量访问（论坛贴出来），CDN 全部 miss，origin 同时收到 N 个相同请求，N 个都去调源 API —— 额度瞬间烧光。origin 侧必须对同一 IP 做 singleflight 合并，只发一次外部请求。

**2. 查询计数必须移出渲染路径。** 一旦 HTML 被 CDN 缓存，origin 就看不到命中请求了，"该 IP 被查询 N 次"这个数字会严重少计。处理方式：计数这个字段**不上报告页**，只在 `/api/*` 和需要 `no-store` 的历史页里出现；或者用 Cloudflare Worker 在边缘计数写 KV。

Go 侧通过 `embed.FS` 打包 `web/`，启动时把 `static/` 挂到 `/static/` 并设置 `Cache-Control`。将来静态资源量大了，把 `/static/` 整个前缀切到对象存储或 Cloudflare R2，Go 侧只需改一行路由。

## 任务入口（Taskfile）

所有操作收敛到 `task`，不靠口头约定的命令行：

| 命令 | 作用 |
|---|---|
| `task build` | 编译到 `bin/ipspot` |
| `task dev` | 构建并启动，读 `configs/config.yaml` |
| `task test` | `go test ./...` |
| `task cover` | 覆盖率报告 |
| `task lint` | `gofmt -l` + `go vet` |
| `task migrate:up` | `bin/ipspot migrate up` |
| `task migrate:down` | `bin/ipspot migrate down 1` |
| `task migrate:version` | 查看当前版本 |
| `task migrate:create NAME=x` | 生成 `NNNN_x.up.sql` / `.down.sql` |
| `task clean` | 清理 `bin/` 与 `data/` |

**迁移走二进制自带子命令，不依赖外部 `migrate` CLI。** `bin/ipspot migrate up` 内部把 `migrate` 当库用，DSN 从同一份 `configs/config.yaml` 读，迁移和程序不会连到不同的库。

`docker:*` 任务到 Stage 5 再加（Stage 0 不做部署脚本）。

## 核心接口

```go
type Observation struct {
    Source  string
    Fields  map[string]any
    Fetched time.Time
    Err     error
}

type Source interface {
    Name() string
    Query(ctx context.Context, ip netip.Addr) Observation
}
```

两处刻意设计：

- **`Fields` 不归一化**：每个源返回什么就存什么，这是"存原始观测不存结论"的类型级落实
- **`Err` 是字段而非返回值**：部分失败能混在成功结果里一起返回，调用方永远看得到哪个源挂了

## 验收标准

- [ ] `task build` 通过，`bin/ipspot` 可运行
- [ ] `GET /healthz` 返回 200
- [ ] `task migrate:up` 建三张表成功，`task migrate:down` 可回滚
- [ ] `task migrate:create NAME=test` 能生成 up/down 文件对
- [ ] 配置文件加载 + 环境变量覆盖生效
- [ ] `task test` 通过（占位测试）
- [ ] `/static/` 下带 hash 的资源返回 `immutable`，不带 hash 的返回 `no-cache`
- [ ] `bin/ipspot migrate up` 与 `task migrate:up` 等价
- [ ] 依赖数 <= 4（gin / sqlite / migrate / yaml）
- [ ] 全部技术决策记录进 ADR

## 实现记录

### 与设计的差异

| 项 | 设计 | 实际 | 原因 |
|---|---|---|---|
| Go 版本 | 1.25 | **1.26.0** | 依赖链（`golang.org/x/sys`、`modernc.org/libc`）已要求 1.26，无法回退 |
| `migrate` CLI | Taskfile 调外部 CLI | 二进制自带 `ipspot migrate` 子命令 | 本机无 CLI，且单二进制更符合部署目标 |
| 迁移连接 | 复用业务 `*sql.DB` | 独立连接 | `migrate/database/sqlite` 在 Close 时会把底层 `*sql.DB` 关掉，复用会废掉连接池 |
| 迁移文件位置 | 仅 `migrations/` | `migrations/` + `internal/store/migrations/`（后者供 `go:embed`） | embed 不能跨包目录。两处由 `task migrate:create` 同步创建 |
| 模板渲染 | `c.HTML` | `render.HTMLRender` 适配 + `c.Render` | 直接 `c.HTML` 需要 `*gin.Engine` 上有 render 实例，测试中构造 `gin.New()` 时会 nil panic |
| 静态资源缓存 | 中间件按路径判定 | 抽成 `cacheControlFor(path)` 纯函数 | 缺失资源被 `http.FileServer` 返回 404 时会清空头，无法在集成测试中断言，纯函数可直接测 |
| IP 校验 | `strings.Contains(".")` | `netip.ParseAddr` | 原实现让 `/favicon.ico`、`/robots.txt` 都返回 200 |
| `/ip/:addr` 校验 | 仅纯文本路由校验 | 报告页同样校验 | 非 IP 路径曾返回 200 + `s-maxage`，可被构造出无限缓存条目 |
| 静态资源缓存头 | 请求前统一设置 | 只在 200 响应上设置 | 404 曾带 `immutable`，缺失的 hash 资源会被客户端缓存一年 |
| `cover` 范围 | `./...` | `./internal/...` | `cmd/` 与空壳包（`web`、`model` 零覆盖率包）拉低整体数字 |

### 技术决策

- 迁移驱动只用 `migrate/database/sqlite`（纯 Go），**不用** `sqlite3`（需 CGO）
- 缓存与配额走接口：`cache.Cache` / `quota.Guard`，单机实现分别为 `Memory` 与 `SQLGuard`
- `Taskfile.yml` 固定 `GOTOOLCHAIN=go1.26.0`：本机 gvm 是 1.25.5，不固定会导致 `-cover` 路径混用两套编译器，报 `compile: version "go1.26.0" does not match go tool version "go1.25.5"`

### 测试证据

- 单元测试覆盖率：**83.3%**（`./internal/...`，`task cover`）
- 每个包：cache 91.7% / httpapi 94.9% / query 90.9% / store 78.1% / quota 78.3% / config 75.6%
- E2E（真实 HTTP）：
  - `task dev` 启动后逐路径验证
  - `/healthz` 200 · `/` 200 · `/ip/1.1.1.1` 200 · `/8.8.8.8` 200（text/plain）· `/2606:4700::1111` 200 · `/static/app.css` 200（`no-cache`）
  - `/favicon.ico`、`/robots.txt`、`/hello.world` 均 404
- 真实环境验证：本机 8080 端口，`curl --noproxy '*'` 逐条确认状态码与 `Cache-Control`
- 干净环境验证：`rm -rf bin data && task build && task migrate:up && task migrate:version` → `version=1 dirty=false`

### 交接说明

- `bin/ipspot migrate up|down -steps N|version` 是唯一迁移入口
- 新增迁移用 `task migrate:create NAME=x`，会同时写 `migrations/` 与 `internal/store/migrations/`
- `configs/config.yaml` 已在 `.gitignore`，`configs/config.example.yaml` 才是模板
- `data/` 与 `bin/` 均已忽略
- 引擎 `query.Engine` 已支持并发拉源，但 `source.Source` 尚无任何实现——Stage 2 从零开始加 adapter

## 回滚方案

只有新增文件，无存量代码。删目录即可。

## 单机部署边界

**架构目标是单实例。** 理由：成本、SQLite 单写者模型、Stage 1 尚未验证需求。但单机会让三样东西只存在一份，必须提前定规矩，否则将来多副本时**静默失效**：

| 状态 | 单机 | 多副本时的后果 |
|---|---|---|
| origin 内存缓存 | 一份，命中率正常 | 各存各的，命中率除以 N，额度消耗乘以 N |
| 每源配额计数 | 准确 | 各算各的，**总计超限且不报错** |
| singleflight | 有效 | 各进程各发一次，防击穿失效 |

**Stage 0 定三条不花成本的规矩：**

1. **缓存与配额必须走接口**（`internal/cache`、`internal/quota`），单机只是其中一个实现
2. **配额计数持久化到数据库**，不放内存——重启不清零，且换 Postgres 后天然支持多副本
3. **singleflight 保留进程内实现，但必须记 `shared` 日志**——多副本时能从日志看出「同一 IP 在不同进程各打了一次」，这是迁移信号

这三条进 Stage 3 的验收项。现在不做多副本，但也不把路堵死。

## 已知局限

- 缓存、CDN、限流全未实现（Stage 3 / 5）
- 无监控（Stage 5）
- 图表方案推迟到 Stage 4 决定（内联 JSON + uPlot 优先）

## 对后续 Stage 的影响

- `Source` 接口定型后，Stage 2 的 6 个 adapter 全部按此实现
- `observations` 表结构决定 Stage 3 的缓存 key 与 Stage 6 的历史对比
- `/static/` 前缀约束意味着 Stage 4 引入的 uPlot/htmx 必须放 `web/static/` 并走哈希命名
- Taskfile 收敛后，Stage 5 的部署脚本也走 `task`，不另起一套
