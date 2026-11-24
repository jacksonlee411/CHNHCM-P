# CHNHCM-109 启停期间专用维护实施与多轮检查

登记信息：[registry.json](../governance/registry.json)。适用范围：[109计划](../plans/109-lifecycle-pair-maintenance.md)。2026-11-02启动、11-03复核；固定范围已完成；修复后最后两轮复核无新增主要问题，实际通过及限制如下。

## 1 基线、业务结果与退出范围

固定wt-dev-b，基线8709cc5bb1ce8cd1a6ce0f6447980e36298e13b9；初始工作区及stash为空，未切分支、未提交或推送。管理员从设置→组织配置→业务单元选中记录，仅新增两个专用操作：

| 操作 | 当前结果与边界 |
| --- | --- |
| 插入停用与重新启用 | 所选启用段内L＜S＜A＜R，无后继受企业/日期类型范围约束；默认最早两个合法日，不足两日仅禁用。弹窗只有两日期和分别空白的两原因，类型固定，无名称编辑；一次保存两条正式记录 |
| 删除停用与重新启用 | 从当前相邻S/R的任一条定位整段，不要求原本同次创建；缺配对无入口。列两日期/类型，实际可读资料发生变化才显示结果；一次删除两条，无保留名称/改类/自动补建选择 |

停用继承前期资料，恢复保存独立快照；删除正常重算期间与停用继承，后继独立快照及旧来源保持。任一条非法整笔不保存；写入未确认按原身份恢复。单条名称/日期/分类仍在更正记录，首条仍按普通两步维护；非法单条拒绝，不增加关联编排回退。

通用编辑器、添加关联操作、公开timeline-revisions写路由、通用预览及Go公开JointInput/PreviewJoint退出。多条分类纠错、日期交换和任意混合编排退役。内部复用已有候选/事务及REVISE_TIMELINE多记录来源存储；当前应用层仅构造固定两项，不新增配对表、迁移、兼容写格式或隐藏通用入口。旧来源、原件、日志及既存终态继续读取。

## 2 环境、数据保护与来源

任务测试使用隔离profile m02-maintenance-review，Web4548/API8548/DB54648，PostgreSQL18.6、Go1.26.0、Node24.19.0、Playwright1.55.1桌面Chromium。复用该任务既有数据库，未清库；真库用例另外使用Testcontainers独立空库。新Compose网络曾受本机地址池耗尽影响，改用既有任务profile，不调整其他容器或网络。

原4382的m02-current-b只重启API接入当前程序，不迁移、不初始化或重建业务数据。重启前后7个对象的头来源及完整时间轴摘要相同；正常菜单逐一复核6个既有样本，包括后继Nroth保护、删除原件、原启停、首条两份资料、旧分类来源和2028闰日。4527来源样本仅只读核对，头来源保持。另直接按原身份resolve读取4382的3个旧通用accepted结果，其中2个包含多目标UPDATE或混合处理，未重执行旧编排。

证据：[重启前](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/109-lifecycle-pair-maintenance/current-data-before.json)、[重启后](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/109-lifecycle-pair-maintenance/current-data-after-api.json)、[6样本菜单结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/109-lifecycle-pair-maintenance/current-samples/ui-result.json)、[旧结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/109-lifecycle-pair-maintenance/legacy-results.json)。公开开发登录预设仍为admin/reader、admin1234567；数据库凭据、cookie及CSRF不进入证据。

## 3 检查轮次、发现和处置

本轮是主代理多轮自查与复测，不是独立代理评审或人工批准。

