# CHNHCM-136 一次性 runner 与 GitHub 托管分钟试点

本地验证基于 `1ddb90055fc587be2e49ceccb6ab0478d466f2e4`，各轮受测代码摘要保存在检查明细。2026-11-22 发起正式 PR 前，仓库级自托管 runner 数为 0，两个试点变量均未设置，仓库级 fork PR 工作流已关闭。[检查明细](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-hosted-minutes-pilot/checks.json) · [执行约定](../governance/137-actions-and-validation.md#12-全量前置回归与一次性-runner-试点) · [原观察窗口](136-actions-observation.md)

## 1 用实账选择迁出范围

基线仅取当前策略的 5 次全量业务交付 #20、#22、#23、#24、#25。每次包含全部 PR 候选、重跑、关联诊断及 main。只读 API 复核了 345 个已分配的 GitHub 托管 job 和 10 个未获配的 job。3713 分钟分为浏览器分片 2791、DB/API 分片 790、quality 与两个汇总 132。浏览器占 75.2%，其余 922 分钟中有 608 分钟发生在失败或取消的 PR 候选。上一个阶段的两次策略发布另列。[按宿主复核与逐 PR 推演](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-hosted-minutes-pilot/hosted-baseline.json) · [逐 job 原账](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-actions-observation/window-jobs.csv)

失败或取消的候选占 2423 分钟。即使把这部分按零估算，5 个最终 PR 加 main 仍需 1290 分钟，平均每次 258 分钟。达到 80% 还需要降低成功交付的托管用量。[候选与主线的成本分解](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-actions-observation/window-deliveries.json)

| 投放范围 | 假定仍由 GitHub 托管的分钟 | 每次交付均值 | 相对基线降幅 |
| --- | ---: | ---: | ---: |
| 原来全托管，5 次交付实测 | 3713 | 742.6 | 基线 |
| 只迁出 11 个浏览器分片 | 922 | 184.4 | 75.2% |
| 再迁出第 1、2 个 DB 分片 | 732 | 146.4 | 80.3% |
| 再加入第 3 个 DB 分片 | 637 | 127.4 | 82.8% |
| 原有 11 个浏览器与 8 个 DB 分片全部迁出 | 132 | 26.4 | 96.4% |

迁出方案按历史已分配 job 的原时长重算。原第 1 至 3 个 DB 分片合计 285 分钟。按 11 个浏览器和 3 个 DB 分片推算，各交付剩余 45、102、99、263、128 托管分钟，#20 降幅为 77.8%。全体目标均值为 **148.52 托管分钟/次**。推演保留全部原失败。前置后 Go 余集重新分片，各片权重、新前置的准备开销、宿主速度及交付组合都会影响实测。设备、虚拟机空闲、存储、网络及维护成本需要单独归集。即使托管分钟下降，总算力和等待也可能增加。

这 5 个最终 PR 从触发到最后一个已分配 job 结束，等待中位数为 38 分 44 秒，范围为 21 分 26 秒至 41 分 52 秒。若宿主速度相同，11 个浏览器和前 3 个 DB 分片相当于转移约 51.3 个实例小时。新窗口用实际客机寿命核算供应成本，同时对照相近测试规模的候选等待。[运行与等待原始时刻](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-actions-observation/window-runs.json)

若还要让 **现金支出** 降低 80%，且这 3713 分钟原本均按同一边际单价收费，5 次交付的总预算相当于 742.6 个托管分钟。目标配置已占其中 637 分钟，留给新增主机、存储、网络和维护的预算只相当于 105.6 个托管分钟，约为原支出的 2.84%。实际可用额度、账单和主机支出决定现金收益。[GitHub 计价口径](https://docs.github.com/en/billing/reference/actions-runner-pricing)

PR #25 最终候选的这 14 个 job 合计实际运行 13,504 秒，最长单片 2,122 秒。按同速客机和即时派单估算，只用 1 个并发实例，单矩阵就至少需要 225 分钟，4 个并发实例至少 56 分钟，8 个至少 35 分钟。这个下界取总时长除以并发数和最长单片两者中的较大值，另需加上 quality、前置和汇总。原候选实际等待 41 分 16 秒。单机单实例适合试跑第 1 片，扩大到 14 片时应以实际供给速度决定数量。[#25 时长与容量推算](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-hosted-minutes-pilot/hosted-baseline.json)

## 2 在 19 个分片之前能发现哪些问题

`make ci-preflight` 先在真实 PostgreSQL 上执行国家字段旧库升级与 T07 迁移集合准入，再用真实 Chromium、独占数据库和原有企业依赖，执行 BU 历史切换、职务窄屏菜单、职级 V01 和职级 V03。四个用例名分别与 #24 后四轮的实际失败 job 对账。职级两个视口都走原件返回焦点和重新启用旅程，部门跨日选择仍在余集内。[窄屏菜单](https://github.com/jacksonlee411/CHNHCM/actions/runs/44549288690/job/135653501667) · [历史切换](https://github.com/jacksonlee411/CHNHCM/actions/runs/44556818695/job/135678721365) · [职级桌面](https://github.com/jacksonlee411/CHNHCM/actions/runs/44563621740/job/135702816123) · [职级窄屏](https://github.com/jacksonlee411/CHNHCM/actions/runs/44570462397/job/135728640579)

在隔离检出中对 #24 首轮 head `24778cad` 运行当前前置选择器，其源码树 `7bbe6e44` 与该轮正式 PR 合并候选记录一致。两个旧库测试根在 15.283 秒的 Go 测试中均复现迁移准入拒绝，进程按预期退出 1。新工作流在安装 Chromium 和放行 19 个分片之前即可发现这个错误。原轮 160 托管分钟中，DB 分片用了 52，浏览器分片用了 99，quality 与汇总用了 9 分钟。新前置自身的远程耗时待实测。[原运行计划及用时](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-actions-observation/window-runs.json) · [原首错和修复](138-geo-area-interactions.md#84-发布候选的迁移回归)

在本地构造的双父提交候选上，默认、只迁浏览器第 1 片、迁浏览器 11 片与 DB 前 3 片三种配置均选择全量。349 个 Go 测试根按前置 2 个、余下 8 片 44/44/44/43/43/43/43/43 个完成清单对账。

当前全量浏览器清单有 573 个不同测试。前置 job 运行 8 项，其余 11 片运行 61、54、68、63、39、52、52、67、62、52、39 项。两阶段合计 617 次，新增的 4 次执行均为新环境需要的企业维护依赖。已在本地实际跑过前置，两个 Go 测试根通过，Chromium 8/8 通过（Playwright 报告 4.1 分钟）。该用时不含整个 CI 的排队、安装和启动。临时数据库容器、网络、卷与私有配置已回收。[受测输入摘要、清单、宿主路由与真实用例](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-hosted-minutes-pilot/checks.json)

## 3 本地执行、首错与复验

隔离检出的 `make check`、`make build`、`make generate` 和生成后的 Git 差异检查通过，actionlint 1.7.12 通过。两包合跑的 `make integration` 在本机各触及 20 分钟总预算。超时时活动根分别为 `TestChangeLogPaginationFiltersPrivacyAndBudget`（16 秒）和 `TestLegalEntity206ExactCountry`（18 秒）。这次执行记为失败。随后按工作流 8 片在隔离检出中执行，8 片退出码均为 0。连同前置逐个核对 349 个不同测试根，336 个通过、13 个按原条件跳过。[命令、时限及各片实际结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-hosted-minutes-pilot/checks.json)

通过全量候选的分片入口执行浏览器第 1 片，初次 58/61 通过，耗时 20.4 分钟。该轮使用共享开发机，期间同机并行运行数据库验证。TL340 触及 30 秒用例预算。另两条桌面与窄屏原生链接用例在新页中等待一级标题，均到达 5 秒断言时限。独占 profile 已回收，首轮按失败留存。[TL340 遮罩截图](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-hosted-minutes-pilot/tl340-first-masked.png) · [原生链接](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-hosted-minutes-pilot/tl327-first-masked.png) · [窄屏链接](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-hosted-minutes-pilot/tl327-narrow-first-masked.png)

从 #25 最终候选成功的第 1 片日志取出测试清单，与本轮按项目、文件、用例 ID 逐项对照。61 个标识及清单顺序完全相同。[#25 原始托管 job](https://github.com/jacksonlee411/CHNHCM/actions/runs/44898855553/job/136831531456) · [逐片核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-hosted-minutes-pilot/checks.json)

针对这三项打开分步计时，分别在新临时环境中运行原断言及其四项企业依赖。开发服务器上 4/7 通过，三项复现，Playwright 报告 3.8 分钟。TL340 从用例正文开始到读账号登录完成已用 10.1 秒。用相同源码生成生产构建后 7/7 通过，Playwright 报告 1.5 分钟，三个业务用例依次为 16.1、10.3 和 7.1 秒。两轮都伴有其他数据库任务，负载随时间变化。这些数字仅描述诊断过程，正式工作流仍使用原开发服务器。[诊断计时](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-hosted-minutes-pilot/checks.json)

待本任务其他验证全部结束，在同一本地候选、相同清单上重新执行完整第 1 片。保留原开发服务器、断言和超时，只开启分步计时。61/61 通过，Playwright 报告 14.4 分钟，含环境准备与回收为 17 分 2 秒。原三项分别用时 28.3、28.9 和 26.0 秒。该对照支持先限制共享主机并发、再按真实交付等待扩容。其余 10 个浏览器分片本地完成清单核对，正式 PR 全量执行待发布。[同候选复验与首轮结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-hosted-minutes-pilot/checks.json)

整包超时后留下两个 PostgreSQL 测试容器。按本批创建日志、Testcontainers 标签和匿名卷归属逐一核对，已回收这两个容器和两个卷。最终复核了本批日志中的 664 个容器 ID、5 次浏览器运行使用的 4 个临时 Compose 项目及对应仓外配置，剩余数量均为 0。[资源复核](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-hosted-minutes-pilot/checks.json)

### 3.1 浏览器开发服务器的耗时对照

将同一产品源码的生产构建用于完整第 1 片，61 个用例 ID 与原正式清单相同。59 项通过、2 项失败，Playwright 报告 7.7 分钟，含构建、环境和回收为 10 分 33 秒。两项失败分别校验登录后迟到的启动 401 和 503 响应，都依赖 React StrictMode 在开发模式下发起第二次读取。正式 CI 继续使用开发服务器，这轮失败不计入节省。日期与体验夹具也有开发模式专用入口。[401 首失败截图](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-hosted-minutes-pilot/production-bootstrap-401-masked.png) · [503 首失败截图](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-hosted-minutes-pilot/production-bootstrap-503-masked.png) · [原 G4 验收边界](204-g4-implementation.md#41-全量覆盖构建与首次失败)

在有 `CI_PLAN` 的临时 `ci-browser` 环境中，Vite 开发服务器预打包工作树内的 `@mui/x-data-grid`。实测优化器的输入为 `packages/mui-x-data-grid/src/index.ts`，输出文件为 1,124,304 字节，随临时环境回收。两条启动回归分别在 5.9 和 9.8 秒通过，开发模式、企业前置依赖、单 worker 和原超时均参与本轮实测。[Vite 7 的工作区依赖预打包选项](https://v7.vite.dev/config/dep-optimization-options#optimizedeps-include)

在同一台机器、本地双父候选 `15cd17ae` 上顺序运行同一份 61 项清单。两轮都从新数据库和新优化器缓存开始，均为 61 项通过、0 跳过、0 重试。采用预打包后，包含清单核验、临时环境准备、运行和回收的时间缩短 **74.4 秒（7.23%）**。此前同清单的独占复验耗时 17 分 2 秒。该差额来自本机这一片，仓库级托管分钟仍按实际 job 计量。[两轮受测输入、缓存和耗时](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-hosted-minutes-pilot/checks.json)

| 第 1 片，单 runner | Playwright 报告 | 含环境的耗时 |
| --- | ---: | ---: |
| 开发服务器，常规依赖加载 | 14.6 分钟 | 1029.4 秒 |
| 开发服务器，预打包表格依赖 | 13.3 分钟 | 955.0 秒 |

在另一套新数据库和预打包缓存上，单独复验原 G4 的 6 个开发模式专用用例、4 个前置浏览器用例及 4 个企业依赖，14/14 通过，Playwright 报告 4.3 分钟。覆盖两种日期区域、体验夹具、启动响应先后顺序和职级的两个视口。以上 4 次实际浏览器运行共使用 3 个临时 Compose 项目，项目资源和对应仓外配置均已回收。[用例与清理复核](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-hosted-minutes-pilot/checks.json)

最终候选重算得到 349 个 Go 测试根和 573 个不同的浏览器用例。`make check`、构建、生成一致性及 actionlint 通过，表格正式来源校验覆盖 501 个源码文件和 500 个声明文件。其余 10 个浏览器分片的远程耗时和全窗口收益待正式发布。[最终检查结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-hosted-minutes-pilot/checks.json)

## 4 投放与验收

2026-11-22 用户同意先推送本批并以 GitHub 托管配置开正式 PR。首轮验证前置回归、浏览器预打包和全量分片，成本按策略发布单列。合并由用户在查看最终三项门禁后决定。后续接入自托管主机时按下列步骤投放。

1. 确认已授权的隔离主机和发布范围。由控制端为每个 job 创建独占 Ubuntu 24.04 x64 客机，提供 Git、Python、make、gh、Docker 与 Compose、sudo 安装依赖及网络下载能力。先准备客机补充与回收能力，再设置浏览器试点变量为 1。一个全量候选的前置和第 1 片分别消耗一个一次性实例。
2. 发布候选从 quality 的正式范围证据核对到前置、两层矩阵、三个必需检查及 main。记录真实 job ID、runner 分组、临时实例回收、排队、失败及遮罩证据。若无可用客机，将变量降到 0，在重新计算路由的正式运行上复验。旧运行照实入账。
3. 根据主机容量扩大浏览器分片，再考虑 DB 第 1 至 3 片。分开统计切换中的候选和目标配置下的自然交付。按 10 次有效交付或 7 天先到者收口，对全量按测试规模、覆盖范围和失败构成配对或归一化。限定范围、策略发布单列。未配对或宿主无法分类时保留限制。

使用已核验成员的交付台账按宿主复算。脚本核对最终 PR、合入 main、完整 attempt 与 job 身份，逐个已分配 job 向上取整，保留失败、取消、等待时间和未分类宿主。自托管和未分类 job 另存 job ID、runner ID 和起止时刻，可与控制端的客机创建、回收记录对账。使用下列命令可重复取得本批 5 次业务交付的基线。[用量脚本](../../scripts/ci_usage.py) · [GitHub job 数据](https://docs.github.com/en/rest/actions/workflow-jobs#list-jobs-for-a-workflow-run) · [GitHub 分钟取整规则](https://docs.github.com/en/billing/reference/actions-runner-pricing)

```sh
python3 scripts/ci_usage.py --repo jacksonlee411/CHNHCM \
  --deliveries docs/verification/data/136-actions-observation/window-deliveries.json \
  --pr 20 22 23 24 25 --rollout
```

`--rollout` 逐次读取对应 quality 日志中的原始计划，核对仓库、运行、事件、提交及 PR，把计划投放范围与实际宿主分钟并列。仅重跑分片时沿用提供矩阵的 quality，重新执行 quality 时记入新的投放范围。投放阶段按正式全量候选区分，main 等价复用时两层为 0 片，其实际分钟仍归同次交付。旧版本无配额字段或日志无法取得时保留空值，已消耗分钟照计。用这条命令两次回读本批 5 次业务交付，均取得 15 个全量候选与 5 个 main 复用计划，两份结果逐字节相同，345 个已分配 job 共 3713 托管分钟。

试点实际托管分钟、同范围节省比例、正式门禁与开发者等待均待首次发布和后续自然交付。

原 136 观察任务继续按原截止条件运行。采集器已能在同一个前置 job 的日志中读取 Go 根和浏览器执行数。按缓存重放此前 11 份清单，结果逐份相同。另以本批前置日志和一次部分重跑的本地样例核对了新读法。

## 5 正式候选全量启动频次

2026-11-22 在 #27 合并树 `09124724` 上复核。按交付清单回读 GitHub job 原件，第一次 quality 发现未同步的 sqlc 模型，花费 8 托管分钟，业务分片获配 0 个。第二次、第三次各有 19 个业务分片实际获配，整轮分别为 307、296 分钟。main 等价复用花费 3 分钟。合计 614 托管分钟，正式 PR 全量启动 **2 次，最终通过的 PR run 前 1 次**。第二轮 19 个业务分片占 292 分钟；提前命中失败才可能避免这笔开销，节省额待新交付实测。[交付清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-hosted-minutes-pilot/pr27-delivery.json) · [失败候选](https://github.com/jacksonlee411/CHNHCM/actions/runs/44998358652) · [最终候选](https://github.com/jacksonlee411/CHNHCM/actions/runs/45003381956)

以下以各合并提交的第一父提交为基线，读取候选当时的映射及消费者源码，使用本轮选择器回放。两列数字依次为 DB/API 和浏览器分片数。末两行是拟独立交付的提交区间。

| 候选 | 变更文件 | 现行 | 试算 |
| --- | ---: | ---: | ---: |
| #22 `a9e5daca` | 784 | 8 / 11 | 8 / 11 |
| #23 `51db9753` | 969 | 8 / 11 | 8 / 11 |
| #24 `94e9aefb` | 180 | 8 / 11 | 8 / 11 |
| #25 `1ddb9005` | 167 | 8 / 11 | 8 / 11 |
| #27 `09124724` | 200 | 8 / 11 | 8 / 11 |
| #27 生成模型 `825016a0..eda83fbe` | 3 | 8 / 11 | 8 / 11 |
| #27 页面修复 `eda83fbe..66f06665` | 4 | 0 / 2 | 0 / 2 |

最后一组可用 `make ci-preview CI_BASE=eda83fbe CI_HEAD=66f06665` 重放，选中 17 个浏览器文件，包含两个时序故障所在文件。五个完整 PR 的合同和迁移均超出首批试算边界，#24 的旧库升级、BU 历史、职务窄屏及职级焦点回归仍在全量中。另在独立副本中修改职务列表成功响应说明并运行锁定版客户端生成器，试算选 4 / 4；单独新增职务读取表索引及配对 Down，试算选 8 / 4。两组各覆盖 20 个浏览器文件；索引同时改变共享授权时回到 8 / 11。史料未包含可降级的真实正例，正式门禁继续使用现行范围。

在临时检出 `825016a0` 运行生成检查，准确报出 access、organization、staffing 三个 `models.go`，原文件保留。当前树的实际 Playwright 清单共 605 个不同测试，前置执行 6 个回归与 5 个企业依赖，11 个余集分片均非空。跨环境重复的 55 次执行均属于声明的企业依赖。两项真实 PostgreSQL 旧库测试及 Chromium 前置 11 项在本地通过。该清单核对覆盖所有分片的分配，实际业务执行范围为前置用例。

将两份控制响应顺序的测试放到修复前的 `eda83fbe` 产品树，5 个企业依赖通过，两项业务断言均失败。结构日期应为 `2027-04-01`，实际停在默认的 `2026-11-22`；390px 返回职务列表时，“更多操作”在列表响应前已经可用。在修复后的产品树上，同样的两项断言均通过。两次使用独立临时库，运行结束后均回收。

另按同一频次口径复核旧窗口。#22、#23、#24、#25 的正式 PR 全量分别启动 3、2、6、3 次，其中最终通过的 PR run 前分别有 2、1、5、2 次。这些是已有运行的实测基线。[逐 job 记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-actions-observation/window-jobs.csv) · [运行时刻与候选](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-actions-observation/window-runs.json)

首次对四文件修复执行真实限定清单核对时，94 个测试因定义位于共享辅助文件而被误判为范围外，另有 6 个入口文件被报缺失。归属现取 Playwright 文件级 suite，测试身份继续使用项目、定义位置及测试 ID。复核 17 个入口得到 221 个不同测试，两片分别执行 114、112 项，两片之间仅重复 5 个企业依赖。两个试算正例的 20 个入口得到 223 个不同测试，四片分别执行 89、32、67、50 项。索引试算检验了选择范围，合成迁移未进入产品或持久库。

本轮本地通过 `make ci-ready`、`make check`、`make build`、文档历史引用检查和 `git diff --check`。草稿摘要已在本地用相同参数生成；本轮没有启动 GitHub Actions，发布后的单次交付分钟与启动频次待观察。

继续核对远程 quality 的步骤时，在临时仓库复现了一个漏检：生成器新增一个应提交的文件，原 `git diff --exit-code` 仍返回 0。quality 现于依赖安装后先执行生成并检查 Git 工作区状态，再运行静态检查和构建。从工作流读取该步骤原文执行，新增、修改、删除三种生成物差异分别返回 1，仅产生忽略的缓存时返回 0。在 `09124724` 上组装本批候选的独立临时检出，执行锁定依赖安装与实际生成后工作区清洁。模拟 PR 的草稿计划为 0 / 0 且汇总拒绝合并，正式计划为 8 / 11。上述验证均在本地执行。

## 6 同一 PR 的后续全量候选

2026-11-23，wt-dev-b 采用已合并 #29 的 `45a4c227` 为基线。对 #27、#29、#30 实际展开过 19 片的 9 次运行，读取 GitHub 作业结果、原计划及实际分片列出的测试 ID 和 Go 根。在各次候选树上重新发现 Playwright 入口，核对去重集合，并按时间顺序将新调度规则应用于这些原件。每轮只向下一轮提供按新规则实际会派发的作业。首轮生成失败、草稿和分片启动前取消的运行单独保留在原 Actions 中。[逐轮回放](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/136-hosted-minutes-pilot/repeated-full-replay.json)

| PR | 原实际全量 | 按新规则计划全量 | 另外需要的业务分片 |
| --- | ---: | ---: | --- |
| #27 | 2 | 1 | 最后一次页面修复执行前置和 6 个浏览器分片，涵盖原第 10 片的两项回归。 |
| #29 | 4 | 2 | 窄屏和撤权修复执行前置和 8 个浏览器分片。随后 22 份说明变更可沿用运行 45101961774 的 quality，业务分片为零。合入 #30 时共享分片脚本发生变化，完整验证。 |
| #30 | 3 | 1 | 两次分别执行第 8 和第 11 个浏览器分片。职级旅程变化还会重做前置，末次慢读取测试修复可沿用前置。 |

这三项交付的 171 个原生业务分片，在固定历史提交序列上会派发 92 个。另有前置、quality 和门禁作业。#30 第二轮历史上发生的业务单元慢读取错误位于当轮未受影响的第 11 片，新规则当轮不会发现这次偶发失败。后续实际修复该测试的提交会选中第 11 片。回放保持历史提交序列，开发者在不同反馈下可能采用不同修复顺序。[历史失败分片](https://github.com/jacksonlee411/CHNHCM/actions/runs/45100729611/job/137455124057)

本地完整清单核对覆盖 357 个 Go 测试根和 616 个不同浏览器用例。每片按原生执行身份与入口核对，允许企业依赖重复，包含前置共 671 次浏览器执行身份。专项验证覆盖同 PR 多次修复、覆盖来源变更、部分失败、分片移动、过期或缺失原件、文档跟进、合并后复用及共享输入变更。最初历史稀疏检出遗漏了四个页面测试共同导入的视口辅助文件，发现阶段报错。补齐仓内原件后，九个候选均与远程清单对上。首次专项测试夹具缺少 PR head 输入，修复夹具后重新验证。新工作流的远程运行与实际用量待发布后观察。

本地 `make ci-ready` 在候选文件树 `40b5f968` 核对生成物，2 项真实 PostgreSQL 旧库升级和 11 项 Chromium 前置全部通过，临时环境已回收。随后对同名嵌套入口补充了归属核验，以本批完整清单和历史回放复查。工作流使用 actionlint 1.7.12 检查，同时执行 `make check`、`make build`、文档历史引用核验与 `git diff --check`。

### 6.1 PR #31 首轮正式运行

2026-11-23，PR #31 的首个候选 `a22c3668` 基于 `45a4c227`，GitHub 测试的合并树为 `fb78693e`。本次修改了 `.github/workflows/ci.yml`，quality 选择全量。同一 PR 可沿用的原作业为 0，前置、8 个 DB/API 分片、11 个浏览器分片、quality 和两个汇总门禁共 23 个作业，首次均通过。[正式运行](https://github.com/jacksonlee411/CHNHCM/actions/runs/45140253998) · [quality 调度记录](https://github.com/jacksonlee411/CHNHCM/actions/runs/45140253998/job/137576182679)

从触发到最后一个已分配作业结束共 45 分 35 秒。23 个作业均使用 GitHub 托管 runner，按各作业运行时长分别向上取整，合计 302 托管分钟。本条验证记录构成同一 PR 的纯文档跟进，后续候选和主线的实际用量汇总在 PR 说明中。[PR #31 交付记录](https://github.com/jacksonlee411/CHNHCM/pull/31)
