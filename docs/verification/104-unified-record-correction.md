# CHNHCM-104 统一更正记录实施验证

登记信息：[registry.json](../governance/registry.json)。适用范围：[104计划](../archive/plans/104-unified-record-correction.md)、[AC031](../specs/m02/011-m02-business-requirements.md#m02-ac-031)及[TL-282—293](../specs/m02/023-m02-timeline-test-cases.md#m02-tl-unified-correction)。

## 1 验证对象与受测基线

固定wt-dev-b，基线7326ce053b91e8158220d75abf1d9c99307e713b及本批工作区差异。该提交先保存用户确认的103规则文档，作者和提交者均为lijianjun <lijianjun@cubebox>。本批实施统一更正，输入见[35项源码摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-unified-record-correction/implementation-inputs.json)及[源码快照](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-unified-record-correction/implementation-inputs.tar.gz)，不把HEAD冒充包含未提交实现。

详情保留一个“更正记录”入口，一次CORRECT修改所选记录名称、原位置内日期及按需展开的业务分类。单条按名称/日期/原新类别分类权限分别核验，不要求联合入口权。名称默认目标自身资料，日期默认原日，分类默认原类型/原因；最终停用名称继承，已有名称草稿须明确处理。日志及查回从准确原新修订读取，一笔来源/事件、完整候选及必要派生修订，不串行保存。

## 2 环境与前置条件

真实DB/API采用Testcontainers PostgreSQL；浏览器主环境为隔离profile m02-unified-correction，Web4542/API8542/DB54642；视口及生命周期补验独立使用m02-unified-correction-ui，Web4543/API8543/DB54643，避免撤权夹具互相影响。两环境均为任务新建合成数据，测试后停止服务并保留卷。

用户环境m02-current-b为Web4382/API8382/DB54373。构建验证后仅restart-api，未运行迁移、初始化、清库或改写业务样本；前端由本工作树服务读取。管理员从“设置→组织配置→业务单元”正常菜单搜索编码，选择时间轴记录进入表单；六个样本和原有第七个BU均保留。开发登录仍为公开预设admin/admin1234567。真实DB凭据、会话、CSRF及浏览器鉴权档案不保存到仓库。

## 3 验证过程与结果

| 检查 | 实际结果与边界 | 证据 |
| --- | --- | --- |
| 静态、单元及类型 | make check通过：65项文档工具测试、登记/引用、边界、运行脚本、Go测试/vet、TypeScript及差异检查 | [最终检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-unified-record-correction/check-final.log) |
| 构建 | make build通过；保留现有资源体积提示，不作为新功能失败 | [构建](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-unified-record-correction/build-final.log) |
| 真实DB/API | make integration最终两包通过，integration 583.763s、bootstrap 283.579s；含组合一来源、整笔拒绝、无联合权的三面权限、原/新类别权限、后续演进后的原结果、取消/超时及原历史回归 | [完整最终输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-unified-record-correction/integration-final-second.log)、[受影响回归](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-unified-record-correction/integration-focused.log) |
| 受影响Chromium | 36项通过；另生命周期6项当前合同复测通过 | [36项](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-unified-record-correction/browser-maintenance-second.log)、[生命周期6项](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-unified-record-correction/browser-lifecycle-final.log) |
| 完整Chromium覆盖及复验 | 全量首轮116项中114通过、2项旧入口断言失败；修订后同主profile仅复跑这2项，均通过。逐项最终结果116通过；原全量命令仍如实记失败，未重写成一轮全绿 | [全量原输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-unified-record-correction/browser-full.log)、[失败项复跑](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-unified-record-correction/browser-last-failed.log)、[逐例执行绑定](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-unified-record-correction/browser-case-results.json) |
| 布局与真实缩放 | 7组全部通过：1440×900、1440×540、390×844、320×900、320×256、原生200%与400%；各组实际组合修改、非法名称400定位、合法保存、日期/原因与新建样式对照、键盘及滚轮；无横向溢出 | [矩阵及截图](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-unified-record-correction/matrix/results.json)、[执行输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-unified-record-correction/viewport-first.log) |
| 原环境业务入口 | 4382六样本逐一搜索，统一表单打开/展开分类/取消无脏输入保护；North/Nroth后继、停用恢复及删除原件、联合日志、首条保留、旧分类、2028闰日前后均通过；4527原六样本头来源保持 | [原环境结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-unified-record-correction/current-samples-final.log)、[初次截图](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-unified-record-correction/current-samples/ui-result.json) |
| 数据与原件 | API更新前后7个BU头来源和时间线SHA256逐项一致；1,541项既有报告/来源/资产摘要保持；原257及103新增24编号保留，再追加12项，293编号/289现行/4退役 | [更新后核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-unified-record-correction/current-data-after-final.json)、[原件摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-unified-record-correction/protected-originals.json)、[一致性](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-unified-record-correction/consistency.json) |

TL-282/285/289/290/291由正常菜单验证初值、只读日期、真实修改、冲突重读及读写撤权；TL-283/286/287/288/292由DB/API与页面组合验证；TL-284采用Go独立手写边界/闰日范围及API/页面代表参数，不声称重新穷举原T07日期全矩阵。TL-293从普通越界拒绝携草稿转既有联合UPDATE，真实保存A/C/B并保持目标身份、后继快照及旧A/B/C来源。

## 4 复核、修复与复验

修复了四项产品风险：组合日期被辅助入口旧白名单拒绝及其状态提示缺失；冲突重读可能把未编辑日期/分类变成旧值回写；撤写权后需要明确移除该面并继续保存其他草稿；快速连续办理可能在新详情就绪前进入旧来源表单。相关DB/API、直接保存、字段定位和浏览器实跑均通过；自查及验证绑定见[复核记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-unified-record-correction/review.json)。

测试迁移保留全部原ID和业务预期。原T06跨节点成功参数移至显式联合，普通日期API同样保序；超时夹具使用可移动末条，不削弱10秒预算及回滚断言。完整DB复跑前纠正取消夹具误用另一组日期，最终全量通过。浏览器将“停用记录无更正入口”改为“统一入口中名称只读、日期/分类按权办理”，TL249改日改走统一入口；2个原失败项复跑通过。产品源码在完整浏览器、视口及后续定点复测间未变，差异仅测试，见[测试修订范围](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-unified-record-correction/test-fix-delta.json)；原失败输出和最初输入摘要仍保留。

## 5 未覆盖范围与限制

103位置新增/插入的24项仍未实施/运行。本轮不优化联合入口或新增“插入历史停用再启用”交互；关联草稿交接只保留现有能力。没有执行全293项TL参数矩阵、容量、Firefox/WebKit、真实消费者/采用、生产恢复或客户验收；本批通过不代表全M02完成。主代理自查不等于独立代理评审或人工业务批准。

## 6 结论与后续事项

统一更正的本轮实施与受影响分层验证完成，无未关闭的本轮发现。当前用户入口为http://127.0.0.1:4382/，六个样本和原第七个BU均可继续核对。103规则文档已提交7326ce0；本轮实现及报告仍在工作区，未提交、推送、PR或合并。原4382服务保留，任务4542/4543服务停止且卷保留。

## 7 证据保存

本报告data/104-unified-record-correction保存非敏感输入、原始失败/最终输出、逐例绑定及掩码截图。原非敏感运行输出位于.local/unified-correction。原环境只读脚本及其合成身份输入已固化，可复核源码和原来源摘要；不得将脚本输出或旧历史报告改写为新的通过记录。后续产品/测试变动应另记证据，不覆盖本报告。
