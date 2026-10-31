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

**决策**：`html/template` 服务端渲染；如需交互则先考虑 htmx，最终手段才是 React/Next。

**理由**：三条硬约束——数据必须在首屏 HTML 里（SEO）、Go 单二进制不应引入第二个运行时、这个站没有客户端状态（一个输入框 + 一次提交 + 一段只读报告）。

**对后续 Stage 的影响**：
- Stage 4 的交互只能是表单提交；如需要局部刷新引入 htmx 而非框架
- Stage 6 的静态页生成可直接复用模板，无需另建渲染管线
- 若将来有登录后页面，可单独在 `/app` 引入，不影响主站

**日期**：2026-10-31

**相关 Stage**：Stage 0 / 4

**考虑过的替代方案**：
1. Next.js（SSR）：能解决 SEO，但需要 Node 常驻进程，两个容器两套依赖，个人项目负担过重
2. Astro 岛屿架构：静态页友好，但查询页仍需服务端数据，会形成两套渲染逻辑

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
