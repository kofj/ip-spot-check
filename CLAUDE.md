# Claude Code 入口

> 本项目使用 RDD（Roadmap Driven Development）框架。

## 快速开始

```
# 查看当前状态
cat docs/stages/stage-roadmap.md

# 查看下一步
cat docs/11-next-steps.md
```

## RDD 命令

| 命令 | 用途 |
|------|------|
| `/rdd-roadmap` | 管理路线图 |
| `/rdd-stage-auto` | 带门禁执行 Stage |
| `/rdd-review-auto` | 自动审查 |
| `/rdd-fresh-check` | 验证新 Agent 能否接管 |
| `/rdd-diagnosis` | 诊断问题 |

## 关键文件

| 文件 | 用途 |
|------|------|
| `AGENTS.md` | Agent 入口（先看这个） |
| `docs/01-charter.md` | 项目愿景 |
| `docs/stages/stage-roadmap.md` | 当前状态 |
| `docs/11-next-steps.md` | 立即行动 |
| `.rdd/config.yml` | RDD 配置 |

## 红线

不出现总分 / 不做主观定性 / 不接商家付费 / 源失败必须可见。

详见 `AGENTS.md`。

## 技术栈

Go 1.25 + Gin + SQLite（modernc，纯 Go）+ golang-migrate + html/template。配置走 YAML 文件 + 环境变量覆盖。所有操作经 `task` 收敛。

完整决策见 `docs/stages/stage-0.md` 与 `docs/08-autonomous-decisions.md`。


## 技术红线

- 不在主站引入 React / Next。需要交互先考虑 htmx
- 静态资源必须放 `web/static/` 且带内容哈希，走 `/static/` 前缀
- 数据库变更必须带 `up.sql` 与 `down.sql`
- 新增直接依赖需先写 ADR

## 工具使用规则

1. 编辑前先读文件
2. 用 Grep / Glob 搜索
3. 提交用精准 `git add`，禁用 `git add *`，必须 `--signoff`
