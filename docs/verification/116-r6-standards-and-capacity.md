# CHNHCM-116 R6接入标准与容量交接

登记信息：[registry.json](../governance/registry.json)。适用范围：[116路线图](../archive/plans/116-m02-shared-dated-maintenance-roadmap.md)R6，上游为[120容量合同](../specs/m02/120-dated-maintenance-technical-contract.md)和[124操作指南](../guides/124-dated-object-admission-and-capacity.md)。

## 1 验证对象与基线

2026-11-06，wt-dev-b，R5已提交587667a1a044d1d99df9baf869d4b0401f07c1e9。本报告对应其后的R6工作区增量。原始文件及资源摘要见[基线](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r6/before.json)。未推送、未创建PR、未合并或部署。本次由主代理执行，未将自动检查表述为人工业务签收。

标准交付为124接入指南、按日对象最小模板、BU/部门真实扩展点及五层测试例子。跨域只评估M03基础资料及M04任职的采用条件，不启动第三对象、跨域业务内核或通用EAV。容量与功能结果分别记录；标准可读、构建通过不等于容量达标。

本批标准包、规定规模实测及交接报告已交付，容量验收未通过。R6-P01冷列表/当前读和R6-P02混合写入为未关闭P1；116保持进行中，不登记全R6通过或归档。下文保留首次失败、修复后的精确复验和未完成项去向；R6工作区增量尚未提交。

## 2 环境与前置

使用新建、明确可丢弃的R6独占环境。正常管理员通过真实HTTP操作；大量背景样本从真实API产生的独立原型按固定身份映射扩展，保留全部约束、封口、来源、证据和结果。背景扩展不计API写入吞吐；测量阶段仅使用当前真实HTTP业务入口。持久环境不写入。

profile为r6-capacity-b-1106，项目chnhcm-bu-39b8a01d24e5，API为http://127.0.0.1:55910，数据库55870；预留Web origin为54694，本批容量环境以no-web启动，不能把该预留地址当作实际页面。浏览器回归另由make smoke建立独占环境。

固定规模为2000个背景BU、20000个背景部门；每个背景部门50个来源、20个当前成员。原型经真实API完成19次新增及30次更正，50个来源合计810个历史成员。复制时所有FK、排斥约束、封口及只增触发器保持启用。复制的旧请求摘要不对应新的HTTP请求；旧authorization_decision_id复用原型事件，其事件目标也不是复制后的身份。因此这些旧合成结果不能计为原提交重放或完整授权审计正确性证据；查回测试使用测量期间新提交的真实身份及当前X03事件。另建100层路径、2000后代、1000来源以及20个无部门引用的BU，后者只用于合法的BU固定启停写入。

测量计时为完整HTTP响应及JSON解码，使用Go单调时钟。普通保存、固定两条和查回的准备读取不进入操作延迟，单列准备/清理失败；失败请求保留在错误率及已计时延迟中。warm和cold、BU/部门/同类动作双对象并发分别统计；warm另执行六类操作同时运行的20会话混合负载，按对象及动作拆分。冷测正常重启自有API、停止自有数据库，再逐文件清理自有卷的页缓存；不清主机缓存、不重建数据。数据库启动和readyz会预热设施页，此边界保留。

普通保存为既有首条记录的名称更正，不改日期/分类/关系；固定两条使用2035-01-01停用、2035-02-01恢复，随后通过真实API成对删除。确切源读取第25源；accepted查回使用本轮新写入的提交身份及来源。混合模式的BU与部门按相同背景BU配对，不能把该项当作互不相关对象的理想并发。测量不断追加合法来源，不为复跑回退或重写既有来源，因此负载后50来源是下界，20当前成员保持不变。

## 3 过程与结果

背景生成已完成，独立SQL核对通过，见[shape before](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r6/shape-before.json)：2000BU/100000来源/40000当前成员；20000部门/1000000来源/400000当前成员。两类背景对象均恰为50来源、20当前成员。另有100层、2000后代、1000来源和20个独立BU；无未验证约束或禁用触发器。

