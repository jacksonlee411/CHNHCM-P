# CHNHCM-500 人员模块实施路线图评审

2026-11-22，Codex 主代理顺序审阅五稿，前四稿各有需要修订的事项，第五稿通过路线图评审。评审范围包括首批独立签收、页面标准继承、领域写入责任和后续联验。第一项执行任务是 T00 的 M01 专项规格。[路线图](../archive/plans/500-personnel-roadmap.md) · [登记表](../governance/registry.json)

## 1 基线与依据

工作树为 `wt-dev-f`，HEAD 为 `063c92fd6568e20f697ad07a64e5a9e85ef99615`。开始时工作区干净、stash 为空，核对的 15 个本地及远程跟踪引用均无 500 号登记。评审针对该基线上的新增路线图。此次还新增本记录、逐稿材料和两处导航，并更新项目登记。各轮均由同一主代理执行，未开展人工签收。

| 依据 | 实际核对与取舍 |
| --- | --- |
| 本仓 M01、M04、004、X03、206、313 与 106 | 用模块合同核对人员登记、编号、用工与实际任职，用 206 区分主体资料与用途资格，用 313 核对职务/岗位模式、整数名额及录用转换。106 登记为草案，路线图把采纳及专项范围列进 M04 开工前置。[M01](../architecture/target/modules/m01.md) · [M04](../architecture/target/modules/m04.md) · [004 目录与端口](../architecture/004-repository-layout.md) · [206 分批](../archive/plans/206-legal-entity-implementation.md) · [313](../specs/m03/313-position-capacity-contract.md) · [106](../adr/106-same-day-personnel-actions-and-daily-state.md) |
| 已交付页面和当前前端接线 | 对照 012、121、124 及 308 逐页继承表，核对 BU／职务的列表、表单、详情、操作、日志、草稿及提交恢复代码。回查已记录的 308 实测范围。本轮的人员页面检查对象是路线图里的任务和预期。[012](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-expression) · [121](../specs/m02/121-dated-object-admission-matrix.md) · [124 页面准入](../guides/124-dated-object-admission-and-capacity.md#page-standard-admission) · [308 整改与验收](308-job-page-standard-implementation.md) |
| 本地 CubeBox 工作树 `074fe044ab41c80abbab41a74aadd2a97b66efcf` | `apps/prototype/src/operations/staffing-list.tsx` 的行键取 assignmentId，部门计数按 personId 去重。核对了员工列表及任职时间轴留存图面、`docs/dev-plans/ui-ux/203-employee-personal-information.md` 的多值资料及 `apps/prototype/src/domain/staffing-multiple.test.ts` 的跨关系样本。三文件 SHA-256 前 12 位依次为 `901ea143a709`、`9d92ad1b5dcc`、`6d1fc9848091`。这些是本机只读观察，项目初始来源包另固定在 2deaf927。[项目来源基线](../reference/README.md) |
| 本地 Fusion/FSCR 资料及 Oracle 产品说明 | 核对 FSCR `d88c993` 保留的概念映射、来源索引 O02–O04 及 Oracle 原文。Fusion 的 Person 通常随工作或联系人关系建立，任职有独立访问范围，再聘匹配原人并建立新关系。本仓采用 M01 独立登记和 M04 按用途的两层主选择。此次参考限资料、源码及留存图面，未操作 Fusion 租户。[本地映射](../architecture/target/07-concept-mapping.md) · [本地来源索引](../architecture/target/data/oracle-release-check.json) · [Person Records](https://docs.oracle.com/en/cloud/saas/human-resources/fawhr/person-records.html) · [Assignments](https://docs.oracle.com/en/cloud/saas/human-resources/fawhr/assignments.html) · [Rehire a Worker](https://docs.oracle.com/en/cloud/saas/human-resources/fawhr/rehire-a-worker.html) |

## 2 五轮结论

| 稿次与审阅角度 | 结论及转入下一稿的事项 | 路线图 SHA-256 |
| --- | --- | --- |
| 一 · 依正常菜单走登记、切换任职、返回和查看历史 | 要求修订。第一稿未把无任职人的发现范围、辅助导航失效、过去记录与只读原件的区别落到用户可见结果，且把原生缩放的 1280×900 写为窗口外框。 | `08c07968f5e6c0aa4ee3b2e280784d030544003cf37daf9f11512dd2c753c732` |
| 二 · 复验页面修订，沿一次入职追踪每个事实的写者 | 页面问题关闭。要求修订 106 采纳与 M04 开工的关系、人员与任职读取的归属、存量岗用量未知时的处理，以及录用份额换人或换提交号的行为。 | `e35f6ea20da3a9c47d044e97006aa0463f72ae961e703c91c33c3d50e674e8c0` |
| 三 · 按交付先后独立走查，用受限主体检索名录 | 第二轮问题已处理。发现按任职先筛选可能带出无人员读权者，直接入职与兼岗共用验收行，离职又预设已交付合同。要求明确两个授权集合及独验/联验分工，补入同日局部撤销的可观察结果。 | `2f1b6d41533e3e02dae3683230edf015726c53490c9309224a09a0fb35cd6ff6` |
| 四 · 复算日期、权限交集与名额，回查两个重送入口 | 第三轮问题关闭。T06 已承诺未来入职撤销，但本稿的招聘场景只核对首次转换，尚无法发现撤销后恢复旧预留、迟到重送消耗新预留。登记重送也缺同号异内容样本。要求修订。 | `54545af7e89e93ebd82104760ee041237b47de786aa20eb5d4c1343b766ca100` |
| 五 · 对准 313 撤销/再次预留及 M01 幂等，复查全稿 | 通过。按同一份额及两次不同预留追踪，加入失配、同号异内容及旧号永久归属的判据。复查七项任务、八个场景、页面参照及独验分工，未发现阻断路线图采纳的未结项。 | `97e467c1022f394a03bbb056738b53cd8c17a5d244a6aee2052ca8f81b36e4b0` |

## 3 终稿复核

| 反例 | 验明的路线图要求 |
| --- | --- |
| 能看同一人的 A 任职，B 才是主任职；另一人的 A 任职可读，但该人档案无权 | A 不被改标成主，隐藏的关系和数量不外露。按 A 筛选人员名录时，任职命中甲、乙，人员范围只有甲，最终列表、位置、计数均只含甲。再从页末进入详情、跨页、返回，查询和焦点有可复验的落点。[页面继承](../archive/plans/500-personnel-roadmap.md#personnel-pages) · [兼岗样本](../archive/plans/500-personnel-roadmap.md#m04-scn-002) |
| 同一表单在长屏可保存，但短屏无法退出历史、撤权后原回执仍显示号码 | 七种视口及原生缩放按角色验。过去的生效记录与明确打开的来源原件分别处理。未提交草稿与已提交未知按不同路径恢复，字段失权时清理正文、日志和回执。[资料与权限](../archive/plans/500-personnel-roadmap.md#m01-scn-002) · [012 视口标准](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-viewport-matrix) |
| 存量岗无使用来源，或用人单位只完成 L1 登记 | 依赖未齐时可单独交付人员登记。M04 产品开工以 106 采纳及专项范围为前置。直接入职需有效主体资格和真实任职政策。岗位名额用量未知时不能据空表放行，已知最后一编并发只成立一份任职。[直接入职样本](../archive/plans/500-personnel-roadmap.md#m04-scn-001) · [313 直接入职竞争](../specs/m03/313-position-capacity-contract.md#m03-ac-071) |
| 只实现仅按职务入职或基本合同续订 | T03 仅按职务可先验；最后一编竞争属于按岗分支，兼岗计数由 T04 承接。T05 用一段真实关系先验续订和合同链，再与 T04 联验多关系和退出。执行结果按实际分支登记。[合同续订样本](../archive/plans/500-personnel-roadmap.md#m04-scn-004) |
| 同日两件独立人事动作，误撤其中一项 | A/G1 先后形成独立的调部门和调职级，撤职级后当日为 B/G1，仍可查看动作和撤销依据。有必要依赖时，后继须另作合法处置。人事、岗位和后续薪酬各域保有自己的事实。[同日样本](../archive/plans/500-personnel-roadmap.md#m04-scn-006) · [架构反例](../architecture/target/10-ddd-and-state-contracts.md#ac03) |
| 先预留后转换，生效前撤销，招聘重新预留，旧请求迟到 | 看原入职日的（预留，已占），依次应为（1，0）、（0，1）、（0，0）、（1，0）。两份旧办理各重送一次，仍为（1，0）。原转换可查。登记重送另按原提交号和内容核对，改号后的旧号永久定位原人。[录用样本](../archive/plans/500-personnel-roadmap.md#m04-scn-005) · [313 原样本](../specs/m03/313-position-capacity-contract.md#m03-ac-097) · [登记样本](../archive/plans/500-personnel-roadmap.md#m01-scn-001) |

## 4 检查、材料与限制

- 五稿原文及样本复算程序以同一份压缩包保存。逐份对比第 2 节 SHA-256，第五稿与当前交付路线图逐字节相同。解包后用 Python 3 运行 `examples.py`，复算可见人员交集、人数/任职数/名额、未来入职日期、最后一编的两种办理次序、两岗对调、同日撤改及招聘份额的四次变化和旧办理重送。手工复核按章引用，检查五张表、七个任务、八个唯一场景锚点及七组视口条件。[逐稿原文与复算程序](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/500-personnel-roadmap-review/reviewed-plans.zip)
- Go 1.26.0 下运行 `make boundaries`，625 个既有 Go 文件的导入边界检查通过。`go test ./scripts/check-boundaries -count=1` 通过。数据库事务所有权和实际页面交互由各产品批次另验。
- Python 3.14.4 下运行 `make docs`，137 项脚本测试通过，仓库检查覆盖 418 份 Markdown、13,186 处本地链接和 231 项保留资产。与 HEAD 比对登记表，原有 87 份文档、248 条验证记录及其他登记内容一致，本次各新增一项。用任务临时 Git 索引把新增材料纳入 `git diff --check`，六个变更文件全部通过。

本轮结论限于路线图文档。M01/M04 产品实现、真实 DB/API、人员菜单浏览器验收及人工业务签收均未执行。T00 冻结首批资料及无任职者的核对范围，T02 接续 M04 专项决定和前置能力。后续报告按实际构建记录业务事实和首错复验。[批次及责任](../archive/plans/500-personnel-roadmap.md)
