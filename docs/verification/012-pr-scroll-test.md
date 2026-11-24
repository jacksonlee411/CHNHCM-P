# CHNHCM-012 PR浏览器滚动采样修正

登记信息：[registry.json](../governance/registry.json)。依据为[012页面合同](../specs/m02/012-m02-functional-and-interaction-design.md)，仅修正真实滚轮旅程的采样前置。

## 1 验证对象与基线

wt-dev-b，基线5f7ff9982f3911851c370fb6f22647083a56427f，开始时工作区干净。本次产品源码不变，仅tests/e2e/bu.spec.ts及本报告登记有差异；最终测试与差异摘要见[执行清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/012-pr-scroll-test/execution.json)和[补丁](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/012-pr-scroll-test/test.patch)。

## 2 环境与场景

沿用b的m02-more-actions环境，Web4382/API8382/DB54372，真实Go/API/DB和桌面Chromium；凭据在仓外。管理员从正常菜单搜索20条合成资料，以真实滚轮滚动列表和短屏详情日志，查看当次来源后浏览器后退，再返回列表。原断言仍要求恢复精确像素位置、页签、选中行及焦点。

## 3 失败与原因

[PR3首轮](https://github.com/jacksonlee411/CHNHCM/actions/runs/41302404284)的quality/integration成功，browser为43/44；日志返回位置205与294不同，见[首轮摘录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/012-pr-scroll-test/ci-first.txt)。不能计为通过。

本地原流程三次通过。控制按钮部分超出360px视口后，旧测试复现：采样206，按钮范围344.1875—380.6875，实际pointerdown已为227；返回同为227。点击先滚入按钮，页面恢复了真实离开位置，旧预期却取自点击隐式滚动之前，见[受控反例](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/012-pr-scroll-test/reproduce.txt)。远程未记录pointerdown，不能断言其所有像素差都来自同一过程；已确认该前置存在确定的错误取样。

## 4 修正与复验

在记录离开位置之前，让“查看当次来源”完整进入视口并断言完整可见。保留真实滚轮、精确位置比较、历史只读、列表恢复及焦点断言；不增加重试、超时或容差，也不修改产品滚动逻辑。临时坐标和事件诊断已去除。

部分可见条件下修正后连续3次通过，见[反例复验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/012-pr-scroll-test/partial-target-after.txt)；最终测试输入另连续3次通过，见[最终复验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/012-pr-scroll-test/final.txt)。

## 5 范围与后续

本地未重跑完整浏览器、独立DB或构建；产品输入不变，专项验证本次前置修正。合并仍要求PR3最新提交的quality/integration/browser全部成功，以实际远程结果为准；本报告不预记该结果。不扩大完整M02、全部TL或100完成范围。

## 6 证据保存

本地原输出在.local/verification/pr-browser-scroll；非敏感摘录、反例和复验归档到本报告data目录，仅去ANSI控制码与行末空白。旧报告和来源证据保持，凭据、cookie及CSRF不归档。
