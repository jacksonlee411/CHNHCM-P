# CHNHCM-103 位置新增与插入规则同步记录

登记信息：[registry.json](../governance/registry.json)。适用范围：用户确认的八项规则及两项补充，仅文档同步。上游依据：[103决策](../adr/103-positioned-record-addition.md)、[011 BR005](../specs/m02/011-m02-business-requirements.md#m02-br-005)、[012位置交互](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-position-addition)、[013技术目标](../specs/m02/013-m02-technical-design.md#m02-tech-position-addition)。

## 1 验证对象与受测基线

2026-11-02，固定wt-dev-b，基线1b660877c0f23982d94094f15a9b2c63261b0ed6及本批未提交文档差异。变更限103决策、011—013、023及010/017/导航/登记/本报告和证据。受测文档清单、摘要见[输入清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/103-positioned-record-addition/input-manifest.json)，完整文档差异输入见[快照](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/103-positioned-record-addition/document-inputs.tar.gz)；这些摘要标识实际工作区，不把HEAD当作包含未提交规则。

## 2 环境与前置条件

本工作树文档检查环境；用户已确认规则，不需要另行征求业务取舍。原工作区干净、无stash。未启动或更改4382/4527及其他运行环境，未修改Go、Web、SQL、迁移、OpenAPI、生成客户端、测试代码、旧验证报告或来源资产。

## 3 验证过程与结果

| 检查 | 预期 | 实际结果 | 证据 |
| --- | --- | --- | --- |
| 规则语义与追踪 | 八项规则及两项补充跨文档一致，类型固定CHANGE/原因空，日期分位置默认，无可插入日期仅禁用且无原因；实现边界明确 | 通过 | [语义自查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/103-positioned-record-addition/review.json) |
| 案例及稳定ID | 原257编号保留；新增258—281共24项待验，AC030前后关联；无业务通过声明 | 通过；281编号/277现行/4退役，新24例未运行 | [一致性](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/103-positioned-record-addition/consistency.json) |
| 文档结构与链接 | make docs；登记/编号/标题/来源与本批链接可解析 | 通过；65项文档工具测试及登记/链接检查 | [最终文档检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/103-positioned-record-addition/docs-final.log) |
| 差异及原件保护 | git diff --check；代码、旧报告及来源摘要不变 | 通过；1,568项受保护原件摘要不变 | [差异](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/103-positioned-record-addition/diff-check.json)、[原件](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/103-positioned-record-addition/consistency.json) |

## 4 复核、修复与复验

首轮文档结构及链接检查通过，见[首轮输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/103-positioned-record-addition/docs-first.log)。语义自查发现并补齐了旧BR016入口/010路径、案例数量与共同锚点输入，以及默认候选非法但合法范围非空时不误禁用的细节；复核八项规则的[逐项追踪](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/103-positioned-record-addition/review.json)通过，无遗留发现。本轮复核重点是替换现行合同中的新增日期初始为空/按输入日决定位置，而不是只追加一段说明；既有定稿报告、带日期的历史进度保留原结果。普通新增目标合同显式锚点，联合ADD/启停/补全协议保持；不得为文档采纳预先标记实现或验证通过。

## 5 未覆盖范围与限制

不运行或宣称Go单元、真实DB/API、浏览器、容量及生产验证。本批只有规则和待验设计，TL-258—281全部未运行，旧T07和102证据不证明新规则已接线；用户数据和运行环境不变。主代理语义自查不等于独立评审或人工产品验收。

## 6 结论与后续事项

规则文档、登记及追踪已同步，make docs、稳定ID/案例计数、语义自查及git diff --check通过。产品/API/客户端及业务案例实现由后续任务承接；010旧完成基线和历史报告保持，不提交、推送、PR、合并或部署。

## 7 证据保存

仅保存文档输入/摘要、检查输出及受保护原件的摘要核对结果，不保存凭据、会话或业务库正文。非敏感原始输出位于.local/position-addition-docs，必要长期证据位于本报告data/103-positioned-record-addition；后续检查另留记录，不改写旧验证结果。
