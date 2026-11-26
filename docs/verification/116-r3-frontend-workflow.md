# CHNHCM-116 R3前端流程与BU接入实施记录

登记信息：[登记表](../governance/registry.json) `governance.verification_records`。
适用范围：R3-A共用前端流程，R3-B BU接入、实际菜单和完整回归；不含部门产品或R4—R6。
上游依据：[116](../archive/plans/116-m02-shared-dated-maintenance-roadmap.md)、[120](../specs/m02/120-dated-maintenance-technical-contract.md)、[121](../specs/m02/121-dated-object-admission-matrix.md)、[范围评审](116-r3-contract-review.md)。

## 1 验证对象与受测基线

2026-11-05，wt-dev-b，基线2c8b9c898938cafcbedb9f6250b1c308673a3280；该本地提交已完成R2收口，无推送。R3增量为前端类型适配、共用机制、BU页面接入及文档/测试；Go、SQL、公开API和存储合同不变。R3增量尚未提交；实际受测输入及最终比较见[输入核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3/input-verification.json)。

## 2 环境与前置条件

主代理执行、逐批复核，非独立评审或人工批准。隔离真实BU环境使用公开开发登录和合成数据；不写日常持久数据，不收集真实凭据。定向恢复分别使用临时profile r3-a-20261105及r3-a-retest-20261105；最终完整浏览器使用r3-reviewed-20261105，Web54896/API54286/DB56952。三组视口另用独占临时profile，实际端口和退出结果随各驱动记录保存。真实DB使用Testcontainers，日常m02-current-b及其他持久环境保持。

## 3 验证过程与结果

| 批次或检查 | 前置与动作 | 预期 | 实际与结论 | 证据 |
| --- | --- | --- | --- | --- |
| R3-A 共用提交机制 | 新建、单条维护、固定两条实际调用SubmissionFlow；执行9项独立机制测试 | 原内容重试、同类型阻塞、跨类型隔离、查回胜过迟到重试、会话/挂载隔离、存储失败保留引用 | 9项通过；不代表部门产品 | [机制结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3/a-submission.txt) |
| R3-A 类型检查 | API适配器使用生成DTO，移除编辑器对BU页面恢复函数依赖 | 无DTO强制转换，无并行旧提交实现 | 通过 | [类型检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3/a-typecheck.txt) |
| R3-A 定向恢复菜单 | 管理员从列表新建/保存；丢响应、查回、撤权、退出、迟到响应 | 原件/字段/会话保护不回归 | 冻结输入后19项通过；首轮17通过/1失败单独保留 | [复验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3/a-browser-retest.txt)、[输入核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3/a-input-verification.json) |
| R3-B BU页面闭环 | 列表/详情/表单共享读取、离开、选择导出及导航；字段和能力由BU映射 | 现有菜单与114/115/122/123行为保持 | 逐批复核通过；单轮187项Chromium通过，零重试，33.1分钟 | [完整输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3/browser-reviewed.txt)、121第8节 |
| 完整真实DB | make integration | 当前BU事实、历史、原子性、恢复、权限无回归 | 通过，525.872s/329.716s；Go输入未变 | [DB输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3/integration.txt) |
| 静态与单元 | 最终产品修订后make check | 文档、边界、Go单元/vet、脚本、提交机制、类型正反例通过 | 通过，73项Python、17项Node（其中本轮机制11项）、236份Markdown及Go检查；Go缓存结果按原输出读取 | [最终产品检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3/check-reviewed.txt) |
| 构建 | 最终产品修订后make build | Go服务/upgrader及前端可构建 | 退出0；Vite保留现有MUI指令及包体积提示 | [构建输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3/build-reviewed.txt) |
| 表单、固定两条、列表视口 | 每组桌面、短屏、390/320窄屏、320×256及真实200%/400%缩放 | 字段直接可见，键盘、滚动、错误定位、原件和返回可达 | 三组分别7/7，共21组通过；真实缩放dpr2/4、visualScale1，未用CSS缩放替代 | [汇总](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3/viewport-summary.json) |
| 日常环境只读菜单 | admin/reader从设置→组织配置→业务单元读取列表和详情 | 当前页面可访问，查询者无新增记录，零BU写入 | 两角色列表/详情均200，businessWrites=0 | [菜单输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3/persistent-menu.txt) |
| 最终文档与格式 | 阶段状态、登记及证据同步后make docs、git diff --check，并检查新增文本 | 当前合同、进度、真实结果一致，历史报告不改写 | 73项脚本及236份Markdown通过；增量格式无发现 | [关闭检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3/docs-close-review.txt)、[最终门禁](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3/docs-final.txt)、[格式核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3/whitespace.json) |

## 4 失败、修复与复验

首轮类型检查发现React 19 useRef必须提供初值，已补undefined；后续类型检查通过，首次输出保留在[a-typecheck-first.txt](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3/a-typecheck-first.txt)。不把第一次失败记为通过。其余发现及复验如下：

