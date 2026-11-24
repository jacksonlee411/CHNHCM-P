# CHNHCM-116 R5多轮评审与修复

登记信息：[registry.json](../governance/registry.json)。适用范围：116 R5及其既有未提交修复。
上游依据：[116路线图](../plans/116-m02-shared-dated-maintenance-roadmap.md)、[119交互合同](../specs/m02/119-department-functional-design.md)、[120技术合同](../specs/m02/120-dated-maintenance-technical-contract.md)、[121接入矩阵](../specs/m02/121-dated-object-admission-matrix.md)。

## 1 验证对象与受测基线

2026-11-06，CHNHCM-wt-dev-b固定wt-dev-b，HEAD为545860a40f9c2c2571258ea4a1a9ba768cea9e96。起始工作区含R5及两批评审修复；原报告与证据保留。本次由主代理按合同建立独立预期、多轮审查及实际复现，不以已有通过记录作为结论，也不代表另一名评审者或人工签收。起始文件摘要与资源身份见[基线](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review/before.json)。

## 2 环境与前置条件

真实PostgreSQL、Go API、桌面Chromium，管理员及查询者的合成样本。浏览器从正常菜单进入部门或BU，撤销真实授权后恢复；仅指定详情重读注入503、断连或延迟。临时profile独占并自动回收；持久m02-department-b不参与业务或授权写入。

## 3 验证过程与结果

| 轮次或检查 | 独立预期与实际 | 结论 | 证据 |
| --- | --- | --- | --- |
| 第一轮，最小引用读权与字段读权分离 | 撤BU/上级最小引用读权后真实详情不含名称，候选200空集；页面仍显示旧名称，两项失败 | 发现R5-F08/P1 | [反例](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review/round1.txt)、[探针](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review/round1-probe.ts.txt) |
| 第二轮，组合编辑与候选请求 | 仅修改名称后打开BU/上级候选，两者都发送changes={}并400 | 发现R5-F09/P2 | [独立复现](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review/f09-before.txt) |
| 新增真实DB/API | 可读但不合资格的选择、搜索/分页独立、部分/全部撤权、不存在身份同形、输入严格性及入口字段权 | 两项通过 | [API结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review/selected-api.txt) |
| 首批组合浏览器 | 首次三项因空关系补丁失败；修复后10项中8通过，2项重新启用准备漏传幂等标识而失败 | 产品与夹具失败分别保留，继续复验 | [首次](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review/f08-browser.txt)、[修复后](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review/f089-browser.txt)、[原夹具](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review/f089-tests-first.ts.txt) |
| 第三轮，组合维护/异步辅助与恢复 | 32项中29通过；重新启用两个候选被辅助错误抢焦点关闭，另一用例假定新BU在首屏而超时 | 新增R5-F10/P2，分页夹具另修 | [32项首次](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review/round3-browser.txt)、[原夹具](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review/round3-tests-first.ts.txt) |
| 第三轮独立延迟响应反例 | 延迟真实部门辅助400，用户已打开候选后释放响应，候选关闭且焦点丢失；BU对照实际为200，首版误期待400 | 部门产品反例成立；BU测试预期修正 | [反例及对照失败](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review/focus-before.txt) |
| 第四轮，搜索输入独立预期 | 搜索其他BU/上级，200候选正确但搜索词被旧所选标签替换，两项失败 | 本轮引用投影修复引入R5-F11/P2回归 | [反例](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review/search-before.txt)、[独立探针](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review/search-probe.ts.txt) |
| 第四轮，主动保存错误定位 | 部门非法名称保存400但failureDetails.locations为空，无法聚焦；真实API独立复现 | 新增R5-F12/P2 | [API反例](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review/name-before.txt) |
| 首次完整DB/API | 两包JSON记载482项通过、4项专项跳过，但外层进程退出143 | 不计命令通过，保留并重跑 | [原始JSON日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review/integration-first.txt)、[退出状态](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review/integration-first-exit.json) |
| 第四轮扩大浏览器 | 91项中90通过；部门名称错误定位失败。布局脚本虽通过，但人工检查发现列表可能仍加载中 | 名称问题另修；加载截图不计有效列表验收 | [91项结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review/round4-browser.txt)、[首次400%列表](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review/layout-first/zoom-400-list.png) |
| 第五轮，权限与交互复核 | 32项：引用部分/全部撤权、失败重读及草稿/来源保护、三种维护动作、搜索保持、异步焦点与主动保存定位、原状态/历史授权修复 | 32/32通过，0重试，5.6分钟；无新增P1/P2 | [第五轮](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review/round5-browser.txt) |
| 第六轮，任务完成与证据有效性 | 18项：两对象正常菜单创建/维护/启停/删除、原结果恢复、迟到响应、137项部门选择导出、已加载长名称列表及七类页面布局 | 18/18通过，0重试，5.1分钟；无新增P1/P2 | [第六轮](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review/round6-browser.txt)、[320列表](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review/layout-final/320-list.png)、[400%列表](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review/layout-final/zoom-400-list.png) |
| 最终名称错误真实API | 创建/更正名称及简称错误位置、拒绝查回、撤读权后的诊断裁剪、业务来源不变 | 通过 | [名称复验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review/name-after.txt) |
| 第二次完整DB/API | 最终Go输入483项及4个专项跳过；提交锁用例实际55P03/503但4.972秒超过测试4秒上界 | 482通过、1失败、4跳过；命令退出1 | [完整结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review/integration-second.txt)、[命令状态](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review/integration-second-exit.json) |
| 提交锁单用例复验 | 同一最终代码、真实数据库，count=3，保持原4秒断言；并行采样墙钟与单调时钟 | 3/3通过；采样发现约-2.924秒墙钟回拨，不据此断言前次失败的唯一原因 | [三次复验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review/lock-recheck.txt)、[退出及采样](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review/lock-recheck-exit.json) |
| 最终跨日浏览器 | 最终构建，上海/洛杉矶浏览器时区，服务端业务日变化、旧选择及详情撤权 | 两时区通过 | [跨日复验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review/clock-last.txt) |
| 最终静态/构建/生成合同 | make check、make build、独立生成OpenAPI类型后按字节比较 | 73项Python、20项Node、Go单元/vet、边界、TypeScript、构建及生成一致性通过 | [check](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review/check-last.txt)、[build](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review/build-last.txt)、[生成一致性](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review/schema-check.txt) |
| 范围与资源保护 | 原验证文件摘要、最终运行输入、Docker身份集合、临时profile配置和监听端口 | 3421份旧验证文件不变；1401运行输入匹配；11个临时profile回收 | [范围及收尾](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review/scope-and-cleanup.json) |

