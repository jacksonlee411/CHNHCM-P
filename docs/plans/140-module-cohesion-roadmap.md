# CHNHCM-140 模块边界、日期显示与实施登记收敛路线图

本计划适用于模块化单体内已接入的 Organization、Access 和 Staffing 切片。Go 负责业务日期、授权、状态与历史。管理员继续从现有菜单维护资料，查询者按当前权限查看。方案采纳与实施进度分别见[文档登记](../governance/registry.json)和[方案评审](../verification/140-module-cohesion-roadmap-review.md)。

T01（A01、A02）、T02（A03、A04）、T03（A05）已分别通过本地验收。受测候选、首次失败及复验见[分项实施记录](../verification/140-module-cohesion-implementation.md)。A02 的括号连接与别名反例、A04 的准确采用名称反例见[阶段评审修复记录](../verification/140-module-cohesion-review-fixes.md)。T04 待下一次相关 M02 历史流程变更触发。

## 1 范围与优先顺序

| 顺序 | 基线上的具体问题 | 交付与责任 | 放行条件 |
| --- | --- | --- | --- |
| T01 先处理跨域读表 | 行政区划候选在 Organization 的 PostgreSQL 适配器内直接查 `access.policy_guard`。X03 在同一请求内已经锁定并读到该版本。 | X03 持有授权事实，M02 经现有授权端口使用。边界检查由架构维护者接入。 | A01、A02 通过，原跨域 SQL 移除。 |
| T02 统一期间显示 | 同一条记录在详情显示最后适用日，单条删除确认显示下一条起日加“之前”。首条改日晚移和成对删除另有两处显示排他终点。 | M02 在 Go 的读取投影提供显示值，共享页面消费，各对象保留自己的动作合同。 | A03、A04 通过，四种实际入口均有核对结果。 |
| T03 校准实施登记 | Organization 仍标 `directory-reserved`，G0 文字仍称产品未实施。Staffing、Access 均标 `planned`。 | 各域提供切片与证据，架构维护者校正上下文登记、目录说明及现行导航。 | A05 通过，每项能力可追到当前入口及相应验收范围。 |
| T04 下次 M02 历史流程修改时实施 | 地点与法律实体的命令都编排提交、版本、修订和回执。两处各自维护相同的历史规则，有漏改风险。 | 当次 M02 历史流程负责人和两对象维护者共同执行，先以两份独立样本固定行为，再收敛共同部分。 | A06、A07 通过，并随触发任务交付。 |

T01、T02、T03 可分别提交和验收。T03 记录已经有证据的能力，可独立于前两项落地。前三项交付后单列“T04 待触发”，140 执行态保留进行中。313 按自己的核编合同和验收批次推进，140 的收敛工作不作为其开工门槛。部署继续采用模块化单体，本轮复用限于已有消费者。所属域公开端口、短事务与 SQL 归属以 [004](../architecture/004-repository-layout.md) 为准。

## 2 T01 行政区划候选经 X03 取得策略版本

正常入口为「设置 → 组织配置 → 地点」，新建或编辑地址，选择有已发布区划的国家/地区，展开区划候选并翻页。该选择器请求 `GET /api/v1/geo-areas/candidates`。[地点选择器](../../apps/web/src/features/organization/LocationRegionPicker.tsx) · [候选服务](../../internal/contexts/organization/application/geoarea/candidates.go) · [M02 候选查询](../../internal/contexts/organization/infra/postgres/geoarea_query.go)

1. X03 从现有 `LockPolicyShared` 的返回值取得当前实例、主体的 `policy_no`，保存在本次事务的 guard。经 X03 公开的小型只读接口返回版本，M02 从当前 guard 取得。移除 M02 的 `Unit.PolicyVersion(ctx)` 和对应 SQL。共享锁、读取授权、错误传播和释放位置与现有事务一致。[Access 实现](../../internal/contexts/access/infra/postgres/access.go) · [X03 查询](../../db/queries/access/queries.sql) · [M02 端口](../../internal/contexts/organization/application/geoarea/ports.go)
2. 保留现有游标签名和绑定，包括主体、会话、实例、业务日、查询条件、授权摘要和策略版本。每次翻页都建立自己的短事务。一个请求中的守卫、授权和候选读取共用事务。策略写入取得排他锁后递增版本。守卫建立失败沿原错误路径结束。HTTP 沿用已有游标，去掉 M02 的重复查询，无需第二条连接、缓存策略版本或迁移数据。[游标实现](../../internal/contexts/organization/application/geoarea/cursor.go) · [区划事务](../../internal/contexts/organization/infra/postgres/geoarea.go)
3. 在 `make boundaries` 中增加定向 SQL 所有权检查。按目录识别 `organization/access/staffing` 的所有者，检查 `db/queries/<所属域>` 的 sqlc 源、各域适配器的 Go SQL 字符串及嵌入的 `.sql`。以本域查询、Go 的 `access.Principal` 引用和 SQL 注释为放行样本，以本次跨域读、带换行的限定表名及嵌入 SQL 中的跨域 JOIN 为拒绝样本。生成物由源覆盖。全局迁移、接线和测试取证按其职责单独核对。动态拼接的表名由维护者追到执行点，静态扫描结论标明覆盖范围。[现有检查](../../scripts/check-boundaries/main.go) · [目录约束](../architecture/004-repository-layout.md)

