# CHNHCM-017 BU首批本地运行与验证

登记信息：[registry.json](../governance/registry.json)。适用范围：本地创建、查询、资料新增/普通更正、停用/重新启用、分类删除、改日及有限联合修订/首条通用维护/分类纠错。依据[010](../archive/plans/010-m02-scope-and-plan.md)、[012](../specs/m02/012-m02-functional-and-interaction-design.md)、[013](../specs/m02/013-m02-technical-design.md)和[025全新实施](../adr/025-greenfield-implementation-policy.md)。当前启停证据见[T04报告](../verification/021-lifecycle-implementation.md)，资料维护证据见[第一批报告](../verification/010-first-record-maintenance.md)；[018多轮评审](../verification/018-multiround-review.md)和[018旧升级记录](../verification/018-source-history-simplification.md)保留原范围。

当前交互依[101](../adr/101-direct-save-and-form-assistance.md)取消全部手动预览，以下第5节历史环境记录仍保留原时点。

wt-dev-b当前日常开发与手工验证基准为[4390](http://127.0.0.1:4390)，采用模型5、profile `m02-current-vanke5-b`；2026-11-14最新模型5固定构建及维护边界见第22节。跨主机独立环境与共同基线见[134](134-parallel-development-baseline.md)。操作时显式指定profile。其余日期段中的环境状态保留为历史记录。


当前[103位置新增/插入](../adr/103-positioned-record-addition.md)已由[105](../archive/plans/105-positioned-addition-and-coverage.md)接线：按所选记录决定新增或插入，带出该位置的资料及默认日期；无可插入日期只禁用按钮。分层验证见[105报告](../verification/105-positioned-addition-and-coverage.md)，第5节历史环境记录保留原时点。

## 1 任务与可用能力

2026-11-03列表已按113/111切换为Community源码组件：紧凑控制、日期/状态面板、原生分页、服务端表头排序及低于900px摘要列，保留#066b68。桌面列菜单可显隐/改宽度，更多操作“列设置…”恢复全部隐藏；不再有列拖动/固定、Pro许可配置或水印。实际组件和合成HTTP结果见[本轮报告](../verification/111-community-t01-t02.md)，T03真实业务与七组条件另见[新报告](../verification/111-community-t03.md)；正常菜单仍为设置→组织配置→业务单元。单列条件增量由[本次验收](../verification/111-single-column-filter.md)记录：桌面列菜单“筛选…”或列表更多操作“列筛选…”进入；编码/名称包含或等于、状态单/多值、展示记录生效日期比较，一个条件与现有搜索/状态为且。编辑不查询，清除亦须应用，取消/Escape保留原条件和搜索草稿；仅应用后回第一页。Excel候选和跨列组合未接入。

管理员从“设置 → 组织配置 → 业务单元 → 列表 → BU详情”办理新建、过去/当天/未来资料新增和普通更正，查询日期事实、日志及只读确切来源；查询者只有读取能力。新增从所选记录继承未修改字段，后继独立快照保持；更正记录默认原日期及分类，可按各修改面权限一次修改名称/日期/分类，不填写更正原因。停用/重新启用已接线；分类删除和改日已接线；当前仅两个固定启停操作；首条维护及单条分类使用更正记录，不提供多目标关联编排，步骤见第6节。

## 2 前置及隔离

需要本仓依赖、Go1.26.0、Node24.19.0、pnpm10.24.0、Python3、Docker/Compose及Chromium。镜像锁为[PostgreSQL18.6-alpine](../../deploy/local/postgres-image.txt)，本地和Testcontainers一致；普通WAL、synchronous_commit=on、数据库UTC、业务Asia/Shanghai。

每个 profile 按工作树路径和名称确定 Compose 项目、卷及仓外凭据目录。首次启动先检查三个端口，相同 profile 再启动沿用原端口、数据库和身份。新工作树在未指定环境时使用 `default`，未配置端口时使用 4380/8380/54370。T07 采用下文的独立配置。

当前 `wt-dev-d` 日常使用 `m03-current-4486-d`，Web 4486、API 8486、DB 54486。在该工作树运行 `python3 scripts/local-bu.py connection` 可查看实际 profile 和端口，运行 `python3 scripts/local-bu.py verify` 可核实在线服务。指定 `CHNHCM_LOCAL_PROFILE` 始终优先。将一个已登记的持久环境设为本工作树默认值，可运行 `CHNHCM_LOCAL_PROFILE=环境名 python3 scripts/local-bu.py set-default`。选择保存在仓外，其他工作树沿用各自的配置。[4486 切换及原环境恢复](../verification/315-department-quota-allocation.md#6-统一验收环境-4486)

日常开发每个工作树通常复用一个当前模型的持久profile；普通代码修改不逐批新建环境。`make smoke`默认新建独占临时profile、自动选择端口、启动当前模型空库和Web/API，验收结束后回收；不使用当前手工验证库。特定故障或容量现场确需留存时才另建持久profile。既有环境保持原分类，临时属性不能用于将已有持久库改成可删除。

128／130 批次按[128决定](../adr/128-authorization-evidence-removal.md)和[130方案](../archive/plans/130-department-storage-optimization.md)安装00001身份及必要认证/安全事件、00002 BU紧凑历史、00003 BU导出授权、00004账号身份、00005部门紧凑历史及关系、00006字典引用侧索引、00007由Go生成的确切来源分段读取索引；00008增加选择草稿。该批 API 要求 m02-compact-5/read-segment-4/query-seal /3 及连续 00001—00008；新开发环境使用当前迁移空库。4390已用00008，但已应用00002未重跑，不能将其原DDL视为新环境当前迁移的逐字副本。同模型完整前缀仅由显式upgrader推进，不同模型、空洞与未知版本拒绝；既有m02-compact-1、m02-department-1、m02-timeline-7等库保留原数据和匹配程序，不自动升级或清空。历史迁移过程保留在原报告与Git；126原结果见[126实施报告](../verification/126-compact-history-implementation.md)，当前删除授权档案和独立候选验证见[128实施记录](../verification/128-implementation.md)。

## 3 启动与操作

### 3.1 启动独立环境

```bash
cd /home/vivobook/projects/CHNHCM-wt-dev-a
export CHNHCM_LOCAL_PROFILE=m02-t07-business
python3 scripts/local-bu.py configure --web-port 4527 --api-port 8527 --db-port 54607
make install
python3 scripts/local-bu.py connection
make local-up
python3 scripts/local-bu.py verify
python3 scripts/local-bu.py credentials-path
```

00058 起，普通新账号可以登录，业务授予为空。显式可丢弃的开发实例首次创建样例时才授予管理员和查询者的样例权限。已有持久实例按实际授权迁入，重复启动保留每个账号现有的权限。国家和地区目录导入使用现有账号权限，无权执行的包会在启动输出中显示跳过。[027 切换验证](../verification/027-public-grant-cutover.md)

开发过程统一使用公开登录预设：管理员`admin`、查询者`reader`，密码均为`admin1234567`，无需逐次人工查询。此约定仅适用于本地开发；生产、客户和共享交付环境另行配置。数据库管理员及API连接密码仍由脚本随机生成，运行配置仅保存在仓外`~/.local/state/chnhcm/<摘要>/runtime.json`，目录0700、文件0600；真实凭据不复制进仓库、日志或截图。迁移管理员与API角色分离；前端不接收数据库凭据。

`up`会将当前模型既有开发账号同步为统一密码，保留账号身份、授权和业务历史；首次更改会使对应旧会话失效，重复执行不再次改密或退出会话。仅改在线环境的登录密码时，在选定profile后运行`python3 scripts/local-bu.py login-passwords`，不停止服务。旧模型环境不直接运行新程序，应保留其数据库和程序。

2026-11-05，b工作树当前日常环境为`m02-department-b`：页面`http://127.0.0.1:4382`、API8382、DB54373，模型m02-department-1。管理本次环境先`export CHNHCM_LOCAL_PROFILE=m02-department-b`；[部门样本记录](../verification/116-r4-local-department-samples.md)提供1个BU及5个部门、当前/历史/未来/停用读取链接和实际核验。R5部门菜单尚未接线，当前登录后通过API查看。旧m02-current-b已正常停止，7个BU及原程序/卷保留，未迁移到新库；二者端口相同，不能同时启动。当前环境继续供用户验证，由wt-dev-b维护者负责，在用户完成验证或下次版本交接时复查。以下日期段保留此前环境事实。

2026-11-02，b工作树已启动当前模型独立profile `m02-current-b`：页面`http://127.0.0.1:4382`、API8382、DB54373。后续操作先`export CHNHCM_LOCAL_PROFILE=m02-current-b`再执行本页命令；在线状态仍须以`verify`为准。此前`m02-more-actions`的旧模型数据库及程序已停止并保留。2026-11-03按用户指令重新编译重启此profile的API，前端及数据库保留；T05再次从管理员/查询者正常菜单验证“名称包含丙”，均HTTP200并命中1条，见[交付记录](../verification/111-community-t05-delivery.md)。此环境供用户日常验证继续运行，由当前工作树实现/验证维护者负责，用户结束本轮使用或下次服务/版本交接时复查；隔离验收用的4393/4394已停机保留数据，不替换日常库。

原T07录制时页面为[独立本机入口](http://127.0.0.1:4527)，API绑定127.0.0.1:8527，数据库绑定127.0.0.1:54607。URL是否仍在线须以本地status/readyz为准；配置入口不自动证明服务正在运行。

### 3.2 正常菜单及结果

1. 登录，桌面从“切换应用 → 设置 → 组织配置 → 业务单元”进入；窄屏使用左上导航抽屉。
2. 列表“新建”填写编码、名称、生效日期并明确选择操作原因，保存到首条。未来首条显示未生效，不冒充今日有效事实。
3. 生效日期轴选中最新一条，点击“新增记录”在其后追加；选中非最新一条，点击“插入记录”在该条与紧邻后一条之间插入。自动带出所选记录的名称：新增默认取今天与所选日期次日的较晚者，插入默认次日；日期限定在该位置内，无可插入日期只禁用按钮。操作类型固定资料变更，原因留空，须自行选择；英文名称可直接清空。
4. 填写后直接保存；日期默认及必要影响自动带出。保存后定位实际记录，列表返回条件保持。
5. 生效日期轴选中记录，再从标题旁“更多操作（⋮）→ 更正记录”进入；名称及生效日期同页，业务分类按需展开，原日和分类为默认，日期保持原相邻位置。更正不新增时间轴节点。
6. 来源说明显示记录/修订/来源、原办理校验日与实际记录时刻。变更日志区分新增/更正，显示准确目标及名称前后值；“查看当次来源”只读，返回当前资料须重新读取权限和版本。
7. 按[123](../adr/123-current-maintenance-and-dated-query-boundary.md)，普通列表和详情没有查看日期入口。列表显示今日资料，未来首条以未生效可查；详情普通进入选今日记录（或未来首条），时间轴和来源链接按明确记录定位，名称/状态/适用期间一致。未来记录显示尚未生效，历史来源保持只读。
8. 单条停用/重新启用：选中当前末条，在更多操作进入与该条生效后状态相符的动作。日期默认取服务端今天与末条次日的较晚者，须在Go返回范围内；恢复名称进入即带出，可按权调整，原因留空待选。非末条用固定成对操作处理停用期间；自动业务错误就地展示，取消初始默认不提示，真实编辑受保护。
9. 返回列表保留筛选、排序、日期、页码、选择和滚动。取消、切换日期/对象、离开或退出有未保存保护；不持久保存表单正文。

版本冲突后保留仍有权的输入，点击“重新核对当前资料”，对照当前资料后另次保存。不能只换令牌自动重放。原提交未知时先“查回提交结果”，同页仍有原正文可“按原内容重试”；刷新只恢复实例/主体/提交引用，不恢复表单值。已有终态按当前对象/历史/字段读权返回，仅撤写权不阻断；撤读权清受限正文并保留待确认引用。迟到回调不能覆盖已确认结果或新表单。

### 3.3 检查与停止

```bash
make generate
make check
make build
make local-up
make integration
make smoke
python3 scripts/local-bu.py status
python3 scripts/local-bu.py restart-api
make local-down
```

除`make smoke`外，上述本地环境命令沿用本节export配置。make integration每例独立Testcontainers；make smoke使用自动创建的独占临时profile及合成BU，默认桌面Chromium，单worker。使用`make smoke SMOKE_ARGS='tests/e2e/bu.spec.ts --grep 关键词'`做定向验证。确需针对已选定的持久profile复查时，先verify，再运行`corepack pnpm test:smoke`并传相同Playwright参数；该入口不自动回收持久库，不与手工登录或同profile其他套件并跑。登录遵守20次/分钟限制，不放宽产品限流；Playwright关闭trace和可能包含密码的自动文本快照，失败截图遮罩账号及登录输入。make load为后置专项，不是本批综合容量验收。

local-down仅停止本profile管理的进程和容器，保留容器和卷；程序/存储合同一致且配置完整时，重新local-up核对相同初始化，不创建重复业务事实。旧存储须按原配置及匹配程序恢复，不能直接用当前构建重启。脚本日志按profile位于.local/runtime；只允许非敏感输出。受控维护通过仓外连接运行 upgrader 的 `access-change`。停用指定 Instance、Login 和 Operation=disable。逐份撤销指定 Operation=revoke 和 GrantID。旧 Action 参数仅在唯一匹配一份单动作授予时可用，重叠授予或多动作角色须明确 GrantID。客户日常管理入口由 027 T04 接入。批次结束的盘点及收尾见第3.5节。

### 3.4 T07专项复现

容量与业务页面分开：`m02-t07-capacity`为API8517/DB54597、Web4517保留但不启动；`m02-t07-business`用于正常菜单。容量数据全部合成；完整准备约10万次API提交，保留约196万来源成员和授权审计，需预留足够磁盘与执行时间。测量时不得并跑浏览器、其他负载测试或初始化。

```bash
export CHNHCM_LOCAL_PROFILE=m02-t07-capacity
python3 scripts/local-bu.py configure --web-port 4517 --api-port 8517 --db-port 54597
python3 scripts/local-bu.py up --no-web
# seed只用于本profile的新空库；测量会追加来源，之后不得用seed回退或覆盖历史。
# 各输出路径必须不存在；既有证据不覆盖。
go run ./tests/capacity -phase seed -out .local/t07-seed.json
go run ./tests/capacity -phase history -out .local/t07-history.json
# 大批追加完成后执行普通可见性/统计维护，不改业务历史；冷测仍重新退页。
go run ./tests/capacity -phase maintain -out .local/t07-maintain.json
go run ./tests/capacity -phase plans -out .local/t07-plans.json
go run ./tests/capacity -phase measure -cache hot -out .local/t07-hot.json
CGO_ENABLED=0 go build -o .local/bin/t07-cache ./tests/capacity/cache
go run ./tests/capacity -phase measure -cache cold -out .local/t07-cold.json
# 当前正式单对象场景为20条；正常API创建、首中尾更正及重启恢复。
go test -tags=integration ./internal/bootstrap -run '^TestModel5Timeline20Maintenance$' -count=1 -v
# 20条取消／存储观察专项，单次日志不作为P95。
CHNHCM_T07_PRESSURE=1 go test -tags=integration ./internal/bootstrap -run '^TestT07CurrentTimeline20Records$' -count=1 -timeout 15m -v
CHNHCM_T07_CLOCK_BROWSER=1 CHNHCM_MIDNIGHT_BROWSER=1 go test -tags=integration ./internal/bootstrap -run '^(TestT07BusinessDayBoundaryBrowser|TestJointMidnightBrowser)$' -count=1 -v
```

原T07在两个新建任务库试验覆盖索引后，将其收敛到原有日期唯一索引的INCLUDE列，保留原唯一键并移除当时新增的重复试验索引；该段为原批次记录。重现须直接从最终当前迁移建立空库，旧profile不得自动升级或清卷。当前Requirements及Grant由Go按请求核验，128已删除永久判权过程档案，原T07审计数量只作历史证据；API池保持12连接，性能数据不能泛化为所有查询/硬件的SLA。

冷测只停止该容量profile的DB，核对其专属卷后清退卷内数据文件页缓存并以mincore核验，再启动同库；不清主机全局缓存、不删数据或卷。定义是“启动前数据文件页冷”，DB启动和ready会重新暖基础页；不宣称硬件及宿主全冷。20条及时间专项使用独立Testcontainers；正常20条检查经公开HTTP逐条准备，另一个取消／观察专项保留应用事务夹具。正式请求仍受64KiB/10秒预算，超限拒绝不算保存成功。时钟测试仅替换任务内Go时钟，不改主机时间。

当前程序存储合同为m02-compact-4，迁移履历必须恰为已应用00001—00007；同模型完整前缀仅由显式upgrader推进，缺履历不补造。00004保存账号身份，来源身份快照由各对象强类型历史持有，00005持有独立部门历史块及关系头，00006补充字典引用侧索引，00007采用read-segment-3派生索引。已有持久环境不自动填充索引；本轮只对已停止API的自有临时候选显式重建，新固定候选从空库初始化，不能将新程序直接接入缺索引的旧环境。既有m02-compact-1、m02-department-1、m02-timeline-7等持久环境保留原程序和数据，不自动重建，不将新程序接到不匹配库。00001中途失败按该文件事务回滚，去除故障后可在该空库重试；00002失败可留下已完成00001，程序与upgrader保持关闭，应保留故障库并另建隔离空库，不能手工补履历绕过检查。

<a id="local-runtime-cleanup"></a>
### 3.5 开发与测试环境收尾

通用要求由[003第4.1节](../adr/003-implementation-stages.md#development-environment-cleanup)持有。已明确留存的容量、故障或旧批次持久环境完成验证后停机；创建时明确可丢弃的环境回收容器、网络和卷。当前用于用户验证的profile可继续运行，在交接中写明入口、用途、责任角色和复查节点。历史报告里的“环境保留”表示数据和复现依据保留，不要求数据库与Web/API永久常驻；普通验收保存报告和源码即可，不默认冻结整套环境。

1. 以`python3 scripts/local-bu.py list`盘点CHNHCM各工作树profile：工作树、profile、用途、状态、端口、网段及temporary属性。列表复用Compose标签和仓外`profile.json`/`ports.json`，不读取含凭据的runtime.json；旧环境无元数据时按持久环境显示，无法还原profile名时显示项目摘要，缺失的Web/API端口不猜测，数据库端口可由实际绑定补足。结合`docker ps`、`docker compose ls --all`及目标profile的`connection/status`核对。不要直接输出完整docker inspect、进程环境或仓外凭据。
2. 保存必要日志和失败证据，确认持久卷、仓外配置/凭据、匹配的程序或构建来源可用于恢复。若旧程序文件已被覆盖而仅剩运行中的可执行文件，确需保留该版本时先归档所需程序并核对摘要；归档不得包含进程环境或凭据。
3. 在目标工作树使用其原 profile 执行受控停止。下面命令以已结束的容量 profile 为例，`down` 实际停止该 profile 的配套进程和 Compose 容器，不删容器或卷；不要套用到仍供用户验证的 profile。

```bash
CHNHCM_LOCAL_PROFILE=m02-t07-capacity python3 scripts/local-bu.py connection
CHNHCM_LOCAL_PROFILE=m02-t07-capacity python3 scripts/local-bu.py status
CHNHCM_LOCAL_PROFILE=m02-t07-capacity python3 scripts/local-bu.py down
```

4. 旧环境若缺少现行端口/进程记录，或脚本报告归属失配，不修改迁移、伪造配置或删除卷以绕过检查。先依据原记录和实际标签/绑定/挂载逐项核对，再对确切容器ID执行停止；配套进程也须通过身份核对后定向停止。无法确认的对象列入待处理，不能标为已清理。
5. 停止后用 `docker ps` 对照运行清单，用目标容器的 `docker inspect --format` 只读取状态和挂载，核对持久卷仍存在且挂载身份不变；确认目标进程/端口释放。对保留的当前 profile 执行 `python3 scripts/local-bu.py verify`，记录停止对象、保留对象及理由、恢复依据和核验结果，不仅记录资源总数。

`make smoke`在启动前记录临时属性，成功、启动/测试失败及可处理取消均进入finally回收；取消时先终止本任务浏览器进程组，再回收Web/API及数据库。清理失败保留元数据供重试并单独报错，不把原测试失败改记成功。CI使用固定任务profile `ci-browser`，并用always再执行一次幂等清理。Testcontainers仍使用已有t.Cleanup/Terminate路径，本次不声称其全部中断场景均已补验。

2026-11-02 收尾后，当前用户验证入口为4527，责任角色为本批实现/验证执行者，下次新建验证环境或任务交接时复查是否仍需运行；wt-dev-b和CubeBox的环境因未纳入本次已确认旧环境范围而保留。此前名称、T04—T08及T07容量profile已停机，容器/卷和复现资料保留；此次实测清理与规范核对见[记录](../verification/003-development-environment-cleanup.md)。

### 3.6 临时环境与失败重试

默认自动验收直接执行`make smoke`，不必预先启动数据库或手工选择端口。程序输出本轮实际profile、页面、API及DB端口；这些入口只在本轮运行期间有效。非敏感日志和遮罩截图保留在.local，临时构建、仓外凭据及配置在成功回收后删除。

确需由任务脚本手动管理独占临时环境时，在首次启动前声明：

```bash
export CHNHCM_LOCAL_PROFILE=accept-my-task
python3 scripts/local-bu.py configure --temporary --purpose '本任务独占验收' --web-port 4560 --api-port 8560 --db-port 54760
python3 scripts/local-bu.py up
# 执行本任务验证；任务脚本使用finally/trap调用destroy，不能只依赖末尾顺序命令。
python3 scripts/local-bu.py destroy
```

`up --temporary`也可标记尚未初始化的命名profile；已有持久数据库拒绝转换，默认profile不能标为临时。`down`始终只停止并保留，`destroy`只回收准确选定且事先标记的临时profile，无逐次确认，不清理其他工作树。启动只完成部分分配、未生成数据库凭据或只剩网络/卷时仍可重试destroy；回收后重复调用不报错。SIGKILL、断电等无法运行finally时，通过list定位本任务残留，并在所属工作树执行`CHNHCM_LOCAL_PROFILE=实际profile python3 scripts/local-bu.py destroy`，不执行全局prune或按时间批量删除。

### 3.7 Docker地址池

同一宿主机统一配置默认地址池，各profile继续使用Compose自动网络，不手工逐批指定IP。候选[配置](../../deploy/local/docker-daemon.example.json)使用`10.246.0.0/16`、`size:24`，可分配256个网络；选择时核对Windows/WSL、VPN和虚拟机当前路由。Docker默认大子网及容量说明见[官方文档](https://docs.docker.com/engine/network/#subnet-allocation)。该配置仅为宿主机开发设置，不进入客户部署合同。

由宿主机维护者将地址池字段合并到`/etc/docker/daemon.json`并执行`sudo dockerd --validate --config-file=/etc/docker/daemon.json`后应用。日常up/smoke不改宿主机配置。当前机器已配置上述地址池，并启用live-restore，显式保留原`bip:172.17.0.1/16`；本次先reload启用live-restore，再重启应用地址池，运行中数据库的PID、启动时间及挂载核对见[验证报告](../verification/017-docker-runtime-lifecycle.md)。已有网络不重新编号，也未清理历史环境。live-restore适用条件见[官方说明](https://docs.docker.com/engine/daemon/live-restore/)，不能据配置声明所有重启场景无中断。

### 3.8 126保留紧凑案例与原大库退役

版本保护：4387及4388保留原m02-compact-1程序和数据库，当前工作区为m02-compact-4。不得用当前脚本对这两个profile执行up、restart-api或migrate并覆盖原程序；下面的启动命令保留126/127当时记录含义。原页面采用共享工作树Vite，不能把保留API/DB称为前后端全栈冻结。当前模型的独立观察候选见第12节。

2026-11-07按用户“启动以上案例”及“用紧凑库，而清理过度大的原库”的要求，建立持久profile `m02-compact-case-b`：页面[http://127.0.0.1:4387](http://127.0.0.1:4387)、API8387、DB54387，模型m02-compact-1。管理员admin、查询者reader沿用第3.1节本地公开登录预设。两个账号均已在真实浏览器从正常菜单打开BU及部门列表/详情，查询者的新建动作禁用。

该环境重建原2023BU、22104部门、1104206来源及36490条原审计；2026-11-07原整理后测得整库约277.85MB，包含当时登录/核对增加的审计和全部索引；这是测量快照，后续访问会继续增加审计与体积。此处是原冻结案例的持久运行，不重建上一轮可丢弃测量末尾新增的20个对象。旧r6-capacity-b-1106中的chnhcm及r6_baseline_completed_125、容器和卷已按用户要求退役，WSL文件系统净释放约115.24GiB；旧对照追加测量数据随旧库退役，原报告和结果保留。原日期段中的停机保留状态属于历史快照。

提交核验时该profile仍保留126原API及迁移00001—00005，没有执行后续00006或25229e7列表修复；当前源码要求00001—00006，与第11节万科固定库已运行的版本区分。`status`/`verify`只核对现状；`up`会按当前源码重建程序并调用upgrader推进同模型迁移，不能视为原二进制不变的重启。后续升级应另记录前后业务身份、来源与回归，本次文档收尾不执行升级。原部门截图停在加载中，本轮另补实际数据行可见的四张列表截图，结果见[126提交核验](../verification/126-compact-history-implementation.md#10-提交整理与保留环境核验)。

```bash
CHNHCM_LOCAL_PROFILE=m02-compact-case-b python3 scripts/local-bu.py status
CHNHCM_LOCAL_PROFILE=m02-compact-case-b python3 scripts/local-bu.py verify
CHNHCM_LOCAL_PROFILE=m02-compact-case-b python3 scripts/local-bu.py up
CHNHCM_LOCAL_PROFILE=m02-compact-case-b python3 scripts/local-bu.py down
```

该profile从创建时即为持久属性，供用户本轮手工验证，由wt-dev-b维护者负责，用户结束验证或下次版本交接时复查。down正常停机并保留数据，up复用同一库和端口。转换用临时库、卷及仓外私有导出已回收。完整表摘要、恢复后的历史保护反例、首次失败/修复复验及清理结果见[126报告第9节](../verification/126-compact-history-implementation.md#9-持久案例启动与原大库退役)。本次未执行WSL离线压缩。

## 4 失败与恢复

| 现象 | 处理及判据 |
| --- | --- |
| 端口占用 | 选择新profile与空闲端口，不停止其他任务 |
| 新筛选报未知或重复查询条件 | 先核对请求是否重复、目标profile与实际Go进程；Vite会更新前端，Go程序需要重新编译。若是旧API未识别新增白名单，在确认同一当前存储模型后执行`CHNHCM_LOCAL_PROFILE=m02-current-b python3 scripts/local-bu.py restart-api`，再用正常菜单验证真实查询；不靠删除条件、关闭白名单或清库解决 |
| 存储合同不匹配 | 新任务使用空库；保留原库及旧程序，不执行兼容升级或清库 |
| readiness503或启动关闭 | 检查当前基线、企业/身份初始化；不绕过程序匹配检查 |
| 结果待确认 | 保留原身份查回；pending、503、超时和权限拒绝都不能据此断言未保存 |
| 当前写权丢失 | 禁止新执行，仍可按当前读权确认原终态 |
| 当前读权丢失 | 清受限输入/正文，保留引用；不显示旧缓存冒充已授权 |
| 来源不存在/采用未接入 | 分别说明，不回退最新资料；只有显式返回当前才改模式 |
| 会话查询临时失败 | 保留cookie，依赖恢复后原地重试；真实401才重新登录 |

## 5 范围限制与历史

T01/T02的分类、联合候选与多目标持久化已随T08接线；T03—T06及T08已实施，分层证据见[T08记录](../verification/022-joint-implementation.md)。T07已完成本版BU采纳范围的代理业务验收与技术验收：241整例通过、3例仅本域参数通过、9例真实消费者前置未接；另4例退役不计通过，见[T07报告](../verification/010-t07-acceptance.md)。Outbox为源落库且未投递，M03/M04消费者未接入，不声明跨域采用、生产备份恢复、对外部署或全浏览器验收。原[首批失败](../verification/010-first-vertical-slice.md)、[018来源调整](../verification/018-source-history-simplification.md)、[019图面对照](../verification/019-cubebox-experience-migration.md)和[多轮复测](../verification/018-multiround-review.md)保持原证据，不能扩大为当前批次通过。

名称输入采用Go同源规则：支持文字、数字及允许的名称标点，禁止emoji和非法不可见字符；新输入NFC后最多200码点。旧来源按原文只读，不能因新规则清洗历史。名称专项使用独立profile `m02-name-current`（页面4406/API8406/DB54486），沿用本页启动命令替换profile并用configure设置对应端口；原m02-first-current及4396环境保留。当前专项证据见[名称规则复测](../verification/011-organization-name-policy.md)。

此前名称专项复测通过时，4396原环境API刷新为当时的名称实现（不含T04）；TL_04a597a4通过普通更正追加来源3，来源2保留原emoji。可在列表搜索此编码检查200码点边界；常规名称旅程可在4406搜索NAME_前缀。

此前T04使用m02-timeline-3存储合同，4416独立空库；旧4396/4406环境不迁移、不清空，保持原二进制及持久库，不能用新程序直接接旧合同。正常入口停用和重新启用均需先选日期及原因，再预览/确认；今日启用仍可安排已有未来停用后的重新启用。状态按业务日自动解释，无需到日再次写库。当前证据见[T04实施报告](../verification/021-lifecycle-implementation.md)。

T04评审修复使用独立profile `m02-t04-fixes`（页面4436/API8436/DB54516），沿用第3节命令替换profile并用configure设置对应端口。菜单及存储合同保持；该环境包含本次修复，4416保留原实施版本。字段撤权重读会清默认、预览和父详情/日志缓存，仍可读输入保留，该T04批次当时需重新预览后保存；当前按101自动辅助并直接保存。正常菜单及撤权专项证据见[评审修复复测](../verification/021-lifecycle-review-fixes.md)；浏览器权限夹具在finally恢复原授权，不与手工验证并跑。

a整合main的当前UI验证使用独立profile `m02-main-ui-integration`（页面4456/API8456/DB54536），m02-timeline-3空库初始化。原a的4436/4416及b的4382环境不清空或接入新程序。新增为主按钮，停用/重新启用及普通更正从标题旁更多操作进入；编辑沿用紧凑标准表单，自动记录名称输入，不恢复字段勾选或常驻机制说明。证据见[本轮整合验证](../verification/021-main-ui-integration.md)。

T05使用当前m02-timeline-4，仍为00001/00002空库基线；新增按类别的删除专用权限，不把旧库接给新程序。此前4456/4436/4416等环境保留原存储/程序及数据。所选非首条记录从“更多操作 → 删除此记录”打开短表单，日期/分类只读；当前按101直接“确认删除”，只读辅助解释必要影响但不作为保存许可。首条无入口；非法后继拒绝。保存后选中原日期的新适用前段；变更日志可看删除前原件或当次来源。丢响应后查回定位准确删除修订，历史原件和当前资料分开，详细证据见[T05报告](../verification/020-delete-implementation.md)。

T05评审修复使用独立profile `m02-t05-fixes`（页面4476/API8476/DB54556），此前第3节曾采用该配置；当前第3节转到T08独立配置。4466保留原实施版本，存储合同仍为m02-timeline-4。拒绝处理在外层提交前取消或提交失败时，页面保留待确认引用；同页可按原内容重试，刷新后可查回，只有确认终态才解除引用。专项故障由测试夹具限定到合成BU并在finally清理，不能与手工验证并跑；见[修复报告](../verification/020-delete-review-fixes.md)。

T06使用m02-timeline-5空库及四类日期专用权。当前正常菜单在所选记录的“更多操作 → 更正记录”编辑生效日期，可与名称/显式分类一笔保存；首条后移明确预览新增无事实期间。仅日期修改保留原资料及分类；更正记录显式组合名称/分类时按各面核验，保存选中同一记录的新日，旧来源仍读旧日；同日无变化也必须核验版本和日期写权。4476/4466等旧环境保留原库和程序，不直接运行本批迁移。原技术验证及样本见[T06记录](../verification/020-redate-implementation.md)；两项发现修复与独立复审通过见[独立评审](../verification/020-redate-independent-review.md)。

T06独立复审修复保持4486及m02-timeline-5，未改迁移或清库。改日首次结果、精确重试及resolve均返回准确原日previousEffectiveDate和本次新日effectiveDate；日期读权撤销时两者同时裁剪。来源冲突后重新核对会清除已失读权的日期/原因及旧重试正文，仍可读输入保留；该T06批次的仅日期命令在日期不可读时禁止隐藏改日；当前104统一更正会清除日期输入及修改面，仍可按现行权限更正可读可写名称。


原T08使用`m02-t08-final`与`m02-timeline-7`空库；4496为该批早期环境，4486为原T06环境，原数据库/程序和证据保留，不据此重新开放当前通用写路径。旧联合编辑、首条补全及分类专用入口的原步骤仅见对应历史报告；109当前只提供第6节两个固定操作。首条维护继续普通更正及按需另次新增/插入。

联合保存使用整笔提交身份，查回原汇总和逐项结果，不逐项重试。重新核对保留当前仍可读输入，清除受限字段及原重试正文；刷新仅保留原提交引用，不恢复业务输入。程序和存储必须匹配当前标识，变更后的开发迁移只在新隔离空库安装。当前验证和独立复审通过结论见[T08记录](../verification/022-joint-implementation.md)。

联合整笔同值返回“没有修改内容”，保持手选记录；同值新增没有新记录身份。显式重新核对期间等待当前资料到达后再继续预览，不沿用旧来源。

2026-11-02 T07验收：本版正常菜单、分层矩阵、冷热容量与长线取消已完成，实际结果和可查看样本见[T07交接](../verification/010-t07-acceptance.md)。专项恢复按以下命令串行执行，不与完整smoke或授权变更并跑；脚本只允许本任务业务profile，使用仓外凭据，结果目录须为新目录以保留原件。

```bash
export CHNHCM_LOCAL_PROFILE=m02-t07-business
export CHNHCM_EVIDENCE_DIR=.local/verification/t07/recovery-new-run
mkdir -p "$CHNHCM_EVIDENCE_DIR"
node tests/scenarios/t07-recovery/tl162.mjs
node tests/scenarios/joint-rejected-recovery.mjs
node tests/scenarios/t07-recovery/tl164.mjs
node tests/scenarios/t07-recovery/tl174.mjs
```

2026-11-02统一更正实施及验证见[104报告](../verification/104-unified-record-correction.md)。该104批次未实施103位置新增，后由105接线；联合入口/历史停用对入口保持。4382原库保留，测试采用独立profile，不以测试数据库替换用户环境。

当前103产品接线由[105计划](../archive/plans/105-positioned-addition-and-coverage.md)实施：选最大生效日记录显示新增记录，其他当前成员显示插入记录；默认带出所选名称与位置日期，类型固定资料变更、原因留空。插入限定相邻日期之间，无可用日期仅禁用按钮。日期手填及保存仍由Go重验，初始带值可直接取消。实际分层结果见[105验证](../verification/105-positioned-addition-and-coverage.md)，原运行记录保留。

首条维护当前按[107](../adr/107-first-record-maintenance.md)办理：例如6月乙先更正为1月甲；确需6月起乙，再另次新增/插入6月乙。两次来源各自保留；107当时保留的其他联合入口现由109退出；本批验证见[107报告](../verification/107-first-record-maintenance.md)。

## 6 当前启停期间专用维护

109将通用联合写入口收敛为两个固定弹窗：插入停用与重新启用只填写两日期/原因，一次保存；删除由当前所选S/R定位另一条，确认两原件及实际可读资料变化，一次删除。无项目编排、类型/名称输入、保留名称或补建选择。单条更正及首条两步继续；原T08/T07环境、报告和旧通用来源均保留历史。当前页面/数据库和测试profile以109报告实际运行记录为准，不凭默认端口认定新功能已运行。

## 7 Community源码与组件检查

当前使用packages/mui-x-data-grid的固定官方TS/TSX，Vite负责转译；官方同版声明仅供TypeScript。React/MUI Core及基础依赖保持锁定发布包，构建自足。临时组件检查入口为test-fixtures/datagrid-community.html，包含合成页及实际虚拟行返回，不是BU业务菜单或真实Go/DB验收。

无需配置商业许可密钥。Pro源码包、x-license、初始化、许可环境声明、releaseInfo插件和datagrid-pro旧夹具已退出运行；历史源码归档、许可证、报告及112清单保留。运行检查使用实际Community，不隐藏水印或过滤console；组件脚本不启动BU profile；真实T03使用独立profile，实际验证入口及停止记录见T03报告。

## 8 前端构建与可选来源核对

纯前端生产构建执行`corepack pnpm build`，不要求本地API/DB profile、运行端口或Python配置就绪；类型检查继续作为构建检查。实际启动dev/preview或业务浏览器检查时才读取本地连接并核对归属，避免连接错误环境。

显式`corepack pnpm verify:mui-source`核对Community运行源码、官方同版声明及两个归档的摘要（501＋500项），不作为每次build门槛。正常`corepack pnpm build`执行应用类型检查和Vite源码构建。定向检查可在构建后运行`node tests/scenarios/datagrid-list-controls.mjs`、`node tests/scenarios/datagrid-community-return.mjs`及`node tests/scenarios/datagrid-list-return.mjs`；均为任务自有临时静态服务/Chromium并自动收尾，分别使用合成HTTP、实际网格/合成页及延迟渲染钩子，不代替真实Go/API/DB。
## 9 Community列表T03复验

先按第3节选择本工作树的隔离profile并核验实际入口；本次m02-community-t03-b验证时使用Web4393/API8393/DB54393，交付前已停止，持久数据保留。不得据此宣称该入口当前在线。

```bash
export CHNHCM_LOCAL_PROFILE=m02-community-t03-b
python3 scripts/local-bu.py up
python3 scripts/local-bu.py verify
corepack pnpm exec playwright test --config tests/e2e/playwright.config.ts community-list.spec.ts
node tests/scenarios/community-list-viewport.mjs .local/111-t03-matrix-repeat
node tests/scenarios/community-list-native-links.mjs .local/111-t03-links-repeat
python3 scripts/local-bu.py down
```

业务用例通过当前API建立唯一前缀的73个合成BU，并写入非敏感fixtures.json；后两项必须在同一profile运行，复验输出目录使用未占用路径。矩阵需要Xvfb、libX11、libXtst及xwininfo；原生复制地址使用当前Chromium上下文菜单与独立X显示的剪贴板。实际结果、限制及资源记录见[111 T03](../verification/111-community-t03.md)。本节不开放列条件或扩展为全产品浏览器验收。


## 10 全查询结果选择与所选CSV

按[114](../adr/114-bu-query-selection-and-export.md)及[115](../archive/plans/115-bu-query-selection-and-export.md)，列表最左侧表头直接选择服务端当日和已应用条件下全部获权BU，覆盖所有页；翻页、20/50/100及仅排序保持选择。取消某行形成全局排除，工具条显示服务端核对数量，可取消或导出所选。搜索草稿不应用，排序若同时应用了不同草稿则按查询范围变化清空。详情返回高亮独立，不自动勾选；刷新、同标签离开及历史再进入清空。

CSV只包含获权编码、名称、状态、生效日期四列。admin经受控初始化显式具bu.export，reader默认只有读权，可以全选但导出禁用；不按角色显示名放权。成员/资料/读取授权变化后提示重新选择，导出前重验且不输出部分文件。一次导出完成前不要把核对中数量当成最终结果，系统不要求额外确认。

本轮已同步日常m02-current-b，实际页面[4382](http://127.0.0.1:4382)、API8382，正常菜单两角色实测均7条：管理员全选并导出7条，查询者全选7条但不可导出；无BU业务写请求。仅作本轮现场结果，后续在线状态仍以verify和实际业务请求为准。独立137/2021样本、真实缩放及资源收尾见[115实施报告](../verification/115-query-selection-implementation.md)。日常环境为用户体验保留，由当前工作树实现/验证维护者负责，用户结束本轮使用或下次交接时复查；不保留已结束临时验收环境。

## 11 万科固定验证环境

本节为127原固定环境记录，版本保护沿3.8节。下面up及重新构建工具的命令不能套用当前m02-compact-4程序；原API、库和匹配程序保留，当前新观察入口另见第12节。

用户于2026-11-07明确要求“保留万科库为固定验证”，并要求重建时大幅提高并发。独立持久profile为`m02-vanke-validation-b`，项目`chnhcm-bu-219715d544d5`，Web4388/API8388/DB54388；全部87344个组织及44958条历史已重建，普通API、真实页面和重启复验通过；实际证据见[127固定库验收](../verification/127-vanke-full-history.md#9-固定库验收)，页面入口为[4388](http://127.0.0.1:4388)。与m02-compact-case-b分别管理，不以日常清理删除此库或卷。

```bash
CHNHCM_LOCAL_PROFILE=m02-vanke-validation-b python3 scripts/local-bu.py status
CHNHCM_LOCAL_PROFILE=m02-vanke-validation-b python3 scripts/local-bu.py verify
CHNHCM_LOCAL_PROFILE=m02-vanke-validation-b python3 scripts/local-bu.py up
CHNHCM_LOCAL_PROFILE=m02-vanke-validation-b python3 scripts/local-bu.py down
```

重建导入进程已退出。普通入口使用正式12连接上限和本地公开admin/reader账号（密码admin1234567）；down仅停机保留数据，删除须用户另行授权。企业初始化的合成名称、编码和1984-01-01起日保存在仓外私有profile配置，up重用它们，由Go核验摘要，不能改日后覆盖已有企业。原始万科文件及客户端恢复材料始终在仓外；只将聚合证据及程序提交至Git。该固定环境由wt-dev-b维护者负责，在下次固定案例验收或用户提出更换/退役时复查。

完整历史重建并通过首次核对后，可重复执行下面的普通API检查。脚本以admin和reader读取全量列表、抽样当前/原始来源、全部37条特殊名称和18条无变化原提交，并核对业务来源数量不变；不会提交业务修改。请求仍会正常形成访问审计，因此物理库大小可能增长。需要追加写入压测时另行记录测试前后状态，不将变更后的库冒称原导入起点。

```bash
mkdir -p .local/vanke-127
go build -o .local/vanke-127/vanke127 ./tests/capacity/vanke127
VANKE_CASE="$HOME/.local/state/chnhcm/219715d544d5/vanke127"
python3 tests/capacity/vanke127/fixed_check.py "$VANKE_CASE" .local/vanke-127/fixed-read-recheck.json
```

<a id="history-import-rebuild"></a>

### 11.1 固定验证库的历史导入与重建

本节适用于本节所列固定验证库，由 M02 数据与环境维护者在重建或更换导入器时核对。源文件始终在仓外私有位置，根组织映射到一个 BU，其余为部门，BU 直属部门没有行政父部门。各阶段、原行号、来源摘要及实际处理结果可逐行追溯。原件没有的交易身份、操作者和时刻由本次真实办理产生。[来源保护](134-parallel-development-baseline.md) · [原批采纳与验收](../verification/127-vanke-full-history.md)

| 原规则 | 重建时执行的约束 |
| --- | --- |
| V127-I01 来源绑定、I02 名称 | 专属导入器按文件阶段和原行号生成提交号，匹配已核对的原编码、日期、名称与父边。显式历史名称按原 UTF-8 字节保存，拒绝无效编码、超过 200 码点、无文字数字、控制或不可见格式字符。普通新建、更正及预览仍使用产品名称规则。 |
| I03 业务保护、I04 更正 | 写入沿用 Go 的日期、关系、当前授权、锁、幂等与审计，锁前锁后均核对完整依赖。上级更正定位原创建或资料日期。若名称相同或只有扩展变化导致无变化，同日父边由有来源映射的日期新增表示，保留原操作、业务日期及实际处理结果。 |
| I05 扩展、I06 后续维护 | 原组织类型和原因留在专属验证来源表并计入整库空间。只变更扩展的行保留来源，产品无需生成虚假版本。原结果复用实际终态。继承字段保留历史原名，显式再改名称、简称时执行普通准入。 |
| I07 日期合并与续跑 | 名称与父边按业务日合并，同日先形成资料再处理父边。按对象追加或更正末日期，使用上一笔真实来源、记录与修订号。每批发送前在仓外保存原请求，取得全批回执后才推进前缀。部分失败用原键和原文恢复，再继续后续命令。 |

重建按父层推进，相关祖先、非叶子及同一对象保持顺序。跨阶段、跨日的独立叶子可并行，前提是全部来源的每对象序列及各行所见祖先的最近动作与原顺序逐项相同。若锁后发现新入边或来源变化，重新发现并取得对应保护。依赖前沿超过 128 个身份时仅为本次读取装载轻量邻接表。持久库专用导入运行器可使用 12—64 个数据库连接，从 32 路 HTTP / 32 个连接试跑，按实测吞吐与失败率调整。普通服务及正式容量测量沿用 12 个连接。重建完成后以普通入口复核真实 API、历史与冻结路径、重启及原提交结果，再交接持久环境。[关系守卫与依赖](../specs/m02/120-dated-maintenance-technical-contract.md)

## 12 当前模型4390基准

2026-11-12用户明确要求4390应用当时最新改动。入口为[4390](http://127.0.0.1:4390)，持久profile `m02-current-vanke5-b`、API8390/DB54392，当时固定构建b82a1027c35b2591aa8a9b86f10246c25f5f3e09（已合并PR #10），采用模型5/read-segment-4/query-seal /3及迁移00001—00007。原模型4完整快照导入独立新库，现场1BU、87343部门、132297来源全部保留；18张原表、全部历史、22表持久复制、数据库重启、实际构建、双角色菜单/历史及无搜索父候选核验见[原更新记录](../verification/131-4390-model5-refresh.md)。2026-11-13为完成用户指定更名而更新的当前构建见第16节，profile、端口和原库继续保留。开发账号仍为admin、reader，公开开发密码admin1234567。

```bash
CHNHCM_LOCAL_PROFILE=m02-current-vanke5-b python3 scripts/local-bu.py status
CHNHCM_LOCAL_PROFILE=m02-current-vanke5-b python3 scripts/local-bu.py verify
# 模型一致时重新构建并启动本profile的API及固定页面
CHNHCM_LOCAL_PROFILE=m02-current-vanke5-b python3 scripts/local-bu.py up --fixed-web
# 正常停止，保留容器和数据卷
CHNHCM_LOCAL_PROFILE=m02-current-vanke5-b python3 scripts/local-bu.py down
```

新环境保留运行，责任为wt-dev-b维护者，复查节点为下次版本交接。原模型4 `m02-current-vanke4-b` 的36234c9程序、固定页面及54391容器/卷正常停机保留供回退；原模型3和4387/4388保持本轮初态。当前模型5源码不能up旧模型4/3 profile；回退须使用匹配的固定程序并处理4390/8390端口互斥。此次运行更新不关闭125容量P1，不恢复后置容量控制器。以下各日期段为原阶段记录，其中“当前”“未获切换授权”等表述仅对应当时状态，本次操作以上述模型5入口为准。

2026-11-09用户要求确保最新改动在4390生效并重启相关服务。原模型3库无法直接接入当前程序，已用完整快照建立保留同一业务身份和历史的模型4持久profile m02-current-vanke4-b：Web4390/API8390/DB54391，实际入口为[4390](http://127.0.0.1:4390)，模型m02-compact-4/read-segment-3、迁移00001—00007，固定构建基线36234c945c198af5067224544a158fb46b119a82（与已合并main同树）。正常菜单为设置→组织配置→业务单元/部门，开发账号admin及reader，公开开发密码admin1234567。

该次数据为1BU、87343部门，原132290来源完整保留；当时浏览器更正和恢复新增2来源。全部原表、历史与路径解码、固定构建、双角色菜单/API、真实保存及重启恢复结果见[原模型4更新记录](../verification/130-4390-model4-refresh.md)。固定页面不会随工作树编辑自动变化；当前模型5的更新命令以上方新入口为准。

原m02-current-vanke-b模型3程序、固定页面和DB54390容器/卷已正常停机保留，作为更新前回退依据；禁止用当前代码up此旧profile。原4387/4388维持原程序和数据库，退役仍另行授权。4390保留为开发、手工验证及后续复测的入口，在线状态以实际核验为准，责任为wt-dev-b维护者，复查节点为下次版本交接；125容量P1和10000成员保存限制保持，不把本次启动核验计为容量达标。完整125合成矩阵另建自有可丢弃案例，不能以万科形状代替规定历史密度。原[125基准采纳记录](../verification/125-4390-baseline-adoption.md)及[125容量结果](../verification/125-capacity-resumption.md)保持。

2026-11-10开始的131 T04/T05接续中，4390/4387/4388在该任务开始时已停止，继续保留原程序、容器和数据库；2026-11-11 06:57北京时间在万科后处理、R40完整回归、R41只读诊断及首批提交验证结束后再次核对原52个容器身份/完整挂载记录/稳定状态一致，三个页面及API端口均未监听，此前挂载数组顺序误报及复核分别保留。上述4390地址是保留的开发入口配置，不表示此刻在线；本轮未获更新或切换4390的指令。容量只在自有可丢弃的模型4 A与模型5 B副本分时执行，每阶段最多两份完整运行库。其当前构建、进展、资源归属、冷暖失败和收尾待项由[131 T04/T05记录](../verification/131-t04-capacity-and-handoff.md)持有；责任为wt-dev-b本任务实现/验证维护者，复查节点为各完整形状验证结束及最终交付。125容量P1保持，未将模型5接入固定模型4库。

2026-11-11按用户决定，wt-dev-b已选定R118并本地提交`f544108`，当前源码为模型5/read-segment-4；[选定方案及实测边界](../archive/plans/131-department-query-reduction.md#39-本阶段选定方案及实测边界)由131持有。上文`up --fixed-web`仅是模型匹配时的操作模板，当前模型5源码不能用于启动这个模型4 profile。R118在本任务自有模型5副本和临时回归环境验证，4390仍保留原模型4构建，未更新或切换；本轮受影响回归已分轮完成，首失败保留；当前两形状正常写入/空间成本、原历史/重启恢复及T05阶段交接已完成，完整容量后置和125P1仍开放。

2026-11-12 07:17，131按用户批准范围结束本阶段：当前两形状各2000笔写入及重启各2000份结果恢复、必要CSV/菜单和全部原历史保护通过，自有运行容器、卷、测试进程及私有连接配置已回收；原52容器身份/完整挂载/稳定状态与六个固定端口核验通过，4390/4387/4388及API仍按任务开始时状态停止保留。原R127/R128调度器已正常退出，19400次未开始容量和1827次全覆盖观察后置，不应自动重启旧控制器。完整原件、私有结果和历史证据继续由wt-dev-b本任务维护者保管；下次明确恢复125容量整改或环境交接时，重新核对模型、构建、输入和归属，再决定运行资源。当前模型5代码不可直接接入固定4390模型4库，本次未获切换4390授权；最终边界见[131 T05交接](../verification/131-t04-capacity-and-handoff.md#41-本次t05阶段交接结论)。

## 13 130自有案例与字典维护

130原实施阶段工作区为m02-compact-4/read-segment-3，开发迁移00001—00007；该批次保持原4390、4387、4388匹配的原程序和数据库。130原计划不提供旧程序双读、在线升级或固定库维护授权；用户后续明确要求更新4390，当前交付见第12节。实际运行、首失败与结果见[130实施记录](../verification/130-implementation.md)。

仓外0700任务目录持有0600配置和源原件；`tests/capacity/storage130`只服务明确temporary的A/B任务容器，拒绝其他容器或卷。A用冻结的模型3构建，B用当前模型4构建；两者程序不得混接。prepare拒绝覆盖既有资源，转换保持业务UUID及源内容，正常触发器、外键和提交锁保持开启，物理XID及identity分配器按130逐列映射重建。重建读取段是显式离线步骤，失败不提升为开发基准。

维护命令依次为maintain-dry-run、maintain和必要时compact；本批命令均需显式`-input`仓外目录。dry-run只报告目标实例的候选数量及行数据字节。实际维护前正常停止API及全部写入者、排空连接；确认全部实例归属，事务锁定两对象的全部字典引用表后重查并限定目标实例删除，发生连接重入、取消、锁超时或逻辑根变化则回滚。连接复查每次刷新PG统计快照，避免事务内缓存漏掉后进入的连接；普通HTTP保存不执行清理。

maintain执行无引用字典回收及普通VACUUM/ANALYZE；compact另在核验剩余空间后逐表FULL，所需临时表、索引和WAL空间纳入预检。普通维护的可重用空页与FULL后文件缩小分别报告，FULL不作为连续增长周期的日常手段。全部来源、原结果和冻结路径在重启清空缓存后核验；shape不删除，完整UUID不改为哈希或截断身份。

每次交付只清理本任务创建的temporary容器及其指定卷，先核对任务标签、用途和进程退出，再正常stop、rm及volume rm并确认资源不存在；禁止全局prune或使用固定profile作为临时资源。本轮两阶段A/B均已完成规定验证并正常回收，仓外原件/提交与证据保留；2026-11-09只读复查4390/4387/4388归属、原模型及就绪通过，4390固定构建清单一致。原125的模型3固定交接/验证脚本属于4390历史资产，本轮未作为模型4交付入口调用；该原批次未交付新固定基准；2026-11-09用户后续要求的4390更新由第12节记录。

## 14 133 M3与M4自有验证交接

本批基于4129d9b后的当前工作区构建，在仓外任务remediation133-m3-20261112的两个完整输入副本执行验证；仍为模型5/read-segment-4/query-seal /3。固定4390/4387/4388未接入本批构建，没有修改其数据库或缓存。阶段采用、实际容量失败及原始结果统一见[133本批记录](../verification/133-m3-m4-implementation.md)。

自有temporary容器为chnhcm-storage130-b和chnhcm-storage126-product，各使用独占同名-data卷；这些名字只在核对本轮任务标签、创建记录及初态不存在后用于清理，不能作为未来无条件删除清单。实际停止、卷回收、原53容器/47卷一致性及私有复制文件处置以报告第10节清理记录为准。保留证据责任为wt-dev-b本任务维护者，下一次明确恢复容量或运行交接时复查；本批不新留持久运行profile。

## 15 133冷态保存与指定页面观察

客户实库案例的组织明细由私有原仓保存。 [本批报告](../verification/133-cold-save-and-storage-design.md)

本批不新增持久运行profile；两个临时容器/卷、API、私有连接及复制输入按报告第5节清理。固定4390用于用户日常观察，由wt-dev-b维护者负责，下次运行版本交接时复查；4387/4388及其他原资源保持。后续分支使用独立数据库和运行环境，共用协议与存储变更先协调；正式计时期间不并行构建或测试。

## 16 客户实库案例

客户实库案例的组织明细由私有原仓保存。

客户实库案例的组织明细由私有原仓保存。 [修复报告](../verification/133-property-rename.md)

本次不新建持久profile，4387/4388及旧模型环境保持。更新前完整数据库备份和旧固定构建保留在仓外remediation133-property-20261113任务目录，用于本次更新回查；临时副本、连接配置和测试进程按报告收尾。当前4390/8390供用户观察，由wt-dev-b维护者负责，下次固定版本交接复查备份与构建；不得把更新前备份覆盖到已成功更名的当前库来完成日常清理。

## 17 c分支4383国家目录验证

2026-11-14按用户要求更新本机c分支已登记的持久手工环境`m02-current-c`，入口为[4383](http://127.0.0.1:4383)，API8383/DB54374。固定构建为d98715b6对应源码，原库追加00008/00009，251国家目录初始化完成；原15张业务表内容保持。双角色正常菜单、来源/历史、BU及部门新建默认1900-01-01，以及实际API构建归属核对通过，见[运行验证](../verification/200-g2-default-port-verification.md)。

正常菜单为设置→基础资料→国家/地区；管理员admin、查询者reader，公开开发密码admin1234567。该次国家目录运行构建尚不含G3行政区划及G4地点，正式容量和原生缩放当时按用户决定后补。固定构建不会随源码编辑自动更新；脚本不指定profile时的4380并非本机c已登记手工入口。

```bash
CHNHCM_LOCAL_PROFILE=m02-current-c python3 scripts/local-bu.py verify --origin http://127.0.0.1:4383
CHNHCM_LOCAL_PROFILE=m02-current-c python3 scripts/local-bu.py status
```

此环境保留运行供手工验证，由wt-dev-c维护者负责，下次运行更新或用户验收结束时复查。更新前完整备份及旧构建保留在仓外chnhcm-g2-4383-20261114目录，用于此次更新回查；不得覆盖后续新增业务结果。浏览器与会话已关闭，本轮未新建临时数据库。4388万科原程序/原库保持，其他分支资源未操作。

2026-11-14后续按用户要求提供本批验证地址，将上述m02-current-c更新为来源资料精简版eb6c3d38，追加00010；23张业务表内容及业务历史保持，双角色页面/资源不再展示提供方来源，运行证据见[本次更新](../verification/200-country-resource-runtime.md)。端口、账号与维护责任不变。

2026-11-13部门普通保存与真实头本批的容量、原回执、完整历史及隔离格式证明见[接续报告](../verification/133-cold-ordinary-and-head-proof.md)。本批没有更新固定4390、没有将原型接入持久库；自有A/B临时容器和卷已回收。仓外performance133-ordinary-20261113仅保留冻结输入、实际回执、完整来源摘要和原始日志供复核，连接配置已删除；原型源码副本和专用测试二进制已回收，可由登记模板／构建摘要复现。保留证据由wt-dev-b本任务维护者负责，下次性能批次交接复查；原前批固定备份与原环境保持。

2026-11-14本次按用户要求将m02-current-c更新到合并源码8b22e8f7，追加00011/00012并完整发布3620区划；原23表保持，双角色完整目录、结构、历史及原组织菜单通过。实际运行与仓外备份交接见[203固定更新](../verification/203-g3-runtime.md)。入口仍为4383；该次固定构建可从设置→基础资料→行政区划验证G3，构建不含G4；G4开发实现与隔离验证另见第21节，G5未实施。

## 18 135真实头隔离验证

2026-11-14本批隔离候选实施与验证已收尾，净收益不成立，不建议采纳。固定4390仍为原模型5程序和原库；工作区模型6只保留候选实现，不可将其程序接入原持久库。

本任务temporary数据库chnhcm-storage126-product及chnhcm-storage130-b已正常停机并删除容器／卷，临时浏览器与测试环境已回收，未保留候选常驻服务。原53容器、47卷的身份／镜像／挂载／状态保持。仓外head135-20261113受限目录保留完整输入、回执、转储与日志，自有已退役连接配置已移除；保留资料不是可随运行资源一起删除的缓存。范围、首错、资源核验及责任交接由[135报告](../verification/135-real-head-product-integration.md)持有，后续若重新评估，由wt-dev-b维护者另开独立当前模型环境；本批无运行环境待续。

## 19 保留模型5及20条正式场景

2026-11-14用户明确纠正单对象10000条生效记录的不合理验收场景，正式采用单个BU／部门各20条记录，并保留模型5（m02-compact-5／read-segment-4／query-seal /3，迁移00001—00008）。10000条专用保存P95／分页及其驱动的模型6改造退出当前任务，不再列为待修复容量项；原测量和失败仍保留为历史证据。20条是验收场景，不是产品硬上限，不缩减企业对象总量、完整来源历史或并发规模；原500ms目标及125其他容量P1保持。

工作区已撤出模型6运行代码、迁移及专用验证入口，当前新空库初始化和API只使用模型5。固定m02-current-vanke5-b（4390／8390／54392）保留原进程、构建及数据；本次没有执行up、down、迁移或写入业务记录。模型6源码及原实验资料保留用于复核，不能直接作为当前程序启动。新测试只使用自有临时环境并收尾；结果见[采纳记录](../verification/135-model5-and-20-record-adoption.md)。


## 20 301提示整改后的45300

2026-11-14，按用户“提交更改，同步更新后，完成实施”要求，wt-dev-d已合入当时origin/main并将301产品更新到既有tree300-synthetic-d，最终产品提交92f164fcc3de8852f2f071d9df202131af7b22fe。实际部门入口为<http://127.0.0.1:45300/settings/organization/departments>，API45301、DB45302，采用固定前端及API；模型5/read-segment-4/query-seal/3、d迁移1—8、10保持。

原a65ab75c890c容器及原库保留；第一次更新前后15张组织表逐行内容与行数一致，BU来源1份、部门来源87350份。最终补修只更新前端查询条件合并、固定期间提示及新建重复说明，API二进制摘要与首次更新后相同。管理员／查询者正常菜单、实际树叶的快速查询与清空关键词均通过，树范围、状态、列条件、排序和页大小保持，零业务写入；实际1460个编译输入与HTTP提供的7个静态文件摘要核对通过。原失败及后续核对见[301实施记录](../verification/301-user-feedback-implementation.md)。

保留运行用于用户手工验收，负责人wt-dev-d，复查节点为本次用户验收结束或下次300／301整改。4390、4387、4388及其他持久profile本批未操作；125／300原容量失败不因提示整改关闭。不把其他工作树或固定环境自动换成此次程序。


2026-11-14后续按用户明确授权，新建`m02-current-vanke-c`接管4383/API8383/DB54374。1个BU、87343个部门及132284份来源已全量初始化，完整历史、重启和双角色页面通过，见[万科固定环境交接](../verification/203-vanke-fixed-runtime.md)。以上m02-current-c运行描述为历史状态，该原环境现已停机，容器、卷、配置、构建和备份完整保留；两个profile不能同时占用固定端口。当前命令为：

```sh
CHNHCM_LOCAL_PROFILE=m02-current-vanke-c python3 scripts/local-bu.py status
CHNHCM_LOCAL_PROFILE=m02-current-vanke-c python3 scripts/local-bu.py verify --origin http://127.0.0.1:4383
```

当前环境由wt-dev-c维护者负责，供用户手工验收，复查节点为用户验收结束或下次固定运行更新。新旧环境备份位置及摘要由上述报告持有；切回须先停新环境，不得用旧备份覆盖后续新增业务。普通API仍采用原2秒守卫与10秒请求预算，离线导入8秒准备预算不接入普通API；20会话500ms目标及125容量P1保持。


## 21 G4地点隔离验收与固定环境边界

2026-11-15用户授权G4实施及本地提交，当前开发程序追加00013地点独立表，模型5/read-segment-4/query-seal /3保持；实际实施与分轮验收由[204记录](../verification/204-g4-implementation.md)持有。

本批只使用自有temporary profile：g4-204-accept-c为4380/API8380/DB54370，g4-204-tail-c为45187/API47153/DB46893；两者的浏览器验收结束后均已销毁，进程、容器、卷、网络和仓外凭据已回收。容量使用Go测试另建空库，正式8组1600次测量及400回执/全部原历史重启核验已完成，该库也已回收；逐组值及回收证据见204记录。任务日志和脱敏证据保留，临时端口不是新的手工验收入口。

4383仍由m02-current-vanke-c持有原00001—00012构建及完整万科来源，原m02-current-c保持停机保留；4388、4390及其他分支环境未更新。源码提交不会自动更新这些运行环境，不能将当前00013程序直接接入未迁移的固定库。后续运行更新须先按既有数据保护与备份流程核验，再记录实际构建、迁移、原历史和双角色页面结果。

## 22 G5首批交付与4383地点运行

2026-11-15按用户授权更新m02-current-vanke-c：Web4383/API8383/DB54374，原库追加00013地点独立表，原30张业务表、132284份组织来源、251国家及3620区划来源保持，147项普通API和双角色页面验收通过。实际备份、构建和完整内容核对见[205运行交付](../verification/205-g5-implementation.md#6-授权4383部署与历史保护)。入口为设置→组织配置→地点，admin/reader沿公开开发密码admin1234567；持久库未增加验收合成地点。

该profile保留运行供用户手工验收，责任wt-dev-c，用户验收结束或下次更新时复查。原m02-current-c继续停机完整保留，不得同占固定端口；4388/4390及其他分支环境未由本批操作。私有回滚备份同节点复查，不能覆盖后续新增业务；G5临时环境已回收，D08/D09及125其他容量P1保持。

2026-11-15后续按用户“4383更新构建”要求，固定Web/API已更新为PR #16合并提交d810757b。原38张业务表、全部原组织/地理来源、主体及授予权限完整保持，迁移00001—00013不变；147项API、双角色8组详情导航和地点未保存草稿通过，业务写入为0。实际构建、私有备份与内容比对见[205运行更新](../verification/205-g5-publication.md#7-合并构建的4383运行更新)。原运行用途、责任、复查节点及其他环境边界保持。

## 23 302详情导航后的45300

2026-11-15，wt-dev-d按用户“调查和解决”要求更新tree300-synthetic-d，页面<http://127.0.0.1:45300/settings/organization/departments>，API45301／DB45302。构建基于d732a05b及302未提交工作区，模型5/read-segment-4/封套3、迁移1—8及10保持。

详情右上显示当前位置／当前匹配总数，普通切换减少重复读取。双角色首／第二页／尾／关键词／单树范围十组通过；原15张组织表内容、87350部门来源和1份BU来源保持。三次诊断约3.8秒降至0.9秒，非P95，125容量P1保持。完整构建摘要、首失败、复验和资源核对见[302实施记录](../verification/302-detail-navigation.md)。

保留运行供用户手工验收，责任wt-dev-d，复查节点为本次验收结束或下次302整改。五个自有临时profile及12个测试容器已回收，4390／4387／4388等其他持久环境未操作。

## 24 已合并模型5版本更新固定4390

2026-11-14按用户“更新固定端口”要求，将PR #12通过三项CI的88c7195构建应用到原m02-current-vanke5-b，源码树与main合并b0fb527相同。Web4390／API8390／DB54392、原实例和数据库保持；模型5／read-segment-4／query-seal /3及迁移1—8不变，未新增迁移或重启数据库。API SHA256为744bd2431ab3156f64be96be6532ef6b9aef82ec887d35560c7f89c3b7e55c8e，固定前端同批更新。

实际构建/HTTP文件、17张组织表原行、132299来源、两份原成功回执及双角色四条菜单/历史旅程通过。两处验证脚本首错保留，修正复验另列；没有新增业务来源或容量P95。完整备份与旧构建留仓外135-4390-refresh-20261114目录，原53容器47卷保持，当前环境继续供用户手工验证。责任为wt-dev-b维护者，下次固定版本交接复查留存材料；不以旧备份覆盖用户后续数据。详细结果见[135固定更新记录](../verification/135-4390-model5-refresh.md)。

## 25 同步main与提示优化后的4390

2026-11-14按用户授权提交并推送61个文件后，原m02-current-vanke5-b更新为64724468固定构建，入口[4390](http://127.0.0.1:4390)、API8390、DB54392；模型5／read-segment-4／query-seal /3保持，原库追加00009—00012及251国家／3620行政区划初始化。原17张组织表、132299组织来源、企业／账号身份及47条组织授权保持，两份原回执和8条双角色菜单路径通过；查询后旧返回定位和缺失提示核验通过。

更新前完整备份和旧构建在仓外135-4390-sync-refresh-20261114目录，原53容器47卷保持，数据库未重启。当前供用户手工验证，由wt-dev-b维护者负责，下次固定更新或本次验收结束时复查。其他持久环境未操作；125容量P1保持。详细构建、迁移、原数据核验及运行记录提交边界见[135第6节](../verification/135-4390-model5-refresh.md#6-同步main与提示优化后的再次固定更新)。

## 26 部门整改后的4390更新配置

2026-11-16用户要求提交部门整改并更新4390。共享内存由仓外profile配置runtime.json的database_shm_size持久保存，local-bu.py将其传给Compose的CHNHCM_POSTGRES_SHM_SIZE；未设置的profile仍采用64m。m02-current-vanke5-b本次使用512m，与已完成的部门容量验收配置一致；这是容量上限，不是启动即预占512MiB内存。更改已有容器需要在备份并确认原卷后由Compose重建该容器，普通stop/start不会改变其共享内存大小。

本次授权仅覆盖4390／8390／54392的原持久profile，保留原实例、数据库卷、账号和全部历史。当前版本另需追加00013地点迁移；不得将新程序直接接入仅到00012的原库。实际更新已完成，固定API／前端为650db772，00013已追加；Web/API中断59.114秒。原30张组织表、132299组织来源、账号凭据及全部原授权保持，新增14条地点授权与8张空表。双角色12条页面流程和4组返回定位通过，业务写入为零，验证脚本首错完整保留。原数据库容器已为512MiB重建，原卷保持，其他52容器／47原卷未改变。完整备份和原构建留仓外125-4390-refresh-20261116，已读取校验、未实际恢复演练。当前4390继续供用户手工验证，责任wt-dev-b，下次固定更新或本次验收结束复查；原125容量失败保持。详见[本批固定更新记录](../verification/125-4390-runtime-refresh.md)。

<a id="24-303部门一级菜单后的45300"></a>

## 27 303部门一级菜单后的45300

2026-11-15按用户要求在wt-dev-d完成部门一级菜单并更新本分支端口。当前持久profile为primary303-current-d，页面[45300](http://127.0.0.1:45300)，API45301、DB48275；固定API及前端来自d345e4a7基线上的303工作区。桌面应用菜单和窄屏抽屉按部门、设置排列，部门直接进入树/列表；设置保留业务单元及地理基础资料。部门详情、历史及刷新显示部门归属，根路径登录仍落业务单元。开发账号admin/reader及公开密码admin1234567保持。

原tree300-synthetic-d的迁移为1—8及旧10，当前已合并分支为1—12。已完整复制原库到独立持久副本，仅对副本移除旧10派生索引与登记，再按正常upgrader应用当前9—12；旧10与当前12的索引SQL逐字相同。原15张组织表全行摘要及身份保持，原1BU、87345部门、87350部门来源不变，当前目录初始化为251国家及3620区划。原DB45302、旧程序及卷正常停机保留，不可直接使用当前代码up旧profile，也不可因旧temporary标记而删除。

```bash
CHNHCM_LOCAL_PROFILE=primary303-current-d python3 scripts/local-bu.py verify --origin http://127.0.0.1:45300
CHNHCM_LOCAL_PROFILE=primary303-current-d python3 scripts/local-bu.py status
```

本环境保留运行供用户手工验收，责任wt-dev-d，复查节点为验收结束或下次导航调整；固定程序不随源码编辑自动更新。实际构建、分批验证、首失败、数据保护和临时资源收尾由[303实施记录](../verification/303-department-primary-navigation.md)持有。其他分支环境未操作，125及300容量遗留保持。

<a id="25-304企业完整维护后的45300"></a>

## 28 304企业完整维护后的45300

2026-11-15按用户后续授权，将primary303-current-d更新为企业完整维护及已同步main的地点版本。入口[45300](http://127.0.0.1:45300)，API45301/DB48275；企业位于设置→组织配置→企业，部门一级入口保持。企业采用00014，地点00013，当前完整迁移1—14，模型5/read-segment-4/query-seal /3保持。

原库及旧固定程序已在仓外备份，原30张组织表逐行摘要、身份、87345部门及87350部门来源保持，仅初始化一份企业维护来源。双角色18项正常菜单、企业更正草稿放弃、原来源刷新、窄屏及地点入口通过，业务写入为零。固定API及HTTP前端与当前产品来源一致；构建后仅T07测试夹具修复，未进入产品二进制。详细首错、实际构建、验证及发布接续见[304报告](../verification/304-enterprise-maintenance.md#7-默认环境与集成接续)。

公开开发账号admin/reader、密码admin1234567保持。当前环境持续运行供用户验收，责任wt-dev-d，复查节点为验收结束或下次企业/组织维护。仓外更新前备份按同一节点复查，不自动还原到当前14迁移；原tree300-synthetic-d仍停机保留。其他分支环境未操作。

<a id="26-地点地区选择优化及法律实体l1的4383运行"></a>

## 29 地点地区选择优化及法律实体L1的4383运行

2026-11-15按用户“继续推进剩余任务”要求，wt-dev-c已本地提交地点优化a9548b47，并更新m02-current-vanke-c（Web4383/API8383/DB54374），同时带入已提交的法律实体L1与00014迁移。先保存原库、角色、profile及匹配构建的仓外私有备份；原数据库容器和38张业务表完整内容、132284份组织来源及目录保持，新增8张法律实体表为空。147项原API、双角色正常菜单和8组导航、地点完整未保存草稿及法律实体L1草稿复验通过；首轮日期步骤错误和复验保留于[实际记录](../verification/204-location-region-picker.md#6-4383实际更新与验收)。

手工入口为[地点](http://127.0.0.1:4383/settings/organization/locations)及[法律实体](http://127.0.0.1:4383/settings/organization/legal-entities)，公开开发账号/密码保持。新建默认1900-01-01不改，当前既有万科企业下界1984-01-01仍生效，需显式选择范围内日期。保留运行用途为用户验收，责任wt-dev-c，用户验收结束或下次4383更新时复查，私有备份同节点复查；原m02-current-c停机保留且不得同时占用端口。自有临时环境、浏览器及核验二进制已回收，其他分支环境未操作。本轮未推送、创建PR或合并；L2/L3及既有容量遗留保持。

2026-11-15后续按用户要求，4383更新为地点列表简化提交d0c64c16：名称后增加国家/地区列，删除独立国家搜索框。两根DB/API、6项受影响浏览器及双角色现场列展示/返回/窄屏通过，原46张业务表完整内容、全部原授权和用户已有地点保持；00001—00014不变。备份、构建及核验由[204第7节](../verification/204-location-region-picker.md#7-地点列表国家地区列简化)持有。原运行用途、维护责任和复查节点保持。

2026-11-15随后按用户明确要求，4383更新为地址精简提交9f4140cc，原库追加00015。页面取消地址采用日期和独立地址补充；原文本、地点历史/采用/回执及其他42张业务表内容保持，地点4张相关表完成预期转换，授权按两地址字段交集迁移。147项原API、双角色原地点/历史/回执/九列CSV、四地新建草稿与更正取消，以及数据库实际重启后复验通过；本批持久环境业务写入0。构建、私有备份、首次脚本失败和资源核验见[204运行接续](../verification/204-location-address-simplification.md#6-4383授权运行更新与实际验收)。原profile、端口、公开开发登录、运行用途、维护责任和复查节点保持；不得将原d0c64c16程序接回已迁移的库。其他固定环境未更新，原容量及L2/L3事项保持。

2026-11-16按用户授权，4383固定前端更新为e79ffd4c，包含64b77cda的地点名称保留、固定行政区划标签、只读完整地址及基本资料合并，并修复现场发现的新建预览漏显。API仍为9f4140cc原程序和原进程，数据库及迁移00001—00015保持。完整make check与双角色现场原地点/历史/更正取消/四地新建草稿/390及320宽度复验通过，原46张业务表完整内容与全部原身份授权保持，业务写入0；首失败、备份及资源收尾见[204第9节](../verification/204-location-region-picker.md#9-地址表达统一的4383提交与运行更新)。保留运行供用户验收，维护责任和复查节点沿本节；未推送/PR/合并。

## 30 306部门路径及真实集团案例

客户实库案例的组织明细由私有原仓保存。

45300从“部门”进入，4383保留“设置→组织配置→部门”。集团是实际部门，可正常选择和维护；路径不隐式拼接企业，也不显示未来/停用说明。

两处固定源码分别为a8dc35e6及e79ffd4c加306补丁，未迁移。c现场00014法律实体、00015地点地址与当前c分支头不同，后续升级须携带306并先协调迁移，不能直接运行当前分支up覆盖。仓外备份位置、实际HTTP/二进制摘要、首失败及离线维护预算见[306实施记录](../verification/306-department-path-simplification.md)。临时验证资源已回收；两个固定profile保留供手工验收，责任d主代理，验收结束或下次运行更新时复查。

## 31 M03职务页面的4380独立验收

2026-11-17用户授权本地提交并更新4380：当前profile `m03-manual-d`，Web4380／API8380／DB54370，固定产品提交`1d65ecea`，迁移1—17不变。双角色20个桌面、窄屏及原生缩放单元通过；原66张业务表、账号凭据及授权保持，业务写入0。入口为“设置→职务配置→职务系列／职务”，沿用公开开发账号。构建、备份、首失败和现场范围见[308运行交接](../verification/308-job-page-standard-implementation.md#m03-308-4380-runtime)。

该profile已标记为持久验收并保留运行，由wt-dev-d负责，用户验收结束或下次更新时复查；禁止按自动测试结束回收。仓外私有备份按同一节点复查；原45300、4383、4388、4390等环境未更新。停止使用原profile的`down`保留数据，重启前核对匹配程序、原卷及迁移，不用新空库替换。

## 32 M03职级G1的4381独立验收

2026-11-17按用户授权在wt-dev-d实施G1，独立profile `m03-grade-d` 使用Web4381／API8381／DB54371，固定构建来自f87b7b9f上的309候选，完整迁移1—18。入口为“设置→职务配置→职级”，沿用公开开发账号admin／reader。实际验收、构建摘要及运行交接由[309实施记录](../verification/309-grade-g1-implementation.md)持有，阶梯及Job职级关系留待G2。4380保持原程序和1—17账本。4381保留供用户验收，责任wt-dev-d，用户验收结束或下次更新时复查，不随自动测试回收。

## 33 PR19合并后更新原4390

2026-11-16按用户授权，在PR #19最终候选全量22项成功、main实际复用且三项门禁成功后，原m02-current-vanke5-b更新为合并提交6ce8a3e9649f3bf33897c002caaf30c933444a27的固定API与前端。入口[4390](http://127.0.0.1:4390)，API8390、DB54392，512MiB共享内存和模型5保持；原库追加00014—00016，企业、法律实体及地址精简纳入当前构建。

原38张组织表、132299份BU/部门来源、身份及两份原回执完整保持；地点授权按两地址字段交集迁移，新增企业/法律实体授权和一份与原企业声明一致的初始来源。当前51张组织表，1BU、87343部门、251国家、3620区划，地点/法律实体为空。双角色16条菜单/历史/未保存草稿等页面旅程与4组返回定位通过，业务写入0；页面后全表摘要、身份、回执和构建再次核验通过。

Web/API中断66.340秒，原数据库容器未重建或重启，53原容器及47原卷保持。更新前完整备份、角色快照、配置和旧构建仅保存在仓外136-actions-publication-20261116目录，已完整读取归档但未实际恢复；旧650db772程序不得直接连接当前16迁移数据库。当前供用户手工验收，责任wt-dev-b，下次固定更新或本次验收结束复查运行和备份。两处脚本首错及最终证据见[交接记录](../verification/136-actions-publication.md#8-原4390固定环境更新与交接)。后续文档收尾提交不改变实际固定构建；125容量P1保持。
