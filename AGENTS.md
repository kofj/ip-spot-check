# Agent 入口

> 所有在此项目工作的 AI Agent 的入口文档。

## 必读顺序

1. **从这里开始**：`docs/01-charter.md` —— 项目愿景与目标
3. **流程**：`docs/03-stage-based-development.md` —— 我们怎么工作
4. **原则**：`docs/02-engineering-principles.md` —— 工程标准
5. **当前状态**：`docs/stages/stage-roadmap.md` —— 我们在哪
6. **下一步**：`docs/11-next-steps.md` —— 做什么
7. **决策**：`docs/08-autonomous-decisions.md` —— 过去的决策
8. **债务**：`docs/12-technical-debt.md` —— 已知问题

## 红线（不可协商）

1. **不出现总分。** 任何形式的合并评分都是违规
2. **不做主观定性。** 只写「AbuseIPDB 报告 47 次」，不写「该 IP 是攻击者」
3. **不接 IP 商家付费。** 广告、赞助、联盟推广、软文一概不接
4. **源失败必须可见。** 不吞错误、不显示旧值伪装正常
5. **不存结论，存原始观测**

动手前先确认方案不在已否决清单里。

## 文档更新义务

完成工作时必须更新：

1. **Stage 文档**（`docs/stages/stage-N.md`）—— 实现差异
2. **决策**（`docs/08-autonomous-decisions.md`）—— 新 ADR
3. **技术债**（`docs/12-technical-debt.md`）—— 新增或已解决
4. **下一步**（`docs/11-next-steps.md`）—— 进度
5. **路线图**（`docs/stages/stage-roadmap.md`）—— 状态
6. **CHANGELOG.md** —— 变更摘要

## Stage 升级规则

完成 Stage 前确认：

- [ ] 验收标准全部满足
- [ ] 单元测试通过
- [ ] 真实环境验证
- [ ] 设计文档与实现一致
- [ ] 技术债已更新
- [ ] ADR 已记录（含对后续影响）

## 审查规则

- 多模型三角验证
- 一半发现是假阳性，逐条核实
- 记录在 review log
- 低置信度交人工
