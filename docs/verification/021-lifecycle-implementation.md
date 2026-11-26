# CHNHCM-021 停用与重新启用实施验证

登记信息：[registry.json](../governance/registry.json)。范围：[010 T04](../archive/plans/010-m02-scope-and-plan.md)、[021决定](../adr/021-bu-lifecycle-and-record-maintenance.md)、011—013及023的本批参数。代理实施与自查，不冒充独立业务批准。

## 1 验证对象与受测基线

2026-11-01，wt-dev-a，基线`a82d85bb67424ae9d21ab7746dbfdc23f2c0fa49`，开始工作区及stash为空。差异包括同一有限候选内核的启停分支、当前授权目录、API/客户端、页面、严格输入及分层验证。当前受测文件与日志摘要见[执行清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-lifecycle-implementation/execution.json)；报告自己的最终提交号不替代脏工作区输入。

## 2 环境与前置条件

Go1.26.0、Node24.19.0、PostgreSQL18.6锁定镜像、桌面Chromium。独立新空库profile `m02-t04-current`，页面4416/API8416/DB54496，Compose项目chnhcm-bu-fee9d3100982。原4396/4406环境、数据库及卷不变；本批不运行迁移到旧开发库。

当前迁移仍00001/00002，存储合同为`m02-timeline-3`，授权目录证据版本为2；新初始化管理员显式获得两项专用权，查询者不获得。当前程序拒绝不匹配的开发存储，不建设兼容路径。凭据只在仓外0700/0600位置或Testcontainers进程内；未归档cookie、CSRF、trace或登录正文。

管理员从“设置 → 组织配置 → 业务单元 → 列表 → BU详情”办理。动作入口表示有权进入表单，最终按目标日期预览/提交；今日启用仍可排已有未来停用之后的重新启用。原因不预选，停用不提供名称输入，恢复可确认默认或在字段权内修改。

## 3 验证过程与结果

### 3.1 业务规则、历史和权限

| 场景及参数 | 实际断言与证据入口 |
| --- | --- |
| 过去/当天/未来启停 | Go用受控D=2026-06-15手写三时结果，并单列当日停用和当日恢复；确认未来安排不提前改变D。主层不依赖主机时钟，也不调用受测算法生成预期 |
| 多轮独立快照 | Go及真实API建立1月甲/Alpha、4月S、5月R乙/Beta、8月S、9月R丙/null、11月CHANGE丁；逐段日期、期间、状态和资料手写核对。更正首条及R只传播至各自后继S，不传播下一独立R/CHANGE |
| 准确继承与来源 | 真实SQL核对每个来源中S引用同来源紧邻前序的最终record/revision；旧来源原资料及当时开放期间保持。7个来源、审计、Outbox和6个身份符合手写预期；Outbox仅落库未投递 |
| 同名恢复 | 省略changes确认默认仍accepted，新独立完整快照，不能返回no_change；显式同值字段仍校验其写权 |
| 完整后继守卫 | 单S使后继CHANGE非法、提前R使既有R重复均拒绝；指明可读冲突日期；重复状态、首条前/同日、企业外及非法日期拒绝，不自动补齐、删除或隐式恢复 |
| 字段及名称 | S禁止changes/name/state夹带，R的changes存在时非空，name不能null；英文省略继承、显式null清空。R沿用emoji拒绝、合法扩展汉字及当前NFC规则；已有创建/新增/更正名称矩阵完整回归 |
| 专用权限 | 无生命周期权返回403；仅S/R基础字段权可确认默认而不能普通更正。普通创建/资料新增/更正权不能替代启停权，伪造ADD类型的保存及预览400；BU专用授权不扩至另一个BU |
| 部分字段/当前撤权 | CN仅改中文，隐藏英文不出默认/响应且存储保留。撤写权仍可精确重试及resolve原终态；新的执行403，原授权快照字节不变；撤历史读权后resolve403。无日期读权的依赖错误不泄露日期或字段位置 |
| 严格协议 | 缺必填、重复JSON键、错类型/原因、未知字段及不适用目标参数拒绝；已存在的严格解码回归保留 |

### 3.2 事务、恢复和正常菜单

