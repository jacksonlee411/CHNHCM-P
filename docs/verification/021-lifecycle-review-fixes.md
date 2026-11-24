# CHNHCM-021 T04评审修复与复测

登记信息：[registry.json](../governance/registry.json)。适用范围：T04字段撤权缓存清理、候选继承影响及受影响M02回归。上游依据：[012](../specs/m02/012-m02-functional-and-interaction-design.md)、[013](../specs/m02/013-m02-technical-design.md)、[023](../specs/m02/023-m02-timeline-test-cases.md)。原[实施报告](021-lifecycle-implementation.md)保持原证据。

## 1 验证对象与受测基线

固定工作树CHNHCM-wt-dev-a、分支wt-dev-a；基线4f353f2101a10d136065cb28cdae97abab841c8c。开始时工作区干净、stash为空。本次为基线外修复与新增断言，不变更存储合同或历史数据。受测产品/测试输入摘要、结果计数及证据摘要见[执行清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-lifecycle-review-fixes/execution.json)。

## 2 环境与前置条件

隔离profile为m02-t04-fixes，页面4436、API8436、DB54516；PostgreSQL18.6，当前m02-timeline-3，空库初始化及重复初始化。凭据在仓外受保护文件；原4416及更早4396/4406库、进程和卷保持。真实DB/API回归使用独立Testcontainers，浏览器为单worker桌面Chromium。

管理员从“设置 → 组织配置 → 业务单元 → 列表 → BU详情”创建1月甲/Alpha、3月停用，再进入4月重新启用。填中文及英文后，以隔离DB夹具收窄当前字段权；所有业务动作仍经正常菜单/API。夹具持有policy_guard锁并递增policy_no，finally恢复原授权数组；不开放生产测试接口。

## 3 验证过程与结果

| 检查 | 实际结果与范围 | 证据 |
| --- | --- | --- |
| make generate / make check / make build | 通过；生成文件无额外差异，文档/边界/Go测试及vet/前端类型检查和构建通过 | [生成](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-lifecycle-review-fixes/generate.log)、[检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-lifecycle-review-fixes/check.log)、[构建](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-lifecycle-review-fixes/build.log) |
| 原代码反例 | 失败，预览缺少保存时的INHERITANCE；评审浏览器已复现英文缓存残留 | [原算法反例](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-lifecycle-review-fixes/impact-before.log)、[原页面反例](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-lifecycle-review-fixes/review-browser-before.json) |
| Go独立预期 | 通过；首条/重新启用更正的完整候选与结构影响、同值不产生事实、输入来源不变 | [领域专项](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-lifecycle-review-fixes/domain-focused.log) |
| DB/API准确影响及取消专项 | 两项各连续3次通过；预览/终态/Outbox影响一致、准确继承、后继独立快照、原来源保护；取消实际释放BU锁5.09—6.03ms | [三轮专项](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-lifecycle-review-fixes/impact-cancel-final.log) |
| make integration | 最终通过：31个根测试（27公开DB/API、3组合根事务/时钟、1HTTP）；容量1项显式跳过；首次取消夹具竞态按第4节处理 | [完整集成](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-lifecycle-review-fixes/integration.log) |
| 隔离环境 make smoke | 桌面Chromium 39/39通过（约3.7分钟）；包含创建/查询、来源/恢复、启停及资料维护回归 | [完整浏览器](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-lifecycle-review-fixes/smoke.log) |
| 新增真实撤字段权旅程 | 通过；预览403后父详情/默认/英文输入均清理；旧200预览迟到无效；中文输入保存成功；恢复读权后原英文Alpha保持 | [专项](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-lifecycle-review-fixes/browser-focused.log)、[修复后截图](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-lifecycle-review-fixes/field-revocation-after.png) |
| 执行预算与历史保护 | 120000条记录、5次取消最大120.427ms，小于1秒；192份既有非导航验证/参考文件与基线相同；运行API与本次构建摘要一致 | [领域测量](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-lifecycle-review-fixes/domain-focused.log)、[历史保护](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-lifecycle-review-fixes/historical-protection.json)、[程序核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-lifecycle-review-fixes/runtime-match.json) |