## 3 T02 将同一期间显示为同一组业务日期

| 入口 | 实施范围 | 对照位置 |
| --- | --- | --- |
| 记录详情、删除单条的「原适用期间」 | 复用 Go 已返回的 `effectiveDate/endInclusive`，统一展示“起日至最后适用日”或“自起日起”。 | [详情](../../apps/web/src/features/organization/DatedPages.tsx) · [删除确认](../../apps/web/src/features/organization/TimelineEditor.tsx) |
| 首条改日晚移的「无适用资料」、删除停用和恢复记录的「删除后的资料变化」 | 核对 `newNoFactRange` 和各对象 pair deletion 可见区间，Go 在对应读取/预览输出给出 `endInclusive`。同步 OpenAPI、生成类型和实际对象适配器。 | [预览投影](../../internal/contexts/organization/application/commands.go) · [成对删除显示](../../apps/web/src/features/organization/LifecyclePairDialog.tsx) · [公开合同](../../api/openapi.yaml) |

T02a 先统一详情与单条删除的期间展示。须同时具备日期读权和有效起日。有限区间使用 Go 提供的最后适用日，显式开放显示“自…起”。BU 记录的两个终点在失权时也可能为 null，其他对象可以省略字段。遇到字段失权就隐藏相应日期，有限段缺显示终点按不可用处理。前端不做日期减法。[BU 记录投影](../../internal/contexts/organization/application/commands.go) · [012 共享体验](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-history) · [013 技术合同](../specs/m02/013-m02-technical-design.md) · [026 日期格式](../adr/026-date-display-and-input-format.md)

T02b 处理其余两个入口。当前明确产生 `newNoFactRange` 的是 BU 和企业。为这两种预览定义只读区间，Go 沿用本域日历日计算。成对删除涉及 BU 的 `PairNameChange` 及企业、部门、地点、法律实体的 `*DataChange`。逐个核对 Go 投影和前端适配器，尤其是手工挑选字段的 BU 与企业。部门在日期失权时可以保留其他获准的变化，地点与法律实体现有成对动作还需读全相应日期及分类字段。分别验证该对象的授权形态。国家/地区、行政区划依其实际动作覆盖。[企业预览](../../internal/contexts/organization/application/enterprise/reads.go) · [BU 适配器](../../apps/web/src/features/organization/businessUnitAdapter.ts) · [企业适配器](../../apps/web/src/features/organization/enterpriseAdapter.tsx) · [共享页面类型](../../apps/web/src/features/organization/dated/presentation.tsx)

存储继续使用 `daterange [start,end)`。公共 `DateRange` 同时用于 `TimelineImpact`、已存回执及历史编码，显示字段只进入读取输出。分开比对旧来源和旧回执原件、当前 API 的授权投影和页面文字。操作生效日、筛选边界以及输入辅助的最小/最大日期仍按原含义表达。[原区间类型](../../internal/contexts/organization/public/contracts.go) · [历史编码](../../internal/contexts/organization/infra/history/codec_generated.go)

## 4 T03 将登记落到可核对的切片

以当前候选的路由、组合根、相应业务合同及最近一次实际验收交叉核对。将 `contexts.json` 的实际范围写至对象或切片，证据链接指向原验证报告。同步 004 的当前目录描述和文档入口中会被理解为已上线的表述。历史报告和原分支签收范围原样保存。[上下文表](../architecture/contexts.json) · [后端接线](../../internal/bootstrap/runtime.go) · [菜单](../../apps/web/src/app/App.tsx) · [文档入口](../index.md)

