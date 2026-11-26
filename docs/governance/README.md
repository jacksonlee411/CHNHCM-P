# 项目规范导航

从本次要改的业务或页面查起。文档的适用范围、维护角色、审阅及替代关系可在[唯一登记表](registry.json)核对。

| 要解决的问题 | 现行约定 | 对象、实现或查证入口 |
| --- | --- | --- |
| 业务对象是谁，什么条件下可以办理、更正或撤销 | [领域与事实归属](../architecture/004-repository-layout.md) · [各模块业务规格](../specs/README.md) · [术语命名](015-terminology-naming.md) | [人员登记及防重](../specs/m01/501-person-registration-requirements.md#m01-registration-examples) · [任职编号与再聘](../specs/m04/506-job-information-records.md#m04-br-009) |
| 业务日、记录时刻、历史修订和采用来源各指什么 | [日期显示与输入](../adr/026-date-display-and-input-format.md) · [主数据首次新建](../adr/202-master-data-default-effective-date.md) · [按日维护及准确来源](../specs/m02/120-dated-maintenance-technical-contract.md) | [普通维护与专项按日查询](../adr/123-current-maintenance-and-dated-query-boundary.md) · [同日人事办理](../specs/m04/506-job-information-records.md#m04-scn-018) |
| 谁能看、能改哪些字段，授权收窄后如何读取历史、控制导出 | [X03 身份权限与隐私](../architecture/target/modules/x03.md) · [必要审计事实](../adr/128-authorization-evidence-removal.md) · [历史访问](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-history-access) | [人员范围、用途和字段权](../specs/m01/503-person-registration-design.md#m01-registration-access) · [BU 选择与导出](../adr/114-bu-query-selection-and-export.md) |
| 列表、详情、表单的顶栏、控制区、字段和滚动区如何安排 | [012 共享体验](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-expression) · [列表](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-list) · [详情正文](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-detail-sections) · [对象表单](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-object-form) | [职务逐页对照](../specs/m03/308-job-foundation-contract.md#m03-page-inheritance) · [人员名录、详情与对象差异](../specs/m01/502-person-directory-interactions.md#m01-page-map) |
| 筛选何时生效，保存、取消、出错和从原件返回时如何操作 | [查询草稿与应用](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-query-state) · [动作与保存定位](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-actions) · [未保存保护](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-protection) · [历史](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-history) | [人员主兼岗入口与页签](../specs/m04/507-job-information-tab.md#job-tab-layout) · [从人员日志原件返回](../specs/m01/502-person-directory-interactions.md#m01-change-log) |
| 改一个控件，应接哪份共用实现，哪些页面需要一起复验 | [页面继承与业务差异的填法](../templates/dated-object-admission.md#page-inheritance) · [职务已采用的共用布局、交互及差异](../specs/m03/308-job-foundation-contract.md#m03-page-inheritance) | [主题](../../packages/ui/src/theme.ts) · [详情布局](../../packages/ui/src/components/ObjectDetail.tsx) · [共享离开保护](../../apps/web/src/shared/dated/useDraftExit.ts) |
| 人员主动切换页签、从原件后退，各应聚焦哪里 | [012 返回定位](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-link-return) · [502 人员页签与日志](../specs/m01/502-person-directory-interactions.md#m01-change-log) · [507 主岗与兼岗面板](../specs/m04/507-job-information-tab.md#job-tab-layout) | [通用面板 ObjectTabs](../../packages/ui/src/components/ObjectTabs.tsx) · [路由适配 PersonTabs](../../apps/web/src/features/personnel/PersonTabs.tsx) · [人员详情](../../apps/web/src/features/personnel/Detail.tsx) · [任职详情](../../apps/web/src/features/hire/AssignmentPage.tsx) · [焦点复验](../verification/012-desktop-page-acceptance.md#2-用例迁移与复验) |
| 调整领域、数据模型、接口、事件或迁移，应由谁实现 | [模块边界、公开端口与迁移](../architecture/004-repository-layout.md) · [上下文登记](../architecture/contexts.json) · [当前模型与历史保护](../adr/025-greenfield-implementation-policy.md) | [公开 API](../../api/openapi.yaml) · [入职事务及跨域采用](../specs/m04/403-hire-technical-contract.md) |
| 评审该看哪些反例、桌面视口、键盘、缩放或容量证据 | [阶段与完成口径](../adr/003-implementation-stages.md) · [桌面视口与原生缩放](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-viewport-matrix) · [按影响选择验证](137-actions-and-validation.md) | [按日对象接入矩阵](../specs/m02/121-dated-object-admission-matrix.md) · [人员名录容量样本与阈值](../specs/m01/503-person-registration-design.md#m01-capacity) · [验证记录](../verification/README.md) |
| 如何交付候选、更新验收环境和交接运行资源 | [验收环境更新](../adr/003-implementation-stages.md#acceptance-environment-update) · [固定分支与交付](005-branches-and-documents.md) · [并行环境与数据范围](../guides/134-parallel-development-baseline.md) | [本地运行与收尾](../guides/017-local-bu-development.md#local-runtime-cleanup) |
| 规则写在哪里，怎样修订、引用证据和归档 | [分类与权威归属](014-document-management.md#2-分类目录与权威归属) · [新建与修订](014-document-management.md#81-新建及修订) · [归档条件](014-document-management.md#82-归档) · [三类资产](../assets/006-preservation-contract.md) | [文档模板](../templates/README.md) · [资产登记](../assets/inventory.json) · [来源资料](../reference/README.md) |

仍由计划持有的现行条目也可直接定位。BU 与部门异常恢复验收见[301 的验收定义](../plans/301-user-feedback-and-recovery.md#m02-ac-057)及[121 的适用映射](../specs/m02/121-dated-object-admission-matrix.md#21-用户反馈与异常恢复验收)。人员登记、资料权限分别见[500 登记场景](../plans/500-personnel-roadmap.md#m01-scn-001)和[500 权限场景](../plans/500-personnel-roadmap.md#m01-scn-002)。

按对象查条款可从[规格目录](../specs/README.md)进入。查某次交付的实际结果，可按文档号在[登记表](registry.json)的 `governance.verification_records` 中找关联报告，再核对其受测版本、结果和限制。

## 计划与文稿

在[登记表](registry.json)的 `documents` 中按任务查询。下表的链接供走查，完整清单按当前登记筛选。

| 入口 | 筛选方法 | 走查样本 |
| --- | --- | --- |
| 在办计划 | 取 `type=plan` 且 `execution_status=in_progress`。尚待启动的计划另查 `not_started` | [500 人员路线及批次](../plans/500-personnel-roadmap.md#2-批次前置与交付结果) |
| 待审文稿 | 按文档 `status` 查 `draft` 或 `in_review`。计划另读 `execution_status` | [135 对象头接入方案](../plans/135-real-head-product-integration.md) |
| 已关闭批次 | 按 `type=plan` 查执行 `completed` 或 `cancelled`，并读 `closure` 和 `archive` | [510 整改结果](../plans/510-personnel-page-remediation.md#4-实施结果) · [首页历史入口](../index.md#验证与历史批次) |
| 原始验收 | 在 `governance.verification_records` 中按 `related_documents` 查同一编号，按范围选择实施复验，再读逐项结果和限制 | [510 的二十项实测](../verification/510-personnel-page-remediation-implementation.md#1-二十项问题的复验记录) |

例如，从仓库根目录可以直接列出在办计划及其两种状态和正文路径。

```sh
jq -r '.documents[] | select(.type == "plan" and .execution_status == "in_progress") | [.id, .status, .execution_status, .path] | @tsv' docs/governance/registry.json
```
