# CHNHCM-010 T07综合验收与交接

登记信息：[registry.json](../governance/registry.json)。适用范围：[010](../archive/plans/010-m02-scope-and-plan.md)本版BU后台维护；规则、页面、技术与案例分别由011—013及[023](../specs/m02/023-m02-timeline-test-cases.md)持有。本记录保留失败与未运行，旧定稿报告不改写。

## 1 结论与验收边界

本版采纳范围的业务、技术检查及独立正式终审均已通过。用户明确授权主代理按合理业务情形先执行业务验收，原单独评审代理复核；两者均为代理，不记人工业务批准。

253个现行案例中，241整例通过；TL-045/089/128仅本域参数通过；TL-135、180—187真实消费者前置未接，不计整例通过。另4个退役编号TL-241/242/246/247不执行、不算通过。逐例主层、必要补层、完整参数、源码行与摘要、最终执行命令见[案例证据表](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/010-t07-acceptance/cases.json)。不能把函数锚点数或旧报告数当通过数。

按010第6.4节，T01/T02剩余的跨动作汇总、规模预算/存储和真实引用清单已逐项落实；未接外域点/段引用、字段冻结、投递/Inbox/采用交由M03/M04接入批次与M02/Access合同维护者承接。010的本版完成不代表整个M02、全部来源资产或生产交付完成。

## 2 受测输入与独立环境

开工固定`wt-dev-a`、HEAD `f8515927f44e316a9a27a607b57059096687b386`，工作区干净、stash为空。受测内容是此基线加本任务工作区；最终提交号不冒充受测基线。[核对记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/010-t07-acceptance/kickoff.json)保留启动时事实，[输入与证据摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/010-t07-acceptance/manifest.json)固定产品/测试/容量输入及原件SHA256。

| 用途 | 独立profile | 端口 | 边界 |
| --- | --- | --- | --- |
| 正常菜单 | m02-t07-business | Web4527 / API8527 / DB54607 | 当前最终产品重启后完整Chromium及恢复验证 |
| 容量 | m02-t07-capacity | API8517 / DB54597 | Web4517仅预留；独占冷热测量 |
| 事务/故障/跨日/长线 | Testcontainers | 临时随机端口 | 独立空库；不修改主机时钟 |

当前schema7，迁移00001/00002；最终日期唯一索引包含record/revision/range列，键/FK/排斥和封口守卫保持。仅两个本任务新库做过试验索引及最终收敛，不把改写迁移运行到旧库，不删除业务行或清卷；DDL审计代码随容量证据保存。其他旧profile、卷及历史报告保持。凭据只在仓外0700目录/0600文件，不入证据。

## 3 业务验收

管理员从“切换应用 → 设置 → 组织配置 → 业务单元 → 列表 → BU详情”实际办理；查询者验证无写入入口及直调拒绝。创建/资料新增/普通更正、启停、分类删除、改日和有限联合使用同一Go候选。主域规则矩阵保留独立手写预期；DB/API分别验证原子性、准确引用、严格DTO及当前授权，浏览器验证正常菜单、反馈、历史、返回上下文及恢复。

TL050以英文`Nroth`建立首条、只改后继中文并新增启停，再更正首条为`North`；后续独立资料/恢复快照及停用继承仍保持相应原值，预览明确提示，旧来源不改。TL249从创建依次停用、恢复、资料新增、更正、改日、删除，七份来源逐份读取并与保存时原件比较。既有初批追溯新增、首条更正、独立后继、同值、字段清空/撤权、日期边界及原终态恢复随完整菜单和DB/API回归执行。

以下仅列本轮明确留证、可直接查看的代表BU；均为合成资料。名称`Nroth`是故意保留的拼写错误，不是非法字符。