| 登记单元 | 应能区分的事实 |
| --- | --- |
| Organization / M02 | BU、部门、企业、国家/地区、行政区划、地点、法律实体按已有批次注明范围。G0 保留其时间背景。法律实体 L1 基础资料与后续用途/消费者分开。[地理与地点](../verification/205-g5-implementation.md) · [法律实体](../verification/206-legal-entity-implementation.md) · [企业](../verification/304-enterprise-maintenance.md) |
| Access / X03 | 标明当前实际使用的本地身份、会话及按操作/字段/范围授权，证据指向使用该能力的业务切片。[部门验收](../verification/116-r5-department-pages-and-acceptance.md) |
| Staffing / M03、M04 | M03 的职类、系列、职务、职级、职级阶梯、岗位按各自验收范围登记。M04 保留自己的状态。313 核编以其实际就绪程度单列。314 的 AC090 只有 308 单独签收，核编独验和并存联验由 313 首批接续。实施 T03 时复核同一候选的新证据，分别更新实际取得签收的分支。[职级阶梯验收](../verification/309-grade-g2-implementation.md) · [岗位验收](../verification/139-position-workspace-implementation.md) · [314 分批验收](../verification/314-job-function-t05-acceptance.md) |

具体改法是保留 17 个上下文和既有模块映射，在这三个已接入上下文的登记项中列明模块、实际能力范围和仓内证据。上下文汇总可表达“部分已实施”。同一上下文内，切片的验收状态分别取证。其他预留域维持原登记。004 与文档入口链接到这份现状，原 G0 计划与登记表中的首切元数据保留其历史用途。复核当前已执行的命令入口及生成客户端，再调整 004 中相应的“预留”描述。[现有登记检查](../../scripts/check-repository.py) · [首个切片验收](../verification/010-first-vertical-slice.md)

## 5 T04 地点与法律实体的历史流程

下一份实质修改 M02 历史写入、修订组装、同号恢复或按日历史维护的任务，在开工清单引用 140/T04。只要涉及两处重复规则中的一处，就在该批让两个真实对象调用同一实现，无需等待两边分别报错。若变更确属其他对象专有规则，须在该任务记录核对位置、原因、T04 仍待触发及接续角色。T02 的文字和读取投影单独交付。两对象的维护命令与历史组装可直接对读。[地点命令](../../internal/contexts/organization/application/location/commands.go) · [法律实体命令](../../internal/contexts/organization/application/legalentity/commands.go)

先核对现有 M02 `domain/dated` 与 `application/submission`。从两对象同时需要的同一段规则入手，例如按影响集保留或分配修订、删除前像与历史摘要。复用处限定在 Organization 内，传入明确类型并保留两条对象入口。地点独有地址采用、区划/邮编与岗位引用保护，法律实体独有登记国家/地区采用与法律身份，分别由对象负责。权限、持久化、提交终态、Outbox 和原采用继续按对象归属。[M02 按日合同](../specs/m02/120-dated-maintenance-technical-contract.md) · [企业地点维护合同](../specs/m02/201-country-region-geo-area-and-location-maintenance-contract.md) · [法律实体合同](../specs/m02/207-legal-entity-design.md) · [历史保护](../adr/025-greenfield-implementation-policy.md)

实施前先对各对象手算业务预期，再跑修改前后的同输入。比较不同样本时按对象、记录和来源的引用关系比对，使两轮独立生成的 UUID 或时刻不影响判断。同一库的旧修订、原采用和回执则以准确标识逐份核对原文或摘要。若某个公共步骤需要依对象种类分支来决定地址、登记地或授权，把该步骤留在各自对象内，只抽已证明相同的部分。[既有提交协调](../../internal/contexts/organization/application/submission/transaction.go) · [地点岗位引用检查](../../internal/contexts/organization/infra/postgres/location.go)

## 6 业务验收与异常

以下是实施时的独立预期。每批记录实际候选、角色、数据、首轮失败、修订与复验，在本批最终候选上签收。只有声称两项共同生效时才在同一候选联验。

