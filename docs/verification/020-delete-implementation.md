# CHNHCM-020 T05分类删除及依赖保护实施验证

登记信息：[registry.json](../governance/registry.json)。适用范围：T05普通、停用、重新启用记录删除及受影响回归。依据：[010](../plans/010-m02-scope-and-plan.md)、[011](../specs/m02/011-m02-business-requirements.md)、[012](../specs/m02/012-m02-functional-and-interaction-design.md)、[013](../specs/m02/013-m02-technical-design.md)、[023](../specs/m02/023-m02-timeline-test-cases.md)、[024](../adr/024-timeline-implementation-simplification.md)和[025](../adr/025-greenfield-implementation-policy.md)。

## 1 对象与受测基线

2026-11-01在CHNHCM-wt-dev-a、固定wt-dev-a实施，基线72f93ea206e7428f43b348ff5fac9c3aed0d17a1；开工工作区干净、stash为空。本报告记录该基线外的T05及受影响共用路径；受测源文件与测试的摘要见[输入清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-implementation/tested-inputs.json)，实现差异另存同目录source.patch.gz。最终交付提交另由Git历史标识，不将报告自身提交冒充所有测试的受测基线。

## 2 环境与前置

Go1.26.0、Node24.19.0、Python3.12.3、PostgreSQL18.6、桌面Chromium140.0.7339.186。独立profile m02-t05-current，Web4466/API8466/DB54546；当前存储m02-timeline-4，00001/00002隔离空库安装。凭据只在仓外0700目录/0600文件。旧4456/4436/4416及更早环境不迁移、不清库、不换用新程序。DB/API测试用独立Testcontainers；浏览器使用任务独立数据库及端口。当前[运行程序摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-implementation/runtime-match.json)与最终make build产物一致。

管理员从“设置 → 组织配置 → 业务单元 → 列表 → BU详情 → 生效日期轴选择记录 → 标题旁更多操作 → 删除此记录”办理；查询者仅可读。目标原日期、资料及分类只读，无删除原因或名称输入。预览检查全部后继、完整期间及今日状态；必须预览通过才可确认，确认后保留查看日期和列表上下文。

## 3 实现与实际验证

### 3.1 业务旅程与保护

| 场景 | 独立预期与实际结果 | 结论及证据 |
| --- | --- | --- |
| 普通中段删除 | 菜单建立1月甲/Alpha、6月乙/Beta、10月丙/Gamma，删除6月后1月至10月为甲/Alpha；10月独立快照保持。原来源6月仍为乙/Beta；新来源可独立选择删除原件，保留原期间至10月，和查看日事实分开。返回列表保留搜索、查看日和所选BU | 通过，[浏览器](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-implementation/smoke.log)、[DB/API](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-implementation/integration.log) |
| 三类日期与启停依赖 | Go以手写F2/F3/F4/F5/F8/F10核对候选、区间及目标前一天/当天/后一天事实。普通尾条、末尾S/R可删；首条唯一/有后继/未来均拒绝；删S留下R、删R留下CHANGE或S拒绝，不级联或改分类。未来S删除今日仍启用 | 通过，[领域](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-implementation/domain.log)，DB/API与菜单取代表参数 |
| 同值及准确继承 | F8同值前序删除仍accepted；数据范围空，但前段期间延长、删除结构及S继承改指前段准确新修订；已有R独立快照不变。预览/保存仅已分配修订ID不同，不带内部candidate项标识 | 通过，领域及DB/API；菜单显示期间调整并实际保存 |
| 原件、来源和同日新身份 | 删除追加证据修订、完整新来源、审计/Outbox及终态；活成员排除删除修订，运行角色不能改写/删除证据。删除后同日新ADD使用新ID；原重放/resolve仍返回原删除修订、原来源及原日的新适用前段 | 通过，DB/API；新成员引用删除修订被真实复合外键拒绝 |
| 当前类别分权 | 普通/S/R分别使用三类专用删除权，write_fields为空可办；普通删除权不替代启停权。撤写权仍读原终态，撤英文读权裁剪，撤历史/对象读权分别安全拒绝 | 通过，DB/API；查询者菜单无删除 |
| 严格输入、竞争和诊断 | 额外字段含空串、缺/null/重复键、过期来源/修订、真实跨BU目标均拒绝。两身份同源并发仅一笔accepted。依赖拒绝原诊断可查回，不将整体错误定位到不存在的日期输入 | 通过，DB/API及[唯一首条补验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-implementation/db-first-record-final.log) |
| 回滚及恢复 | 三类删除各执行写后409、技术失败后重试先到/resolve先到/竞争，共12参数；整笔业务savepoint无残留，原锁安全拒绝或保持未知，唯一终态、迟到请求封闭。菜单accepted丢响应后原内容重试和刷新查回准确定位原删除修订 | 通过，真实事务/故障夹具与浏览器；不宣称COMMIT通信未知专项已全部覆盖 |
| 仅撤字段读权 | 取得删除原件后只撤日期读权，保留删除权；预览200及同时来源冲突409都清理目标、父详情、日志和输入缓存，重读后重新预览才可保存。裁剪日期不显示假开放终点；迟到响应保护保持 | 通过，[修复复验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-implementation/browser-read-revocation-final.log)及完整浏览器 |
| 跨午夜及重启 | 三类DELETE用注入Go Clock及DB登记时刻跨午夜，原办理日/状态及重放不改。最终菜单样本在真实API重启前后读取两个来源、两个查看日、日志及原终态，完整摘要一致 | 通过，[DB/API](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-implementation/integration.log)、[重启摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-implementation/restart-readback.json) |

