# CHNHCM-511 项目规范实施、归档评估与验收

## 1 范围与判定依据

2026-11-26 在 `wt-dev-f` 以已采纳的 `ea8e1e7f40091886796b82c5080bd3823ed010c0` 为起点核验本批工作区。基线存量取 511 方案所列 `04fb43be9d34745195659e1cf28c8530cf13f6ce`。Codex 主代理承担修订和复核，分别从方案作者和审阅者的入口走查，未另行组织独立人工签收。方案采纳、治理实施、历史产品验收分别查[方案采纳记录](511-project-standards-acceptance.md)、[本批方案](../archive/plans/511-project-standards.md)和各原批报告。

| 方案任务 | 交付物和判定 |
| --- | --- |
| 入口和模板 | 首页按查规范、查计划、查验收、查历史分流，规格按模块定位。登记生成全量清单，目录漂移进入 `make docs`。计划、功能规格、验证模板用相同条款、差异、实现与观察结果交接。[首页](../index.md) · [分类目录](../governance/catalog/README.md) · [模板](../templates/README.md) |
| 文档状态规则 | 014 第 6 节列明七种文档状态的触发、判定角色及证据。归档前从原文逐项提炼业务和工程规范，另查有无文档状态规则，记录适用对象、前后状态、触发、角色及证据。仍有效条款无获采纳的去向时暂缓，已否决事项和原验收结果保留。[014 第 8.2 节](../governance/014-document-management.md#82-归档) |
| 职务视口冲突 | 从 012 与 M03-AC-028 各自排测，普通桌面均为 1440×900、1440×540 的 100%。AC-029 对关键操作要求原生 200%，400% 按风险或客户要求。A 的修改保存／取消、R 的原件读取／返回以及权限专项保持。复查已采纳规格中的“五种／五组 100% 视口”等旧措辞后，012 的替代说明和 308 §13.1 的原 28 单元均保留历史用途。[012 视口](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-viewport-matrix) · [308 对象 AC](../specs/m03/308-job-foundation-contract.md#m03-ac-028) · [原矩阵](../specs/m03/308-job-foundation-contract.md#131-尺寸和角色矩阵) |
| 详情页签及真实变更 | 012 承接通用面板与编辑草稿，502、507 承接人员跨路由焦点和主兼岗差异。新增按唯一人员编号办理的双角色回归，并用键盘从日志打开原件再后退。实际用例、角色和边界见第 5 节。[012 页签](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-tabs) · [人员](../specs/m01/502-person-directory-interactions.md#m01-change-log) · [任职](../specs/m04/507-job-information-tab.md#m04-ac-016) |

本次登记只更新表内涉及的实质修订范围。各合同前次审阅仍可按本批起点与原证据核对。[011](200-geography-plan.md) · [012](012-desktop-page-acceptance.md) · [013](200-geography-plan.md) · [014](011-effective-date-terminology.md) · [118](200-geography-plan.md) · [119](200-geography-plan.md) · [120](125-path-stage-target-adoption.md) · [121](308-job-page-admission-contract.md) · [308](308-job-function-contract-review.md) · [500](500-job-information-delivery-review.md) · [502](507-personnel-action-register-design-review.md) · [507](500-job-information-delivery-review.md)

## 2 首页检索与反例

| 查询 | 方案作者从首页直达 | 审阅者从首页进入规范导航后的落点 |
| --- | --- | --- |
| 人员主兼岗入口 | 当前规则 → 人员主兼岗入口 | 筛选与返回 → [507 任职入口](../specs/m04/507-job-information-tab.md#job-tab-layout) |
| 普通页面视口 | 当前规则 → 普通页面桌面视口及原生缩放 | 评审视口 → [012 现行矩阵](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-viewport-matrix) |
| 计划归档条件 | 查历史计划或办理归档 → 归档前逐项提炼与状态判定 | 文档治理 → [014 归档](../governance/014-document-management.md#82-归档) |

三组路径分别为一跳和两跳，两种入口均到同一文件的同一锚点。在 [登记分类](../governance/catalog/README.md) 中核对以下正反例。

- 500 在办，510、135 执行已关闭。510 从关闭清单到第 4 节，由[原实施复验](510-personnel-page-remediation-implementation.md#1-二十项问题的复验记录)核对 20 项实际结果。
- 135 在待审及已关闭两处出现。原 `in_review / completed`、空整稿审阅和关闭理由可查。[017](../guides/017-local-bu-development.md#19-保留模型5及20条正式场景)持有模型 5；[135 原决定](../archive/plans/135-real-head-product-integration.md#8-用户最终采用决定)保留模型 6 的退出。
- 141、126 均可在已关闭清单找到。待审清单排除 126，现行预检从 [137](../governance/137-actions-and-validation.md#13-减少正式候选的全量启动次数)进入。
- 以 304 §2 企业原初始化声明为反例。只沿共用按日合同无法判定其专属覆盖规则，应由维护者补齐后复查。原位保留、整份替代和物理归档分别用 008、301、138 正向核对。

<a id="plan-dispositions"></a>

## 3 基线 43 份已完成计划逐份交接

按登记的 `type=plan / execution_status=completed` 从方案基线取样。逐份查正文、表格、决定、验收条件、原审阅、执行关闭条件和引用方。下表列原条款、仍适用的接收文档、一次性结果或未结事项的去向。每行末数为本次实施开始时直接引用该原路径的定稿报告份数，逐份去重。原报告的编号和采用范围可从登记表及原文回查。301 的方案评审与后来执行分别记录。

6 份移入 2026 年计划归档目录，31 份原位保存（其中 301 整份替代），6 份暂缓。后者仍显示原有执行关闭状态，触发条件见各行。盘点时，32 份计划有定稿报告直接引用，比 511 初评多涉及 135。新增实施报告自身及分类目录不计入这项原路径统计。511 本批另行关闭并原位保存，以供三份方案审阅记录查回原条款。交付后登记共有 44 份已关闭计划。

涉及文稿状态的通则覆盖 001/008 的形成、预留转草稿，010 等计划的验收与执行关闭，135 的单项采用及 301 的整份替代。作者形成正文时将 `planned` 转为 `draft`，送审进入 `in_review`，审阅者据实际结论采纳或退回。执行者据任务关闭原因登记 `completed`，验收者按受测基线逐项记录。负责维护者完成逐条交接、核验引用及原报告后登记移动。其余各表中的部署、来源包发布、业务启停、测试结果和实验模型是各自对象的状态。逐份查证后沿用 014 生命周期。[文档状态、触发和证据](../governance/014-document-management.md#6-生命周期与完成口径)

### 3.1 物理归档（6 份）

| 原编号与正文 | 原文位置、仍适用条款、接收方或复查条件 | 代表证据 · 原路径引用数 |
| --- | --- | --- |
| <a id="plan-131"></a>[131 部门列表与父候选查询减负计划](../archive/plans/131-department-query-reduction.md) | §2—6 的查询完整性、单语句捕获、父候选、同输入 A/B 归 120 §3.4—3.8，验收和容量边界归 121 §14—17。125 按用户原决定继续持有余项，原失败可查。 [120](../specs/m02/120-dated-maintenance-technical-contract.md#34-131查询完整性原型与接入约束) · [121](../specs/m02/121-dated-object-admission-matrix.md#17-131-t04最终构建验证与容量边界) · [125](../plans/125-r6-capacity-remediation.md) | [原记录](131-t04-capacity-and-handoff.md) · 0 |
| <a id="plan-138"></a>[138 行政区划目录交互实施方案](../archive/plans/138-geo-area-interactions.md) | §1—5 目录查询、直接下级和逐级返回归 201 §6.3，澳门层级及来源准入归 201 的 BR201/202，共用列表归 012。可选联动与原固定包按原范围查。 [201](../specs/m02/201-country-region-geo-area-and-location-maintenance-contract.md#63-g3查询候选与页面合同) · [201](../specs/m02/201-country-region-geo-area-and-location-maintenance-contract.md#m02-br-201) · [201](../specs/m02/201-country-region-geo-area-and-location-maintenance-contract.md#m02-br-202) · [012](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-list) | [原记录](138-geo-area-interactions.md) · 0 |
| <a id="plan-203"></a>[203 G3必要行政区划接入实施方案](../archive/plans/203-g3-geo-area-admission.md) | §2、6 的 G3 分层与 V01—12 是目录首批输入。来源装载、候选资格、准确路径、授权与性能按 201 的 BR201—211 和验收矩阵，G4/G5 已有接续，非首批能力仍由 201 定界。 [201](../specs/m02/201-country-region-geo-area-and-location-maintenance-contract.md#m02-br-201) · [201](../specs/m02/201-country-region-geo-area-and-location-maintenance-contract.md#m02-br-210) · [201](../specs/m02/201-country-region-geo-area-and-location-maintenance-contract.md#geography-acceptance) · [200](../archive/plans/200-geography-and-location-roadmap.md) | [原记录](203-g3-implementation.md) · 0 |
| <a id="plan-204"></a>[204 G4地点纵向交付实施方案](../archive/plans/204-g4-location-implementation.md) | §2.2、6 的 G4-V01—18 是地点当批检查。地点地址、采用及准确来源归 201 §6.4、7.5、9，按日共用接入归 120/121；G5 已交付，余下容量归 200、125。 [201](../specs/m02/201-country-region-geo-area-and-location-maintenance-contract.md#geography-acceptance) · [201](../specs/m02/201-country-region-geo-area-and-location-maintenance-contract.md#75-g4地点授权和准确地址投影) · [120](../specs/m02/120-dated-maintenance-technical-contract.md) · [121](../specs/m02/121-dated-object-admission-matrix.md) · [125](../plans/125-r6-capacity-remediation.md) | [原记录](204-g4-implementation.md) · 0 |
| <a id="plan-205"></a>[205 G5地理联合验收与固定运行交付](../archive/plans/205-g5-geography-acceptance-and-runtime.md) | §2—6 为 G5 联合实测和 4383 运行交接。AC200—211 归 201，固定环境维护归 017 §22，D08/D09 保留 200 的责任与输入，旧 20 会话未达项可查。 [201](../specs/m02/201-country-region-geo-area-and-location-maintenance-contract.md#geography-acceptance) · [017](../guides/017-local-bu-development.md#22-g5首批交付与4383地点运行) · [200](../archive/plans/200-geography-and-location-roadmap.md) · [125](../plans/125-r6-capacity-remediation.md) | [原记录](205-g5-implementation.md) · 0 |
| <a id="plan-306"></a>[306 部门路径精简与案例集团根部门补全](../archive/plans/306-department-path-simplification.md) | §1—6 所有对外路径只含真实部门、授权裁剪和独立状态归 118/119/120，真实案例固定库运行归 017 §30。原补录操作及保护计数保留在验证记录。 [118](../specs/m02/118-department-business-requirements.md#m02-br-028) · [119](../specs/m02/119-department-functional-design.md) · [120](../specs/m02/120-dated-maintenance-technical-contract.md) · [017](../guides/017-local-bu-development.md#30-306部门路径及真实集团案例) | [原记录](306-department-path-simplification.md) · 0 |

### 3.2 原位保存和整份替代（31 份）

| 原编号与正文 | 原文位置、仍适用条款、接收方或复查条件 | 代表证据 · 原路径引用数 |
| --- | --- | --- |
| <a id="plan-001"></a>[001 本地建仓交付记录](../archive/plans/001-repository-bootstrap.md) | §2—4 建仓选择、工具及交付口径。技术栈、目录、编号、资产分类分别由 002、004、005、006 持有，阶段验收归 003。目录及原样本的建仓事实保存原报告。 [002](../adr/002-technology-stack.md) · [003](../adr/003-implementation-stages.md) · [004](../architecture/004-repository-layout.md) · [005](../governance/005-branches-and-documents.md) · [006](../assets/006-preservation-contract.md) | [原记录](bootstrap.md) · 1 |
| <a id="plan-008"></a>[008 下一步：M02 四份详细文档](../archive/plans/008-next-steps.md) | §1—5 四份文稿的依赖与交付结果留作编写史。阶段、来源、预留转草稿和有据采纳分别由 003、006、014 承接，产品范围与规则归 010—013。两份定稿报告仍使用原路径。 [003](../adr/003-implementation-stages.md) · [006](../assets/006-preservation-contract.md) · [014](../governance/014-document-management.md) · [010](../archive/plans/010-m02-scope-and-plan.md) · [011](../specs/m02/011-m02-business-requirements.md) · [012](../specs/m02/012-m02-functional-and-interaction-design.md) · [013](../specs/m02/013-m02-technical-design.md) | [原记录](014-numbered-format-migration.md) · 2 |
| <a id="plan-010"></a>[010 本版实施范围与计划](../archive/plans/010-m02-scope-and-plan.md) | §2、4、6 的 BU 首批范围与 241 整例、部分案例及退出项是原批结论。业务、页面、技术和逐例预期归 011—013、023，后续消费者按原任务接续。 [011](../specs/m02/011-m02-business-requirements.md) · [012](../specs/m02/012-m02-functional-and-interaction-design.md) · [013](../specs/m02/013-m02-technical-design.md) · [023](../specs/m02/023-m02-timeline-test-cases.md) | [原记录](010-t07-acceptance.md) · 30 |
| <a id="plan-019"></a>[019 CubeBox共享体验迁移与页面对比计划](../archive/plans/019-cubebox-experience-migration.md) | §2—6 十组体验映射、创建与查询为当批范围。资产来源归 006，布局、详情与前端接线归 012/013；原图面仍按原批条件查。 [006](../assets/006-preservation-contract.md) · [012](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-expression) · [013](../specs/m02/013-m02-technical-design.md) | [原记录](019-cubebox-experience-migration.md) · 1 |
| <a id="plan-100"></a>[100 页面体验优化实施计划](../archive/plans/100-ui-experience-improvement.md) | §4、7 布局、滚动、键盘与缩放归 012，历史七组结果保留在原报告。后续维护与日志按 011/013。 [012](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-viewport-matrix) · [011](../specs/m02/011-m02-business-requirements.md) · [013](../specs/m02/013-m02-technical-design.md) | [原记录](100-ui-experience-improvement.md) · 5 |
| <a id="plan-102"></a>[102 M02维护日志与专用表单体验修订](../archive/plans/102-bu-maintenance-usability.md) | §2、4、6 的无变化、返回、日志及专用紧凑表单归 101、012，六项专项是本批结果。 [101](../adr/101-direct-save-and-form-assistance.md) · [012](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-object-form) | [原记录](102-bu-maintenance-usability.md) · 1 |
| <a id="plan-104"></a>[104 统一更正记录入口实施](../archive/plans/104-unified-record-correction.md) | §2、6 的字段联合更正与权限归 011 的 AC031，参数案例归 023。原 116 个浏览器实例的结果仅覆盖当批参数。 [011](../specs/m02/011-m02-business-requirements.md#m02-ac-031) · [023](../specs/m02/023-m02-timeline-test-cases.md) | [原记录](104-unified-record-correction.md) · 1 |
| <a id="plan-105"></a>[105 位置新增插入实施与更正覆盖补全](../archive/plans/105-positioned-addition-and-coverage.md) | §2、6 的所选位置插入、默认、后继及撤权归 103、011 的 AC030，补例与结果保持原输入。 [103](../adr/103-positioned-record-addition.md) · [011](../specs/m02/011-m02-business-requirements.md#m02-ac-030) · [023](../specs/m02/023-m02-timeline-test-cases.md) | [原记录](105-positioned-addition-and-coverage.md) · 2 |
| <a id="plan-108"></a>[108 M02维护改造多轮一致性复核](../archive/plans/108-maintenance-consistency-review.md) | §2—7 是四轮一致性抽查，长期的单条维护、首条、旧来源与界面判据分别由 107、011、012 及 023 承接。首败与分轮通过随报告保留。 [107](../adr/107-first-record-maintenance.md) · [011](../specs/m02/011-m02-business-requirements.md) · [012](../specs/m02/012-m02-functional-and-interaction-design.md) · [023](../specs/m02/023-m02-timeline-test-cases.md) | [原记录](108-maintenance-consistency-review.md) · 1 |
| <a id="plan-109"></a>[109 启停期间专用维护与通用联合退出](../archive/plans/109-lifecycle-pair-maintenance.md) | §2—3、6 限定两项固定启停期间操作并退出通用联合编排。规则与反例归 011 的 BR019、013 和 023，原批通用旧入口不得成为新依据。 [011](../specs/m02/011-m02-business-requirements.md#m02-br-019) · [013](../specs/m02/013-m02-technical-design.md) · [023](../specs/m02/023-m02-timeline-test-cases.md) | [原记录](109-lifecycle-pair-maintenance.md) · 1 |
| <a id="plan-111"></a>[111 BU列表体验改造实施计划](../archive/plans/111-bu-list-experience-upgrade.md) | §6、8—9 是列表迁移、单列与交付历史。现行 Community 接线归 113，查询归 011、012；原 Pro 探索及许可边界依旧可查。 [113](../adr/113-bu-list-community-adoption.md) · [011](../specs/m02/011-m02-business-requirements.md) · [012](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-list) | [原记录](111-community-t01-t02.md) · 13 |
| <a id="plan-115"></a>[115 BU全查询结果选择与导出实施计划](../archive/plans/115-bu-query-selection-and-export.md) | §2、6 全查询固定选择、排除和导出失败零文件归 114、011 的 BR022/AC032。运行事实按原样本回查。 [114](../adr/114-bu-query-selection-and-export.md) · [011](../specs/m02/011-m02-business-requirements.md#m02-br-022) · [011](../specs/m02/011-m02-business-requirements.md#m02-ac-032) | [原记录](115-query-selection-implementation.md) · 2 |
| <a id="plan-116"></a>[116 M02共用按日维护实现路线图](../archive/plans/116-m02-shared-dated-maintenance-roadmap.md) | §3—7 双对象阶段和 R0—R6 是本批历史；现行接入、授权、历史及矩阵归 117—121。用户结项决定将跨批容量与未达项交 125。 [117](../adr/117-m02-dated-maintenance-and-department-boundary.md) · [118](../specs/m02/118-department-business-requirements.md) · [119](../specs/m02/119-department-functional-design.md) · [120](../specs/m02/120-dated-maintenance-technical-contract.md) · [121](../specs/m02/121-dated-object-admission-matrix.md) · [125](../plans/125-r6-capacity-remediation.md) | [原记录](116-r6-standards-and-capacity.md) · 19 |
| <a id="plan-122"></a>[122 BU资料操作摘要与变更日志表格化实施计划](../archive/plans/122-bu-operation-summary-and-change-log.md) | §3、6 的摘要、分组日志、筛选及来源归 011 的 BR035、012 的操作摘要和 013。原三批产品和后续定点复验各存原报告。 [011](../specs/m02/011-m02-business-requirements.md#m02-br-035) · [012](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-operation-summary) · [013](../specs/m02/013-m02-technical-design.md) | [原记录](122-operation-summary-implementation.md) · 6 |
| <a id="plan-126"></a>[126 完整历史紧凑存储研究与实施](../archive/plans/126-compact-history-storage-research.md) | §1、3—6 的强类型历史、守卫与全新模型归 013、120、025。取消 200 MB 硬门槛的决定和空间、延迟首败保留；容量 P1 继续由 125 负责。 [013](../specs/m02/013-m02-technical-design.md) · [120](../specs/m02/120-dated-maintenance-technical-contract.md) · [025](../adr/025-greenfield-implementation-policy.md) · [125](../plans/125-r6-capacity-remediation.md) | [原记录](126-storage-feasibility.md) · 1 |
| <a id="plan-133"></a>[133 过度防御与高成本处理整改路线图](../archive/plans/133-overdefensive-processing-remediation.md) | §2—6 的必要守卫和去过度设计分别归 025、128、120；132 仍在审。用户关闭路线后，未达成本与容量交 125，原实验和消息审阅继续可查。 [025](../adr/025-greenfield-implementation-policy.md) · [128](../adr/128-authorization-evidence-removal.md) · [120](../specs/m02/120-dated-maintenance-technical-contract.md) · [125](../plans/125-r6-capacity-remediation.md) | [原记录](133-performance-continuation.md) · 2 |
| <a id="plan-135"></a>[135 真实对象头产品接入与验收计划](../archive/plans/135-real-head-product-integration.md) | §6—8 仅模型 5、20 条正式场景有独立采纳依据，017 §19 持有长期规则。模型 6 退出、万条场景撤销和原失败保留。in_review / completed、空整稿审阅及原关闭原因照原登记。 [017](../guides/017-local-bu-development.md#19-保留模型5及20条正式场景) · [014](../governance/014-document-management.md) | [原记录](135-model5-and-20-record-adoption.md) · 2 |
| <a id="plan-139"></a>[139 岗位列表、岗位关系与结构视图实施方案](../archive/plans/139-position-workspace.md) | §2—6 按部门找岗、管理上级与历史归 312；H01—H14 属原批。资产台账以该路径及原受评文本指纹定位来源，原位保留以便校验。 [312](../specs/m03/312-position-contract.md) · [012](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-list) | [原记录](139-position-workspace-implementation.md) · 0 |
| <a id="plan-141"></a>[141 本地预检结果与生成物核对优化计划](../archive/plans/141-local-preflight.md) | §1—3 三项本地预检、输入与生成物核对归 137 §1.3，运行归 017。旧首错与复验留原报告。 [137](../governance/137-actions-and-validation.md#13-减少正式候选的全量启动次数) · [017](../guides/017-local-bu-development.md) | [原记录](141-local-preflight.md) · 1 |
| <a id="plan-200"></a>[200 国家地区行政区划与地点实施路线图](../archive/plans/200-geography-and-location-roadmap.md) | §3—7 的阶段及正反例追踪首批交付，目录、来源、地点与地址归 201，默认日期归 202。D08/D09 待相同输入复测，原路线保留责任和证据。 [201](../specs/m02/201-country-region-geo-area-and-location-maintenance-contract.md#geography-acceptance) · [202](../adr/202-master-data-default-effective-date.md) · [125](../plans/125-r6-capacity-remediation.md) | [原记录](205-g5-implementation.md) · 2 |
| <a id="plan-301"></a>[301 用户提示与异常恢复整改方案](../archive/plans/301-user-feedback-and-recovery.md) | §3 的恢复规则及 §6.1 全部 17 项均有已采纳后继。按整份替代登记，保留原执行关闭日期和原因。定稿报告依赖原稿旧标题，两个 6.1 入口分别定位原评审与实施。 [011](../specs/m02/011-m02-business-requirements.md#m02-feedback-acceptance) · [118](../specs/m02/118-department-business-requirements.md#m02-department-feedback-acceptance) · [012](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-feedback-recovery) · [013](../specs/m02/013-m02-technical-design.md#m02-feedback-errors) · [119](../specs/m02/119-department-functional-design.md) · [120](../specs/m02/120-dated-maintenance-technical-contract.md) · [114](../adr/114-bu-query-selection-and-export.md) · [121](../specs/m02/121-dated-object-admission-matrix.md) | [原记录](301-user-feedback-plan-review.md) · 1 |
| <a id="plan-302"></a>[302 详情对象导航位置与响应整改](../archive/plans/302-detail-navigation.md) | §2—4 的位置、总数、同查询与局部失败归 012 的列表及详情，部门归 119；原 45300、双角色和计时仅作交付记录。 [012](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-list) · [119](../specs/m02/119-department-functional-design.md) | [原记录](302-detail-navigation.md) · 1 |
| <a id="plan-303"></a>[303 部门一级菜单调整](../archive/plans/303-department-primary-navigation.md) | §1—4 的一级部门菜单、导航、角色可见性归 119、012，部署过程及原 15 表核对留原报告。 [119](../specs/m02/119-department-functional-design.md) · [012](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-expression) | [原记录](303-department-primary-navigation.md) · 1 |
| <a id="plan-309"></a>[309 M03职级与职级阶梯实施方案](../archive/plans/309-m03-grade-ladder.md) | §2—6 的职级、职级阶梯与使用资格归 310；G1/G2 分别保存实施结果、原条件性跳过和固定环境信息。 [310](../specs/m03/310-grade-ladder-contract.md) · [017](../guides/017-local-bu-development.md) | [原记录](309-grade-g2-implementation.md) · 4 |
| <a id="plan-311"></a>[311 M03岗位定义与岗位层级实施方案](../archive/plans/311-m03-positions.md) | §1—5 的岗位定义、层级及读写权限归 312，占编归 313；两批签收分别回查原执行报告。 [312](../specs/m03/312-position-contract.md) · [313](../specs/m03/313-position-capacity-contract.md) | [原记录](311-position-hierarchy-signoff.md) · 1 |
| <a id="plan-314"></a>[314 M03 职类实施批次路线图](../archive/plans/314-m03-job-function-roadmap.md) | §2—5 的职类和双维映射归 308 §14；跨职类编制按 313，原 AC090 限定结果及后续联验分别保留于 314、315。 [308](../specs/m03/308-job-foundation-contract.md#14-独立职类与职务双维分类) · [313](../specs/m03/313-position-capacity-contract.md) | [原记录](314-job-function-t05-acceptance.md) · 5 |
| <a id="plan-315"></a>[315 部门与岗位编制后台实施计划](../archive/plans/315-staffing-quota-administration.md) | §3—6 的岗位编制、部门额度、占用与并发归 313；固定验收环境的交接归 003 及原报告第 6 节，原联验可回查。 [313](../specs/m03/313-position-capacity-contract.md) · [003](../adr/003-implementation-stages.md#acceptance-environment-update) | [原记录](315-department-quota-allocation.md) · 4 |
| <a id="plan-504"></a>[504 M01 个人资料扩展实施方案](../archive/plans/504-personal-profile-extension.md) | §2—5 九栏目、脱敏、弹框和整页保存归 505，名录与容量归 502、503。后续真实主兼岗交 500 T04，原各批样本仍可查。 [505](../specs/m01/505-personal-profile-contract.md#profile-maintenance) · [502](../specs/m01/502-person-directory-interactions.md) · [503](../specs/m01/503-person-registration-design.md#m01-capacity) · [500](../archive/plans/500-personnel-roadmap.md) | [原记录](504-personal-profile-fourth-batch.md) · 6 |
| <a id="plan-508"></a>[508 人员职务信息评审问题修复实施方案](../archive/plans/508-job-information-remediation.md) | §2—5 关系级转正、同日先后、原件和办理列表归 106、506、507，升级和六项原结果留在固定环境交接。 [106](../adr/106-same-day-personnel-actions-and-daily-state.md) · [506](../specs/m04/506-job-information-records.md) · [507](../specs/m04/507-job-information-tab.md) | [原记录](508-job-information-implementation.md) · 3 |
| <a id="plan-509"></a>[509 在职变动、结束与再聘复评问题修复方案](../archive/plans/509-t04-remediation.md) | §3—7 退出、再聘、转正和同日原件归 506，权限、办理与列表归 507，占用归 313，27 项修复及六项回归存原报告。 [506](../specs/m04/506-job-information-records.md#m04-br-009) · [507](../specs/m04/507-job-information-tab.md) · [313](../specs/m03/313-position-capacity-contract.md) | [原记录](509-t04-remediation-implementation.md) · 2 |
| <a id="plan-510"></a>[510 人员页面二十项问题整改与验收](../archive/plans/510-personnel-page-remediation.md) | §1—4 的查询与页面判据归 012、501—503、507。五种 100% 视口是原批条件，现行见 012 矩阵。二十项、两套容量与固定环境验收留原记录。 [012](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-viewport-matrix) · [501](../specs/m01/501-person-registration-requirements.md) · [502](../specs/m01/502-person-directory-interactions.md) · [503](../specs/m01/503-person-registration-design.md) · [507](../specs/m04/507-job-information-tab.md#job-tab-layout) | [原记录](510-personnel-page-remediation-implementation.md) · 2 |

<a id="archive-deferred"></a>

### 3.3 暂缓归档（6 份）

| 原编号与正文 | 原文位置、仍适用条款、接收方或复查条件 | 代表证据 · 原路径引用数 |
| --- | --- | --- |
| <a id="plan-127"></a>[127 万科全量组织及历史验证](../archive/plans/127-vanke-full-history-verification.md) | §7—8 的历史导入映射、重建分层和固定库并发授权有环境专用条件。普通运行和原始资料保护已有指南，M02 数据与固定环境维护者在下次重建或替换该导入器时逐条核对并固化专用规则。 [017](../guides/017-local-bu-development.md) · [120](../specs/m02/120-dated-maintenance-technical-contract.md) · [134](../guides/134-parallel-development-baseline.md) · [125](../plans/125-r6-capacity-remediation.md) | [原记录](127-vanke-full-history.md) · 0 |
| <a id="plan-129"></a>[129 业务单元去过度设计与部门合理实现对齐计划](../archive/plans/129-bu-simplification-and-department-alignment.md) | §3 六项对齐规则已分别进入 011、013、120；§6.2 的 V15 同输入测量和原环境约束仍供 125 回交。由组织域及容量维护者在 V15 首次正式复测时核对通用阈值与仅当批的诊断条件。 [011](../specs/m02/011-m02-business-requirements.md) · [013](../specs/m02/013-m02-technical-design.md) · [120](../specs/m02/120-dated-maintenance-technical-contract.md) · [125](../plans/125-r6-capacity-remediation.md) | [原记录](129-bu-simplification-plan.md) · 0 |
| <a id="plan-130"></a>[130 部门存储去重与索引收敛优化方案](../archive/plans/130-department-storage-optimization.md) | §3、6 的部门关系去重、索引收敛与保护归 120、121。两种形状增长及成本对照仍是 125 复测输入，由组织存储与容量维护者在下次完整基线结算时分别接收或退出旧阈值。 [120](../specs/m02/120-dated-maintenance-technical-contract.md) · [121](../specs/m02/121-dated-object-admission-matrix.md) · [125](../plans/125-r6-capacity-remediation.md) | [原记录](130-implementation.md) · 0 |
| <a id="plan-300"></a>[300 部门列表左侧部门树实施计划](../archive/plans/300-department-tree-implementation.md) | §2、6 的树范围与固定选择归 118、119、120；原 60 组计时、明显未达的 9 组与共用余项仍由 121/125 指回。组织与容量维护者在 125 复测对应矩阵时核对各组去向。 [118](../specs/m02/118-department-business-requirements.md#m02-br-037) · [119](../specs/m02/119-department-functional-design.md) · [120](../specs/m02/120-dated-maintenance-technical-contract.md) · [121](../specs/m02/121-dated-object-admission-matrix.md) · [125](../plans/125-r6-capacity-remediation.md) | [原记录](300-closeout.md) · 1 |
| <a id="plan-304"></a>[304 企业完整按日维护](../archive/plans/304-enterprise-maintenance.md) | §2.1—2.2 的唯一企业初始化声明、期间覆盖和企业守卫具有对象专属约束。120、124 覆盖共用日期；M02 企业维护者在下次企业合同修订时补齐专属判断并核对原初始化反例。 [120](../specs/m02/120-dated-maintenance-technical-contract.md) · [124](../guides/124-dated-object-admission-and-capacity.md) · [202](../adr/202-master-data-default-effective-date.md) | [原记录](304-enterprise-maintenance.md) · 0 |
| <a id="plan-307"></a>[307 M03职务基础实施方案](../archive/plans/307-m03-job-foundation.md) | §2、6 的职务系列和职务规则归 308，原批容量分组仍被后续合同引用。M03 与容量维护者在下一次对应负载测量时接收或标明各组退出条件。 [308](../specs/m03/308-job-foundation-contract.md) · [124](../guides/124-dated-object-admission-and-capacity.md) | [原记录](307-m03-job-foundation-implementation.md) · 2 |

物理归档的六份文稿也分别核过状态条款。131 的执行关闭依据是用户将性能余项移交 125 的明确决定。138 的目录版次描述来源数据，203 的目录发布描述装载校验，204 的办理状态描述地点的生效与维护，它们的文稿采纳与计划关闭均以各自审阅及执行证据判定。205 的固定环境交付须同时核对 G5 联验、未达项去向和运行责任。306 的案例完工须核对两套案例及原来源保护。六份均沿用原采纳、审阅和执行关闭依据，本次新增的是保存位置、日期及逐条交接。131 原交接报告只有私有版本保留 §3.6、4.1，已按固定 Git 对象核验两个标题并把原章节链接接回。

## 4 两份归档试点与在办场景

**008。** 四份文稿的编写顺序已履行。来源增量由原台账持有，阶段、正式模板及状态判定分别归 003、014。原文追加去向并保持原位，核对 010 原范围与 014 格式迁移两份定稿报告均可打开对应内容。[008 交接](../archive/plans/008-next-steps.md#5-交付与后续交接)

**301。** 按旧 ID 比对触发、角色和每个必须观察到的结果。057—062、065—073 共 15 项归 011，063—064 两项归 118。121 第 21 节分别链接 17 项。保存确认、失败分区和权限恢复归 012/119，Go 错误分类与传输归 013/120，BU 固定选择归 114。301 整份登记为 `superseded / completed`，后继均已采纳，原执行关闭日期 2026-11-14 和原理由保留，物理归档字段为空。[逐项入口](../specs/m02/121-dated-object-admission-matrix.md#21-用户反馈与异常恢复验收) · [历史定义](../archive/plans/301-user-feedback-and-recovery.md#61-实施验收)

按固定 Git 对象 `061d28105966bd10eb1e3b0032cf2976a6448de4` 读取原仓 `draft-3.json` 的文本，核对轮次、SHA-256、同名章节和该节 17 项。该稿的「6.1 待执行验收」与原方案评审“17 项未运行”相符。旧片段回到原稿定位说明，新片段仍抵达实施验收。原稿摘要及获权入口见[受评历史位置](../archive/plans/301-user-feedback-and-recovery.md#61-待执行验收)，查后来结果用[实施报告](301-user-feedback-implementation.md)。

**500。** M01 两项登记、权限以及 M04 六项入职、兼岗、再聘、合同、招聘交接、同日动作，仍从原 8 个锚点读取。501 的登记与权限调用保持。500 第 5 节已逐项明确接收合同；劳动合同和招聘交接分别待 T05、T06 的事实边界落实，500 继续在办。[场景及交接](../archive/plans/500-personnel-roadmap.md#5-验收样本与关闭条件)

## 5 人员、共享组件与职务消费者的实际验证

产品代码采用本工作树基线，在一次性 PostgreSQL、API 与浏览器环境从正常菜单运行。固定构建用于产品页面，开发夹具单独运行。验收从条款、对象实现、真实操作回查，旧 510 的数值与 308 原 28 单元只用于定位历史。浏览器回归源可复跑。[人员名录用例](../../tests/e2e/first-hire.spec.ts) · [日志用例](../../tests/e2e/personnel.spec.ts) · [通用编辑夹具用例](../../tests/e2e/experience.spec.ts) · [职务两对象](../../tests/e2e/staffing308.spec.ts) · [职务原生缩放](../../tests/e2e/staffing308-zoom.spec.ts)

| 条款与操作 | 实际断言 |
| --- | --- |
| 501、502、507 从「人员」按唯一编号办理 | 经办人先见 3 份任职、1 人，只看兼岗得 2 份、1 人。键盘打开 2 号、返回后编号、条件和 2 号链接焦点恢复。另开资料读者会话，同号全部类型得一条人员行、零任职，进入可读个人资料。名录行与授权结果使用真实 API。[名录页面](../../apps/web/src/features/hire/EmployeeDirectory.tsx) |
| 012/502 主动切签与从原件后退 | 方向键移动焦点，Enter 或空格激活变更日志。唯一可见面板和页签双向关联，主动切签后已选页签聚焦。用 Enter 从筛选后的日志第 2 页打开「查看当次资料」再后退，原页数据及触发链接焦点恢复，链接纵向位置差最多 2 CSS px，再前进、后退可重复。[路由页签](../../apps/web/src/features/personnel/PersonTabs.tsx) · [人员详情](../../apps/web/src/features/personnel/Detail.tsx) |
| 012/507 主岗、兼岗与角色 | 两个 100% 桌面视口和关键路径原生 200% 上，主岗在当前授权内切换各面板，兼岗独立显示职务信息，原查询可返回；两角色人员页另验证短屏末项、菜单、表单及日志。[任职详情](../../apps/web/src/features/hire/AssignmentPage.tsx) |
| 012 通用面板与编辑夹具 | 1440×540 上切换面板保留输入及展开状态，错误摘要选中相应页签、展开隐藏分组并定位字段，短屏可滚动。实际职务消费者另查职务与职务系列。[通用面板](../../packages/ui/src/components/ObjectTabs.tsx) · [编辑夹具](../../apps/web/test-fixtures/experience.tsx) · [职务详情](../../apps/web/src/features/staffing/Detail.tsx) |
| 308 对象验收与原批 | 职务系列、职务 × 管理员、查询者 × V01、V02 共 8 个桌面组合，另按受影响路径测原生缩放。管理员与读者的操作断言随角色执行。原 28 个单元的构建及历史结果保持在原位置。[原批结果](../specs/m03/308-job-foundation-contract.md#131-尺寸和角色矩阵) |

## 6 检查与交接

完整分类以本次登记重新生成。校验覆盖现行 AC 的唯一定义、旧报告与原锚点、在办场景、三个任务的双入口、六份归档文件的路径和逐份处置。现场运行及检查结果如下。

- 三个独立环境分别通过 24、1、10 项浏览器执行，合计 35 次，按项目与用例标题去重为 29 项。两轮产品包含相同的 5 项企业维护前置，人员日志在加入 Enter 打开原件的断言后重跑。三轮均无跳过、重试或意外失败。人员与职务的关键路径实测原生 200%，本批没有触发 400% 专项。三个临时环境均自动回收，固定手工环境继续由原责任方维护。[逐项结果、时间和复跑参数](data/511-project-standards-implementation/browser-results.json)
- 核验 43 个原编号与逐份结论完全对应，六份归档各自保有原状态、关闭证据、审阅、锚点和反向交接。比对 301 的 17 条原始场景及结果，和 011/118 的现行条款逐字一致；121 逐项指向对应的唯一定义。抽查 131 时发现两个旧片段在公开版摘录中没有正文，已核实固定版本并接回原章节。
- 分类目录逐组与登记反查。现行标准 54 份、在办计划 6 份、待启动 0 份、待审文稿 2 份、执行已关闭计划 44 份、其他历史文档 2 份、预留 0 份，另有 297 份验证记录。135 同时满足待审与执行已关闭两个条件。此前 296 份报告及其历史登记均已核对保留。
- 首轮文档检查定位到 012 新增 §2.4.2 排在 §3 标题之后。调整章节位置后复核了两级目录、旧锚点和引用，并重跑文档门禁。目录生成检查已接入仓库统一检查器，已有报告标题改变时，回归夹具验证过期目录会被检出。
- 本批运行 `make check`，覆盖文档检查、Go 依赖与 SQL 边界、运行工具测试、生成物核对、Go 测试与 vet、前端类型检查及补丁空白检查。所改产品场景走真实 API、数据库与 Chromium。历史容量及专属企业、导入合同的未结项由第 3.3 节逐份接续。[检查入口](../../Makefile) · [文档检查器](../../scripts/check-repository.py)