| 风险 | 受测过程及结果 |
| --- | --- |
| 同来源并发 | S和R各用两个独立提交并发，均恰一201一409，只新增一来源 |
| 写后409 | 组合根任务夹具调用真实Persist后返回409；两动作均回滚整笔业务savepoint，七张业务表完整摘要不变，无孤立身份/半来源；原锁内留下唯一安全rejected终态 |
| 整体回滚与终结竞争 | 两动作分别在真实写后返回DeadlineExceeded，整体回滚且没有补写拒绝；原身份重试先到、resolve先到和真实并发均验证唯一终态，not_applied封闭迟到原请求且业务七表不变 |
| 原办理日跨午夜 | 通过数据库Clock适配和事务内来源时间触发夹具，对ADD/S/R分别固定采样在上海6月14日23:59:59，实际recorded_at为6月15日00:00:01；保存及resolve始终保留原办理日/原状态，主机时钟不变 |
| 有界执行/取消 | 当前同一内核120000条当前记录、5轮取消，及真实DB超时/HTTP取消释放BU锁的共同框架回归；实际测量见domain/integration日志。夹具仅证明测量范围，不计T07全容量通过 |
| 正常菜单 | 5项T04新增Chromium旅程覆盖未来多轮与默认确认、当天停用/未来恢复、过去补登记、未保存保护、S/R丢响应刷新查回原source+记录、冲突日期聚焦和保留输入、查询者无入口；历史来源无编辑入口 |
| 受影响回归 | 完整Chromium覆盖既有创建/查询、普通新增/更正、名称准入、无变化、返回条件、权限清理、会话和迟到响应；原结果链接始终定位当次来源，未用当前资料代替 |
| API重启 | 冻结后读取合成LIFE_样本当前来源、原来源和各期间；只重启本profile API后重新登录核对相同内容。结果见[重启查回](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-lifecycle-implementation/runtime-readback.json) |

### 3.3 检查和TL映射

| 检查 | 结果及证据 |
| --- | --- |
| make generate | 通过；sqlc、名称属性及OpenAPI客户端一致，见[生成](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-lifecycle-implementation/generate.log) |
| make check | 通过；文档工具/登记/边界、Go测试/vet、前端类型及差异，见[检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-lifecycle-implementation/check.log) |
| make build | 通过；API、upgrader和页面，见[构建](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-lifecycle-implementation/build.log) |
| GOFLAGS=-v make integration | 30项通过（26项公开DB/API、3项组合根事务/时钟、1项HTTP）；容量1项显式跳过，见[完整集成](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-lifecycle-implementation/integration.log) |
| 最终权限专项 | 通过；普通入口伪造S/R均拒绝，只有启停权时readOnly=false，见[权限补验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-lifecycle-implementation/permissions-final.log) |
| 当前Go独立预期与取消 | 通过；120000条、5次取消最大53.891ms，小于1秒；真实DB取消释放BU锁5.278ms，见[领域](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-lifecycle-implementation/domain.log)及集成日志 |
| 独立profile make smoke | 最终桌面Chromium 38/38，约3.5分钟，见[浏览器](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-lifecycle-implementation/smoke.log) |
| API重启/原结果精确重试 | 三个来源×三日期和两笔终态摘要一致；补充样本T04_VERIFY_65ea6791，见重启查回及[正常菜单样本](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-lifecycle-implementation/menu-samples.json) |
| 最终文档、暂存diff与历史保护 | 通过；见执行清单和最终文档日志，旧报告及211项来源身份/原件摘要保持 |

最终完整集成包含入口类型限制及其反例；随后readOnly汇总修复仅改变详情的能力汇总，受影响权限专项在修复后重跑，完整集成其他未变参数复用。最终Chromium和构建均在全部产品代码修复后执行。最后新增多轮边界测试只改手写断言（下述标签修正），Go域测试及最终check重跑，未无故重复未变DB/页面全集。

TL按实际参数与层次登记在执行清单，以下不表示253例整体关闭：