### 3.2 检查与测量

| 检查 | 实际结果 | 证据 |
| --- | --- | --- |
| make generate | 通过；sqlc及OpenAPI客户端当前格式一致 | [generate.log](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-implementation/generate.log) |
| make check | 通过；docs、边界、运行脚本、Go测试/vet、名称表、前端类型及差异检查。最终文档封存后再执行，输出单列 | [check.log](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-implementation/check.log)、[final-check.log](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-implementation/final-check.log) |
| make build | 通过；既有前端主块大于500kB提示保持，未改变构建门槛 | [build.log](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-implementation/build.log) |
| make integration | 37根测试通过、TestBUSliceLoad按T07范围跳过；含新T05六根及创建/查询、名称、启停、来源、授权、当前安装和失败关闭回归。之后仅补强领域F5/F10及唯一首条测试，分别独立通过，业务源文件未变 | [integration.log](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-implementation/integration.log)、领域/唯一首条补验 |
| 隔离make smoke | 桌面Chromium完整59/59通过，其中T05八项，包含200/409两种撤字段读权反例；创建/查询、普通维护、启停、页面上下文和原恢复回归保持 | [smoke.log](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-implementation/smoke.log) |
| 七组视口及原生缩放 | 1440×900、1440×540、390×844、320×900、320×256、真实200%及400%全部通过；键盘可达、取消焦点、真实滚轮完整读取删除后资料，400%实际保存。固定窗口1280×1039；CSS640×450/DPR2及320×225/DPR4，visualScale=1，非裁图或设备倍率替代 | [矩阵及14张遮罩图](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-implementation/matrix-final/results.json)、[脚本](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-implementation/viewport.mjs) |
| 执行与清理预算 | 120000记录DELETE取消5次，最大本地停止77.591115ms，小于1秒。ADD/DELETE累计SQL超限分别10.0011/10.0020秒；真实取消DB锁释放分别5.17/5.55ms，原来源不变、无业务残留；显式resolve前无补写终态 | 领域及integration日志；不是T07全容量结果 |
| 历史/资产保护 | 683份既有报告、证据及来源字节不变；211项来源身份、原保留成员、整项状态及旧采用/证据保持，仅追加7项partial采用 | [历史核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-implementation/history-protection.json)、[资产核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-implementation/asset-protection.json) |

七组布局测量早于最后的读字段收窄修复；后改动仅触及当前读字段变化时的缓存清理、错误响应元数据及隐藏无权日期。矩阵主体具有完整读权，相关条件为真、布局/CSS和键盘路径未改，复用其布局证据；最新完整浏览器另实测两种收窄路径和320×256。未把早期缺少真实滚轮断言的结果作为最终矩阵。

### 3.3 案例映射

