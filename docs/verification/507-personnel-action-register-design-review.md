# CHNHCM-507 人事事务办理列表设计复核

## 1 范围与受审基线

2026-11-24，Codex 在 wt-dev-f 的 `56189a0e59a1a8f2f35c34f5ae2d8b8e408ca37a` 及已有职务信息文稿上复核本次修订。受审范围是 502 的菜单、506 的办理计数、507 的列表与原件返回、403 的查询及 500 的交付次序。此前的主兼岗和首单复核分别保留原记录。[原职务信息复核](506-job-information-design-review.md) · [按职务首单的实测](400-first-hire-integration.md#decision-first-hire) · [登记表](../governance/registry.json)

基线的「人员」路由有员工列表、个人详情和本人入职页。OpenAPI 有按办理身份读取原件和撤销未来入职的接口。现有首单历史查询按客户和人员读取。T04 增加跨人员集合查询，并将列表行关联到原办理。[现行路由](../../apps/web/src/app/App.tsx) · [现行接口](../../api/openapi.yaml) · [首单读取与撤销](../../internal/contexts/staffing/infra/postgres/hire_history.go) · [T04 查询合同](../specs/m04/403-hire-technical-contract.md#personnel-action-query)

## 2 设计走查

| 业务问题 | 复核结论 |
| --- | --- |
| 部门月底核查，怎样找到调走前发生的晋升 | 周宁调动当日的 0 号主岗在产品部。以发生部门查晋升归研发部，查调动可分别指定调出研发部或调入产品部，均定位原调动。[办理列表与字段](../specs/m04/507-job-information-tab.md#personnel-action-register) |
| 三份任职、两名员工、两个日期条件怎样计数 | 周宁三份任职中的转正引用同一工作关系办理。周宁同日三笔加陈禾地点更改共四笔。陈禾的办理时间落在 2026-12-03，生效日落在 2026-12-04。[两人四笔样本](../specs/m04/506-job-information-records.md#m04-scn-017) |
| 两个入口维护后，如何知道哪笔发生了变化 | 两端读取同一办理身份。有效更正在原笔产生修订，原件仍可追溯。只获部分任职及明确办理读权者，两端的行、数量、部门候选和深链均用当前范围。列表返回恢复原条件和原行焦点。[双入口验收](../specs/m04/507-job-information-tab.md#m04-ac-017) |
| 办理、历史和待办如何分工 | 新事项可在职务信息直接保存，跨员工核查从办理记录进入。申请审批或工作分派产生的待办按 M20 的后续交付处理。[记录操作](../specs/m04/506-job-information-records.md#m04-br-010) · [M20 业务待办](../architecture/target/modules/m20.md) |

## 3 验证与交接

按两个员工的事项、任职、发生部门、生效日及首次登记时刻独立复算，四笔、三笔、关系转正一笔、受限读者一笔的预期均相符。核对当前工作区 14 份新增或改写文稿中的 185 处仓内链接及锚点，5 个未跟踪文件的空白和末行通过。`make docs` 的 180 项回归和仓库资料检查、`make boundaries` 及 `git diff --check` 均通过。

跨人员办理记录及两个入口的联验列入 T04。真实数据库、API、受限角色菜单、返回焦点及原生缩放待产品实施后按相应批次验收。[T04 分批交付](../archive/plans/500-personnel-roadmap.md#job-information-delivery)
