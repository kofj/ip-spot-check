# ip-spot-check

多源并列的 IP 检测工具。不提供总分。

## 文档

RDD 框架入口：`AGENTS.md`（AI Agent）、`CLAUDE.md`（Claude Code）。当前状态：`docs/stages/stage-roadmap.md`。

### 文档

| 文档 | 内容 |
|---|---|
| [stage-roadmap](docs/stages/stage-roadmap.md) | 路线图与 Stage 状态 |
| [stage-0](docs/stages/stage-0.md) | Stage 0：技术框架定型 |
| [stage-1](docs/stages/stage-1.md) | Stage 1：分歧验证 |
| [stage-3](docs/stages/stage-3.md) | Stage 3：查询编排与缓存 |
| [01-charter](docs/01-charter.md) | 项目章程 |
| [02-engineering-principles](docs/02-engineering-principles.md) | 工程原则 |
| [11-next-steps](docs/11-next-steps.md) | 下一步 |
| [12-technical-debt](docs/12-technical-debt.md) | 技术债 |

## 开发

```bash
cp configs/config.example.yaml configs/config.yaml
task build          # 编译到 bin/ipspot
task migrate:up     # 建表
task dev            # 启动，默认 :8080
task test           # 单元测试
task cover          # 覆盖率
task lint           # gofmt + go vet
```

本机 go 为 gvm 1.25.5，模块要求 1.26.0；`Taskfile.yml` 已全局固定 `GOTOOLCHAIN=go1.26.0`。

## 一句话

本站不接受 IP 商家付费，数据全部来自公开源并可复现。
