# 现行规范

按编号、标题或导航中的条款名称查找。检索整份正文时，在仓库根目录运行 `rg -n -F -- '关键词' docs`。

| 编号与正文 | 条款入口 | 部分条款的替代去向 |
| --- | --- | --- |
| [002 继承 CubeBox 技术方向](../../../adr/002-technology-stack.md) | — | — |
| [003 全新实施的阶段与完成口径](../../../adr/003-implementation-stages.md) | [验收环境更新](../../../adr/003-implementation-stages.md#acceptance-environment-update) | — |
| [004 目录、领域及依赖](../../../architecture/004-repository-layout.md) | — | — |
| [005 固定分支及文档编号](../../005-branches-and-documents.md) | — | — |
| [006 三类资产保留与重新验证](../../../assets/006-preservation-contract.md) | — | — |
| [007 首个切片 M02](../../../architecture/007-m02-entry.md) | — | — |
| [009 M02 大陆组织角色与工作关系术语](../../../adr/009-mainland-organization-terms.md) | — | — |
| [011 业务需求与流程说明](../../../specs/m02/011-m02-business-requirements.md) | — | — |
| [012 功能规格与交互设计](../../../specs/m02/012-m02-functional-and-interaction-design.md) | [历史访问](../../../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-history-access) · [012 共享体验](../../../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-expression) · [列表](../../../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-list) · [详情正文](../../../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-detail-sections) · [对象表单](../../../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-object-form) · [查询草稿与应用](../../../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-query-state) · [动作与保存定位](../../../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-actions) · [未保存保护](../../../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-protection) · [历史](../../../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-history) · [012 页签与面板](../../../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-tabs) · [012 返回定位](../../../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-link-return) · [桌面视口与原生缩放](../../../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-viewport-matrix) | [026 统一业务日期显示与输入格式](../../../adr/026-date-display-and-input-format.md) |
| [013 技术详细设计](../../../specs/m02/013-m02-technical-design.md) | — | — |
| [014 文档管理规范](../../014-document-management.md) | [分类与权威归属](../../014-document-management.md#2-分类目录与权威归属) · [新建与修订](../../014-document-management.md#81-新建及修订) · [归档条件](../../014-document-management.md#82-归档) | — |
| [015 术语中文命名规范](../../015-terminology-naming.md) | — | — |
| [017 BU首批本地运行与验证](../../../guides/017-local-bu-development.md) | [本地运行与收尾](../../../guides/017-local-bu-development.md#local-runtime-cleanup) · [017](../../../guides/017-local-bu-development.md#history-import-rebuild) | — |
| [018 生效记录与来源追溯决策](../../../adr/018-effective-records-and-source-history.md) | — | [025 全新实施与当前模型交付策略](../../../adr/025-greenfield-implementation-policy.md) · [123 基础资料维护取消通用查看日期与按日查询边界](../../../adr/123-current-maintenance-and-dated-query-boundary.md) |
| [020 M02 已生效记录删除与生效日期更正](../../../adr/020-effective-record-delete-and-redate.md) | — | [021 M02 多次启停与记录维护规则](../../../adr/021-bu-lifecycle-and-record-maintenance.md) · [022 M02 有限联合修订与历史补全](../../../adr/022-joint-timeline-revision.md) · [025 全新实施与当前模型交付策略](../../../adr/025-greenfield-implementation-policy.md) · [101 M02直接保存与自动表单辅助](../../../adr/101-direct-save-and-form-assistance.md) · 其余替代范围见正文 |
| [021 M02 多次启停与记录维护规则](../../../adr/021-bu-lifecycle-and-record-maintenance.md) | — | [022 M02 有限联合修订与历史补全](../../../adr/022-joint-timeline-revision.md) · [025 全新实施与当前模型交付策略](../../../adr/025-greenfield-implementation-policy.md) · [101 M02直接保存与自动表单辅助](../../../adr/101-direct-save-and-form-assistance.md) · 其余替代范围见正文 |
| [022 M02 有限联合修订与历史补全](../../../adr/022-joint-timeline-revision.md) | — | [024 M02时间轴实现与验证收敛](../../../adr/024-timeline-implementation-simplification.md) · [025 全新实施与当前模型交付策略](../../../adr/025-greenfield-implementation-policy.md) · [101 M02直接保存与自动表单辅助](../../../adr/101-direct-save-and-form-assistance.md) · [107 首条资料维护收敛到通用操作](../../../adr/107-first-record-maintenance.md) · 其余替代范围见正文 |
| [023 M02 时间轴全面测试案例](../../../specs/m02/023-m02-timeline-test-cases.md) | — | — |
| [024 M02时间轴实现与验证收敛](../../../adr/024-timeline-implementation-simplification.md) | — | [025 全新实施与当前模型交付策略](../../../adr/025-greenfield-implementation-policy.md) · 其余替代范围见正文 |
| [025 全新实施与当前模型交付策略](../../../adr/025-greenfield-implementation-policy.md) | — | — |
| [026 统一业务日期显示与输入格式](../../../adr/026-date-display-and-input-format.md) | — | [123 基础资料维护取消通用查看日期与按日查询边界](../../../adr/123-current-maintenance-and-dated-query-boundary.md) |
| [101 M02直接保存与自动表单辅助](../../../adr/101-direct-save-and-form-assistance.md) | — | 见正文的替代说明 |
| [103 按所选位置新增与插入记录](../../../adr/103-positioned-record-addition.md) | — | 见正文的替代说明 |
| [106 M04同日人事动作、日级状态与统计口径](../../../adr/106-same-day-personnel-actions-and-daily-state.md) | — | — |
| [107 首条资料维护收敛到通用操作](../../../adr/107-first-record-maintenance.md) | — | 见正文的替代说明 |
| [110 BU列表交互收敛与DataGrid Pro采用](../../../adr/110-bu-list-and-datagrid.md) | — | [113 BU列表改用Community并保留既定规则](../../../adr/113-bu-list-community-adoption.md) · [114 BU全查询结果选择与导出决定](../../../adr/114-bu-query-selection-and-export.md) · [123 基础资料维护取消通用查看日期与按日查询边界](../../../adr/123-current-maintenance-and-dated-query-boundary.md) |
| [113 BU列表改用Community并保留既定规则](../../../adr/113-bu-list-community-adoption.md) | — | [114 BU全查询结果选择与导出决定](../../../adr/114-bu-query-selection-and-export.md) · [123 基础资料维护取消通用查看日期与按日查询边界](../../../adr/123-current-maintenance-and-dated-query-boundary.md) |
| [114 BU全查询结果选择与导出决定](../../../adr/114-bu-query-selection-and-export.md) | — | [123 基础资料维护取消通用查看日期与按日查询边界](../../../adr/123-current-maintenance-and-dated-query-boundary.md) |
| [117 M02共用按日维护与部门边界](../../../adr/117-m02-dated-maintenance-and-department-boundary.md) | — | [123 基础资料维护取消通用查看日期与按日查询边界](../../../adr/123-current-maintenance-and-dated-query-boundary.md) |
| [118 M02部门业务规则与场景](../../../specs/m02/118-department-business-requirements.md) | — | — |
| [119 M02部门功能与交互](../../../specs/m02/119-department-functional-design.md) | — | — |
| [120 M02共用维护与部门技术接入合同](../../../specs/m02/120-dated-maintenance-technical-contract.md) | [120 企业约束](../../../specs/m02/120-dated-maintenance-technical-contract.md#enterprise-dated-contract) | — |
| [121 M02双对象接入验证矩阵](../../../specs/m02/121-dated-object-admission-matrix.md) | [121 逐项映射](../../../specs/m02/121-dated-object-admission-matrix.md#21-用户反馈与异常恢复验收) | — |
| [123 基础资料维护取消通用查看日期与按日查询边界](../../../adr/123-current-maintenance-and-dated-query-boundary.md) | — | — |
| [124 按日维护对象接入及容量验证指南](../../../guides/124-dated-object-admission-and-capacity.md) | [124 测量口径](../../../guides/124-dated-object-admission-and-capacity.md#bu-department-capacity-comparison) | — |
| [128 授权过程与关联跳转过度设计整改](../../../adr/128-authorization-evidence-removal.md) | — | — |
| [134 并行开发共同基线与独立环境](../../../guides/134-parallel-development-baseline.md) | — | — |
| [137 GitHub Actions使用与验证规范](../../137-actions-and-validation.md) | — | — |
| [201 国家/地区目录、行政区划目录与企业地点维护技术合同](../../../specs/m02/201-country-region-geo-area-and-location-maintenance-contract.md) | — | — |
| [202 配置与主数据默认生效日期](../../../adr/202-master-data-default-effective-date.md) | — | — |
| [207 法律实体基础维护合同](../../../specs/m02/207-legal-entity-design.md) | [法律实体基础验收](../../../specs/m02/207-legal-entity-design.md#legal-entity-l1-acceptance) · [主体用途与真实采用](../../../specs/m02/207-legal-entity-design.md#legal-entity-purpose-handoff) | — |
| [308 职类、职务系列与职务基础合同](../../../specs/m03/308-job-foundation-contract.md) | [职务逐页对照](../../../specs/m03/308-job-foundation-contract.md#m03-page-inheritance) · [308 容量分组](../../../specs/m03/308-job-foundation-contract.md#job-capacity-matrix) | — |
| [310 职级与职级阶梯对象接入合同](../../../specs/m03/310-grade-ladder-contract.md) | — | — |
| [312 岗位定义与岗位层级对象接入合同](../../../specs/m03/312-position-contract.md) | — | — |
| [313 编制管理合同](../../../specs/m03/313-position-capacity-contract.md) | — | — |
| [401 直接入职业务规则与验收](../../../specs/m04/401-direct-hire-requirements.md) | [直接与未来入职](../../../specs/m04/401-direct-hire-requirements.md#m04-scn-001) · [录用转入职](../../../specs/m04/401-direct-hire-requirements.md#m04-scn-005) | — |
| [402 入职准备与办理页面合同](../../../specs/m04/402-hire-interactions.md) | — | — |
| [403 入职来源采用与事务技术合同](../../../specs/m04/403-hire-technical-contract.md) | — | — |
| [501 人员登记与名录业务规格](../../../specs/m01/501-person-registration-requirements.md) | [人员登记及防重](../../../specs/m01/501-person-registration-requirements.md#m01-registration-examples) · [登记与编号](../../../specs/m01/501-person-registration-requirements.md#m01-scn-001) · [个人资料与权限](../../../specs/m01/501-person-registration-requirements.md#m01-scn-002) | — |
| [502 人员登记与名录页面交互规格](../../../specs/m01/502-person-directory-interactions.md) | [人员名录、详情与对象差异](../../../specs/m01/502-person-directory-interactions.md#m01-page-map) · [从人员日志原件返回](../../../specs/m01/502-person-directory-interactions.md#m01-change-log) | — |
| [503 人员登记与名录技术合同](../../../specs/m01/503-person-registration-design.md) | [人员范围、用途和字段权](../../../specs/m01/503-person-registration-design.md#m01-registration-access) · [人员名录容量样本与阈值](../../../specs/m01/503-person-registration-design.md#m01-capacity) | — |
| [505 M01 个人资料九栏目维护合同](../../../specs/m01/505-personal-profile-contract.md) | — | — |
| [506 M04 职务信息与任职记录业务规则](../../../specs/m04/506-job-information-records.md) | [任职编号与再聘](../../../specs/m04/506-job-information-records.md#m04-br-009) · [同日人事办理](../../../specs/m04/506-job-information-records.md#m04-scn-018) · [兼岗与人员计数](../../../specs/m04/506-job-information-records.md#m04-scn-002) · [调动、离职与再聘](../../../specs/m04/506-job-information-records.md#m04-scn-003) · [同日动作与经理](../../../specs/m04/506-job-information-records.md#m04-scn-006) | — |
| [507 人员详情职务信息与人事事务办理记录页面设计](../../../specs/m04/507-job-information-tab.md) | [人员主兼岗入口与页签](../../../specs/m04/507-job-information-tab.md#job-tab-layout) | — |
| [512 工作关系合同业务规格](../../../specs/m04/512-employment-contracts.md) | [合同续订](../../../specs/m04/512-employment-contracts.md#m04-scn-004) | — |
