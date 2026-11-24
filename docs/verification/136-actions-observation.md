# CHNHCM-136 正常交付的 Actions 成本观察

登记信息见[登记表](../governance/registry.json)。本记录接续 [136 发布](136-actions-publication.md)与[分层选择](136-impact-selection.md)，采用 [137 用量口径](../governance/137-actions-and-validation.md)。第 1 至 5 节保留主线 `1ddb9005` 时的 7 次交付快照。终期覆盖至主线 `c8b63983`，共 10 次有效交付。[窗口收口](#6-窗口收口)

## 1 观察窗口与交付口径

窗口从 #19 合入生效的北京时间 2026-11-16 15:40:07 起算，到第 10 次有效交付或 2026-11-23 15:40:07 先到者为止。按合并时刻归属，每个正式 PR 计一次，最终候选与对应 main 的必需门禁均成功时计入有效次数。合入窗口的 PR 归集从首轮候选起的全部开销，包括 #19 在起点前启动的候选。策略发布计入窗口总账，正常业务交付另算单位成本。[起点](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-actions-publication/merge.json) · [快照边界](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-actions-observation/snapshot.json)

截至北京时间 **2026-11-21 23:34:24**，7 次合并均完成核验。其中 #19、#21 为策略发布，另有 5 次业务交付。26 次运行共返回 453 个 job 记录，439 个获配 runner，14 个是 main 预期跳过的矩阵。各次运行均为 attempt 1，窗口内无额外手动诊断或未归属运行。逐个已分配 job 以起止秒数除以 60 向上取整，按 job ID 去重，纳入失败、取消、quality 和汇总的实际用量。等待秒数单独按触发至最后一个已分配 job 结束计算，可能与 GitHub 稍后写入的 run 状态时刻相差数秒。原始时刻为 UTC，正文使用北京时间。[逐 job 台账](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-actions-observation/window-jobs.csv) · [各轮范围、等待和原始时刻](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-actions-observation/window-runs.json)

## 2 实际成本、范围与等待

| PR | 失败／取消候选分钟 | 最终 PR 分钟 | main 分钟 | 交付合计分钟 | 最终 PR／main 等待秒 | 首轮触发至 main 完成秒 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| #19 首批策略及部门减负 | 395 | 196 | 7 | 598 | 1448／313 | 5769 |
| #20 部门路径 | 0 | 197 | 6 | 203 | 1286／280 | 1980 |
| #21 分层选测与复用 | 0 | 195 | 3 | 198 | 1355／76 | 1559 |
| #22 职务系列与职务 | 291 | 221 | 3 | 515 | 1643／83 | 8154 |
| #23 职级阶梯与岗位 | 327 | 283 | 3 | 613 | 2324／89 | 5320 |
| #24 行政区划目录 | 1278 | 296 | 3 | 1577 | 2512／85 | 12412 |
| #25 编制、招聘与职类 | 527 | 275 | 3 | 805 | 2476／89 | 6585 |
| **全部 7 次** | **2818** | **1663** | **28** | **4509** | | |
| **其中 5 次业务交付** | **2423** | **1272** | **18** | **3713** | | |

业务交付平均 742.6 runner 分钟，失败或取消的候选占 65.3%。#19 与 #22 的交付总成本已分别与原发布核算交叉核对。等待是用户经历的墙钟时间，交付全程还包含本地整改、提交和合并间隔。[交付总表](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-actions-observation/window-deliveries.json) · [#19 原核算](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-actions-publication/publication-cost.json) · [#22 原核算](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/308-publication/costs.json)

7 份最终 PR 的原始 JSON 均已从 GitHub 下载，逐份比较压缩包 SHA-256 与平台 digest，并另存原始 JSON 的摘要。#19、#20 为 schema 1，#21 至 #25 为 schema 2。7 份正式计划均为 full，各完成 8 个集成、11 个浏览器分片及三项门禁。逐次核对了 main 计划引用的源 run、合同、实际受测树和两个父提交，main 的三项门禁全部成功。主线合计避免重复执行 133 个业务分片。限定范围和纯文档的正式交付样本均为 0。[范围原件](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-actions-observation/evidence) · [门禁与摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-actions-observation/bindings.json) · [实际受测 Git 对象](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-actions-observation/tested-trees.json)

7 次最终 PR 的 Go 测试根依次为 284、285、285、310、343、343、349，日志中的条件跳过依次为 8、8、8、10、13、13、13 项。浏览器不同用例从 396 增至 573，含前置依赖的逐片实际执行从 436 增至 613，各片执行结果已与 quality 的清单对账。条件跳过保留原测试名称。[清单、执行结果及 job ID](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-actions-observation/inventories.json)

## 3 PR #24 的五轮重做

每个候选的完整差异均包含 OpenAPI 合同，正式计划均选全量。四次标记为 cancelled 的运行各有业务分片先失败，随后的修复候选在前一运行结束前已启动。平台记录能证明这段重叠和已消耗的分钟。取消操作者、假定提前结束所能省下的分钟，现有材料无法确定。[逐轮计划和起止时刻](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-actions-observation/window-runs.json) · [首错、整改及本地复验](138-geo-area-interactions.md#84-发布候选的迁移回归)

| 原候选 | run 结果 | 分钟 | 已发生的首错及处理 |
| --- | --- | ---: | --- |
| [24778cad](https://github.com/jacksonlee411/CHNHCM/actions/runs/44546598283) | 取消 | 160 | DB 第 7 片。两项旧库升级测试保留了较新的迁移登记，升级器拒绝不连续版本。修复测试重建的旧库状态。 |
| [c78a5ea7](https://github.com/jacksonlee411/CHNHCM/actions/runs/44549288690) | 失败 | 283 | 浏览器第 8 片。390 像素时职务用例误走桌面入口，改为按视口选择并等待菜单。 |
| [bcd144e3](https://github.com/jacksonlee411/CHNHCM/actions/runs/44556818695) | 取消 | 280 | 浏览器第 7、3 片。BU 历史页未就绪便切日志，补齐页面前置。部门选择清空发生在北京时间跨日附近，原运行缺服务端业务日，直接原因仍待证实。 |
| [8ff2abb9](https://github.com/jacksonlee411/CHNHCM/actions/runs/44563621740) | 取消 | 282 | 浏览器第 9 片。独立生产构建复现了真实产品缺陷，人员编制日志曾提前显示缺少版本上界的原件返回链接，已修复。 |
| [f3489d3d](https://github.com/jacksonlee411/CHNHCM/actions/runs/44570462397) | 取消 | 273 | 浏览器第 9 片。职级重新启用时必填原因尚未返回便提交，受控响应复现后，测试等待正确原因显示再保存。 |

五轮合计 1278 分钟。最终候选 296 分钟，main 复用 3 分钟。五轮均有新提交或后续修订，其中一轮还揭示产品缺陷；无法把 1278 分钟归为可全额消除的浪费。[最终 PR](https://github.com/jacksonlee411/CHNHCM/actions/runs/44576926720) · [主线](https://github.com/jacksonlee411/CHNHCM/actions/runs/44584442932)

## 4 可比历史与收益

旧策略下 #1 至 #18 共 59 次运行，按 PR 重新归入候选、已关联诊断及 main。主线复用历史对照取其中最终 PR 和 main 均实际使用 8／11 且两者受测树、父提交一致的 #14 至 #17。#18 的最终 PR 为 8／4，保留在整次交付台账。此前 3944 分钟的历史快照有不同的截止时刻与事件构成。[历史交付](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-actions-observation/historical-deliveries.json) · [历史运行](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-actions-observation/historical-runs.json) · [历史 job](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-actions-observation/historical-jobs.csv)

| 旧交付 | 最终 PR／main 分钟 | 含失败与诊断的整次交付分钟 | 最终 PR 的 Go 根／浏览器执行次数 | main 结果 |
| --- | ---: | ---: | ---: | --- |
| #14 区划及 CI 分片发布 | 145／145 | 582 | 251／336 | 成功 |
| #15 部门详情导航 | 141／147 | 288 | 252／343 | 成功 |
| #16 地点与导航 | 155／148 | 485 | 259／364 | 成功 |
| #17 企业维护与部门导航 | 178／169 | 369 | 269／415 | 失败 |

**主线复用的估算。** 保持本期 5 次业务交付的所有 PR 尝试成本不变。历史同树 main 与最终 PR 的合计分钟比为 `609 / 619`，用于估算这 5 次若重复跑全量 main 的成本。推算为 `1272 × 609 / 619 = 1251.5` 分钟，较实际的 18 分钟多约 **1233.5 分钟**。相对于这一反事实整次交付基线，差额占 **24.94%**。分别代入四个历史比率，结果为 24.27%–26.05%。#17 的 main 失败也计入历史成本。该范围体现对历史对照的敏感性，实际扣费无可用账单。[计算输入与结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-actions-observation/comparison.json)

**整次交付的可比样本。** #15 与 #20 均为部门导航或路径维护，最终验证包含同一对执行层，PR 首轮通过且 main 成功。按本期 Go 根数与浏览器实际执行次数分别折算旧成本。#15 共消耗 DB 77、浏览器 199、quality 和汇总 12 分钟；按 #20 规模得 `12 + 77 × 285 / 252 + 199 × 437 / 343 = 352.6` 分钟。#20 实际 203 分钟，较这一归一化历史对照低 **42.4%**。折算假定层内平均每项成本可迁移，包含固定准备开销及前置依赖差异，只有 1 对样本。[旧 #15](https://github.com/jacksonlee411/CHNHCM/pull/15) · [本期 #20](https://github.com/jacksonlee411/CHNHCM/pull/20) · [分层明细](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-actions-observation/comparison.json)

#22、#23、#25 涉及的 M03 业务在这批旧全量交付中没有对应覆盖。#24 包含目录迁移及编制原件返回缺陷，旧区划或地点交付的消费者与重做构成不同。整次交付可配对 **1/5**，限定与文档两类仍无样本。因此本窗口的整体 25%–33% 节省目标保留为**证据不足**。分片数和发布门禁继续依现行 137；136 执行状态保持进行中。

## 5 窗口接续

wt-dev-e 负责本次观察。按原起点继续采集后续正式交付，达到任一截止条件即冻结成员，归集这些交付的关联尝试及主线结果。收口时保留策略发布、正常业务与各层选测范围，报告未能配对的交付、条件跳过及失败。仓内本次快照保留原值，后续收口证据单列在同一数据目录的 `window-result/`。本批只读查询已有运行，无新增 Actions 或固定环境操作。

本机用户定时器 `chnhcm-136-window-20261116.timer` 每半小时只读复查，登记时已实际运行一次，仍为 7 次。到期或满额后生成终期总表并停用。离线期间依赖该用户下次登录补查；成员截止仍取原时刻。收口前若工作树或初始快照已更换，结果保存在仓外的任务状态目录，避免覆盖并行修改。

首次采集按新版名称查找产物，未识别 #19 和 #20 的旧版凭据，等价校验中止。修正为按 schema 核对原名称后，7 份压缩包及 JSON 均通过摘要和内容核验。只读 GET 的临时 EOF 通过重新获取处理；这些读取未重跑 CI。

逐 job 独立复算覆盖本期 453 条和历史 697 条记录；与原 136 快照重叠的 56 个已结束 run，分钟逐一相同。独立核对了 7 份原件、133 个计划中的分片身份和全部必需门禁。本地使用隔离输出目录和缩短的截止条件，验证了 7 次真实交付的到期汇总、保留原快照及重复调用；真实窗口仍在进行。`make docs` 的 128 项检查、资料校验及 `git diff --check` 通过。

## 6 窗口收口

#28 于北京时间 2026-11-22 20:02:06 合入。北京时间 2026-11-22 20:10:10 的快照确认它是第 10 次有效交付，窗口按合并时刻截止，共覆盖 #19 至 #28 的 34 次运行。10 份最终 PR 原件的 SHA-256 均与登记摘要一致。10 个最终候选均为全量，候选与对应 main 的 quality、integration、browser 门禁全部成功，main 均复用了对应候选。[窗口边界](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-actions-observation/window-result/snapshot.json) · [原件与门禁核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-actions-observation/window-result/bindings.json) · [受测树与父提交](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-actions-observation/window-result/tested-trees.json)

采集器将 #28 的 298 runner 分钟列为“尚未归类”。该 PR 交付了全量分片前的生成物校验和高风险回归入口，涉及 CI 工作流、脚本及规范。下表将其计入策略发布。按 569 条 job 的分配状态和起止时刻复算，543 条获配 runner，逐次交付分钟均与台账一致。[采集时的汇总](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-actions-observation/window-result/closing-summary.json) · [#28 变更](https://github.com/jacksonlee411/CHNHCM/pull/28/files) · [逐 job 台账](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-actions-observation/window-result/window-jobs.csv)

| 交付类型 | PR | 次数 | runner 分钟 | 其中失败或取消的候选分钟 | 其中 main 分钟 |
| --- | --- | ---: | ---: | ---: | ---: |
| 业务交付 | #20、#22、#23、#24、#25、#27 | 6 | 4327 | 2738 | 21 |
| 策略发布 | #19、#21、#26、#28 | 4 | 1366 | 395 | 16 |
| 合计 | #19 至 #28 | 10 | 5693 | 3133 | 37 |

#26、#27、#28 的最终候选分别等待 2732、2742、2683 秒，各记录 13 项 Go 条件跳过。#27 有两轮候选失败，生成物校验一轮耗用 8 分钟，浏览器第 10 片失败的一轮耗用 307 分钟。[交付与等待](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-actions-observation/window-result/window-deliveries.json) · [逐次运行](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-actions-observation/window-result/window-runs.json) · [测试清单与条件跳过](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-actions-observation/window-result/inventories.json)

6 次业务交付平均 721.2 runner 分钟。按第 4 节的方法，主线复用估算节省 1521.7 分钟。整次交付只有本期 #20 与历史 #15 可按同类业务配对，限定范围和纯文档交付样本均为 0。整体降本 25%–33% 的结论为**证据不足**。80% 托管分钟试点另有观察窗口。[历史配对与估算](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-actions-observation/window-result/comparison.json) · [托管分钟试点](136-hosted-minutes-pilot.md)

归档时一并复算了历史 59 次运行的 697 条 job，逐次运行的分钟均与原台账一致。`make docs` 的 137 项测试和仓库资料检查通过，`git diff --cached --check` 通过。[历史 job 台账](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-actions-observation/window-result/historical-jobs.csv)