| 编号 | 输入和操作 | 通过条件 |
| --- | --- | --- |
| A01 | 准备同一父级或搜索条件下至少 51 条可见候选，地点选择器每页 50 条。在稳定资料上翻页。再做两组独立变更，一组只撤非必需字段读权，另一组改变策略后恢复到完全相同的授权。 | 稳定时连续两页无重漏。两组旧游标均返回 409 `M02_CURSOR_CHANGED` 且无新页，页面清掉失效批次的选项，重试以原条件无游标取首页，按新权展示。核对浏览器请求、页面选项及取消零地址写入。 |
| A02 | 两个 PostgreSQL 连接用同步点交换读取和策略写入的先后。另在发起续页前撤掉必要的区划读取权，并分别测试会话失效、守卫读取故障。 | 先持读锁时写方等待读事务结束。写方先提交时新请求见新版本。失去必要读权按原顺序返回 403 `ACCESS_DENIED` 并清除候选，会话失效按原路径返回 401，数据库故障不返回候选或伪造版本。核对按域 SQL 放行和拒绝样本。 |
| A03 | 管理员建 2026-01-01、06-01、10-01 三条 BU 记录。查看六月记录，打开删除确认后取消。再看末条、跨闰日记录、查询者的受限字段。 | 六月记录两处均为 `2026-06-01 至 2026-09-30`，取消零写。开放末条用“自…起”。2024-03-01 的前一日为 2024-02-29。无日期权时无日期泄露。 |
| A04 | BU 的企业起日须不晚于 2024-02-28。另备企业样本，确保在 03-01 之前没有阻止后移的下游事实。分别把首条从 02-28 改至 03-01，查看影响再取消。成对删除选会产生资料变化的样本，按对象覆盖有限段、开放段、日期失权及无资料变化。 | 无适用资料为 `2024-02-28 至 2024-02-29`。成对删除在有变化时才列可见变化，区间终点依同一显示规则。检查服务端 JSON、适配后的实际页面、窄屏操作与取消无写。对可删除样本另行确认，按原来源读回准确修订，核对旧回执和历史原件。[原日期样例](../../tests/e2e/date-format.spec.ts) · [原删除样例](../../tests/e2e/delete.spec.ts) |
| A05 | 从登记正向跟到源码与证据，再从正式菜单或授权调用反查切片。分别假设 M04 随 Staffing 自动完成、只有合同的 313 随 308 通过、一个确已实施的切片缺少证据链接。 | 三例均能被识别并要求纠正。结构检查覆盖模块唯一、能力属于登记域和证据文件存在。维护者核对证据正文、候选及签收范围。只验过的切片才能标注相应验收结果。 |
| A06 | 地点、法律实体各设一条有独立后继的时间线。操作人更正前序、停用/恢复，删除可删中段并回看原来源。分别试缩短已被岗位采用的地点期间、在登记国家改名后读取原法律实体采用。 | 停用继承准确前序及其修订，独立后继和既有采用保持。受影响记录才增加修订，删除保留前像，回执区分被删目标和返回后选中记录。地点的引用约束生效，法律实体按原国家来源与当前权限投影。每次成功操作只有一份来源、操作和对象事件。 |
| A07 | 两对象各测无变更、取消、发起时无权、业务冲突、提交前存储故障、提交成功后丢响应、撤权后查回。另令同一主体在两个对象类型使用相同提交号。 | 无变更留终态且无新事实。取消与前置拒权均无新增业务事实或提交终态。可持久化的业务拒绝只留拒绝回执。提交前故障回滚且不发布临时结果。提交后按原号重送/查回只有一次写入。原号异载荷冲突，同号跨类型各有自己的结果。撤权后按当前权限裁剪或拒绝。[地点回滚及丢响应](../../internal/bootstrap/location204_acceptance_integration_test.go) · [法律实体竞争样例](../../internal/bootstrap/legal_entity206_races_integration_test.go) |

A07 的取证顺序是先在独立连接读取已提交的事实、头和终态，再调用恢复。提交前故障的样本，获权人首次用原号 `resolve` 可以落 `not_applied`，此后按原号已关闭处理。提交已成功但响应丢失的另一份样本，则应查回原成功结果。分别核对两份样本，避免恢复操作本身改变取证结论。[恢复合同实现](../../internal/contexts/organization/application/submission/recovery.go)

## 7 签收、资源与交接

T01 由 Access、Organization 以边界正反例、真实 DB/API 及地点选择器路径共同签收。T02 由 M02 与体验维护者核对授权字段、Go 投影、生成客户端、实际页面及 012 的视口/键盘要求。T03 由各域维护者提供切片证据，架构维护者复核链接、枚举及受影响的登记消费者。T04 随触发批次由地点、法律实体一起签收。结构检查采用 `make docs`、`make boundaries` 和 `git diff --check`，产品变更另按影响运行 `make check`、真实 DB/API 与浏览器场景。首次失败和未运行项分别留存。[实施阶段](../adr/003-implementation-stages.md) · [分层验证](../governance/137-actions-and-validation.md)

T01 可复用[地点现有翻页样本](../../internal/bootstrap/location_region_picker_integration_test.go)并增加权限变更及双连接情形。T02 从 `api/openapi.yaml` 运行 `corepack pnpm generate:api` 后检查类型和投影，更新 013、120 或相应对象合同中实际受影响的显示字段约定。实施验证使用 `make integration` 和测试自有 profile 的 `make smoke`，按影响记录所选用例。任一批次发现权限扩大、原件改变、重复提交产生新事实或登记提前签收，暂停该批交付，保留首失败，在修订候选上重验。

实施使用任务自有临时 profile。完成或中断时按 [017](../guides/017-local-bu-development.md#local-runtime-cleanup) 核对进程、容器和卷的回收结果。保留的用户验收环境及数据由原责任方管理。阶段结果写入对应验证记录，长期合同由 004、012、013、120、201、207 及各源域维护。全部触发任务验收并有接续证据后，登记维护者再办理计划关闭。
