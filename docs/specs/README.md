# 业务与功能规格

按模块查业务事实、操作方式和技术合同。跨模块的布局、交互、共用组件及验收方式可从[项目规范导航](../governance/README.md)查找。[文档首页](../index.md) · [检索现行规范](../governance/catalog/standards/README.md)

## M01 人员登记、名录与个人资料

[登记、编号及权限](m01/501-person-registration-requirements.md) · [人员名录、资料面板及日志返回](m01/502-person-directory-interactions.md#m01-change-log) · [跨域端口及容量](m01/503-person-registration-design.md) · [个人资料九栏目](m01/505-personal-profile-contract.md)

## X03 本地账号与授权

028 拟定首位客户管理员的引导与恢复、可委派范围和逐份授予规则，并以部门与单岗位核编给出正常菜单、例外处理和验收样例。当前处于文档评审阶段。[028 客户管理员合同](x03/028-local-account-and-authorization.md) · [027 分阶段实施计划](../plans/027-x03-local-access-rollout.md)

## M03职务基础

| 办理对象 | 本模块合同 | 本模块引用的共享条款 |
| --- | --- | --- |
| 职务系列、职务 | [308 职务基础](m03/308-job-foundation-contract.md) · [逐页继承对照](m03/308-job-foundation-contract.md#m03-page-inheritance) · [字段与办理差异](m03/308-job-foundation-contract.md#m03-page-differences) | [012 共享体验](m02/012-m02-functional-and-interaction-design.md#m02-ui-expression) · [对象表单](m02/012-m02-functional-and-interaction-design.md#m02-ui-object-form) · [查询草稿与应用](m02/012-m02-functional-and-interaction-design.md#m02-ui-query-state) · [动作与保存定位](m02/012-m02-functional-and-interaction-design.md#m02-ui-actions) · [变更日志](m02/012-m02-functional-and-interaction-design.md#m02-ui-change-log-table) · [未保存保护](m02/012-m02-functional-and-interaction-design.md#m02-ui-protection) · [026 日期显示与输入](../adr/026-date-display-and-input-format.md#2-决策) |
| 职类目录、职务双维分类 | [308 职类与双维分类](m03/308-job-foundation-contract.md#m03-job-function) · [职类菜单、查找与导出](m03/308-job-foundation-contract.md#m03-br-032) · [沿用的职务系列页面](m03/308-job-foundation-contract.md#m03-page-inheritance) | [012 对象表单](m02/012-m02-functional-and-interaction-design.md#m02-ui-object-form) · [121 页面标准继承与 M03 接入映射](m02/121-dated-object-admission-matrix.md#24-页面标准继承与m03接入映射) |
| 职级、职级阶梯与职务可用职级 | [310 职级与阶梯](m03/310-grade-ladder-contract.md) · [逐页继承](m03/310-grade-ladder-contract.md#m03-br-023) · [成员顺序与集合维护差异](m03/310-grade-ladder-contract.md#82-有依据的业务差异) | [012 共享体验](m02/012-m02-functional-and-interaction-design.md#m02-ui-expression) · [列表](m02/012-m02-functional-and-interaction-design.md#m02-ui-list) · [动作](m02/012-m02-functional-and-interaction-design.md#m02-ui-actions) · [历史](m02/012-m02-functional-and-interaction-design.md#m02-ui-history) · [026 日期显示与输入](../adr/026-date-display-and-input-format.md#2-决策) · [121 按日维护共用规则](m02/121-dated-object-admission-matrix.md#2-共用规则矩阵) |
| 岗位、岗位上下级 | [312 岗位定义与层级](m03/312-position-contract.md) · [页面接入与岗位差异](m03/312-position-contract.md#4-页面接入) · [岗位工作区与管理汇报入口](m03/312-position-contract.md#6-岗位工作区接续) | [012 列表](m02/012-m02-functional-and-interaction-design.md#m02-ui-list) · [动作与保存定位](m02/012-m02-functional-and-interaction-design.md#m02-ui-actions) · [历史来源](m02/012-m02-functional-and-interaction-design.md#m02-ui-history) · [层级页面的共享体验](m02/012-m02-functional-and-interaction-design.md#m02-ui-expression) · [121 按日维护共用规则](m02/121-dated-object-admission-matrix.md#2-共用规则矩阵) |
| 岗位核编、部门编制分配 | [313 编制管理](m03/313-position-capacity-contract.md) · [核编日期与历史](m03/313-position-capacity-contract.md#m03-br-026) · [部门分配与同日调剂](m03/313-position-capacity-contract.md#m03-br-040) · [岗位核编与部门统计入口](m03/313-position-capacity-contract.md#5-正常入口与应用合同) | [012 共享体验](m02/012-m02-functional-and-interaction-design.md#m02-ui-expression) · [123 专项按日查询准入](../adr/123-current-maintenance-and-dated-query-boundary.md#25-专项按日查询准入) · [026 记录时刻与历史证据](../adr/026-date-display-and-input-format.md#23-记录时刻与历史证据) · [121 页面接入要求](m02/121-dated-object-admission-matrix.md#24-页面标准继承与m03接入映射) |

上述页面共用 [012 桌面视口与原生缩放验收条款](m02/012-m02-functional-and-interaction-design.md#m02-ui-viewport-matrix)。

## M02当前规格

[业务单元规则、场景及验收](m02/011-m02-business-requirements.md) · [共享体验、表单与布局](m02/012-m02-functional-and-interaction-design.md#m02-ui-expression) · [桌面视口与缩放](m02/012-m02-functional-and-interaction-design.md#m02-ui-viewport-matrix) · [技术设计](m02/013-m02-technical-design.md) · [时间轴案例](m02/023-m02-timeline-test-cases.md)

## M02部门及共用接入合同

[部门规则](m02/118-department-business-requirements.md) · [部门交互](m02/119-department-functional-design.md) · [共用日期、历史、选择和权限](m02/120-dated-maintenance-technical-contract.md) · [按日对象接入](m02/121-dated-object-admission-matrix.md) · [异常恢复逐项验收](m02/121-dated-object-admission-matrix.md#21-用户反馈与异常恢复验收) · [国家/地区目录、行政区划目录与企业地点维护技术合同](m02/201-country-region-geo-area-and-location-maintenance-contract.md) · [法律实体基础验收](m02/207-legal-entity-design.md#legal-entity-l1-acceptance) · [主体用途与真实采用](m02/207-legal-entity-design.md#legal-entity-purpose-handoff)

## M04 入职、任职与人事事务

[直接入职与未来撤销](m04/401-direct-hire-requirements.md#m04-scn-001) · [已接受录用转入职](m04/401-direct-hire-requirements.md#m04-scn-005) · [工作关系合同](m04/512-employment-contracts.md) · [准备与办理页面](m04/402-hire-interactions.md) · [来源采用、权限与事务](m04/403-hire-technical-contract.md) · [职务记录、主兼岗及再聘](m04/506-job-information-records.md) · [任职入口与页签](m04/507-job-information-tab.md#job-tab-layout)

## M09 薪酬体系配置设计

[薪酬体系配置业务方案与制度样例](m09/211-compensation-configuration.md) · [后台页面与交互设计](m09/212-compensation-configuration-pages.md) · [可点击桌面原型](m09/prototype/compensation-configuration.html) · [设计及实施路线图](../archive/plans/210-compensation-configuration-roadmap.md)

## M09 员工首次定薪与按日调薪

[个人基本工资办理与页面](m09/215-assignment-base-pay.md) · [准确采用、权限和并发合同](m09/216-assignment-base-pay-contract.md) · [二期实施计划](../archive/plans/214-assignment-base-pay-rollout.md)

## 实施与验证去向

[在办计划](../governance/catalog/README.md#catalog-in-progress) · [待启动计划](../governance/catalog/README.md#catalog-not-started) · [待审文稿](../governance/catalog/README.md#catalog-review) · [按关联号查方案评审和实施验收](../governance/catalog/README.md#catalog-evidence)

## 历史编写记录

[已关闭计划及保存位置](../governance/catalog/README.md#catalog-closed) · [历史规格及替代关系](../governance/catalog/README.md#catalog-history) · [原始登记与审阅](../governance/registry.json)