生成后数据库约54.764GiB，其中部门来源成员17.071GiB、路径节点16.013GiB、路径期间9.518GiB、路径边5.128GiB；数据来自[seed second](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r6/seed-full-second.json)的after快照。该次生成窗口WAL增量约79.121GiB，包含同环境诊断活动，不是HTTP业务吞吐或每次保存的WAL。正式冷暖容量使用[固定输入](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r6/capacity-measurement-inputs.json)；生成期间probe和EXPLAIN仅作诊断。

### 3.1 已实施的优化和保护

1. 部门单对象读写按实际变化根、反向后代及必要祖先读取；增加拟选父节点不再展开其无关兄弟部门。BU写仍覆盖全部当前/未来启用使用者。共用守卫按稳定身份批量取共享依赖锁，目标写锁保持原顺序，锁后重规划与最终资格规则不变。
2. 不可变快照建立请求内身份索引；更换集合时重建索引，避免线性查找和候选覆盖污染。列表同一SQL固定资料/关系头，随后仅取确切不可变事实。事实缓存绑定数据库连接池及实例，限定序列化预算；当前日、状态、路径、字段和权限仍由本次Go/X03处理。
3. X03批量只读投影使用同一个判定器，压缩重复Grant和结果，同时保留每项目标、动作、允许/拒绝及字段的完整对应。此格式不授权写入或结果恢复，不跨请求缓存权限。
4. 源成员增加窄来源索引并覆盖身份/范围字段，部门原日期唯一约束只增加INCLUDE列。SQL使用来源定向读取，所有实例/企业/对象范围过滤保留。只对本任务临时数据库同步候选索引，既有持久库未改写。
5. M02短事务设置LOCAL jit=off。实测某单对象计划总耗时约5.8秒，其中LLVM编译约5.23秒；关闭本事务JIT后同类目标查询约69ms，确切源约55ms。这些是生成期间诊断值，不是HTTP p95。
6. BU和部门提交身份锁均补上Go单调2秒预算。空HTTP请求体不再安装读取截止时间，避免后台读连接在5秒后取消正常GET并影响同一keep-alive连接。请求、取锁及SQL原有业务预算未延长。

### 3.2 执行入口

从当前模型的新临时profile开始；本批预留端口记录在第2节，重跑须先按017核对端口及归属，不覆盖正在使用的环境。以下输出目录须新建且不包含旧证据；驱动拒绝覆盖已有结果文件。

```bash
export CHNHCM_LOCAL_PROFILE=r6-capacity-b-1106
go build -o .local/bin/r6-capacity ./tests/capacity/r6
CGO_ENABLED=0 go build -o .local/bin/r6-cache ./tests/capacity/cache
.local/bin/r6-capacity -phase prototype -out .local/r6-retest/prototype.json
.local/bin/r6-capacity -phase seed -count 20000 -sessions 0 -out .local/r6-retest/seed.json
.local/bin/r6-capacity -phase fixtures -out .local/r6-retest/fixtures.json
.local/bin/r6-capacity -phase verify -sessions 0 -out .local/r6-retest/shape-before.json
.local/bin/r6-capacity -phase vacuum -sessions 0 -out .local/r6-retest/vacuum.json
.local/bin/r6-capacity -phase measure -sessions 20 -cache warm -out .local/r6-retest/warm.json
.local/bin/r6-capacity -phase measure -sessions 20 -cache cold -out .local/r6-retest/cold.json
.local/bin/r6-capacity -phase special -out .local/r6-retest/special.json
.local/bin/r6-capacity -phase cancel -out .local/r6-retest/cancel.json
.local/bin/r6-capacity -phase verify -sessions 0 -out .local/r6-retest/shape-after.json
python3 scripts/local-bu.py destroy
```

measure非零退出表示错误或目标未达，不把它跳过；保存该结果后继续需要的语义/收尾检查。index、lookup-index、plans和cache-plan为本轮诊断专用入口：新空库已由迁移建立索引，无需重复执行索引变更。本批顺序与失败复验另见过程记录，不将上面的标准重跑顺序冒充本轮一次成功执行。

