# CHNHCM-111 Community列表T05交付与收尾

登记信息：[registry.json](../governance/registry.json)。上游为[111计划](../archive/plans/111-bu-list-experience-upgrade.md)、[110](../adr/110-bu-list-and-datagrid.md)/[113](../adr/113-bu-list-community-adoption.md)及012/013/023。本记录按[验证模板](../templates/verification.md)追加，不覆盖既往报告。交付范围为Community列表及用户另行采纳的单列条件增量。

## 1 对象与受测基线

2026-11-03，固定wt-dev-b，受测基线`b63a2ded1920318b87cbb4a3ac5744567ef77e7a`及其已验收、尚未提交的单列增量。起点工作区dirty、stash/index为空；72个路径来自该增量。原[31份输入](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-single-column-filter/checks.json)逐一核对一致，其中17份运行输入保持原字节。T05仅修订当前资料/资产/登记、追加证据并按用户授权本地提交，不修改运行代码或供应商实现。

[基线及保护摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t05-delivery/baseline.json)记录运行输入、起点路径及314份历史报告/证据摘要；[最终检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t05-delivery/checks.json)记录本次最终输入与结果。本报告的基线不使用自身提交号代替。原报告中的“当时未实现/未提交/未关闭”仍为该时点事实，不倒填为当前通过。

## 2 环境、角色及前置

