# CHNHCM-116 R3前端流程合同增量评审

登记信息：[登记表](../governance/registry.json) `governance.verification_records`。
适用范围：R3-A共用流程及R3-B BU闭环；不含部门运行时、R4—R6或生产验收。
上游依据：[116](../archive/plans/116-m02-shared-dated-maintenance-roadmap.md)、[120](../specs/m02/120-dated-maintenance-technical-contract.md)、[121](../specs/m02/121-dated-object-admission-matrix.md)。

## 1 验证对象与受测基线

2026-11-05，CHNHCM wt-dev-b，R2已本地提交2c8b9c898938cafcbedb9f6250b1c308673a3280。提交前原日志及补丁仅做保留原字节的压缩包装，依据见R2 commit-packaging.json；R2产品代码及测试结果未改。R3起始工作区干净。合同增量覆盖116、120、013、121、023及导航，不改变011/012业务和页面行为。

## 2 环境与前置条件

主代理对照源码、合同及既有案例进行范围评审，非独立评审或人工验收。此次范围评审只读代码及修订文档；运行资源由实施报告记录。原2728份验证文件保存摘要以保护历史证据。

## 3 验证过程与结果

| 检查 | 前置与操作 | 预期与实际 | 结论 | 证据 |
| --- | --- | --- | --- | --- |
| 范围与批准 | 对照用户完成R3指令、116阶段表 | 共用流程、类型适配及BU真实接入；R4—R6未启动 | 通过 | 116第4.4节 |
| 类型与字段边界 | 对照BU API、现有表单、120 | 共用层只接强类型端口；BU字段/能力/DTO/路由及投影留适配器；无部门空端点 | 通过 | 120第4.3节 |
| 提交身份与恢复 | 对照三种提交表单及既有恢复测试 | 原BU三元引用和键保留；kind隔离事件；会话/挂载/提交归属及原内容重试统一 | 通过 | 120第5.2节 |
| 页面、列表及历史 | 对照122/123和114/115 | 共用离开、读取、列表核对/导航；原字段、直接保存、日志、来源、选择和CSV语义不变 | 通过 | 023第28节 |
| 证据与关闭 | 对照121和116退出门槛 | 机制测试不冒充部门产品；BU完整DB/浏览器及视口独立留证 | 通过 | 121第8节 |
| 文档检查 | make docs | 首轮发现报告results尚未填；补齐实际结果后73项脚本及236份Markdown通过 | 通过 | [复验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3/contract-docs-retest.txt) |

## 4 失败、修复与复验

源代码复核识别到详情和前后对象导航尚未统一使用会话代次守卫；R3在共用读取流程中补齐。该项是实施保护范围，不将源码观察写为已发生生产故障。提交/类型初检及实际回归的失败、修复分别由实施报告记录。

## 5 未覆盖范围与限制

本记录只关闭R3合同范围，不替代类型检查、机制测试、真实BU菜单和数据库回归。不修改R0/R1/R2/122/123历史报告；部门39项继续not_run，无第二真实对象接入或人工业务签收结论。

## 6 结论与后续事项

语义范围复核及文档门禁通过。按R3-A、R3-B逐批交付，阶段完成只在实际退出证据齐备后登记。

## 7 证据保存

[基线](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3/baseline.json)、[历史保护摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3/protected-before.json)及[实施记录](116-r3-frontend-workflow.md)。