### 3.3 暖缓存完整负载

[暖缓存原始结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r6/warm-full.json)共4800次计时请求，8次未获得预期成功响应，准备/清理错误为0。前18组各200次，混合组各对象100次；六类动作同时执行的mixed-actions共1200次，每对象每动作100次。p95单位为毫秒，星号表示超过原目标，查回只报告耗时且必须语义正确。

| 负载/对象 | 列表300 | 当前300 | 确切源500 | 保存500 | 固定两条500 | 查回 |
| --- | --- | --- | --- | --- | --- | --- |
| BU单对象 | 161.32 | 101.23 | 49.65 | 576.03* | 358.88 | 59.03 |
| 部门单对象 | 9656.64* | 883.19* | 764.86* | 597.47* | 1506.09* | 408.57 |
| 同动作混合中的BU | 679.78* | 227.52 | 112.80 | 917.86* | 683.50* | 86.90 |
| 同动作混合中的部门 | 5616.61* | 419.52* | 183.60 | 659.75* | 1470.91* | 241.47 |
| 六动作混合中的BU | 207.62 | 146.69 | 120.52 | 2233.14* | 367.84 | 167.72 |
| 六动作混合中的部门 | 3657.46* | 298.52 | 229.35 | 405.99 | 1934.69* | 305.88 |

部门单对象列表为5/200次503 M02_RESULT_PENDING；六动作混合中的BU保存为3/100次同码响应，不能将pending解释为已确定业务拒绝。其他组无响应错误。总错误率8/4800约0.167%，不能以该总体比例掩盖列表2.5%或混合BU保存3%的分组失败。[负载后形状](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r6/shape-after-warm.json)通过：两类背景仍各20当前成员，固定两条已清理，来源只能追加，约束/触发器仍完整。

测量本身453.38秒，4505次资源采样无采样错误。API RSS峰值约3.187GiB、PSS约3.186GiB，数据库cgroup峰值约4.134GiB（含文件缓存）；采到12个活动连接、最多11个锁等待/被阻塞连接。100ms采样不能证明更短等待不存在。数据库从约54.799GiB增至57.465GiB，WAL增量约5.553GiB；这些窗口包括准备、清理、授权证据和恢复，不是单次业务写入成本。

正式测量未与其他主动回归并行；暖阶段07:58曾正常停机并移除前两轮总超时遗留的两个本任务测试容器，见[归属及收尾](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r6/test-timeout-resource-cleanup.json)。该资源操作与暖测量有时间重叠，不将本轮描述为完全无外部活动的主机基准，也不据此消除任何失败。

### 3.4 冷缓存完整负载

[冷缓存第二轮](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r6/cold-full-second.json)使用[更新后的容量工具输入](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r6/capacity-retest-inputs.json)，产品源码与完整DB/API回归相同。90次API重启/自有数据库逐文件清理均通过，最终驻留页全部为0；补上逐文件fsync后，本轮没有文件需要第二次驱逐。每组5轮×20会话，共1800次计时请求；混合组每对象50次。冷测不运行连续mixed-actions，后者只在暖缓存单列。

| 负载/对象 | 列表300 | 当前300 | 确切源500 | 保存500 | 固定两条500 | 查回 |
| --- | --- | --- | --- | --- | --- | --- |
| BU单对象 | 297.25 | 191.17 | 133.42 | 862.74* | 628.02* | 149.14 |
| 部门单对象 | 10012.10* | 512.00* | 426.30 | 894.78* | 3788.64* | 644.34 |
| 同动作混合中的BU | 357.49* | 236.26 | 138.66 | 1179.16* | 1141.64* | 199.96 |
| 同动作混合中的部门 | 10100.11* | 379.02* | 298.29 | 935.11* | 3433.99* | 434.03 |