本轮TL映射按实际参数，不关闭253例整体或后续动作：

- TL-048/049：G补更正首条或R时S的INHERITANCE独立预期、同值保持；D/A补R→S的预览/保存/Outbox一致、准确修订、后继R及原来源不变。三时/普通资料及历史参数随既有分层测试回归。
- TL-062：新增B代表参数为1月甲/Alpha→3月S→拟4月R；读取默认后收窄英文读写权，保留中文、清全部展示副本，旧预览不能恢复受限值，保存后完整快照仍有Alpha。主G的部分字段矩阵继续执行；不据此宣称全授权管理已实现。
- TL-155/156：仅补单条重新启用的旧预览迟到和当前撤权参数；联合、真实跨域引用等组合未完成。
- TL-220/221/224/225：既有10秒整体写预算、真实锁释放、原来源保护及长线取消回归；T07全容量不在本次范围。

可人工复核样本：页面4436列表搜索`LIFE_9e987e10`，BU为`53c8670f-4cfc-43ec-98ff-409c546f80b8`。其1月甲/Alpha、3月停用、4月保留中文/Alpha已保存；测试结束恢复原字段权，当前手工打开能看到Alpha。撤权时刻的裁剪以真实测试日志和遮罩截图为证。

最终补充[文档检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-lifecycle-review-fixes/final-docs.log)和暂存差异检查；make check后仅有文档收尾、测试注释及已由完整集成执行的取消夹具调整，无需重复未受影响的构建/浏览器全集。

## 4 失败、修复与复验

P1：403重读原先仅清输入字段，前序默认值和父详情仍显示已撤权英文。现清除默认/预览/原重试包，按最新详情清字段并同步父详情、清变更日志缓存和失效旧日志请求。保留可读中文输入；当前字段权约束默认展示，旧预览和重读错误均检查请求代次。

P2：前序资料更正时候选仍带旧修订标识，原预览缺少后继停用的INHERITANCE，而存储分配修订后才识别。现完整期间确定后按时间顺序识别待生成修订；待生成前序的继承引用留空，Record.Revision保留原基准定位。统一Compare识别结构影响，保存分配后绑定准确修订；不制造候选持久ID，不增加第二套算法。同值更正仍无变化，后续独立快照与旧来源保持。

完整DB回归首次出现`TestTimelineClientCancellationReleasesLocks: cancel changed outbox`。检查发现客户端取消返回后，夹具立即放开Outbox屏障；这不能保证服务端已收到取消，允许提交先完成，实际混入丢响应竞态。现保持屏障直到真实BU行锁释放，再解除屏障核对业务表和resolve=not_applied；未放宽超时或无残留断言。调整后该测试连续三次通过，完整回归另行复跑。原失败单独保存，不算通过。

## 5 未覆盖范围与限制

T01/T02跨后续动作、T05删除、T06改日、T08联合/首条补全/分类纠错、T07全容量仍未完成；未运行Firefox、生产恢复或真实M03/M04消费者。TL-241/242/246/247退役，不计通过。既有报告/来源资产保持，不提升未完整采用资产状态。构建主块体积提示不代表性能验收通过。

## 6 结论与后续事项

两项评审问题已修复并完成分层复测，T04按本批已接通范围通过。另修正取消夹具的同步竞态，最终完整回归通过。T01/T02跨后续动作、T05/T06/T08和T07未关闭；本批结束后停止。

## 7 证据保存

非敏感原日志保存在.local/verification/t04-fixes；归档到data/021-lifecycle-review-fixes，仅去除日志行末空白。评审前失败日志、JSON及截图另存before文件，后续通过不覆盖它们。凭据、cookie及CSRF不归档。保留评审失败和修复通过的分别记录，不覆盖原定稿报告。
