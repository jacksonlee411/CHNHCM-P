# Community完整CI运行与返回上下文补验

登记信息：[registry.json](../governance/registry.json)的同路径verification_records。关联[111](../plans/111-bu-list-experience-upgrade.md)、[005门禁](../governance/005-branches-and-documents.md)及[运行指南](../guides/017-local-bu-development.md)。

## 1 对象与原结果

原head为8a580da5b23ce72a9555f2786a59b9b1dd1ba60a，[PR7原CI](https://github.com/jacksonlee411/CHNHCM/actions/runs/41633997124)的quality/integration成功；browser最终结论为cancelled，注解明确超过30m0s。保留[原日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-ci-browser-budget/ci-cancelled.txt.gz)及[注解](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-ci-browser-budget/annotations.json)，不改记通过。

套件156项中，取消前有135条明确通过输出、1条删除首例失败输出，另20条没有完成结果。失败用例为“正常菜单删除中段、保留查看和返回上下文、原件与新来源、同日新身份”；取消发生在最终错误汇总之前，原上传条件failure()也没有上传其截图。原CI缺少堆栈，不能仅从标题判定具体断言。后续本地同名失败及导航追踪见第3节；原CI是否恰为相同断言仍保留未知。

## 2 有限修订

仅把browser作业总时限由30改为45分钟；quality/integration预算、156项用例、workers=1、retries=0、原单例时限和全部断言保持。已测本地完整155项耗时29.8分钟，CI在环境准备后尚未执行完，原总预算不足。

失败或取消时上传已生成的遮罩PNG，仍仅保留7天；不上传Cookie、trace、错误文本快照或整个.local。测试结果异常时立即输出用例标题、状态、源文件行号，不输出断言值、请求或会话内容；避免作业后来取消后丢失定位信息。运行代码、业务合同、Go/API/数据库结构不变。

## 3 复现与门禁

删除首例在独立自有临时环境重复3轮均通过，见[输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-ci-browser-budget/delete-repeat-three.txt.gz)。按Community已有73对象场景与删除首例交替重复3轮，原测试5通过、1失败；加入临时导航追踪再次5通过、1失败，稳定定位到删除首例最终搜索输入为空，见[原复现](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-ci-browser-budget/delete-populated-failed.txt.gz)、[导航追踪](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-ci-browser-budget/delete-navigation-trace.txt.gz)、[本地遮罩截图](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-ci-browser-budget/delete-return-masked.png)及[临时追踪源码](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-ci-browser-budget/delete-traced-source.ts.txt)。

追踪显示：成功轮次先收到q=DEL…的200响应，再进入back.q相同的详情；失败轮次URL刚更新q即点击仍可见的旧结果，进入详情时back.q已为空，此后删除/历史/返回均保留该旧值。根因是测试没有等待真实查询完成和链接更新，不是删除过程改掉已带入的返回条件。临时逐次导航日志未进入最终测试。

修订只在进入详情前等待指定q的真实GET成功并核对count=1、当前URL的q和原生链接back.q；然后继续原删除、历史、同日新身份及返回断言。不删旧断言、不加产品阻断或固定sleep。修订后相同分页数据条件交替3轮、6项全部通过，2.1分钟，见[复验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-ci-browser-budget/delete-corrected-six.txt.gz)。

同时修订该用例的异步前置；不以重复或旧结果替代完整验收。新的PR7 head仍须quality/integration/browser全部成功后合并，实际结果以[PR7](https://github.com/jacksonlee411/CHNHCM/pull/7)为准。主代理定位及复核，不冒充独立评审或人工批准。

本轮make check通过，见[输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-ci-browser-budget/check.txt.gz)；提交前再执行登记及差异检查。临时复现profile均自动回收。
