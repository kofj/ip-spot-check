# Changelog

本项目的重要变更记录在此。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)。

## [Unreleased]

## [Stage 0] - 2026-10-31

### Added
- Go 1.26 项目骨架：Gin + SQLite（modernc）+ golang-migrate（作库）+ html/template
- `cmd/ipspot`：HTTP 服务与 `migrate up|down|version` 子命令
- `internal/` 七个包：config / model / source / query / store / cache / quota / httpapi
- `0001_init` 迁移：`observations` / `queries` / `quota_usage` 三张表，含 down 脚本
- `Taskfile.yml`：build / dev / test / cover / lint / migrate:* / clean
- 路由：`/healthz`、`/`、`/ip/:addr`、`/static/*`、`/:addr`（纯文本）
- 缓存与配额接口及单机实现（尚未接入查询路径）

### Changed
- Go 版本从设计稿的 1.25 改为 1.26.0（依赖链要求），Taskfile 固定 `GOTOOLCHAIN`
- 迁移从「调外部 CLI」改为「嵌入二进制子命令」

### Technical Debt
- Introduced: TD-01 迁移文件存在两份副本
- Introduced: TD-02 配额与缓存尚未接入查询路径
- Introduced: TD-03 首页表单用内联 JS
- Introduced: TD-04 `cmd/` 无测试

### Metrics
- 测试覆盖率 83.3%（`./internal/...`）
- 空壳启动并逐路由验证通过

## [0.1.0] - 2026-10-31

### Added
- RDD 框架初始化
- Stage 0 / Stage 1 设计文档
