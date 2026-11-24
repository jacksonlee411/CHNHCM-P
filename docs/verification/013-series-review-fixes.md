# CHNHCM-013 系列方案评审修订验证

登记信息：[registry.json](../governance/registry.json) 的 `governance.verification_records`。适用范围：[011 业务需求](../specs/m02/011-m02-business-requirements.md)、[012 交互](../specs/m02/012-m02-functional-and-interaction-design.md)、[013 技术设计](../specs/m02/013-m02-technical-design.md)的历史返回边界，以及 [009](../adr/009-mainland-organization-terms.md)/[DDD 合同](../architecture/target/10-ddd-and-state-contracts.md)的后续规格承接。

## 1 受测基线与范围

2026-10-31，工作树 `/home/vivobook/projects/CHNHCM-wt-dev-a`，分支 `wt-dev-a`，HEAD 为 `4a9844498b9a51330d6011ab4a5efe7556616709`。开始时已有前轮 20 项未提交或未跟踪文件；用户要求修正系列评审问题后本地提交全部更改。本报告的受测对象是该基线与当前差异，不使用报告自己的最终提交号替代受测基线。

本轮修订前 8 份资料保存在[快照](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/013-before-series-review.zip)，当前变化及未改资料摘要见[本轮输入](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/013-series-review-inputs.json)。初次 [013 编写验证](013-technical-design.md)、[表设计补验](013-table-design-review-fixes.md)及其摘要保持，不将旧结论扩大为本轮输入已通过。

## 2 问题、修订与独立预期

| 发现 | 原问题及触发场景 | 本轮修订与预期 |
| --- | --- | --- |
| P2：尚未登记结果可能展示后来资料 | 013 合并 NO_FACT/NOT_RECORDED 并共同允许展示未来首条；T1 才登记的资料可能出现在早于 T1 的历史查询 | 拆开响应分支；NOT_RECORDED 无已解析来源、记录或时间轴，只保留当前授权下必要定位；NO_FACT 只能展示实际解析来源的未来首条。012 明确切换时清空资料/来源区，011 的 AC008 增加正反例，013 第 8.2 节增加接线验证 |
| P2：M04 后续设计误交 013 | 009 与 DDD 将工作关系字段/事务及 RegisterAssociation 接口交给 013，与 010 的首批 BU 范围冲突 | 009 入口、正文及维护说明和 DDD 合同统一交给后续 M04 专项规格，待立项确定编号；本批 010—013 仅承接 M02 BU；角色与关系规则保持 |

历史独立样例为：T1 首次登记未来首条 A，T2 更正为 A2，查看日均早于首日。T1 前截点不得返回后来资料；T1/T2 之间及确切 T1 来源只能展示 A；当前读取展示 A2。样例是新增待实施预期，本轮没有执行 API、数据库或页面来证明它通过。

目标 DDD 文本修改由 [local-revisions.json](../reference/local-revisions.json) 的 CHNHCM-009 第 2 次修订衔接；原链条、来源包、模块/术语数据及图稿保持。未引入 M04 实现、审批、工作台或消费者采用。

## 3 检查与复核结果

| 检查 | 实际结果与证据边界 |
| --- | --- |
| 独立历史返回复核 | `/root/m02_012_flow_review` 回读确认原 P2 在规格层关闭，四种查询预期、显示清理及响应分支一致，无新增矛盾；未运行产品 |
| 独立范围/DDD 复核 | `/root/m02_plan_contract_review` 回读确认原 P2 关闭，仅澄清承接去向；组织角色、关系语义及首批范围保持，修订链前后摘要匹配 |
| 文档与引用检查 | make docs 的 46 项 Python 回归通过；补齐当前验证入口后，最终仓库检查核对 117 个 Markdown、2773 个本地链接、53 张目标图及 197 项资产，固定分支引用与 git diff --check 通过 |
| 历史保护和受测输入 | 8 项快照与修订前摘要匹配；本轮 9 项输入及 302 项未改文件摘要匹配，旧 20 份验证登记和 9 条修订链条目保持；前两轮分别 305/308 项输入及保护文件可从当前文件和快照重建 |

两项发现均在静态文档范围关闭，独立复核者不参与修改。新增业务预期、数据库查询及缓存清理仍为待实施验证，不能把文档检查记为产品通过。

## 4 交付及限制

010—013 保持 draft，009 保持已采纳的角色决定；本轮仅关闭上述交接和历史返回歧义，文档状态、实施进度及资产采用状态不提升。原法律/Oracle 事实没有改变，也未重新核验其时效。

没有修改 Go/前端业务代码、SQL 迁移、工具依赖或检查器，没有启动服务、数据库或浏览器。来源捕获、约束、权限与历史查询的真实验证仍由后续实施完成。用户授权的提交是本地 Git 提交，不包含推送或发布。

## 5 输入保存边界

保留原 20 份验证登记、报告及输入摘要；本报告和本轮摘要自身排除于输入摘要，避免循环引用。前轮摘要中已发生变化的文件，可以通过本轮快照及更早的表设计快照取得对应旧字节；本轮保护摘要核对未修改范围，不覆盖旧失败或旧结论。