| 轮次 | 发现与修订 | 关闭依据 |
| --- | --- | --- |
| R1 接线与原子性 | 固定请求及公开通用退出；拒绝配对的元数据/错误码须能由当前固定授权正确解码及重放，补原拒绝查回 | 新真库日期/配对/拒绝重放、原结果、原件及故障边界测试 |
| R2 回归与权限 | 完整真库首轮3处旧测试假设未迁移：旧入口权期待、已移除jointEdit解引用、池预算夹具/预览输入；按现行普通或固定入口修订。删除类型显示按当前读权裁剪，未确认退出保留引用 | 完整真库重跑通过；固定表单隐藏字段、两方实际权限与恢复用例 |
| R3 反向语义与交互 | 发现仅两项启停权时入口被普通维护外层条件隐藏，修复菜单条件并补正常菜单用例；取消焦点纳入pairMode；恢复日下界与Go窗口一致。清理当前文档中的跨节点/多条分类/联合预览正向承诺及退役追踪 | 权限专用用例、日期联动及字段读权测试，7组视口；011—013/023语义复查 |
| R4 范围和证据复核 | 无新增主要产品发现。第一次完整130项浏览器有1项超时：主代理同时运行的权限探针共用测试账号，短时撤权影响另一进程的新建入口；该次129通过/1失败不能记完整通过。随后禁止同profile并行浏览器，顺序重跑完整132项 | 首次失败保留；随后完整132项顺序回归通过，原失败保留 |
| R5 最终冻结复核 | 完整132项顺序回归通过；无新增主要发现，当前应用/测试源未再改变 | 最终223份输入摘要、文档及历史原件反查、原端口数据与运行核对 |

视口脚本前3次另发现测试准备问题：选择记录后尚未等详情刷新就用键盘聚焦、窄屏仅处理紧凑菜单而漏了日期省略菜单。改为等待所选资料实际显示，并经原有日期菜单选择；未修改产品业务规则。第4次7组全通过，首轮及各轮结果保留。

## 4 实际验证结果

| 检查 | 实际结果与证据 |
| --- | --- |
| make check | 通过：文档/边界/脚本测试、Go单元及vet、TypeScript、diff；[最终命令输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/109-lifecycle-pair-maintenance/check-final.txt) |
| make build | 通过；[构建输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/109-lifecycle-pair-maintenance/build-final.txt)，保留现有MUI打包指令及体积提示，未调整构建阈值 |
| 完整真实DB/API | `go test -tags=integration -timeout=20m ./tests/integration ./internal/bootstrap -count=1`通过：integration 397.118s、bootstrap 214.142s；[完整输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/109-lifecycle-pair-maintenance/integration-all-round2.txt) |
| 固定与共用故障保护 | 覆盖固定两条的来源/修订/配对/日期拒绝、准确重放、恢复快照、双侧权限、隐藏字段、原件与封口、真实COMMIT/断连/取消/清理/锁及重试resolve竞争；不把控制故障栈记产品失败 |
| Chromium正常菜单 | 完整132项顺序回归全部通过，17.7m；[完整输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/109-lifecycle-pair-maintenance/browser-final.txt)。不沿用首轮129/130或旧146项通过 |
| 受控跨午夜菜单 | 固定两条通过：原校验日2026-06-14、实际登记跨至上海次日，原结果恢复仍为未来安排；[实测](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/109-lifecycle-pair-maintenance/midnight-browser.txt) |
| 视口与输入 | 7组通过：1440×900、1440×540、390×844、320×900、320×256，原生200%为640×450、400%为320×225；21张页面图加2张浏览器校准图。核对键盘、真实滚轮、错误原因聚焦、取消焦点及两条实际保存/删除；[逐组结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/109-lifecycle-pair-maintenance/viewport-fourth/results.json) |
| 文档/原件反查 | 322稳定案例，278现行/44退役；新16例，旧通用39例退出、原5退役保持。1924份既有来源/资产/报告文件未改；26份退出测试/工具原文件与基线逐字一致；[R4审计](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/109-lifecycle-pair-maintenance/audit-round4.json) |

## 5 测试迁移与覆盖边界

新固定领域/应用/真库用例以及8项页面旅程覆盖成功、取消、日期空档/历法/上界、继承/独立快照、双侧权限、失败整笔、冲突重读和未知恢复。现行调用与OpenAPI/生成客户端同步，不以旧入口别名维持测试。原通用协议的24个Go测试函数退役或按新请求重新设计；分阶段回滚、原结果读取和当前授权等共同保护在受支持的普通/固定命令上复验。没有声称旧函数的全部参数一对一迁移。

