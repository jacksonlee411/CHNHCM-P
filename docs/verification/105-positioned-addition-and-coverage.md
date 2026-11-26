# CHNHCM-105 位置新增插入与更正覆盖验证

登记信息：[registry.json](../governance/registry.json)。适用范围：[105计划](../archive/plans/105-positioned-addition-and-coverage.md)、103的AC030/TL-258—281及104的AC031/补充TL-294—301。原103文档同步和104实施报告保持原证据。

## 1 验证对象与受测基线

固定wt-dev-b，先以提交`80431acdaf55061ce21fa022ab27c17b8f365ebe`保存统一更正，提交者及作者为`lijianjun <lijianjun@cubebox>`。随后按用户明确答复“一并实施，并补充测试”，实施103位置新增并补齐104覆盖。当前105实现及补测在工作区，尚未追加提交或推送。

Go提供同一来源中的所选位置、下一条、可用性、默认名称和日期范围；普通ADD强制显式anchorRecordId，最终保存先核对来源，再验证该位置及完整候选。Web只绑定此上下文：最新一条显示新增记录，非最新一条显示插入记录；初始带值不算修改，原因留空。空档不足仅禁用按钮，不加说明。未改字段不声明写入，空changes按完整守卫返回no_change；原结果保存原锚点，不因后续删除或来源推进改定位。

受测源码、调用和测试冻结见[输入清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/105-positioned-addition-and-coverage/implementation-inputs.json)及同目录压缩包。浏览器首次全套收集后只调整TL020旧默认断言并补两个测试；受测产品代码保持，逐例采用最近执行结果，不把失败命令改写为成功。

## 2 环境与前置条件

Go/Testcontainers使用真实PostgreSQL及HTTP。浏览器使用隔离m02-positioned-addition（4544/API8544/DB54644）及UI环境（4545/API8545/DB54645），样本均合成。原4382仅重启当前API、不迁移清库；4527来源环境保持。凭据、会话及CSRF不进入证据，截图遮罩账号。

角色为有权管理员及按用例裁剪读写/联合权的主体；正常入口为设置→组织配置→业务单元→搜索或新建→时间轴选中记录→新增记录/插入记录/更正记录。成功回到实际记录，查看日期及列表条件独立保持；异常保留可读草稿、准确定位字段，来源冲突须显式重读，不自动换锚点或重新套默认。

## 3 验证过程与结果

[初始覆盖评估](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/105-positioned-addition-and-coverage/initial-coverage-audit.json)发现103只有设计、产品未接线；104已有主流程和历史恢复证据，但缺八组参数。已新增TL-294—301，案例总数为301（297现行、4退役）。[覆盖映射](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/105-positioned-addition-and-coverage/coverage-review.json)列出各层实际执行及未穷举参数。

| 检查 | 实际结果 | 证据 |
| --- | --- | --- |
| 静态、边界、工具测试、Go单元/vet、TS及差异 | make check通过 | [检查输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/105-positioned-addition-and-coverage/check-final.log) |
| 构建 | make build通过；Vite仍提示包体较大 | [构建输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/105-positioned-addition-and-coverage/build-final.log) |
| 完整真实DB/API回归 | make integration两包通过，分别560.705秒及249.850秒 | [集成输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/105-positioned-addition-and-coverage/integration-final.log) |
| Chromium浏览器 | 首轮130项：129通过、1项旧断言失败；修正及补例后相关17项全部通过，132个不同测试最近结果均通过 | [逐例绑定](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/105-positioned-addition-and-coverage/browser-case-results.json) |
| 位置新增视口 | 桌面、短屏、390、320、320×256及原生200%/400%共7组通过；实际保存、越界拒绝、取消、禁用、键盘滚轮及字段样式 | [矩阵](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/105-positioned-addition-and-coverage/matrix/results.json) |
| 原端口正常菜单 | 六个样本原业务结果、统一更正及11次位置入口默认/取消核对通过；4527六个来源头保持 | [页面复核](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/105-positioned-addition-and-coverage/current-samples/ui-result.json) |
| 原数据及历史资产 | 4382全部7个对象来源头和时间线摘要前后一致；1614份既有验证/来源/架构/资产文件与提交基线一致 | [数据复核](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/105-positioned-addition-and-coverage/current-data-after.json)、[保护清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/105-positioned-addition-and-coverage/protected-originals.json) |

103参数包括按生效日排序、过去/当天/未来默认、月末/年末/闰日/9999上界、空档/单日、停用段、独立查看日、空changes、英文清空、字段裁剪及越权、锚点必填/范围、并发新后继及删除、原结果重放/查回、跨上海午夜、后继独立快照与停用继承。104补例包括真实边界落库、全部七种非空修改面组合无需联合权、显式同值仍授权、严格新DTO负例、辅助503直接保存、三个单面权限、分类读写撤权及双向冲突重读。

## 4 失败、修复与复验

首次集成失败来自旧普通ADD调用缺锚点、前条之前的拒绝状态旧断言及撤日期读权后测试辅助仍从受限返回读取日期。已把夹具位置显式写入，区间外按400而占日仍409，受限角色使用前置已知锚点；完整集成最终通过。空changes已是合法无变化输入，不再归入非法对象例表。Go测试运行器总时限因容器用例增加调为20分钟，单命令10秒及SQL预算断言保持。

浏览器准备阶段修正空原因的MUI隐藏值断言、旧历史插入夹具未选择前条，以及冲突注入前未等待真实自动辅助完成的竞态。首轮完整运行唯一失败是TL020仍要求新增日期为空，已改为服务端默认日期并独立断言三月查看日保持，相关17项复跑通过。没有宣称单次完整132项命令全绿。

视口首轮桌面/短屏通过，窄屏脚本仍点击桌面时间轴按钮而超时；脚本改为使用现有紧凑下拉菜单，最终7组全部通过。原端口新增核对脚本先因携带非API查询参数、随后因必填标签精确匹配失败，均修正后完成六样本。首轮输出保留，不计为产品通过。

## 5 未覆盖范围与限制

本批执行针对103产品接线及104八组缺口，不是023全部301编号/所有参数的整版验收。TL-272新增表单逐项撤读权的浏览器参数未单独运行，本批API裁剪/拒绝及既有通用撤权浏览器回归通过；TL-277专门执行并发插入及删除锚点，没有穷举锚点改日、后继改日的所有交错。对应最终来源/位置守卫及既有改日回归通过。

未优化联合入口、未增加历史停用再启用专用交互；未运行Firefox/WebKit、真实消费者或生产验收。当前为主代理实施与自查，不称独立评审。旧T07、103及104通过记录均不替代新增参数。

## 6 结论与后续事项

103位置新增/插入已接线，104补充八组测试的已执行参数通过；原4382可继续验证，六样本业务结果及全部7对象事实保持。任务测试profile停止并保留卷，原4382/4527保持。当前105变更待用户后续提交指令，不推送、PR或合并。

## 7 证据保存

证据位于[data目录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/105-positioned-addition-and-coverage/initial-coverage-audit.json)：输入冻结、覆盖评估、逐例浏览器结果、完整成功及首次失败输出、遮罩视口/样本截图、原数据摘要、历史资产保护清单和自查结论。非敏感原始输出保留于.local/correction-coverage。可复用入口为tests/scenarios/positioned-addition-viewport.mjs及本目录verify-current-samples.mjs，旧104资产不改写。