| 场景 | BU编码 | 本机验证入口 |
| --- | --- | --- |
| TL050 | `JOINT_6b1a1542` | [详情](http://127.0.0.1:4527/settings/organization/business-units/a4a7d8eb-e547-4cb5-9b3e-29847c29e64d?businessAt=2026-11-01) |
| TL249 | `JOINT_64c6ea2d` | [详情](http://127.0.0.1:4527/settings/organization/business-units/46857dee-4e84-4ae0-b44f-10fea37a6ed2?businessAt=2026-11-01) |
| tl162 | `IND_SESSION_3bda600c` | [详情](http://127.0.0.1:4527/settings/organization/business-units/6472d8a5-651c-46e9-8056-4835d6e3aec5?businessAt=2026-11-01) |
| tl163-menu | `JOINT_REJECT_87ade073` | [详情](http://127.0.0.1:4527/settings/organization/business-units/2e13443e-b75e-4069-bc88-938a27451e9a?businessAt=2026-11-01) |
| tl164 | `IND_LIMIT_827cc0ea` | [详情](http://127.0.0.1:4527/settings/organization/business-units/c09ab916-e96e-4034-9f78-254aededf2d3?businessAt=2026-11-01) |
| tl174 | `IND_READ_RESULT_e5815fc8` | [详情](http://127.0.0.1:4527/settings/organization/business-units/ca9e04dc-3c5f-469d-9891-a54fedd1e3ad?businessAt=2026-11-01) |

TL162验证真实重新登录后迟到accepted、受控旧403不会覆盖新会话输入/提交身份；TL163在有效预览后注入超长名称形成真实服务端rejected并丢响应，刷新后按原item定位，后续新来源不改原诊断；TL164的受控503信封只证明UI恢复，真实DB超时另由D覆盖；TL174实际撤全部写权后仍按当前读权恢复原accepted，最后恢复测试授权。TL202真实not_applied封闭后丢弃迟到409形成网络未知，页面仍可新办理。所有恢复脚本均串行，未使用mock信封宣称真实事务通过。

显式受控时钟验证自然到日、两浏览器时区及跨午夜联合保存的原办理日说明。TL008主G为Go应用层四时刻独立预期表，经真实A/D观察，不称无DB纯领域测试。TL132未选保留选项的提示/禁存发生在填早期资料之前。TL003依据011企业前置及013/025把主层A校正为M：未初始化Open拒绝，歧义/有限企业由真实DDL拒绝；保留原全部异常参数和阻断预期，不伪造不可达HTTP运行态，也未将其退役。

## 4 发现与修复

| 编号 | 问题与最终行为 | 原失败与复验 |
| --- | --- | --- |
| R1 | 最大迁移号不能识别履历缺洞；Open/Ready检查各版本最新应用状态恰为1/2，已有结构缺履历不补造 | admission-01/02、独立runtime-gap及完整integration |
| R2 | 联合预览连接池超时反馈不符合合同；单条/联合统一预算错误映射 | budget-01/02及完整integration |
| R3 | 更正前段缺后继独立快照提示；Go返回successorSnapshotsPreserved，页面按服务端提示 | TL050正常菜单及Go独立预期 |
| R4 | Detail正文钉旧source后能力读新head；能力复用同一份原始source成员 | read-snapshot-02原失败/03及完整integration |
| P1 | 规定规模下列表、详情及20项保存超标；改为单SQL分页/准确日期定位、同请求能力判定去重、完整summary一次授权落证、集合写入与准确结果批量读取 | hot01/02、cold01—06失败原件保持；cold07/hot04全项通过 |

最终仍完整核验每一显式要求；一份匹配Grant须覆盖全部必要字段，不拼接写权。summary保留Items、全部Requirements、完整grants与相关匹配依据；算法证据版本3区别于旧原件。集合写入顺序为source、revisions、members、direct items，沿用同一savepoint、逐行约束、封口和显式提交意图。原结果仍按当前对象/历史/字段读权读取，当前写权仅用于新执行；未知提交不另起事务补写拒绝。

其余探索或夹具失败原日志均保存，包括初始化样本、时间字段断言、并发夹具、wire/read-snapshot编译、长线准备、浏览器取消按钮选择器、mock Requirements、以及最终check01领域测试误引os。后者将观测文件读取移到组合根测试，不放宽领域依赖规则；最终check02通过。frozen-races01正则未匹配测试，明确不计通过，最终完整integration执行修正后的准确F2竞争与rejected断连分支。

## 5 检查与容量结果

### 5.1 最终门槛

make generate、make check、make build、make integration、独立环境make smoke均通过；完整默认桌面Chromium为97项，无重试。check包含docs、boundaries、Go测试/vet、前端类型与生成差异；build仅有既有包体积提示。显式clock browser、TL162/163/164/174恢复及长线压力单独运行，不把默认跳过记通过。旧TestBUSliceLoad默认跳过，本次以更完整的TL255容量CLI执行，不把旧单来源样本当全规模验收。最终diff检查及文档门槛另留记录。

实际命令、退出码、日志及案例关联见[执行索引](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/010-t07-acceptance/cases.json)；[日志目录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/010-t07-acceptance/logs)同时保留原失败。没有执行Firefox/WebKit、生产恢复或真实跨域消费者。

原门槛wrapper使用time.time壁钟观测；integration的523.685秒与Go自产包时长570.444/276.782秒、smoke的892.125秒与Playwright自产16.2分钟存在口径差异，不将wrapper值作为总耗时，差异原因未另行测定。案例执行索引使用observed_wrapper_seconds并单列日志时长，原gate JSON/日志保持原字节；见[计时口径说明](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/010-t07-acceptance/execution-time-annotations.json)。容量HTTP与取消预算使用Go单调经过时间，未采用这些wrapper值。原始归档日志的空白按.gitattributes免风格检查以保留字节，不放宽产品或测试源码检查。

### 5.2 冷热规定规模

初始真实HTTP建立2000BU×50份来源，首份1成员、其余20成员，共100000来源/1962000成员；另一个BU有1000次历史来源。测量持续追加来源，实际起止规模见JSON，不将后续测量冒称仍恰好100000源。Ryzen7 5800H/16逻辑CPU、约8GB WSL、PG18.6/shared_buffers128MB、API池12；完整环境及计划在原件。

| 请求 | 冷p95 ms | 热p95 ms | 目标 ms |
| --- | ---: | ---: | ---: |
| 默认编码排序列表 | 233.83 | 91.92 | 300 |
| 当前详情 | 254.34 | 112.17 | 300 |
| 确切来源25 | 135.13 | 43.97 | 500 |
| 单项保存 | 343.41 | 137.61 | 500 |
| 两项保存 | 368.59 | 179.43 | 500 |
| 20项保存 | 492.17 | 222.00 | 500 |
| 同值项与实变项联合保存 | 357.69 | 114.97 | 500 |

最终[cold07](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/010-t07-acceptance/capacity/cold-07.json)/[hot04](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/010-t07-acceptance/capacity/hot-04.json)使用同一最终产品65项摘要；每类冷100、热200个完整HTTP响应样本，20会话，全部零错误。保存前读依据不纳入保存计时；热读先预热。冷测是停止专属DB后退数据文件页并核验mincore，不清主机全局缓存；启动/ready重新暖基础页，不宣称硬件及宿主全冷。普通VACUUM/ANALYZE维护不删除事实。最终[plans08](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/010-t07-acceptance/capacity/plans-08.json)显示原日期唯一索引Index Only Scan，同instance/BU/source复合范围直接定位，完整revision键关联。

冷20项保存仅约8ms余量，不是生产SLA。列表性能只覆盖无筛选code排序跨页；筛选/名称排序只声明功能回归。冷数据库增长48,791,552B/WAL253,665,392B，热增长106,823,680B/WAL299,822,792B；WAL包含准备/审计/冷重启checkpoint等。50ms锁采样冷1/热3个等待样本，不据采样间隙推断无短锁。20连接试验反而变慢，最终恢复12；hot03仅热通过、cold01—06不通过的原件均保留。

### 5.3 长当前线、取消与清理

120000成员实际Calculate候选、标准sort边界、继承及Compare四阶段逐点取消；本轮最长本地停止7.325326ms，低于1秒，不改标准库排序。原事务三条独立SQL各4秒，分别小于5秒但累计触发10秒整体预算；ADD/DELETE/REDATE与联合验证零业务残留、独立有界清理及原身份resolve。真实wire吞COMMIT或ROLLBACK TO回复验证提交未知/2秒清理截止，不将连接本地结束冒称DB回滚确认。

200/2000/10000当前成员由受控应用存储夹具准备，明确绕过HTTP准入仅用于准备；正式单项/全期间HTTP请求仍遵守64KiB、SQL5秒、整体10秒及清理预算。精确测量如下，400超正文和503预算拒绝均不记为保存成功；各项核对完整原来源摘要和成员数，取消后先以NOWAIT观测真实锁重获再resolve。

```text
t07_pressure_integration_test.go:117: HTTP current=200 whole=false items=1 inputBytes=290 outputBytes=1613 elapsed=76.779929ms status=200 allocDelta=1895368 heapAfter=3124416
t07_pressure_integration_test.go:117: HTTP current=200 whole=true items=200 inputBytes=43874 outputBytes=214409 elapsed=243.338006ms status=200 allocDelta=21051368 heapAfter=3846328
t07_pressure_integration_test.go:222: current=200 cancelToServiceReturn=83.775µs cancelToDBLockReacquired=6.186149ms; NOWAIT observed before resolve; original identity resolved and exact original source digest preserved
t07_pressure_integration_test.go:117: HTTP current=2000 whole=false items=1 inputBytes=290 outputBytes=1612 elapsed=797.604451ms status=200 allocDelta=15489776 heapAfter=15493864
t07_pressure_integration_test.go:117: HTTP current=2000 whole=true items=2000 inputBytes=438074 outputBytes=97 elapsed=5.371878ms status=400 allocDelta=1589056 heapAfter=13616800
t07_pressure_integration_test.go:222: current=2000 cancelToServiceReturn=44.083µs cancelToDBLockReacquired=6.158777ms; NOWAIT observed before resolve; original identity resolved and exact original source digest preserved
t07_pressure_integration_test.go:117: HTTP current=10000 whole=false items=1 inputBytes=290 outputBytes=110 elapsed=5.019103017s status=503 allocDelta=6024296 heapAfter=76559680
t07_pressure_integration_test.go:117: HTTP current=10000 whole=true items=10000 inputBytes=2190074 outputBytes=97 elapsed=16.920014ms status=400 allocDelta=6917240 heapAfter=61352632
t07_pressure_integration_test.go:222: current=10000 cancelToServiceReturn=49.671µs cancelToDBLockReacquired=2.839739ms; NOWAIT observed before resolve; original identity resolved and exact original source digest preserved
```

## 6 独立复核及交接

正式结论见[独立评审原件](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/010-t07-acceptance/independent/final-review.json)，固定输入见[受审manifest快照](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/010-t07-acceptance/independent/final-reviewed-manifest.json)，独立文档检查见[日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/010-t07-acceptance/independent/final-docs-confirmed.log)。

原单独代理完成逐参数审计、冻结12组缺口与最终代码候选v10，并在完整执行、计时口径补正及证据复核后正式通过：无遗留P1/P2/P3问题。评审者独立执行make docs及git diff --check；其余重负载验证为只读核验主代理实跑原件，包含重算冷热原始样本p95、源码/合同/构建/证据摘要和资产旧证据保护，不冒称独立重跑全部DB或浏览器。

代码、证据和文档只在wt-dev-a本地提交，不推送、PR、合并或部署；完成后停止。下一批由用户选择真实消费者引用/采用与投递、审批/工作台或生产运行能力的独立交付范围，本次没有自动建设授权平台或启动其他模块。凭据查看方法和隔离运行命令见[017](../guides/017-local-bu-development.md)。
