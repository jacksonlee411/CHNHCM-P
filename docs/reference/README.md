# 来源基线与验证范围

公共仓库提供产品构建、合成测试及可分发的依赖。客户实库记录、历史运行包和仅供本地使用的来源包留在私有原仓。下文的私仓链接需要原仓访问权限。

人员登记与名录 T00 留存了 CubeBox 任职列表和多值资料设计的两份原文。501/502 记录采用范围，资产台账保留规格提炼与产品接线的区别。[人员来源包](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/reference/sources/cubebox-personnel-074fe044.zip) · [T00 评审](../verification/500-t00-personnel-specifications.md) · [资产台账](../assets/inventory.json)

国家/地区251条运行资料已纳入Git：[可审阅JSON](../../internal/contexts/organization/infra/countryseed/countries.json)、[版次及许可说明](../../internal/contexts/organization/infra/countryseed/NOTICE.txt)。提供方来源资料保存在版本资产中；国家HTTP资源及页面不再重复展示，业务修订历史保持。

2026-11-13补充国家/地区与四地资料：[200来源索引](200-geography-sources.json)登记本机CubeBox20份原字节及[离线包](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/reference/sources/cubebox-geography-2deaf927.zip)；[Odoo只读观察](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/200-geography-plan/odoo-observation.json)分别记录当前运行库与旧导出数据的差异。来源使用范围见[200路线图](../plans/200-geography-and-location-roadmap.md)，本轮未将旧原型通过或源数据数量计为本仓产品验收。

截至 2026-10-30 建仓采集时，采用的 FSCR 提交为 `d88c993d13d325b18943549f120c5f03e7864205`，核对的 CubeBox origin/main 为 `2deaf9276086f68e07aaa0c2cf7f35025046a9ee`。这些是固定输入，不持续跟随来源仓 HEAD。本仓保存离线资料及逐文件 SHA256，不依赖相邻工作区。

China HCM 采用上述提交包含的命名增量，状态为**待整合验证**。R25 冻结摘要 `ceff0834ac7bb7ead53b3b4b7171aa4a2c0cf953c6810353955d71e0ba87743e` 已通过范围评审，d88 的 26 项源输入与该冻结版本不同；该评审没有被扩展为 d88 输入或后续本仓 HEAD 全量通过。此决定保留用户建仓时的采用范围。

术语数区分来源口径：R25 冻结 JSON 为 405 项，d88 来源 JSON 及初始采用文件实际为 437 项；CHNHCM-009 增加三个角色术语后为 440 项，CHNHCM-011 首次补登记五个记录/查询公共用语后为 445 项；后续登记操作类型、操作原因、记录处理三个公共用语，本仓现行 [17 术语表](../architecture/target/17-terminology-glossary.md)为 448 项。新增仅说明业务分类和记录处理的区别；M02 固定目录仍归 M02，M04 人事目录职责保留，未增加共享聚合或通用配置平台。`baselines.json` 的 `counts` 保留源完成报告统计，`observed_terminology_counts` 保留各次直接核对结果；本次变化另记修订链，不改写来源数量、冻结摘要或旧评审范围。

M02 后续按用户决定统一为“删除记录”；17 中“删除未来记录”保留原稳定词条 ID 和英文映射，旧中文仅供历史检索，不构成另一入口或功能。术语总数仍为 448；修订范围、历史保护及检查见[删除用语修订验证](../verification/011-delete-record-terminology.md)。

按后续采纳的 B&B 设计，M02 新业务选择操作原因，更正/删除继承目标分类并自动留痕，不另填处理原因；此前删除用语报告保留其当时输入与结论，现行分类合同及本轮静态来源核对见[分类简化验证](../verification/011-record-classification-simplification.md)。448 项词条身份和来源冻结范围保持。

[CHNHCM-018](../adr/018-effective-records-and-source-history.md)取消 knownAt，保留 448 个词条身份；Known-at Time 仅供历史定位，现行查询使用业务日期、确切来源与采用引用。DDD、M01/X01/X02/X06 及 E04 图的修订另记本仓修订链，不改写来源包、初始采用摘要或旧验证结论。

- [基线及逐项摘要](baselines.json)
- [本仓设计修订链](local-revisions.json)：从初始采用摘要逐项衔接后续变更；不覆盖来源包或把新设计字节归入旧评审。
- [文档生成工具来源](document-tooling.json)：本仓独立维护的格式化及渲染入口。
- [M02 术语外部依据](m02-terminology-sources.json)：CHNHCM-009 于 2026-10-30 实际核对的法律条文、Oracle 资料渠道与项目解释边界。
- [China HCM 来源包](sources/china-hcm-d88c993.zip)：采用内容的原字节、源 README、UPL、R25 冻结输入及评审/状态记录；只作证据。
- [CubeBox M02 与共享资产来源包](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/reference/sources/cubebox-m02-2deaf927.zip)：197 个三类优先来源及技术选型原件；不是全部产品资产。
- [本仓目标设计](../architecture/target/README.md) 与 [资产台账](../assets/inventory.json)
- [未采用的历史引用路由](not-imported.md)

源包不可原地覆盖。未来采用新来源应新增包/记录，评估差异并更新受影响合同。源 JSON 中的路径和历史状态属于原记录，不作为本仓可执行指令；源包内的 AGENTS 也不替代根 AGENTS。建仓时未重新核验 Oracle 在线文档或当前属地政策；CHNHCM-009 仅核对上述组织角色与工作关系的直接依据，继承的其他资料链接不是现行政策准确性的证明。

019补充共享导航、日期轴、布局与离线字体来源；独立原件和摘要见[补充来源](ui-sources.json)、[UI来源包](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/reference/sources/cubebox-ui-2deaf927.zip)及[字体来源包](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/reference/sources/cubebox-font-2deaf927.zip)。原初始包及其摘要不变，实际采用和验证由[019](../plans/019-cubebox-experience-migration.md)追踪。

122操作摘要与日志表格化的[来源索引](122-operation-summary-sources.json)记录Microsoft/Oracle官方依据、PeopleSoft职务代码概要实机观察及不采用项；[概要截图](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/reference/122-operation-summary/job-code-profile-overview.png)和[底部局部](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/reference/122-operation-summary/job-code-update-footer.png)保留原字节。来源观察不构成本仓功能、权限或多修订验收。
