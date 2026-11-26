# CHNHCM-116 R2事务与恢复提炼实施记录

登记信息：[登记表](../governance/registry.json) `governance.verification_records`。
适用范围：R2-A共用事务端口和PostgreSQL事务设施；R2-B提交重试、查回、当前读取授权。BU必须真实使用，部门运行时后置。
上游依据：[116](../archive/plans/116-m02-shared-dated-maintenance-roadmap.md)、[120](../specs/m02/120-dated-maintenance-technical-contract.md)、[121](../specs/m02/121-dated-object-admission-matrix.md)、[023](../specs/m02/023-m02-timeline-test-cases.md)、[合同评审](116-r2-contract-review.md)。

## 1 验证对象与受测基线

2026-11-05，工作树 `/home/vivobook/projects/CHNHCM-wt-dev-b`，分支 `wt-dev-b`，基线 `90dce9927c9e3541898b045070982054a11099b1`。本轮增量已实施并分批复核；受测输入、差异及哈希见第7节。R1修复已本地提交；本轮R2增量留在工作区，未提交或推送。

## 2 环境与前置条件

运行环境为Linux/amd64、Go 1.26.0、Node 24.19.0、pnpm 10.24.0、Playwright 1.55.1及固定镜像PostgreSQL 18.6。开始时存在3个持久PostgreSQL容器，见[data/116-r2/runtime-before.txt](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r2/runtime-before.txt)。既有持久环境保留；本轮测试使用独占环境。管理员从“设置 → 组织 → 业务单元”执行创建、维护、查回；查询者及字段受限角色检查当前读取与历史权限边界。本轮完整DB使用独占Testcontainers，命令按默认通道运行；浏览器临时profile为r2-20261105，Web56338/API54470/DB55374；测试输出独占`.local/playwright-116-r2`。既有日常profile m02-current-b先核验Web/API归属及就绪，完整DB通过后重新编译并重启本profile的API；数据库、前端和持久卷保留。

## 3 验证过程与结果

| 场景或检查 | 前置与操作步骤/命令 | 预期结果 | 实际结果 | 结论 | 证据位置 |
| --- | --- | --- | --- | --- | --- |
| R1修复后基线 | go test ./internal/contexts/organization/... -count=1 | 基线通过 | 通过 | 通过 | [baseline-go.txt](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r2/baseline-go.txt) |
| R2-A端口与事务 | 有界清理、保存点、提交回执、纯读取安全审计 | 不泄露未确认结果 | Go、导入边界及8项真实DB顶层检查通过 | 通过 | [a-integration.txt.gz](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r2/a-integration.txt.gz)、[a-go-final.txt](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r2/a-go-final.txt)、[a-boundaries.txt](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r2/a-boundaries.txt) |
| R2-B共用合同 | 非BU结果类型、原编码常量、授权优先级、字段副本 | 复用原结果且不依赖BU DTO | Go及race检查通过 | 通过 | [submission-contracts.txt](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r2/submission-contracts.txt) |
| R2-B完整DB/API | 四类终态、身份隔离、重试/查回竞态、撤权与字段裁剪 | 复用原提交且执行当前读权 | tests/integration 542.227秒、internal/bootstrap 343.803秒，均退出0 | 通过 | [integration.txt](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r2/integration.txt) |
| 完整Chromium回归 | 正常菜单、恢复、最近操作者、日志来源、当前资料维护 | 保持既有表达及权限语义 | 单轮186/186通过，32.7分钟，退出0 | 通过 | [browser.txt](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r2/browser.txt) |
| 日常环境更新 | m02-current-b核验、restart-api及管理员/查询者菜单读取 | 当前API可用，保留既有事实 | 两角色均7条，列表/详情HTTP200，零BU写请求 | 通过 | [persistent-restart.txt](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r2/persistent-restart.txt)、[persistent-menu-retest.txt](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r2/persistent-menu-retest.txt)、[persistent-after.txt](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r2/persistent-after.txt) |
| 原证据与资源收尾 | 对照2717份原材料、1328个运行输入及本任务资源 | 原材料/受测代码不变，临时资源归零 | 哈希均一致；容器/网络/卷/私有配置均无残留，3个临时端口关闭 | 通过 | [protection-and-cleanup.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r2/protection-and-cleanup.json)、[runtime-after.txt](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r2/runtime-after.txt) |
| 静态门禁 | make check | 文档、边界、工具/Go、vet、类型及差异检查通过 | 通过 | 通过 | [check.txt](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r2/check.txt) |

### 3.1 实现与复核边界