失败为部门单对象列表100/100，以及混合组部门列表50/50；其余响应成功，全部准备/清理错误为0。原始样本保留M02_DEPENDENCY_UNAVAILABLE等503结果；失败样本仍计入延迟及错误率，不能把约10秒的超时p95当作成功响应能力。完整暖缓存也已有列表超标，不能仅用预热掩盖冷启动不可用。

本阶段692.58秒，5608次资源采样中2937次恰逢数据库停止/重连而无法查询，缺测不能解释为无活动或无锁。已取得样本的API RSS峰值约0.476GiB、PSS约0.475GiB，数据库cgroup峰值约4.035GiB；最大活动连接12，采到最多4个锁等待/被阻塞连接。WAL窗口增量约0.846GiB，包含90次重启/检查点、准备和清理；与暖阶段不是单次写成本的直接对比。

### 3.5 特殊规模与语义

修正驱动后的[special第二轮](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r6/special-second.json)完成以下检查。语义均通过，但100层/2000后代根详情仍超过300ms目标，整个special按失败退出，不改写成容量通过。

| 项目 | 样本与实际结果 | 边界 |
| --- | --- | --- |
| 100层路径 | 20次读取均得到企业节点及100个不同部门节点，叶身份正确；p95 544.01ms | 结构完整，当前读目标未达 |
| 2000后代根详情 | 20次成功，p95 609.34ms | 当前读目标未达 |
| 2000后代祖先更名 | 1次真实更正约1770.36ms；子部门来源不变、当前路径变化、旧来源完整路径结构不变 | 单次关系/来源语义，非p95 |
| 1000来源部门 | 当前20次p95 83.58ms；第25/1000准确源各20次，p95 62.41/64.72ms，准确源及只读标识均正确 | 三组均达到各自读取目标 |
| 137条CSV | 从20000项查询中跨7页选择137个稳定身份，解析137，导出138行/8列，路径值完整；14872字节，约1868.11ms | 一次完整导出，SHA-256及列名保留；不是CSV p95或生产承诺 |

[完整背景的取消专项](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r6/cancellation-full-background.json)通过真实外部持锁屏障确认等待发生；BU和部门取消后无来源或业务终态、头不变，原身份resolve得到not_applied。该项单次耗时不与上述p95混合。

### 3.6 混合负载原身份恢复补验

[mixed-actions定向补验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r6/mixed-recovery-retest.json)共1200次计时请求、每对象每动作100次，使用同一产品实现及新增提交号/查回观测。BU保存74/100次503（73次M02_RESULT_PENDING、1次M02_DEPENDENCY_UNAVAILABLE），每笔均保留原提交号，随后全部查回not_applied；恢复错误及准备/清理错误为0。这只证明这些失败提交按原身份封闭，74次原失败仍全部保留，不能以恢复成功计作保存成功。

| 对象 | 列表p95 | 当前p95 | 确切源p95 | 保存p95 | 固定两条p95 | 查回p95 |
| --- | --- | --- | --- | --- | --- | --- |
| BU | 205.53 | 131.99 | 190.14 | 2215.85* | 407.93 | 173.64 |
| 部门 | 3426.10* | 359.62* | 241.26 | 459.31 | 3429.75* | 367.04 |

其余1100次动作以及BU保存中26次获得预期成功响应。该轮与初轮完整暖测的3次混合BU失败分别保存；测试顺序、已追加来源及缓存经历不同，不作只改变一个变量的性能对照，也不能挑选较好的初轮结果关闭混合写入问题。所有指标的可机器读取摘要见[metrics summary](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r6/metrics-summary.json)。

## 4 失败与复验

首个合成样本的UUID映射不符合API版本/变体约束，原型保存结果和失败probe保留；仅销毁并重建当次明确可丢弃的R6环境，再修正Go/SQL一致映射。该失败不计为当前正确样本的业务能力。

首次8,900余部门的背景生成主动中断，以提高准备并发；按对象事务原子提交，已完成身份保持，未完成事务回滚，第二次按稳定身份续建。原中断记录保留，不将首次生成称为成功。第二次结果文件与末行日志记录完成，命令会话最终返回143；差异单列于[会话记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r6/seed-second-session-completion.json)，不声称其外层退出0。随后独立SQL已核对全部实际数量和约束，不能只据进度计数认定样本完整。

