# China HCM 目标设计基线

采用 FSCR `d88c993d13d325b18943549f120c5f03e7864205` 的最新设计及命名增量作为 CHNHCM 输入；**命名增量待整合验证**。R25 是已通过的冻结评审，不表示当前所有输入已复核。来源原件、摘要及状态见[基线记录](../../reference/README.md)。

本目录是本仓采用的目标设计，后续变更由 CHNHCM 决策与规格维护；来源历史资料只作证据，不能覆盖本仓阶段规则。导入时只调整失效的相对链接，未改领域含义或重绘图。

初始导入后的本仓变更单独登记：[CHNHCM-009](../../adr/009-mainland-organization-terms.md)已明确大陆用人/用工/关联/付款角色及工作关系语义，相关词表和图按同一源数据维护；[本仓修订链](../../reference/local-revisions.json)保留初始采用摘要。该局部修订不表示源 R25 后全部命名增量已验证。

[CHNHCM-018](../../adr/018-effective-records-and-source-history.md)取消任意系统时间截点查询及提交证据采集，保留生效记录、确切来源和消费者采用；E04、X01、DDD 及17术语表已同步。历史评审范围保持，当前验证见[018报告](../../verification/018-source-history-simplification.md)。

[CHNHCM-128](../../adr/128-authorization-evidence-removal.md)已撤除默认永久保存逐次AccessDecision/RecordAccessEvidence及详情关联快捷跳转的设计。本文链接的X03、DDD、聚合/术语目录、源数据和图中相关旧条款不再作为现行要求；保留当前判权、必要安全事件和真实业务历史。当前源JSON、DDD/概念/聚合/术语、X03/X04/X05/X06和两张源图/SVG已按本仓修订链同步；实施与验证进展见[128实施记录](../../verification/128-implementation.md)。原导入包与历史验证不改写。

- [G02 整体业务架构](diagrams/g02-target-domain-map.svg)
- [G05 DDD 限界上下文](diagrams/g05-context-map.svg)
- [完整图集：整体 → C01–C05 → 模块 → 关键设计](diagrams/README.md)
- [目标架构](02-target-architecture.md) · [一致性合同](03-consistency-contracts.md)
- [DDD 合同](10-ddd-and-state-contracts.md) · [聚合与对象归属](14-aggregate-catalog.md)
- [命令验收问题](16-command-acceptance-index.md) · [术语表](17-terminology-glossary.md)
- [首个模块 M02](modules/m02.md)

53 张图、20 个 M 模块、6 个 X 能力、17 个上下文是目标范围；154 组对象和 131 项概念不是产品实现完成数。源图及 SVG 均在本仓，无需启动原项目即可阅读。