[测试迁移记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/109-lifecycle-pair-maintenance/previous-test-routing.json)和[案例范围](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/109-lifecycle-pair-maintenance/case-scope.json)区分退出与继续承担的责任。关键原测试及旧运行工具分别保存在[原测试归档](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/109-lifecycle-pair-maintenance/previous-general-tests.tar.gz)、[原工具/容量负载归档](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/109-lifecycle-pair-maintenance/retired-general-runners.tar.gz)，其余被调整测试的完整原文可由基线Git追溯，不修改旧验证报告。当前容量工具已改用普通逐条新增/更正，退出任意2/20项及全时间线同时修改；新工作负载仅构建/静态检查，未重跑2000BU冷热容量专项，不继承旧约200万成员的测量结论。

未执行Firefox、全浏览器、人工验收、生产恢复或真实消费者接入；未宣称278现行案例的所有层次/参数整例通过，也不宣称全M02完成。旧T07及104—108仍只持有原批次结果。

## 6 冻结输入与最终状态

[受测输入清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/109-lifecycle-pair-maintenance/tested-inputs.json)及[原始输入归档](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/109-lifecycle-pair-maintenance/tested-inputs.tar.gz)包含223份当前程序、配置、测试及本轮核验脚本；4份未改SQL另与基线逐字比对，见[DB输入](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/109-lifecycle-pair-maintenance/unchanged-db-inputs.json)。最终回归期间冻结，之后不再改应用或测试源。旧定稿证据保护及退出原件核验见[基线归档审计](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/109-lifecycle-pair-maintenance/retired-input-audit.json)。最终132项回归及输入复核通过，见[冻结核验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/109-lifecycle-pair-maintenance/final-input-check.json)。R4、R5及最终范围反查无新增主要问题；已完成的发现均复测关闭。任务4548环境停止、卷保留，原4382仍运行并核对归属及readiness；未提交、推送、PR或合并。

## 7 发布准备整合与验证边界

2026-11-03用户另行授权提交、push、PR、跑绿合并与同步。先以56c201c提交本轮实现，再保留并整合wt-dev-a的a9a40a7环境收尾规范提交；登记表保留双方验证记录，013保留当前固定操作/容量范围及003收尾要求，未恢复旧通用编排。原223份受测输入和归档不回写；整合仅改变AGENTS资源规范及文档/模板/原收尾记录，Go、页面、API、SQL与测试源均保持本轮受测版本。发布前重新执行本地质量/生成检查，远程quality、integration、browser须在PR最新提交全部成功后合并，实际发布结果由GitHub持有。

提交检查发现7份新入库的Testcontainers原始输出带库打印的行尾空格。保留原件字节，并按既有原始日志归档惯例仅对这7个准确路径设置whitespace属性，不豁免源码或一般文档。原检查与首次失败记录保持，不把发布准备整合说成已完成远程CI。

任务m02-maintenance-review已停止且持久卷保留，恢复按017使用该profile；4382/m02-current-b保留用于本次管理员验证，交付代理已核对归属和readiness，用户验收结束或下一开发任务开始复查其保留需要。4527仅作为原样本来源对照，本轮未改其业务数据或服务；由来源工作区维护，不纳入本轮停止范围。

发布范围相对origin/main的检查另发现此前已提交的3份105原始Testcontainers日志同样带工具行尾空格；追加这3个准确路径的归档属性，原始文件字节保持，未扩大源码豁免。

PR #5首轮远程CI 41538914877的quality/integration成功，browser因整项任务20分钟时限被取消，日志124项通过、0项失败，余项未完成不计通过。发布修订只将browser job预算改为30分钟，保留全部132项、单worker、零重试及原断言/单例超时；新head仍须重新三项跑绿后合并。此次取消及资源停止结果保留在GitHub原运行记录，不改写前述本地132项通过证据。