- TL-048/049：G中R纠错→S派生而下一R保持，首条→S准确继承；D补充最终修订及原来源；B补充多轮正常菜单。
- TL-055：A直调停用名称更正拒绝，B无入口。
- TL-059—063：G三时、同名恢复、中文部分输入/英文清空及D状态；B补今日启用仍可安排未来恢复，A/D补专用权、完整快照及来源。
- TL-064：本批仅G单条S/R拒绝及B安全错误定位，联合两项成功留T08，整例不关闭。
- TL-065—070：G非法后继、重复S/R、首条前/各类同日、12格状态及新身份CREATE；A/D补原子拒绝及并发，不在浏览器机械重跑12格。
- TL-071：主A夹带/空changes/null/缺字段/重复键；G补规则。
- TL-072：G合法两种S原因/一种R原因及交换/缺失、固定标签；A实际目录、拒绝及原因快照，B不预选。
- TL-073：G完整6段链及三时状态解释，D逐次来源只增1，B未来多轮及纠错独立性；未纳入删除/改日参数。
- TL-074：主D过去合法分步、本域旧来源/事件不变；未接入消费者，采用验证不算通过。
- TL-173：主A省略英文可R、显式同值英文无权拒绝，补B确认默认；TL-174仅新增S/R的accepted撤写权后读取参数，联合及其B补层仍未闭合。TL-178主D补原授权快照字节保持，TL-218补无日期读权的安全依赖诊断及撤历史读权拒绝，完整终态×撤权组合继续按后续矩阵执行。
- TL-196/198—203/208/211：补S/R原结果、丢响应、重启、整笔回滚后重试/resolve/竞争及迟到封闭；9阶段故障矩阵未在本批重建，TL-209仅复用共同事务回归，不能计全参数完成。
- TL-220/221/224/225/226：长线Go取消、真实DB整体预算和锁释放为共同框架回归；TL-226新增S/R单条时钟参数，联合及B跨午夜仍留T08。
- TL-228/234/240/243—245：本次两DTO专有边界、单条终态、当前安装及严格匹配/解码；多目标、联合授权不计。TL-248只补G矩阵，A单项联合未实现；TL-249只补B创建/启停/更正/历史旅程，改日/删除段仍未闭合。TL-168/169及TL-187的改日/删除/真实跨域引用矩阵本批未执行。

## 4 失败、修复与复验

首轮API专项发现生命周期命令未映射为内核ADD项，返回unsupported；补齐固定映射后专项通过，保留[首轮失败](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-lifecycle-implementation/integration-first-failure.log)。新日期诊断测试对安全省略的field断言误要求空字符串，修正为接受省略或空串后通过，不是业务泄露。

自查发现扩展候选类型后，普通ADD入口也须锁定CHANGE以保护专用授权；已对保存及预览增加固定映射校验，并用保留普通权但撤S/R权的主体分别验证伪造两类请求拒绝。另修正仅有启停权主体的readOnly汇总字段，专用能力不被普通动作状态覆盖。收尾验证及证据复用边界在执行清单记录。新增多轮日期边界断言曾把对外“未生效”标签误写成内核守卫的no_fact；按现行StatusAt合同改为not_effective后通过，保留[断言失败](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-lifecycle-implementation/domain-boundary-assertion-failure.log)，未修改产品状态语义。

封存时首次文档检查发生在execution清单写入前，报告链接因目标尚不存在被正确拒绝；补齐清单和归档后重新执行最终check/docs。该次失败见[结构检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-lifecycle-implementation/check-first-archive-failure.log)，未把缺失证据算通过。

## 5 未覆盖范围与限制

不实施T05删除、T06改日、T08联合/首条补全/分类纠错；T01/T02相应内核/JSON/多目标终态和完整矩阵保留。AC006/015只覆盖T04启停分支，不能按整项完成；TL-064联合成功留T08。TL-241/242/246/247已退役，不执行、不记通过。

未运行T07全容量、Firefox、M03/M04消费者、生产备份恢复、外部系统交换或完整X03授权管理界面。本域实际引用范围仍是企业归属及BU内部准确修订，不以恒空替身宣称跨域引用保护完成。当前名称上下文和历史准入边界不扩大。

前端主块仍有大于500KB建议值的构建提示，构建成功不代替性能验收；无独立人工业务批准、推送、PR、合并或对外部署。

## 6 结论与后续事项

T04停用/重新启用完整纵向流程已接通并完成本批分层复测，当前名称规则及既有创建/查询、资料新增/普通更正回归通过。T01/T02跨后续动作逐项保留。后续顺序为T05/T06/T08，再T07综合验收；完成本批后停止，不自动开始下一批。

## 7 证据保存

data目录保存非敏感命令日志、输入/日志摘要、TL参数映射和合成样本；归档日志仅去行末空白，原输出在.local/verification/t04保留。凭据与会话正文不归档。旧定稿报告和全部来源资产保留原字节，不提升未完整采用的资产状态。
