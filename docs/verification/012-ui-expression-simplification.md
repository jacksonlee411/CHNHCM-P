# CHNHCM-012 页面表达简化验证

登记信息：[registry.json](../governance/registry.json)。基线为b固定分支的e7fa5a3，开始时工作区干净；使用既有m02-more-actions隔离环境（Web4382/API8382/DB54372）。本次为用户明确要求的设计取舍，原报告和来源证据保留。

## 1 变更与原则

AGENTS及[012](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-expression)明确“页面设计应避免过度表达”：默认只展示当前任务所需字段与反馈，不把内部机制、正常系统保证及可自动处理的选择变成常驻说明和额外确认。

取消“明确修改中文/英文名称”勾选、后继快照/分类/来源/采用保证、提交字段说明及重复帮助。直接输入自动记录修改；预览默认只显示目标结果，其他生效记录按需展开。保留必要的日期回显、错误、未保存、权限及结果待确认反馈。013/017/023及010进展同步。

## 2 独立预期

管理员从正常菜单新建及进入新增/更正；没有字段提交勾选或常驻机制说明。只改中文不回传英文，英文清空仍发送null，改回原值仍由Go判定无变化；预览其他时间线默认收起、可正常展开。原业务旅程继续验证真实保存、历史、撤权、并发及恢复。

## 3 检查

静态检查、构建、44项浏览器及受影响七组视口全部通过，产品和用例输入在执行期间保持不变。图面复用原仓内脚本，原文件不修改；凭据在仓外。

| 检查 | 结果 | 证据 |
| --- | --- | --- |
| make check / make build | 通过，17.841s / 10.974s | [检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/012-ui-expression-simplification/check.txt)、[构建](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/012-ui-expression-simplification/build.txt) |
| make smoke | 44/44通过，244.248s | [浏览器](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/012-ui-expression-simplification/smoke.txt) |
| 七组视口 | 全部通过，75.293s，含固定外框的原生200%/400% | [矩阵](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/012-ui-expression-simplification/matrix.json)、[图面包](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/012-ui-expression-simplification/viewports.zip) |

执行输入和保护范围见[执行清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/012-ui-expression-simplification/execution.json)，当前界面见[新增预览](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/012-ui-expression-simplification/after.png)。正常菜单、键盘、滚轮及实际保存均由原采集流程验证；自动字段跟踪、清空、同值和必要保护由完整浏览器旅程验证。

## 4 范围与结论

本次简化已完成。Go/API/存储和共享UI原语未变；独立DB专项、完整M02、全部TL、Firefox及生产交付未重跑。两类返回操作的改造继续暂缓，100保持原完成状态。修改尚未提交、推送或合并。
