# CHNHCM-140 模块边界、日期显示与实施登记收敛路线图评审

2026-11-22，Codex 主代理按用户选定的方式，用仓内合同、实际调用链、独立日历算例和登记反例逐轮复核。前四轮要求修改，第五轮接受终稿。评审结论覆盖路线图的实施边界、顺序与验收设计。四项产品任务均待实施。[实施路线图](../plans/140-module-cohesion-roadmap.md) · [登记](../governance/registry.json)

## 1 受审对象与基线

固定工作树为 `CHNHCM-wt-dev-b`，分支 `wt-dev-b`，基线 `f2c6edf79909a9de8ab18dc35c6cd212d37b5d74`。开工时工作区和 stash 均为空，本地分支相对本机跟踪引用领先四个提交。受审对象是该基线上新增的 140 方案。方案五个版本以 SHA-256 标识。各轮原始修订差异已从终稿逐轮反向应用，逐字节复得前四稿。[实际修订差异](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/140-module-cohesion-review/revisions.txt)

| 轮次 | 输入稿 SHA-256 | 判定 |
| --- | --- | --- |
| R1 | `bbfd7ea095887bfc201144b5141820e5a00d151f912955b89fee85ff4b159326` | 修改授权续页和边界检查条件，形成 v1。 |
| R2 | `3744103aedcdb596992582435401b187f7fe3f56eac16269750df3e032262f03` | 修改日期投影、权限及适用对象，形成 v2。 |
| R3 | `e0faf5b44e905f6aad38bf8b418b7ee5650919f4fab96d2b3c77e026a1c85fc4` | 修改登记粒度、历史触发和行为比对，形成 v3。 |
| R4 | `3d861a344e42161d814c6bd60c57097622da1a0bdb0f6ee6652c87ff76de12a5` | 修改取证先后、样本前置和阶段出口，形成 v4。 |
| R5 | `476e27d97c994ecfc023aa1aaf1671569427f87e61954a4e86e9b6789488b072` | 接受方案。既有发现已落实，终稿复核无新增主要问题。 |

## 2 首次发现与修订

| 轮次 | 独立反例与观察 | 写入终稿的处置 |
| --- | --- | --- |
| R1 授权与 SQL | 只撤非必需字段，使用者仍可打开候选。再把授权恢复为原来的集合，若只比较授权摘要，旧游标就可能被误用。选择器对 403 清旧项，对其他续页错误会保留上一页。原 Go 检查只解析包导入。 | T01 保留 X03 同一事务已锁定的版本，移除 M02 重查。A01、A02 区分 409、必要权限丧失后的 403、会话失效与故障，要求 409 后清旧批次再取首页。加入按域 SQL 源、Go 字符串、嵌入 SQL 的放行和拒绝样本。[授权守卫](../../internal/contexts/access/infra/postgres/access.go) · [区划候选](../../internal/contexts/organization/application/geoarea/candidates.go) · [地点选择器](../../apps/web/src/features/organization/LocationRegionPicker.tsx) |
| R2 日期与历史 | BU 在失去日期权时仍可能有两个 null 终点。BU、企业的前端适配器手选成对删除字段，单改页面或生成类型都会遗漏。预览使用的 `DateRange` 还参与历史编码。将 03-01 一律显示为最终适用日，在闰年会多算一天。 | T02 分成已有记录投影和专用区间投影。A03、A04 覆盖有限、开放、失权、跨闰日、真实适配器、取消及旧回执。限定两种无资料区间生产者和五种成对删除对象，保留对象原有权限条件。[BU 投影](../../apps/web/src/features/organization/businessUnitProjection.ts) · [区间与回执](../../internal/contexts/organization/public/contracts.go) · [部门受限投影](../../internal/contexts/organization/application/department/capabilities.go) |
| R3 登记与历史收敛 | 同属 Staffing 的 M03 和 M04 可以处于不同阶段。314 签收表将 AC090 分为三项。地点和法律实体命令各 222、226 行，逐行匹配 203 行，但只有地点的持久化路径检查岗位引用。两个独立运行也会产生不同 UUID。 | T03 为实际切片列归属、范围及证据。T04 在首个触及重复规则的任务内实施，保留对象专有规则。A06 比较来源关系及旧原件，A07 分开前置拒权、业务拒绝、无变更、提交前故障和响应丢失。[314 签收](314-job-function-t05-acceptance.md) · [地点写入](../../internal/contexts/organization/infra/postgres/location.go) · [法律实体写入](../../internal/contexts/organization/infra/postgres/legal_entity.go) |
| R4 验收能否执行 | 提交前故障后，获权人查一个尚无终态的原号，可以写入 `not_applied`。先查回再统计终态会把两次操作混为一谈。企业首条后移还可能被更早的下游事实挡住。把所有批次绑在一起也会让 313 和前三项等待未触发的 T04。 | A07 先从另一连接读已提交事实，再按原号恢复。A04 将 BU 和企业分为各自合法的样本。T01—T03 独立签收，T04 待触发时保留计划进行中，313 按自己的合同推进。[恢复实现](../../internal/contexts/organization/application/submission/recovery.go) · [地点回滚样例](../../internal/bootstrap/location204_acceptance_integration_test.go) · [企业后移样例](../../internal/bootstrap/enterprise304_integration_test.go) |