作用域提炼首轮遗漏空父标识过滤，创建返回503；随后候选预览因拟选对象不在原作用域而失败，已恢复候选/预览所需的完整事实读取。集合索引首轮在BU反向检查收窄Departments后保留旧索引，真实DB回归复现panic，已在集合变化后重建并定向复验。以上首次输出均保留。

2000后代样本第一次在1059项时碰到原有广泛依赖耗时，返回M02_RESULT_PENDING；定向依赖修复后续建至2000成功。100层及1000来源均已建成，见[special fixtures first](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r6/special-fixtures-first.json)和[second](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r6/special-fixtures-second.json)。

HTTP真实连接对照先运行原HEAD实现：空体GET在约5秒返回503，POST控制组成功；修复后GET/POST都完成5.2秒应用处理并复用同一连接。该延迟服务只验证HTTP截止时间，不能代替业务验收。提交锁测试通过外部事务真实持锁并在被测连接关闭PG锁计时，仍受Go预算限制；resolve合同为202 pending，测试首轮503预期错误已单独修正。

完整make integration首轮与背景生成和浏览器并发，tests/integration触发20分钟套件总超时；bootstrap包完成。原超时不是单个业务断言失败，也不记为全量通过。后续定向部门范围/授权/守卫回归通过；第二次完整运行安排在背景生成结束后，仍与独立环境的浏览器回归部分重叠。正式容量计时等待两者结束。

首轮完整Chromium为233通过、4失败，原输出见[smoke first](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r6/checks/smoke-first.txt)，遮蔽截图见[browser first failures](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r6/browser-first-failures)。其中候选重试测试误把第一页当完整集合，已按编码检索；启停测试沿用自动辅助抢焦点的旧预期，已按R5采用规则验证主动点击错误链接；表单外观测试在创建能力加载期间测量禁用字段，已等待可编辑后测量。另一个清空父关系案例碰到候选查询503，本轮后端读取优化后的真实环境定向复验通过。四项定向复验均通过，见[browser targeted](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r6/checks/browser-targeted.txt)；不能将这四项补验拼成一次237项全绿。第二轮完整Chromium为232通过、5失败，见[完整输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r6/checks/smoke-second.txt)及[browser second failures](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r6/browser-second-failures)。失败包括登录阶段的30秒总超时、固定期间之后的详情503，以及三个关系候选等待失败；API记录四次55P03取锁超时，不能把所有失败归因于同一个未证实原因。所有失败发生在背景仍在生成时。随后先经真实HTTP建立219个BU背景，再执行原五项断言，5/5通过；列筛选beforeAll还会建立自己的73个BU。见[额外背景](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r6/browser-retest-population.json)、[复验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r6/checks/browser-large-retest.txt)及[命令/收尾](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r6/browser-large-retest-status.json)。未改这五项断言或超时，也未改产品代码；分批结论不等于单轮237项全绿。两轮完整布局分别保存在browser-first-layout和browser-second-layout，第二轮含42张已加载页面截图。

冷缓存准备首轮辅助进程非零退出，旧包装未转存其stdout，仅有[原异常](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r6/checks/cold-reset-preparation.log)，无法据该次输出判断剩余驻留页的具体文件。现已补上失败JSON及残留文件诊断，失败仍非零退出。第二次准备核验1618个文件、59,901,560,181字节，重启前驻留页为0，见[第二次](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r6/cold-reset-preparation-second.json)。这是清理方法准备检查，不是正式冷负载结果。

正式冷测第一轮在任何计时业务前失败，见[cold first](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r6/cold-full.json)：1630个文件中base/16384/17886和base/16384/18203.2各剩1页，业务workloads为空，不能据此声称冷请求成功或失败。仅FADV_DONTNEED可能保留脏页/回写页；工具现先对自有停止数据库的每个文件fsync，再逐文件清理和mincore核对，最多三次有界驱逐，逐次残留另存，最终仍要求零页。该方法不调用全局drop_caches或全文件系统syncfs，不重建数据或放宽阈值。

