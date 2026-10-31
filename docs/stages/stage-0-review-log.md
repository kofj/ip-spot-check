# Stage 0 评审日志

**评审类型**：设计
**评审日期**：2026-10-31
**评审者**：主 Agent（自评）+ 规则检查
**Stage**：0

## 评审摘要

设计整体可执行，但有两个会让实现阶段直接卡住的硬问题：依赖版本与目标 Go 版本不匹配、migrate CLI 未安装却写进了 Taskfile。均已修正。

## 发现

| # | 严重度 | 发现 | 处理 |
|---|---|---|---|
| 1 | **Critical** | `golang-migrate/migrate/v4@latest`（v4.20.1）要求 Go >= 1.25.11，本机为 1.25.5 | 固定 `v4.19.0`（要求 1.23.0）。已改设计并写入 go.mod 约束 |
| 2 | **Critical** | Taskfile 的 `migrate:*` 直接调 `migrate` CLI，但本机未安装，Docker 镜像里还要额外装 | 改为 **migrate 作为库 + 二进制自带 `migrate` 子命令**，单二进制零外部依赖 |
| 3 | High | `database/sqlite` 与 `database/sqlite3` 两个驱动易混：前者纯 Go（modernc），后者需 CGO | 明确只用 `database/sqlite`。已核实其 import `modernc.org/sqlite`，无 CGO |
| 4 | Medium | 静态资源"必须带内容哈希"没有可执行的校验手段 | 中间件按文件名正则判定：含 `.<8位以上hex>.` 才发 immutable，否则 no-cache。规则可测 |
| 5 | Medium | 设计里 `quota` 包只给了接口，但表在建 | 补最小 SQLite 实现，用真实表验证迁移产物 |
| 6 | Low | 裸路径 `/<ip>` 与 `/ip/<addr>` 并存，路由易冲突 | gin 中显式顺序：先注册静态路由组，再用 `/:addr` 兜底并校验是否合法 IP |
| 7 | Low | `Source.Query` 返回结构体而非 `(T, error)` 是刻意设计，但未说明调用方如何判断失败 | 约定 `Err != nil` 即为失败，`Fields` 此时可为空；测试覆盖该契约 |

## AI 预过滤结果

无过滤项。7 条全部核实为真实问题（1、2 通过实际拉取模块与检查本地环境确认；3 通过阅读驱动源码确认）。

## 规则检查

| 规则 | 结论 |
|---|---|
| 记忆偏差（依赖未声明的上下文） | 已修：设计里"migrate CLI"默认它存在，实际不存在 |
| 逻辑谬误 | 无 |
| 范围蔓延 | 无。补的 quota 实现是为了验证建表，不越界 |

## 解决摘要

**总发现**：7
**已修**：7（其中 1、2 在实现前改了设计）
**延期**：0
**不修**：0

设计文档已同步更新。

---

## 代码评审（Stage 0）

**评审类型**：代码
**评审日期**：2026-10-31
**评审者**：主 Agent（自评）+ 集成测试

### 审查文件

- `cmd/ipspot/main.go`
- `internal/config/config.go`
- `internal/model/model.go`
- `internal/source/source.go`
- `internal/query/engine.go`
- `internal/store/store.go`、`migrate.go`
- `internal/cache/cache.go`
- `internal/quota/quota.go`
- `internal/httpapi/server.go`、`render.go`
- `internal/store/migrations/0001_init.{up,down}.sql`
- `web/templates/*.html`、`web/static/app.css`
- `Taskfile.yml`

### 发现

| # | 严重度 | 发现 | 处理 |
|---|---|---|---|
| 1 | **Critical** | `migrate/database/sqlite` 的 `Close()` 会关闭底层 `*sql.DB`，与业务连接池共用会导致**后续所有查询报 `sql: database is closed`** | 迁移改用独立连接（`withMigrator` 内部 `sql.Open` + `defer Close`）。已用 `TestMigrateUpIsIdempotent` 等 4 个测试覆盖 |
| 2 | **Critical** | `c.HTML` 在 `gin.New()` 构造的 router 上 nil pointer panic（`c.engine.HTMLRender` 为 nil） | 改用 `render.HTMLRender` 适配器 + `c.Render`，并在 `Router()` 里 `SetHTMLTemplate` |
| 3 | High | `looksLikeIP` 用 `strings.Contains(".")`，导致 `/favicon.ico`、`/robots.txt`、`/hello.world` 全部返回 200 | 改 `netip.ParseAddr`，新增 `TestNonIPPathIs404` 覆盖 4 条路径 |
| 4 | High | 干净环境首次 `task migrate:up` 失败：`unable to open database file (14)` | `withMigrator` 内先 `MkdirAll` 数据目录。已用「删 `bin` 与 `data` 后重跑」验证 |
| 5 | Medium | `Taskfile` 的 `sources` 声明在 `bin/` 被删后仍判「up to date」，导致构建被跳过 | 去掉 `sources`，`build` 每次都编译（成本可忽略） |
| 6 | Medium | 静态资源缓存的集成测试失败：`http.FileServer` 对缺失文件返回 404 时会清空 `Cache-Control` | 抽出 `cacheControlFor(path)` 纯函数并单测，集成测试只断言存在的资源 |
| 7 | Medium | 本机 gvm 为 go1.25.5、模块要求 1.26.0，`go tool cover` 与 `go test -cover` 混用编译器报 `version does not match` | `go.mod` 加 `toolchain go1.26.0`，`Taskfile.yml` 全局 `GOTOOLCHAIN=go1.26.0` |
| 8 | Low | 覆盖率统计把 `cmd/` 与空壳包算进总分，从 83.3% 被稀释到 62.5% | `task cover` 限定 `./internal/...` |
| 9 | Low | 测试断言写反（`TestRunPreservesPartialFailure` 里 `!Failed()` 判断） | 已修正 |

### 解决摘要

**总发现**：9
**已修**：9
**延期**：0
**不修**：0

**所有 Critical 已修**：是
**所有 High 已修**：是