| 发现 | 实际影响及修复 | 复验/状态 |
| --- | --- | --- |
| R3-01 类型初值 | React 19 useRef补显式undefined | 后续类型及完整check通过 |
| R3-02 开发期热更新污染定向运行 | 同时编辑产品源码导致旧会话案例填写原因时页面重载；非冻结受测输入，不能用17项成功冒充批次通过 | [Vite重载记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3/a-hmr.txt)及首轮失败保留；冻结1339文件后19项通过 |
| R3-03 迟到查回拒绝 | 同页原内容重试已确认后，过时resolve失败不得广播失权并清理新草稿；错误路径补引用仍存在检查 | 新增同一详情的新草稿浏览器反例通过，纳入完整187项 |
| R3-04 清理在途正文 | 放弃输入后若退出未完成，不能仍显示无限保存中；清正文同时失效原响应并转待确认，三元引用保留 | 独立机制第10项及最终check通过；完整187项菜单回归通过 |
| R3-05 旧视口脚本漂移 | 更新123已退出的查看日期/businessAt、122来源链接及115勾选列预期；不恢复旧产品入口 | 按当前菜单实测，原历史报告不改 |
| R3-06 视口脚本错误 | 新增CHANGE夹具选项误写其他资料调整，真实117为其他数据调整；脚本超时发生在选择选项，未保存 | 原输出与脚本压缩保存；另修详情等待点和独立seed的baseURL。表单/固定两条各7组通过；列表修正窄屏勾选列旧预期后完整7组通过，合计21组 |
| R3-07 文档登记与阶段表述 | 新报告空results未通过门禁，补实际结果；023第25章旧表述同步为R3验证BU、R5验证部门 | 初轮复验及最终状态同步后的文档门禁均通过 |
| R3-08 结果与错误表达 | 共用提交保留存储故障中文提示；非accepted/no_change的成功HTTP响应仍视为结果未明，不显示已保存 | 第11项机制反例及最终check通过；修订后单轮187项通过 |

完整浏览器首轮在27项通过后为追加上述复核修订主动中断，退出130并回收临时环境；该次不记完整通过，原输出见[browser-full.txt](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3/browser-full.txt)。后续修订后的单轮187项全部通过，完整输出独立保存，不拼接此前成功项。

R3-A范围复核通过：三类表单均调用共用提交机制、无BU页面恢复函数反向依赖、无部门DTO转换；原BU存储键/三元引用及精确API保留。首批9项机制、类型和19项正常菜单反例通过；后续复核补充两项机制后最终11项通过，并由R3-B最终完整回归覆盖当前输入。

R3-B范围复核通过：BU正常菜单的创建/查询、位置新增、统一更正、启停、分类删除、改日及固定两条，均实际调用当前流程；122摘要/日志、123日期语义、114/115选择/CSV、权限裁剪、来源/返回及未知提交恢复按本轮证据通过。复核发现均已修复并有对应复验；该结论为主代理范围复核，不是独立评审或人工签收。

### 4.1 当前调用与代码复核

[接线核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3/wiring-review.json)确认新建/单条/固定两条均调用useSubmission，三处离开保护共用useDraftExit，上下文/原因/列表/详情共用useOwnedRead；列表选择/CSV与前后对象分别调用useQuerySelection/useObjectNavigation。共用目录不引用BU页面/DTO/业务路径，BU强类型适配器持有全部真实写路由、字段目录及原结果转换。旧页面内提交/恢复和两份BU专用选择/导航实现已删除，无并行回退。

[后端核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3/backend-input-verification.json)确认190个后端/API/DB测试输入与R2提交完全一致；真实DB回归仍在本轮重新执行，不复用R2通过。011/012业务及交互语义不变，本轮长期技术合同同步120/013，验证与阶段映射同步023/121/116。

完整浏览器的[起始摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3/reviewed-inputs.json)与[最终摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3/final-runtime-inputs.json)比较：1339项中1338项不变，产品源码、默认E2E、API及后端输入均未变；唯一变化为不被默认E2E引用的独立community-list-viewport脚本修正勾选列预期，该脚本随后单独完成7/7复验。没有将视口脚本首轮失败计入187项通过。

## 5 未覆盖范围与限制

合成字段/结果类型仅验证端口，不计Department/Hierarchy实现。部门39项仍not_run；R4、R5双对象菜单及R6容量/接入标准后置。默认DB回归不含显式容量和附加跨日浏览器通道。本轮为本地默认Chromium及所列真实缩放检查，无远程CI或生产验收结论；单元/类型检查和历史通过不能代替本轮菜单验收。

## 6 结论与后续事项

R3-A和R3-B已完成并逐批复核通过；116整体仍in_progress，R4—R6未开始。共用前端已由BU实际使用，原页面内提交/恢复及BU专用选择/导航流程退出；不据此宣称第二对象或全M02完成。

日常m02-current-b（http://127.0.0.1:4382）已核验所有者与ready；admin/reader从正常菜单完成列表/详情读取均200，零BU写入，见[persistent-menu.txt](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3/persistent-menu.txt)，收尾[verify](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3/persistent-final.txt)退出0。该环境保留供wt-dev-b日常开发和手工复核；下次模型变更、运行维护或用户要求停机时复查。

[收尾核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3/cleanup.json)确认本轮七个临时profile的容器/网络/卷、私密配置及构建产物已移除，三个视口显示资源已退出；三个原持久容器均保留。2727份旧报告/证据原字节保持，仅验证导航README新增R3入口，见[历史核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3/history-protection.json)。已保存脱敏日志和合成页面证据，不回收原历史资产或其他环境。

## 7 证据保存

[基线](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3/baseline.json)记录R2提交和起始状态；[保护摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3/protected-before.json)覆盖2728份原验证文件。[最终增量](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3/final-changes.json)及[压缩补丁](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r3/implementation.patch.gz)保存实际代码/文档输入，证据目录另行保留，不递归写入补丁。本轮执行日志、失败、复验及合成页面截图保留在data/116-r3，临时本地输出不作为唯一长期依据。
