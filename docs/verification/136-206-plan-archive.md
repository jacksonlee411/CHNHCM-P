# CHNHCM-136、CHNHCM-206 计划关闭与归档交接

## 1 关闭范围

2026-11-26，按用户要求在 `wt-dev-f@7f71e22fba934481fb9b00dbe02298e381a9c258` 复核并关闭两份计划。136 的节流工作流、发布及首轮用量观察已有交付记录，后续托管分钟试点移交 137。206 的基础维护和直接入职所需的主体用途、真实采用已有产品依据，后续用途由 207 指定的维护角色在接入时验证。两份计划的文档采纳状态均保留 `active` 及原审阅，执行状态登记为 `completed`，正文按原编号移入原目录对应的归档位置。[136 原计划](../archive/plans/136-actions-cost-reduction.md) · [206 原计划](../archive/plans/206-legal-entity-implementation.md) · [归档规则](../governance/014-document-management.md#82-归档)

<a id="actions-closeout"></a>

## 2 GitHub Actions 节流与托管分钟试点

136 第 1 至 4、6 至 11 节的 T01–T04 有本地实现和反例记录。T05 的分片及准备成本由验证维护者按实际覆盖、用时和等待比较。T06 的首批发布已在 PR #19 验证最终门禁、main 复用及 4390 固定环境，原观察窗口至 #28 收口。以下保留 V01 至 V11 各自的适用范围，后续修订从 137 进入。[首批实现](136-actions-implementation.md) · [发布与固定交接](136-actions-publication.md) · [原验收条件](../archive/plans/136-actions-cost-reduction.md#42-实施必须完成的验收)

| 原条件 | 现行条款及复核时机 |
| --- | --- |
| V01、V05、V06 | 修改事件、范围或候选身份时，核对草稿转 ready、文档任务明确不适用、不同 PR 相互独立及最终受测候选。手动诊断和旧候选各留自己的结果。[正式触发](../governance/137-actions-and-validation.md#3-提交同步与触发) · [同一 PR 后续候选](../governance/137-actions-and-validation.md#14-同一-pr-的后续全量候选) |
| V02、V03 | 修改分类或消费者映射时，对账完整差异、重命名、删除、未知项及多业务消费者，分别核对 DB/API 和浏览器覆盖。[影响范围与清单](../governance/137-actions-and-validation.md#4-按影响选择验证) |
| V04、V10 | 修改汇总、分支保护或集成流程时，以失败、取消、缺失及非预期跳过核对三项正式门禁。main 必需验证失败由集成维护者暂停新集成。[范围与门禁](../governance/137-actions-and-validation.md#42-范围清单与门禁) · [完整回归](../governance/137-actions-and-validation.md#52-全量的必要场景) |
| V07、V08 | 137 已将合并后等价复用扩展到获准的全量、限定及文档 PR。核对实际合并树、覆盖及来源，证明不足时恢复相应检查。环境故障按同一运行有界补验，业务断言、换候选按各自规则处理。首批未演练的远程事件和故障在相关变更或真实故障时留证。[合并后复用](../governance/137-actions-and-validation.md#51-同一合并候选的范围复用) · [失败与重跑](../governance/137-actions-and-validation.md#6-失败与重跑) |
| V09、V11 | 分片变化时核对用例并集、必要依赖、环境隔离及回收。按周核对片时、总用量、等待及失败。原 4/6/8 分支示例是成本推演，正常交付按实际成员和范围计量。[分片与资源](../governance/137-actions-and-validation.md#7-分片缓存与运行资源) · [用量记录](../governance/137-actions-and-validation.md#8-用量记录验收与维护) |

原窗口包含 10 次交付。按固定账本重新求和，6 次业务交付为 4,327 个 runner 分钟，4 次策略发布为 1,366 分钟。6 次中只有 1 次可与同类历史整次交付配对，原 25%–33% 整体降幅的结论为证据不足。11 月 26 日的全量运行有 5 个 DB/API 片和 16 个浏览器片，21 片各在 15 分钟内完成，25 个已分配作业合计 268 托管分钟。这是单次运行的覆盖和片时证据。[原窗口核算](136-actions-observation.md#6-窗口收口) · [21 片实际用时](136-hosted-minutes-pilot.md#71-2026-11-26-对超出-15-分钟的分片复核)

原计划第 12 至 14 节的试点、前置和同 PR 复用由 137 第 12 至 14 节承接。80% 试点仍待取得目标配置下的正常交付数据。wt-dev-e 的 CI 实施及验证维护者按获准投放逐次归集，集成维护者核对门禁和复用。目标配置首次承接正常业务交付后，到 10 次有效交付或 7 天先到者，在既有试点记录追加结果。对照五次原业务交付的 3,713 分钟，均值为 742.6，目标上限为每次平均 148.52 个托管分钟。试点记录继续以 `draft` 登记，归档时未产生新的远程运行、账单或降本结论。[137 试点责任与窗口](../governance/137-actions-and-validation.md#hosted-minutes-observation) · [接续台账](136-hosted-minutes-pilot.md#4-投放与验收)

<a id="legal-entity-closeout"></a>

## 3 法律实体基础、用途与真实采用

| 206 原范围 | 验收、现行规则与后续责任 |
| --- | --- |
| 第 1 至 5 节的基础身份、日期、国家采用、授权与恢复 | L1-T00 至 T05 已在原批次签收。207 第 1 至 6 节持有对象与全套维护规则，120、121、124 持有共享接入、日期与容量口径。原批报告保留数据库、API 和菜单的受测范围、缺陷及复验。[207 对象合同](../specs/m02/207-legal-entity-design.md) · [L1 实施记录](206-legal-entity-implementation.md) |
| 第 6 节的 L1-V01 至 V15 | 原 15 项编号逐项进入 207 第 6.1 节。V01–V08 验身份、权限、期间及国家采用，V09–V12 验查询、选择、恢复及真实存储，V13–V15 验菜单、共用回归和独立容量。V13 的历史视口按原报告保存，今后按 012 的现行视口规范验收。[206 原矩阵](../archive/plans/206-legal-entity-implementation.md#6-验收与关闭条件) · [207 现行矩阵](../specs/m02/207-legal-entity-design.md#legal-entity-l1-acceptance) |
| 第 6、7 节的容量限制 | 206-D01 至 D03 的 P95 时延曾超标。125 后续使用 1,000 个背景主体和另 20 个各有 20 条记录的主体，实测 8 组、800 次请求，均达到法律实体的原目标。地点 D08、区划 D09 和 125 其余容量 P1 仍按各自范围接续。[独立容量口径](../guides/124-dated-object-admission-and-capacity.md#6-法律实体接入实例) · [限定复测原件](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/125-target-adoption-and-current-validation.md#34-法律实体当前限定补验) |
| 第 2、7 节的 L2 与 L3 | 直接入职所需的主体用途、全期资格及必要 BU／地点绑定已在原法律实体菜单验收。M04 实际保存首单验证两种竞争顺序。先收窄资格，原入职请求返回 409。先提交入职，与既有安排冲突的收窄返回 409，原关系与采用继续有效。[L2 维护与供给](400-hire-simplification.md#t02-results) · [L3 双连接实测](400-first-hire-integration.md#first-job-only-hire) |

206 原先以三阶段分别关闭为整份计划的出口。本次按用户要求以已交付范围关闭。其余真实设立单位、委托或派遣角色，以及计薪、扣缴、参缴、付款和成本用途，分别交给 M02 与对应消费域，在首次有明确使用场景时按 207 第 7 节核验依据、期间、权限和历史。固定环境由原责任方在下次更新或用户验收结束时复查。[各用途接续责任](../specs/m02/207-legal-entity-design.md#legal-entity-purpose-handoff)

## 4 状态与本次复核

136 的 CI 结果、试点观察和 206 的产品验收由各次验证者按受测证据判断。文档维护者按 014 保存采纳状态、原审阅、执行关闭原因及归档路径。计划中的分批放行和固定环境交接继续按 003、137 与原报告核对。[文档状态](../governance/014-document-management.md#6-生命周期与完成口径) · [实施完成与证据](../adr/003-implementation-stages.md#4-完成与证据规则)

本次在独立 PostgreSQL 测试容器和真实 API 上重跑四项已有用例，均通过，合计 76.711 秒。L1 复验基础维护、成对操作与字段权限，L2 复验入职准备，L3 复验真实入职与资格收窄的双向保护。浏览器、容量和固定环境结果沿用各原报告的受测范围。

```sh
go test -count=1 -tags=integration -timeout=8m \
  -run '^(TestLegalEntity206Core|TestLegalEntity206PairsAndPermissions|TestHire400Readiness|TestFirstHireAndPurposeProtection)$' \
  ./internal/bootstrap
```

与归档前的基线逐份比对了 299 份既有验证记录及其登记。两份定稿报告和已归档的 500 计划各调整一处链接目的地。136 和 206 原有的 27 个标题全部保留。206 原引用的容量报告在本仓摘录中仅有来源说明，已核对固定提交中的原第 3.4 节并将链接指向该节。现行入口逐项对齐 15 项法律实体验收条件和 11 项 CI 验收条件，已关闭并物理归档的计划从 45 份增至 47 份，执行状态仍为 `in_progress` 的计划为 125、140、400。

`make docs` 的 185 项回归及仓库文档检查通过，8 项 CI 用量脚本回归通过。`make check` 和 `git diff --check` 通过。