用户日常profile为m02-current-b：页面[4382](http://127.0.0.1:4382)、API8382、DB54373。进入路径为登录→设置→组织配置→业务单元，管理员及只读查询者分别验证；使用本地开发公开预设，凭据/会话/CSRF不写入证据。本轮没有造数、修改授权、调用业务保存或重建数据库；仅开发登录及真实读取。浏览器上下文及临时Chromium在finally关闭，日常Web/API/DB按用户指令保留。

真实业务与资源补验记录为[daily-runtime](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t05-delivery/daily-runtime.json)、[运行身份](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t05-delivery/runtime-identity.json)及[资源清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t05-delivery/resources.json)。当前API进程、源码参数及实际请求相互核对，不只依靠readyz。

## 3 阶段交付及实际结果

[阶段台账](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t05-delivery/stages.json)逐项列出依据与复用理由，不将组件实验、浏览器组数、业务案例和视口条件折算累加。

| 任务 | 交付及结论 | 证据与边界 |
| --- | --- | --- |
| T01 | Community8.29.3固定未转译源码及同版声明、自足依赖、MIT/来源、退出Pro许可运行路径；通过 | [原T01/T02](111-community-t01-t02.md)的冻结安装和来源/构建；包/锁及供应商未变，本轮[源码核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t05-delivery/source-audit.txt)再次通过501源码/500声明。旧Pro许可项按选型退出，不计通过 |
| T02 | 紧凑控制、四列/摘要、原生分页/列设置、A/D及名称链接和返回；通过 | 原组件17＋10及辅助7项仅各层证据；桌面普通行最终39px、长文auto，见[样板行高](111-cubebox-desktop-row-height.md)，52px记录只作历史；不含列拖动/固定 |
| T03 | 当前12例TL323—334、10组列表与4组维护代表、3项DB检查、七组视口及原生复制链接；通过 | [T03](111-community-t03.md)按原基线；本次不重跑整组原生复制链接或全维护。之后受影响10组列表已在单列批次复验 |
| T04 | 单条件原生面板/应用外壳/必要UI与Excel差距评估；通过 | [原评估](111-column-filter-evaluation.md)的7项仅组件实验，当时选择维持现有查询；后续实施授权单独记录 |
| 单列增量 | 编码/名称、状态单/多值、展示记录纯日期；统一应用/取消；真实Go/API/SQL/客户端；通过 | [单列报告](111-single-column-filter.md)：8新增＋10旧列表、7个集成根测试及七组含原生200%/400%；17/18原组合与TL333定向复验各自保存，不能称整轮18/18 |
| T05日常补验 | 管理员/查询者名称包含丙及取消，均HTTP200、1条、返回名称匹配、未调用业务写入口；通过 | 本轮[daily-runtime](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t05-delivery/daily-runtime.json)，两角色0个pageerror；仅追加当前日常合同接通的证据，不替代整个功能验收 |
| T05资料/资源 | 当前合同/计划/入口/资产与登记对齐；隔离环境停机，日常环境保留；通过 | 最终资料/引用/分轮语义检查及[资源清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t05-delivery/resources.json)；原件保护、后置范围及本地提交边界保持 |

条件与保护由011 BR009/BR010/BR015、012查询A/D及013单列合同持有，023 TL335—342持有新案例。T05闭合111第6节的阶段退出证据，不新增规则ID或用计划completed扩大342例的实际验收数。

## 4 运行同步问题、修复与复验

用户在日常4382打开“名称、包含、丙”曾收到“未知或重复的查询条件”。上一轮只读诊断在前端代理及8382直连分别复现；三个新增参数单独发送亦为400，旧API二进制不含filterField/filterOperator/filterValue，前端已包含。原因为Vite已更新前端，日常Go进程仍运行旧构建，未识别新白名单；值“丙”尚未进入查询执行。独立4394/8394验收通过没有证明日常环境已经同步。

按用户明确指令已执行同profile的restart-api，重新编译并只重启API；前端及数据库保留。此前授权重启后已正常查询，本轮再从管理员及查询者正常菜单补验：HTTP200、均1条、名称匹配、取消保留原A；没有业务写请求。此前诊断来自上一轮工具现场，本轮未启动旧程序制造失败；[问题闭合记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t05-delivery/runtime-sync-incident.json)明确区分其来源和本轮新观测。017补充Go改动后的重编译/重启及实际业务查询判据；不移除参数白名单、添加兼容回退或新版本平台。

原类型、表单布局、首轮中断及TL333旧预期失败仍见原单列报告/日志；314份既往文件受保护，不用本轮结果覆盖失败。T05[首次暂存检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t05-delivery/cached-diff-first.txt)发现原typecheck-second.txt含末尾空行；保持日志原字节，仅为该单一证据路径设置whitespace=-blank-at-eof归档属性后复验，其他源码空白检查继续执行。

## 5 检查、复用与限制

本轮运行输入与原单列验收相同，故复用其真实DB、18组最后业务结果、七组真实缩放、构建及静态检查；不为关闭资料重复登录/造数/写库或全量重跑。基线内未改的T01包/锁/来源继续引用原冻结安装。当前另执行公开源码核对、日常正常菜单两角色、资源盘点、提交前静态/单元/类型检查、资料和Git引用、语义复核及diff检查；具体结果由[最终检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t05-delivery/checks.json)持有。

本次由主代理执行并分轮复核，非独立评审或具名业务人员签收。不声明全部342案例、完整M02、整项CubeBox文件/资产、真实跨域消费者、全维护组合、Firefox/macOS或生产/容量SLA验收。各原报告保留自己的输入及未覆盖项；Excel候选/联动计数、跨列组合、通用高级查询、收藏/分享增强、分组和导出后置，列拖动/固定继续取消。

## 6 资源收尾、结论及提交边界

m02-community-t03-b（4393/8393/54393）及m02-column-filter-b（4394/8394/54394）两个自有测试容器均exited、六端口释放，持久卷和恢复依据仍在；先前矩阵的Xvfb219/221已关闭。本轮只读复查，不重复启动已结束环境或清理他人项目。

m02-current-b的Web4382/API8382/DB54373按用户日常验证需求继续运行；责任角色为当前工作树实现/验证维护者，用户结束本轮使用或下次服务/版本交接时复查是否停机。持久业务数据默认保留。T05的临时浏览器已关闭；保留运行用途及责任明确，不能把日常环境误当结束任务的临时profile关闭。

111的Community T01—T05及用户另采纳单列增量按限定范围交付完成，登记execution_status为completed，长期规格/ADR继续active。局部资产追加交付证据，整项来源资产complete_asset_verified保持false；没有删除原采用条目。后置功能须另获采纳并定义合同/计划，不能从本次关闭推导已获授权。

本次按用户指令提交当前已验收实现、文档及T05记录到固定wt-dev-b，仅本地commit；不执行push、PR、合并或切分支。提交后以Git实际提交及工作区状态回报，不在报告中预填尚未生成的提交号。

## 7 证据保存与复现

持久证据在[data/111-community-t05-delivery](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t05-delivery)，本地非敏感原始输出在.local/111-community-t05-delivery。日常补验脚本[原件](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t05-delivery/daily-verification.mjs)恢复到该.local路径后，从本工作树运行`CHNHCM_LOCAL_PROFILE=m02-current-b node .local/111-community-t05-delivery/daily-verification.mjs`；只登录和读取，不记录原始业务正文/凭据/令牌。日常在线判据同时包含脚本归属/就绪和正常菜单实际查询；仅有URL不证明后续一直在线。

原报告及图面引用仓内原路径，源仓/Pro历史档案与MIT当前源码保持各自许可身份；最终输入摘要、分轮复核和提交前结果提供可复核交付边界。