首轮[special](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r6/special-full.json)的路径断言误用path.path，合同实际字段为path.departmentPath：100层20次被误报不完整，祖先名称对比也错误比较了两个空值。修正驱动并增加段/节点身份检查，历史路径改为完整结构对比；产品实现未改。该轮137条CSV、1000来源的第25/1000准确源读取成功，完整special仍记失败，后续复验另存。

暖测3次pending保存的原始样本缺少提交号，不能由这份输出逐笔证明最终结果；独立取消专项仍可证明其自己的原身份恢复，不能替代这3次。驱动已为每次写入记录非敏感提交身份；失败响应之后单独按原身份resolve，记录耗时、终态和来源，保持原计时样本失败，不换号重发。后续只复验mixed-actions时显式标注-mode mixed-actions，不能冒充重跑暖缓存全套。

完整背景下BU和部门取消/原身份恢复均通过，见[实际锁等待与取消](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r6/cancellation-full-background.json)。此项在独立浏览器和DB回归期间执行，只据其屏障、数据库零写入及恢复结果认定语义，不将单次耗时作为容量p95。

第二轮完整DB/API在20分钟总上限处结束，157顶层及325子测试通过、5顶层专项跳过，没有已记录的业务断言失败；末项仍在执行，不能计全范围通过。第三轮以相同两个包及全部测试、40分钟套件上限执行，162顶层及328子测试全部通过，共490项通过、5顶层专项跳过，见[摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r6/integration-third-summary.json)及[完整输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r6/checks/integration-third.txt)。tests/integration耗时1170.374秒，bootstrap耗时473.588秒；调整的仅为整套看门狗，业务请求/事务/取锁预算保持原值。

新增事实缓存的跨日专项通过，见[真实双时区浏览器](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r6/checks/clock-browser-r6.txt)：上海和洛杉矶均按服务端业务日转移部门授权归属，旧选择/投影清除、旧来源拒绝，业务来源未增加。该专项与完整默认套件中的skip分别记录。

完整背景的暖缓存准备见[原始结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r6/warmup-full-background-preparation.json)：前6次请求超时，第7次成功约2.34秒；当时与其他独立环境回归重叠，不能当作安静环境p95。这些失败不从证据删除，也不计为目标通过。

### 4.1 最终回归范围

| 检查 | 实际结果及证据 | 结论边界 |
| --- | --- | --- |
| 静态、边界、单元及前端检查 | [make check最终轮](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r6/checks/check-final.txt)通过，约21.89秒，输入见[final inputs](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r6/final-inputs.json) | 包含最终容量工具；此前第三轮及其后5个工具文件差异分别保留，产品源码未变 |
| 构建与生成 | [make build](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r6/checks/build-second.txt)、[make generate](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r6/checks/generate-final.txt)通过 | 对应当前产品与查询，不代替业务运行 |
| 完整真实DB/API | 第三轮单次490项通过、5专项跳过 | 前两轮总超时保持失败；默认专项不计为通过 |
| 显式跨日浏览器 | 双时区专项通过 | 上海/洛杉矶覆盖授权转移和旧投影清理；与默认套件skip分开 |
| 完整Chromium | 第二轮232通过、5失败；扩大背景后原5项全部通过 | 不是单轮237项全绿；完整布局42张截图另存 |

主代理复核了缓存身份/来源固定、集合替换后的索引、授权逐项证据、依赖发现及锁顺序、提交恢复和临时环境归属。实际发现与修复列于3.1及第4节；未将本次主代理自检标为独立代理或人工业务签收。

## 5 边界

R6不启动跨域接入、生产交付或外部消费者。

既有验收报告和证据不因本轮改写。冷暖、单轮完整与分批复验分别计数；诊断过程中数据规模持续增长，不能把前后probe直接当同规模优化比。300ms/500ms目标保持原合同，未通过项不得以调整样本规模、关闭约束、放松授权、弱化历史或增大业务超时关闭。