第五轮32项与第六轮18项为两批执行，不能写成单轮50项；第四轮90项通过也不能与后两轮相加成一次全量通过。第四轮之后仅补当前引用展示值稳定、部门名称错误字段及布局取证等待；此前通过的BU日志、选择、固定期间等范围保留其当批输入，最终两轮再次覆盖实际受影响流程。第二次完整DB/API与提交锁3次复验分别记录，不能把完整回归记为全绿；本轮未放宽时间阈值、删除测试或改写失败结果。

## 4 失败、修复与复验

R5-F08（P1）：候选为空并不证明当前选择不可读，原页面却持续回退到旧基线标签。现在请求可带selectedId，Go在当前授权快照中独立投影selectedReference，不赋予候选资格；页面用最新投影更新标签。发现标签或链接权收缩时先清除缓存中的关系名称及路径，再重读；保留关系ID、仍可读字段输入和准确来源。读取失败及取消编辑不能回显旧标签；仍可读但不合资格的原关系不能被自动清空。

R5-F09（P2）：共用编辑器从名称变更提取关系补丁后留下空对象，违反严格DTO的缺省/空对象区别。现只在实际存在BU或上级补丁时传changes，显式parentDepartmentId=null继续表示清空上级；没有关系改动时省略。更正、位置新增及重新启用共用此修复。

R5-F10（P2）：自动辅助的错误字段复用了主动保存的焦点副作用，重新启用在原因未选时异步400把焦点移到原因并关闭正在操作的关系候选。修复将错误展示与主动保存的聚焦请求分开；辅助仍显示Go诊断，只有明确保存失败才定位字段。BU的未选原因辅助按其真实200合同对照，不改为部门400行为。

