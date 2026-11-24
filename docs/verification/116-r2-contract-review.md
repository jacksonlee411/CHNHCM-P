# CHNHCM-116 R2事务与恢复合同增量评审

登记信息：[登记表](../governance/registry.json) `governance.verification_records`。
适用范围：R2-A事务提炼、R2-B提交恢复；不含部门运行时和R3—R6。
上游依据：[116](../plans/116-m02-shared-dated-maintenance-roadmap.md)、[120](../specs/m02/120-dated-maintenance-technical-contract.md)、[121](../specs/m02/121-dated-object-admission-matrix.md)。

## 1 验证对象与受测基线

2026-11-05，CHNHCM `wt-dev-b`，基线 `90dce9927c9e3541898b045070982054a11099b1`。该提交仅收口R1评审修复。本次合同增量覆盖116、120、013、121、023及导航；执行报告另行记录运行结果。

## 2 环境与前置条件

本工作树单代理进行范围及源代码对照评审。基线工作区干净，stash为空；未启动、停止或重建环境。合同评审不涉及资源清理。

## 3 验证过程与结果

| 场景或检查 | 前置与操作步骤/命令 | 预期结果 | 实际结果 | 结论 | 证据位置 |
| --- | --- | --- | --- | --- | --- |
| R2范围与批准边界 | 对照116阶段表、用户推进R2指令 | 只实施事务和恢复 | 分R2-A/R2-B；R3—R6及部门运行时后置 | 通过 | 116第4节 |
| 事务职责 | 对照120、013及当前Store/Unit | Go保持业务权威、SQL保持对象归属 | 共用事务/保存点；BU候选、表组和SQL留适配器 | 通过 | 120第5.3节 |
| 原结果恢复 | 对照commands/Resolve与120 | 原提交身份、原终态及当前读权均保留 | 明确摘要、锁键、原来源、确认提交边界与当前字段投影 | 通过 | 120第5.3节 |
| 验收与证据 | 对照121、023及122/123回归边界 | 不以纯端口测试代替真实对象接入 | 复用稳定案例ID；部门39项仍未运行 | 通过 | 121第7节、023第27节 |
| 文档门禁 | make docs | 登记、关联及结构通过 | 通过 | 通过 | [contract-docs.txt](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r2/contract-docs.txt) |

## 4 失败、修复与复验

发现013仍描述按HTTP状态自动提交拒绝事务，已按现有明确提交意图及读拒绝标记修订；规格入口仍有“R1—R6未启动”的旧描述，已与R1完成/R2启动同步。上述为合同陈旧描述修复，不宣称运行缺陷已发生。

## 5 未覆盖范围与限制

本报告只评审合同增量，不替代代码复核、真实数据库故障注入或浏览器验收；不修改已定稿的R0、R1、122、123历史证据。

## 6 结论与后续事项

语义范围评审及文档门禁通过；检查73个脚本用例、234份Markdown及7329个仓内链接。后续按[执行记录](116-r2-transaction-and-recovery.md)分批实施、验证和复核；不得提前将R2登记为完成。

## 7 证据保存

基线见[data/116-r2/baseline.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r2/baseline.json)，原证据保护清单见[data/116-r2/protected-before.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r2/protected-before.json)。原报告及来源资产只读保留。
