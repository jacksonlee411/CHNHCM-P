# CHNHCM-125 R6容量整改方案评审修订记录

登记信息：[registry.json](../governance/registry.json)的governance.verification_records。适用范围：[125方案](../plans/125-r6-capacity-remediation.md)三项P2评审意见的文档修订及核对；按[验证报告模板](../templates/verification.md)记录。原[编制记录](125-r6-capacity-plan.md)和[R6产品实测报告](116-r6-standards-and-capacity.md)保持各自时点，不改写其结论。

## 1 对象与基线

2026-11-06，CHNHCM-wt-dev-b，分支wt-dev-b，HEAD为587667a1a044d1d99df9baf869d4b0401f07c1e9。工作区已有R6产品/工具及125方案未提交增量；本轮只按用户采纳的三项建议完善方案和导航登记。

编制前核对cwd、分支、HEAD、工作区及登记；stash为空。保存5517份原文件SHA-256，其中3823份位于docs/verification。原方案保存为[plan-before.txt](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/125-r6-capacity-plan-review-fixes/plan-before.txt)，摘要为1dbd3d533e12a36bd51cffdcf978107fffd017da610d04dbf7dcc342fb297dd1；输入与保护范围见[inputs.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/125-r6-capacity-plan-review-fixes/inputs.json)。

## 2 环境及范围

读取本仓源码、现行合同、原方案和评审意见；仅执行文档/差异检查。本次不启动API、数据库、浏览器或容量环境，没有新增运行资源，资源启停/清理不适用。产品代码、SQL、测试驱动和120/013/121现行正文保持；R6容量P1不因方案修订关闭。

## 3 三项意见与修订

| 意见 | 原问题与代码依据 | 方案修订及交接 | 本轮结论 |
| --- | --- | --- | --- |
| 125-R01（P2）页面context未进入容量验收 | 原6.1复用六类负载；[Context](../../internal/contexts/organization/application/department/reads.go)在有创建Grant时加载ListSnapshot，正常页面仍可能在列表改善后报错或无法新建 | 3.2.1单列能力判定必需事实；6.1.1增加管理员/查询者各自20真实会话的冷暖context共600次、10个接口冷轮及独立真实菜单/创建表单；对应T00/T02/T06、R6C-V11 | 方案层已补齐；新增产品验证未运行 |
| 125-R02（P2）HTTP成功不能证明大规模结果正确 | [容量驱动](https://github.com/jacksonlee411/CHNHCM/blob/587667a1a044d1d99df9baf869d4b0401f07c1e9/tests/capacity/r6/workload.go)计时列表主要检查HTTP成功，确切来源也仅额外比ID；原137条CSV不能替代其他页 | 6.1.2明确独立样本/角色/命令预期、每页ID/排序/count及字段检查、详情/保存断言、混合负载读取时点和计时后存储核验；错误样本必须计失败并使组非零退出；对应T00/T01/T06、R6C-V12 | 方案层已补齐；判定器及错误响应反例未实施 |
| 125-R03（P2）加载并发未明确连接所有权 | [外层事务](../../internal/contexts/organization/infra/postgres/transaction/scope.go)先获取连接，[API池](../../internal/bootstrap/runtime.go)上限12；外层持连接后再向同池借连接可能互相等待 | 3.2.2首轮固定一请求一条既有事务连接串行分批；并发只列T04条件分支，先于事务全局准入/预留份额，所有API及加载池计入12连接基线；取消/超时/饱和后恢复对应R6C-V13 | 方案层已补齐；连接压力与恢复验证未运行 |

本轮复核由主代理执行，不标为外部独立评审或人工业务签收。上述“方案层已补齐”只说明问题、设计要求、任务及验收参数已衔接，不能推导运行时问题已修复。

## 4 修订核对与检查

### 4.1 语义核对

已按原评审逐项核对以下内容：

- context能力仍由X03按真实BU范围、完整创建声明及结果读/历史要求判定；管理员/查询者用不同真实主体，600次分组单列，菜单少量结果不冒充p95；冷轮预期准备在清理前完成。
- 原4800/1800/1200负载保留，明确它们不含新增context组；语义错误不得从失败率或耗时分母移除，读写结果不能仅靠HTTP 200通过。
- 独立预期不以被测列表或优化投影函数为唯一真值；混合变动内容不能用结束时最新值误判，无法唯一判断的快照关系交由明确屏障专项，覆盖范围如实记录。
- 首轮串行在原外层事务内执行；条件并发的全局额度包括所有共享池API，独立加载池不增加隐含连接预算；观察器连接另记。人为耗尽/阻塞专项与正常容量错误率分开。
- T00—T06和R6C-V11—13承接实际实施；125保持待评审、产品实施未开始，现行合同与旧报告不被本轮修订替代。

### 4.2 实际检查

| 检查 | 实际结果 | 证据 |
| --- | --- | --- |
| 文档登记、格式、链接及检查器回归 | 通过：73项Python检查，254份Markdown、7989条本地链接；本次结果补录后的最终结构检查另存 | [首次输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/125-r6-capacity-plan-review-fixes/docs-first.txt) |
| git diff --check及新增维护文本空白检查 | 通过：差异检查退出0，8份修改/新增维护文本无行尾空白或缺末尾换行 | [首次输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/125-r6-capacity-plan-review-fixes/diff-first.json) |
| 原代码、合同、报告、数据、HEAD/索引/stash保护 | 通过：5517份原文件中仅6份预定方案/登记/导航文件变化，其余5511份一致，含3822份原验证资料；HEAD、索引及stash保持 | [保护核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/125-r6-capacity-plan-review-fixes/protection.json) |
| context容量、真实菜单、语义判定反例及连接饱和/取消 | 未运行；本轮仅方案修订 | 125第4/6节明确后续任务 |

## 5 未覆盖项及完成边界

本轮没有实施context读取优化、容量判定器或连接准入机制，没有新增性能或业务通过结果。300ms/500ms原目标、20会话和完整背景规模保持，所有新增运行参数仍待实际执行。对三项意见的修订不等于正式采纳全部候选结构调整；125仍由登记表管理审阅状态，116继续持有R6-P01/P02。

## 6 证据与交接

本目录保存修订前方案、输入摘要及实际文档检查输出；最终内容的检查见[docs-final.txt](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/125-r6-capacity-plan-review-fixes/docs-final.txt)和[diff-final.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/125-r6-capacity-plan-review-fixes/diff-final.json)，按实际输出读取。原125编制报告/data与116 R6原件不覆盖。后续T00先建立context真实角色负载和独立语义判定，T01固定参数，T02执行首轮单连接读取方案；采用并发分支前再完成3.2.2要求的具体设计和证明。