R5-F11（P2）：本轮接通selectedReference后，查询每次产生等值的新选择对象，触发组件回填原标签，覆盖用户搜索词。修复保持相同展示值的选择对象稳定，当前授权投影仍独立更新；实际标签变化与失权清理不被缓存阻挡。

R5-F12（P2）：部门名称Normalize返回错误时遗漏name字段位置，简称已有位置，BU对照也已有。现补齐Go诊断字段；创建、更正及原拒绝查回一致，当前失权后仍由既有保护裁剪位置，不在前端复制名称规则。

R5-F13（P2，验证缺口）：布局脚本只等待网格可见，加载中状态也能生成“通过”截图，且长名称样本可能不在当前页。现先按样本编码实际查询，确认长名称链接已加载并滚入视口，再采集列表；保留原[脚本](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review/layout-before.ts.txt)及42张原截图，原矩阵的通过标记不代表本轮采纳其列表截图。

初次TypeScript检查发现Autocomplete推断包含空值，已显式指定Choice类型；复验通过。首次10项浏览器中的两项准备失败发生在API停用调用，未进入重新启用页面；补齐幂等标识后按新输入复验，不归为产品问题。

焦点修复首版误删错误链接仍使用的fieldID，make check在TypeScript阶段发现；该批浏览器结果不能计为修复通过。[首次检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review/check-first.txt)保留，恢复错误链接的字段定位后重新验证。

该首版浏览器实际9通过、5失败，见[修复首批](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review/focus-fix-first.txt)。后续第五/六轮的完整通过来自修正后的最终输入。第四轮外层进程也返回143，Playwright记录90通过/1失败且临时资源正常回收；本报告仍按失败批次保存，不将末尾清理成功当作测试成功。

## 5 未覆盖范围与限制

R6容量、生产、外部消费者、远程CI与人工签收不在本次范围内。原报告的完整验收不自动计为本轮通过；分批验证分别记录输入与结果。未重跑整个默认浏览器集合，第四轮91项及最终32/18项属于明确选取的范围。

提交锁耗时失败与后续3次通过同时保留。该用例的BU提交锁实现本轮未改变，实际返回仍为503/pending，但本轮不能据此宣称共享执行环境的时钟稳定性或全部锁等待上界已完成单轮验收。采样发现墙钟回拨，仅证明环境存在时钟扰动，不把它当作消除失败记录或跳过检查的理由。

## 6 结论与后续事项

共完成六轮评审与修复。R5-F08/P1及R5-F09—F13/P2已关闭，其中F11是本轮修复引入后再消除的交互回归，F13为验证缺口。第五、六轮在本报告R5范围内均无新增P1/P2，满足本轮退出条件；不是证明产品不存在任何缺陷或替代业务签收。新增验收参数映射到121的R5-13—18。未执行提交、推送或发布。

11个自有临时profile的容器、网络、卷、私密配置和临时监听端口均已回收；Docker恢复原45个容器、3个运行容器、36个网络、59个卷，身份集合无增减。m02-department-b仅只读核验，DB健康、Web/API运行，未向其写入业务/授权样本、迁移或重建；继续由wt-dev-b维护者用于日常观察，下次模型或环境维护时复查。持久环境状态见[收尾输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review/persistent-status.txt)。

## 7 证据保存

新增证据位于[data/116-r5-multiround-review](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review)。首次失败、夹具修正和最终复验分别保存，既有R5及评审修复报告不覆盖。最终[运行输入摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review/runtime-inputs-last.json)覆盖1401份文件；[已记录原件的源码增量](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review/recorded-source-delta.patch)只比较本轮保存的原件，[完整已跟踪差异](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review/working-tree.patch)包含起始未提交R5工作，配合[未跟踪运行源码](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review/untracked-runtime-source.json)标识当前实际输入。22份失败截图保存在[脱敏索引](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review/failure-screenshots.json)；最终布局42张截图与原42张分别保留，不覆盖或混计。文档首次检查因最终日志尚未落盘而链接失败，保留[首次输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review/docs-first.txt)，补齐后复验通过；文档收尾检查另见[docs](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r5-multiround-review/docs-final.txt)。