## 3 终稿复核与本地验证

| 核对内容 | 本轮实际结果 |
| --- | --- |
| SQL 所有权 | 在三个域的 `db/queries/<域>` 和 `internal/contexts/<域>` 筛查显式 FROM、JOIN、UPDATE、INTO 限定表名，排除测试及生成源。共 281 份 Go/SQL 文件，唯一跨域匹配是 Organization 读 `access.policy_guard`。另对照锁查询、候选授权与游标解码顺序。人工范围之外的动态 SQL 仍需实施时核对。[筛查与反例输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/140-module-cohesion-review/source-checks.txt) |
| 日期与对象 | 用独立日历算出两个排他终点的上一日分别为 2026-09-30 和 2024-02-29。沿 Go 投影、OpenAPI、适配器、页面回读 A03/A04，逐个核对两种首条后移预览及五种成对删除投影。 |
| 登记反例 | 从现行检查脚本提取并运行三条上下文断言。实际登记通过。只在内存副本中把 Staffing 标成覆盖 M03、M04、313 且已全部验收，现有断言仍通过。给同一副本重复加入 M03 则被拒绝。该差别支持 T03 增加切片语义复核。[现有登记检查](../../scripts/check-repository.py) |
| 历史与阶段 | 对读两个对象的提交、组装、存储和恢复，验证最长连续相同段为 50 行。逐条回读 T01—T04 和 A01—A07，确认授权收窄、日历边界、原采用、删除目标、提交故障、双对象同号和 313 分批签收均有独立预期。 |
| 基线 Go 检查 | `make boundaries` 通过，覆盖 628 个 Go 文件。用 `go test -count=1` 运行 Organization 的 `domain`、`domain/dated`、`application/submission`，三个包均通过。原跨表查询仍能通过既有导入检查。 |
| 文档交付 | `make docs` 通过 128 项测试，仓库核对 421 份 Markdown 和 13,276 个本地链接。`python3 -B scripts/check-repository.py --check-refs` 通过六个固定分支号段检查。`git diff --check` 与四份新增文件的独立空白检查通过。反向还原四轮差异得到全部既有稿件摘要，登记相对基线只新增 140 及本评审记录。 |

## 4 结论与执行交接

140 方案评审通过，实施从 T01 接续。切片证据校准由 T03 持有，核编独验和并存联验由 313 首批持有。前三项各自签收后跟踪 T04 的触发任务。

本轮修改限于路线图、评审证据、登记与文档导航。数据库、API 和浏览器的 A01—A07 属于后续实施验收，本轮未执行，也未启动应用 profile、修改固定环境或触发远程 CI。