### 5.1 容量遗留任务及关闭条件

以下由116继续持有，不转成已通过的124接入条件。职责为实现角色，未虚构个人指派。R6标准、测量及交接的交付状态不替代这些容量任务的验收。

| 任务 | 实际缺口及下一步 | 负责角色与关闭条件 |
| --- | --- | --- |
| R6-P01（P1）部门列表及冷启动 | 全量头、事实、逐项授权和完整证据在每次列表的固定开销仍高；部门冷列表150/150次失败，深层/大扇出详情也超标。按取头/载入/Go投影/X03证据分段定位，评估减少重复工作。不得缓存跨请求授权或由SQL接管业务日期 | Organization/X03实现者；同规模冷暖及20会话的列表/当前读p95≤300ms、零业务错误，并通过跨日/撤权/集合版本回归 |
| R6-P02（P1）保存、关系与混合负载 | 普通保存及固定期间在部分负载超过500ms；最后混合BU保存74/100次503且查回not_applied，目标负载下写入可用性未满足。进一步分离依赖读取、守卫等待、验证和落库开销，共同守卫及锁后重读保持 | Organization/存储实现者；BU、部门及混合负载各自p95≤500ms、零业务错误，保持双向竞争、2秒单调取锁及原身份恢复，准备/清理无遗留 |
| R6-P03 来源与路径存储成本 | 现有100万部门来源的完整成员及路径证据占用显著；评估物理共享或索引成本前先固定原件/封口/准确源不变量 | Organization/存储维护者；提交可复核设计和同规模磁盘/WAL/内存对照，任何方案须保留旧来源原值、独立表组及不可变历史 |

R6-P01/P02按原目标重新测量前保持未关闭。R6-P03为容量成本改进项，不另造原合同没有的磁盘达标阈值。跨域实施、真实外部消费者、人工业务签收及生产恢复仍在原计划后置范围。

## 6 收尾

最终[数据形状](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r6/shape-after-retests.json)通过：2000BU共101223来源/40000当前成员，20000部门共1002600来源/400000当前成员；每对象当前成员仍为20。100层、2000后代、1000来源及20独立BU均在，未验证约束及禁用触发器均为0。测量前50来源的固定背景与负载后合法新增来源分别记录，未重置或删除历史来维持表面计数。

自有容量profile正常销毁，见[清理输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r6/checks/capacity-destroy.txt)和[最终资源/原证据核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r6/final-resource-and-evidence-audit.json)。原45个容器（3运行）、36个网络、59个卷逐身份保持，新增及丢失均为0；本任务55910/55870/54694均无监听，临时私有配置和profile元数据已移除。前两轮测试总超时遗留的2个测试容器已按其日志及Testcontainers归属正常停机并移除，所有临时浏览器环境由各自finally回收。

3535份既有验证文件逐一SHA-256核对不变。最终产品源码与完整DB/API第三轮输入相同，之后仅5个容量工具文件变化；新工具已经实际执行本轮冷测、特殊规模及恢复补验。当前任务两项私有值扫描未在325个变更/证据文件及API日志中命中，具体值未输出或入仓，见[扫描摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r6/private-value-scan.json)。

最终make docs及git diff --check通过；另核对54个维护源码/文档（含未跟踪文件）的行末空白、Go格式和最终检查输入，均无差异或问题。检查后再核对原资源身份和3535份原证据仍不变，见[最终收尾摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r6/final-closeout.json)；原始日志按原字节保留，不为消除日志空白改写失败证据。

既有持久m02-department-b保留供用户观察R5部门维护，责任为wt-dev-b，下一次联调或环境更新前按017复核归属、模型和数据；本轮未对其迁移、重建或更新API。其他原有环境保持原责任与用途。本批不保留新增运行环境，不推送、发起PR、合并或部署。

## 7 证据

证据保存于[data/116-r6](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r6)。
