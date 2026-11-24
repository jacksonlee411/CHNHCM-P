# CHNHCM-315 岗位核编实施计划架构复核

## 1 复核范围与结论

2026-11-22 复核 313 的后台实施计划。313 持有业务合同，315 持有开发顺序、接口和验收分工。本次将岗位核编与获权部门汇总列为可独立交付的一步。部门上限和同日多岗调剂列为第二步。修订稿附六栏任务表，逐行列明写入方、直接复用的实现、最小增量和验收批次。[315 修订稿](../plans/315-staffing-quota-administration.md) · [任务表](../plans/315-staffing-quota-administration.md#6-实施任务表) · [313 合同](../specs/m03/313-position-capacity-contract.md)

| 复核问题 | 对修订稿的核对结果 |
| --- | --- |
| 无人员和招聘时能否独验 | 已写入管理员与只读者的正常菜单，以及未核编、核定 0、用量未知和字段受限各自的页面与 API 结果。部分可见时只展示获权小计。 |
| 只调一个岗位的核定数时谁要写 | 现有 Record 含职类、职级和岗位引用，Unit 含层级动作。计划新增额度类型与窄事务，只写额度来源、变更、Outbox 和回执。岗位定义、职务与管理汇报版本列为对照项。 |
| 规则和接口归属是否明确 | 保存由 Go 重验全部受影响日期段、引用、现权及已接管的用量。PostgreSQL 负责锁、现行区间及原子提交。前端使用各动作的 OpenAPI 类型，详情与确认采用 Go 的最后适用日。 |
| 用量未知与竞争如何验 | 两人首次核编和原号查回列入首批。后续用量按来源和期间确认，改日按低额度实际延长的期间核对，停岗检查免占编任职。最后一编的两种抢锁顺序分别给出原需求、任职及预留应保留的状态。 |
| 跨对象如何对账 | AC090 的岗位独验与两批并存分开记结果。并存样本先核编，再只更正旧职务的职类，比较三次快照。另有管理员和只读者走查同日更正、独立后继及旧版本的样本。每项分别列明 Go、数据库/API 与浏览器证据。 |

AC090 的交接依据在本树可核对到 308 合同、314 路线图及 T03 实施记录。所引 `314-job-function-t05-acceptance.md` 在本地工作树、本仓已知提交和核对时的远端 wt-dev-b 分支快照中均未找到。“仅 308”的原验收结果仍归 314，315 据合同承接其余两行，实施时按实际候选登记。[308 AC090](../specs/m03/308-job-foundation-contract.md#m03-ac-090) · [314 交接条件](../plans/314-m03-job-function-roadmap.md#5-验收环境关闭与接续) · [本树 T03 记录](314-job-function-t03-upgrade.md)

## 2 复算与实际检查

固定工作树为 wt-dev-d，HEAD 为 `6bc37020e436274cbb7244a55db85efdb8bc74f1`。开工时暂存区和 stash 为空，工作区已有上一轮的 24 份文档及附件。上一轮 315 计划的 SHA-256 为 `7a3bde604fa88b3bd445367bd70af77d27e8f1cdc4f53c9812436c67ab3ba366`。历史方案复核继续保留其当时的交付口径。[上一轮复核](315-staffing-quota-administration-review.md)

- 用合成额度分别按三个业务日取值。更正后依次为 2、3、4，四月原来源为 1。独占终点 2027-06-01 的最后适用日为 2027-05-31。另一组把 2027-04-01 增额移至 2027-04-20，原低额多覆盖 19 天，样本中两份占编的 5 天从 2027-04-10 开始。获权范围分别复得 4 与 3，两种串行争用顺序的获胜方各取得 1 编。部门与岗位的既有复算程序亦通过。[部门算例](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/315-staffing-quota-administration/recalculate.py)
- Go 1.26.8 实际执行并通过 `TestStaffing307IndependentSuccessors`、`TestPositionUseChangesAreDatedAndFieldSpecific` 和 `TestPositionFutureJobGradeBoundary`。其证据覆盖已有独立后继、岗位按日引用及未来资格变化。新额度写入、双连接竞争与真实菜单均在实施批验证。[原领域测试](../../internal/contexts/staffing/domain/timeline_test.go) · [原岗位测试](../../internal/contexts/staffing/domain/position_test.go)
- 修订前后 `make docs` 的 128 项测试均通过。修订前覆盖 417 份 Markdown、13,109 条本地链接，修订后覆盖 418 份、13,152 条。计划与本记录的 64 处链接另逐一核对目标文件和指定章节。七行任务均有完整六栏，架构生成检查核对 57 份文本一致，候选差异通过空白检查。

## 3 单列的存量事项

| 既有范围 | 实施边界及复查责任 |
| --- | --- |
| 目录对象的 Record、Unit 同时承载多种资料和层级发布 | 核编通过专用类型接入。若后续拆分既有类型，由原 Staffing 目录或岗位层级批次验证其原调用方。[记录](../../internal/contexts/staffing/domain/timeline.go) · [事务接口](../../internal/contexts/staffing/application/ports.go) |
| 现有目录前端用一套输入拼动作路径，再断言为各对象请求 | 编制用生成的动作类型。既有六类目录调用的整理交原页面批次核对各动作。[现有发送器](../../apps/web/src/features/staffing/api.ts) |
| 既有组织详情和部分维护表单使用不同的期间表述 | 组织详情已有最后适用日，部分维护表单显示独占终点“之前”。该处由 M02 独立复核。315 直接验编制详情、确认的最后适用日一致。[组织详情](../../apps/web/src/features/organization/DatedPages.tsx) · [组织维护表单](../../apps/web/src/features/organization/TimelineEditor.tsx) |

315 的计划状态与产品结果分别登记。实际数据库升级与 API、两个管理员首次写、响应丢失查回及浏览器两角色签收均待实施。[岗位独立签收条件](../plans/315-staffing-quota-administration.md#5-验证与交接)