[TL及参数映射](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-implementation/tl-mapping.json)记录主要层、补充断言、实际参数和未完成部分。TL-075—083、085—088的单条合同按规定层验证；TL-084留T08，TL-089只计当前本域真实历史关系。公共权限、恢复、故障、预算及午夜案例只记录本批实际单条参数，不据代表参数关闭整例联合矩阵或所有动作。257个编号中的4项退役不运行，其余253例不笼统标通过。

## 4 失败及复验

首轮纯结构DB专项发现单条预览带内部candidate项标识、保存未带。单条已有确切recordId，现清除内部itemId，领域层仍保留有限项定位；预览/保存仅正式新修订ID不同。[首轮失败](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-implementation/db-delete-first-impact-failure.log)与后续通过分别保留。

首次本地up因尚未显式configure而拒绝端口环境变量；按现行017先configure再up，不绕过归属检查或改动其他环境。[原拒绝](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-implementation/runtime-first-config-rejection.log)保留。首次make check发现新报告未被导航，补齐docs入口后通过，[原输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-implementation/check-first-navigation-failure.log)保留。

收尾增添真实撤日期读权反例，先后复现成功预览仍保留旧缓存、来源冲突响应缺少当前读字段两个问题。[200失败](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-implementation/browser-read-revocation-before.log)和[409失败](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-implementation/browser-read-revocation-conflict-before.log)及遮罩图保留。Go返回当前读范围，成功和已授权4xx采用同一客户端清理/重读路径；当前读范围仅响应元数据，不进入不可变终态，历史正文继续取当前对象/历史字段交集。两例修复后及完整59项均通过。

七组矩阵第一次只有键盘可达断言；补真实滚轮测量时，采集器拒绝覆盖已有图片而报EEXIST，[原输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-implementation/viewport-evidence-collision.log)保留。改新任务输出目录后七组通过，旧图及原始输出保留在.local，未覆盖它们；这次拒绝不算产品业务失败。

归档后的暂存差异检查发现补丁上下文中的空格/制表和两份日志尾部空行触发空白规则；原输出见同目录staged-evidence-first.log。源码差异改用确定性gzip保留原字节，归档日志仅清理格式空白，原始.local输出不改。修复后重新核对文档和暂存差异，不重跑未变的业务矩阵。

## 5 未覆盖范围与限制

T06改日、T08联合/首条补全/分类纠错未开放；T01/T02相应分支、联合项与多目标终态仍保留。TL-084、完整联合恢复/故障/跨午夜矩阵和T07全容量未完成。实际DB取消只声明可观测的正常连接清理，网络失联/COMMIT通信不明等未观测分支不凭本地返回推定通过。TL-241/242/246/247退役，不计通过。

真实引用仅企业归属和BU内部来源/修订；M03/M04等消费者未接入，原采用保护不冒充跨域业务验收。没有完整权限配置模块、后台失败终结器、通用投递、生产恢复或Firefox验收。工具自查和分层执行不冒充独立人工业务批准；没有本轮推送、PR、远程CI、合并或部署。

## 6 结论与后续

T05单条普通/停用/重新启用分类删除及本报告明确的共用回归达到当前批次完成条件：同一Go候选、首条和完整后继保护、三类专用权限、原子追加及原件历史、精确终态恢复和正常菜单均有实际证据。最终完成核对见[交付审计](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-implementation/completion-audit.json)，检查结果汇总见[执行清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-implementation/execution.json)。仅关闭T05本批，不关闭010或T01/T02整项。

下一步依次为T06全类别改日、T08有限联合修订及T01/T02剩余能力、T07综合验收与交接。本批本地提交后停止，不自动开始后续任务。

## 7 证据保存

非敏感原始输出位于.local/verification/t05，长期证据归档在[data/020-delete-implementation](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-implementation)。保存最终命令输出、明确的首次失败、TL映射、遮罩图、脚本、来源/原终态重启摘要及输入摘要；[证据摘要清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-implementation/evidence-sha256.json)不包含自身，避免循环。source.patch.gz与tested-inputs.json共同记录脏工作区的受测源码/测试，不包含凭据、Cookie、CSRF或登录正文。旧定稿报告和来源资产不覆盖。