| 复核点 | 实际实现及业务含义 | 本批证据 |
| --- | --- | --- |
| R2-A 提交边界 | BU创建、普通维护和查回均经submission.Run；外层提交失败时清空临时结果和业务拒绝，客户端仍按原提交查回 | 四类终态取消及21项动作/中断组合；a-integration |
| R2-A 保存点 | Scope处理外层事务和业务保存点；BU查询器在业务保存点内替换、退出恢复；回滚不明确时关闭连接，不补写终态 | panic、取消、保存点丢失、连接中断及共享清理截止时间 |
| R2-B 精确恢复 | Lookup串行化同类型本人提交；Replay保留原摘要、封闭终态优先，再做当前授权和诊断读取；Resolve只在本类型无结果时写not_applied | 独立锁键/摘要常量、非BU合成结果及真实BU多连接锁检查；完整DB通过 |
| R2-B 当前读取 | BU提供对象/历史动作及安全错误；AuthorizeHistory取两类字段交集并复制返回，不修改X03判定字段；无原写权重检 | 非BU字段夹具和原始判定不变断言；真实撤权/恢复、损坏结果优先级完整DB通过 |
| 对象边界 | BU表组、SQL、编解码、原件定位、候选与完整动作/字段写权仍归BU；通用层无BU DTO、sqlc或动态表名 | [scope-review.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r2/scope-review.json)；导入边界 |

011/012、117—119及API合同不增加新业务含义，故本批不制造重复合同增量。013/120持有技术合同，116/121/023持有阶段和验证映射。资产清单未新增来源资产或提升完整资产验收状态。

### 3.2 正常菜单的122/123回归

管理员从设置菜单进入BU，正常新增、更正和删除后，基本资料仍以所选修订显示操作者与时间；变更日志逐字段表达差异，查看当次资料、来源明细及删除前原件留在表格内。无变化和拒绝不刷新摘要；未来首条不被当作今日事实。查询者、独立历史权及字段撤权反例由本轮真实Go/API/DB浏览器套件执行。

本轮截图另存[桌面日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r2/screenshots/log-desktop.png)、[320宽日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r2/screenshots/log-320.png)、[删除原件](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r2/screenshots/deleted-original.png)、[未来首条摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r2/screenshots/future-summary.png)，未覆盖原122证据。主代理检查这4份截图的页签、轻量摘要、表内链接和受限宽度呈现；截图只是已执行旅程的补充，不代替权限及原子性断言。

## 4 失败、修复与复验

R2-A：共用事务设施和强类型提交边界已由BU调用。主代理对照基线逐分支复核外层提交、保存点恢复、取消及有界清理；业务SQL和终态落库仍归BU，未发现未关闭问题。真实DB定向8项顶层检查通过；Resolve最终改用相同Run边界后，Go检查通过，其最终真实DB验证已在R2-B完整回归通过。R2-B同样完成原身份、授权顺序、字段副本及对象边界复核；无未关闭实现发现。

日常环境补充脚本首轮失败：登录已触发列表读取，脚本却到点击同一路由菜单时才监听请求，因此等待新响应超时；见[persistent-menu.txt](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r2/persistent-menu.txt)。修复仅把观察器放到登录前，不修改产品代码、断言或权限；复测两个角色均通过：正常菜单列表/详情HTTP200，均读到7个BU，BU业务写请求为0；见[persistent-menu-retest.txt](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r2/persistent-menu-retest.txt)。完整186项浏览器回归独立运行，不与该附加脚本合并统计。

## 5 未覆盖范围与限制

不含R3前端流程提炼、R4部门、R5双对象验收、R6收口及生产验收。完整integration默认不运行显式容量及两项附加跨日浏览器通道；本轮不据此重算T07容量或声明R6完成。合成结果类型仅证明共用端口不绑定BU；命名空间验证不代表部门已接入。数据库迁移和公开BU协议本轮保持现有合同。

## 6 结论与后续事项

R2-A和R2-B均已完成实施、逐批复核与本轮验证；BU实际使用共用机制，完整默认DB和单轮Chromium186项通过，当前没有未关闭的R2发现。该结论由主代理执行与复核，不冒充独立评审、人工业务签收或整个116完成。R3—R6及部门39项产品样本保持未实施/未运行。

Testcontainers及r2-20261105临时容器、网络、卷、私有配置和配套进程已回收，56338/54470/55374关闭；原3个持久数据库容器保持相同身份及运行状态。日常m02-current-b在[4382入口](http://127.0.0.1:4382)运行本轮API，管理员/查询者只读核验通过；该环境供用户日常验证，由当前工作树实现/验证维护者负责，在用户结束使用或下次服务/版本交接时复查。其他持久环境未变更，未清理其卷。

下一阶段为116 R3：在现有BU菜单中提炼对象定义、类型API适配和办理流程；沿用本轮提交端口，继续保护122/123和114/115，不提前接入部门表或通用树平台。

## 7 证据保存

长期证据置于[data/116-r2](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r2/baseline.json)；原2717份报告、验证及来源文件的保护哈希见[protected-before.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r2/protected-before.json)。运行源码快照见[runtime-inputs.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r2/runtime-inputs.json)，相对90dce99的可复核实现差异见[implementation.patch.gz](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r2/implementation.patch.gz)。R2-A源码另存[a-source-manifest.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r2/a-source-manifest.json)。命令及逐项结果见[execution.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r2/execution.json)，最终文件摘要见[final-changes.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r2/final-changes.json)，范围复核及收尾见[scope-review.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r2/scope-review.json)和[protection-and-cleanup.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r2/protection-and-cleanup.json)。后续文档收口仅复验文档门禁，不重跑或重记已冻结源码的产品测试。
