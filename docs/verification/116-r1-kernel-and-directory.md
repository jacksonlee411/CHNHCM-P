# CHNHCM-116 R1共用内核与分类目录分批实施验证

登记信息：[registry.json](../governance/registry.json)的verification_records。适用范围：116 R1，BU实际接入共用纯Go内核和117目录；R2—R6另行交付。上游：[116](../plans/116-m02-shared-dated-maintenance-roadmap.md)、[120](../specs/m02/120-dated-maintenance-technical-contract.md)、[121](../specs/m02/121-dated-object-admission-matrix.md)。

## 1 验证对象与受测基线

2026-11-04至2026-11-05，wt-dev-b，起点4f4fc0eed3b4d94f5ef1c3ca8ac80393f4fe7559，初始工作区及stash干净。本轮受测内容为该基线加本工作区R1差异，未创建提交。完整回归开始时的89份运行代码、API、脚本及测试文件见[受测清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1/tested-source.json)和[文件快照](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1/tested-source.tar.gz)；之后仅修正一处浏览器定位，见[补丁](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1/lifecycle-retest.patch)及[最终源码清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1/final-source.json)。文档与验证材料单独维护，不混入运行源码快照。

保护清单见[protected-before.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1/protected-before.json)，覆盖2634份已有验证、引用及资产；[最终比对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1/protection-final.json)确认全部原字节保持，分支、HEAD及stash未变。122、123及R0报告保留各自原始结果。

## 2 环境与前置条件

B0只运行本地Go与文档检查，不启停容器。后续使用Go 1.26.0、Node 24.19.0、PostgreSQL 18.6及WSL Chromium；DB使用Testcontainers，浏览器使用本任务4个独占临时profile。管理员、查询者及受限权限场景沿正常菜单“设置→组织配置→业务单元”进入，覆盖创建、维护、来源及返回上下文。

日常持久profile为m02-current-b，入口[http://127.0.0.1:4382](http://127.0.0.1:4382)。本轮刷新API并完成管理员登录、正常菜单及新目录只读检查，未在该库写入业务事实；[实际结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1/daily-runtime.json)与[页面截图](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1/daily-directory.png)留存。执行及复核身份均为本工作树主代理，不称独立评审或人工业务签收。

## 3 验证过程与结果

| 批次 | 范围与预期 | 实际结果 | 结论 | 证据 |
| --- | --- | --- | --- | --- |
| B0 | 122/123衔接、范围与历史保护、实施前Go基线 | Organization包通过，复用Go缓存；合同复核、73项文档测试及关联检查通过 | 通过 | [基线](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1/baseline.json)、[Go](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1/baseline-go.log) |
| B1 | 强类型资料/候选，BU真实接入，输入与候选不可变 | Organization Go全包重新执行通过，强类型及深复制反例通过；主代理复核通过 | 通过 | [B1](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1/b1-review.json)、[Go](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1/b1-go.log) |
| B2 | 生命周期/位置/继承/数据与结构影响，关系资料扩展及授权映射 | 全Go、关系资料独立预期、120000条阶段取消及导入边界通过；主代理复核通过 | 通过 | [B2](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1/b2-review.json)、[Go](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1/b2-go-reviewed.log) |
| B3 | 117目录、旧来源/终态保护、合同/客户端、受影响DB和菜单 | 当前目录和旧标签隔离；旧accepted/rejected/no_change准确重放；完整DB通过，浏览器按下列分轮证据闭合 | 复验后通过 | [最终复核](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1/final-review.json) |

B1交付强类型资料、补丁及候选，B2交付继承、生命周期及影响计算；均由BU实际调用Organization的dated纯Go包，保留BU公开协议/存储映射，不保留旧规则引擎回退。对象策略负责资料克隆、业务值及对象修订结构比较；关系合成资料验证指针、切片、映射和准确引用版本。对象动作映射迁至application/buauth，权限决定仍由X03持有。B3将117目录同步至011—013、023、API、客户端和测试；Enterprise初始化独立于BU分类目录。

| 检查 | 实际结果与边界 | 证据 |
| --- | --- | --- |
| 静态、单元、导入边界、vet、类型及文档 | 修复后make check通过；Go全包曾以-count=1重跑，最终make check允许缓存 | [最终检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1/final-check.log)、[退出记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1/final-check-exit.json)、[全Go](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1/b2-go-reviewed.log) |
| API生成及前端构建 | 通过；保留构建器已有大于500kB的分块提示 | [生成](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1/api-generate.log)、[构建](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1/b3-build.log) |
| 首次完整DB | tests/integration失败，bootstrap通过；空ADD回归见第4节 | [首次DB](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1/b3-integration-first.log) |
| 第二次完整DB | 两包输出通过，但外层退出143；本次记中断，不作为整轮通过 | [输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1/b3-integration-reviewed.log)、[退出记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1/b3-integration-reviewed-exit.json) |
| 最终完整DB | make integration退出0；integration 564.409s、bootstrap 348.092s | [最终DB](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1/b3-integration-final.log)、[退出记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1/b3-integration-final-exit.json) |
| 首轮分类浏览器及定向复验 | 35通过/1失败；修复空ADD并补旧标签案例后2项定向复验通过 | [首次](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1/b3-browser-first.log)、[定向复验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1/b3-browser-reviewed.log) |
| 完整Chromium 186项 | 185通过，1项旧来源按钮定位超时；该轮失败，耗时34.6分钟 | [整轮](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1/b3-browser-full.log)、[失败证据](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1/browser-full-first-failure/finding.json) |
| 启停文件修复后复验 | 8项全部通过，耗时1.7分钟；只修改一处测试定位，生产源码与整轮相同 | [复验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1/b3-browser-lifecycle.log)、[受测差异](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1/lifecycle-retest-source.json) |

浏览器结论由整轮185项通过和启停文件8项复验共同支持，覆盖原186项预期案例；不记作一次186项全绿。122无日志权的摘要、日志字段权、原件入口及123当前维护行为均在本轮受影响回归中实际执行，截图见[data/116-r1/browser-full](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1/browser-full)。

## 4 失败、修复与复验

B1初审误把空CHANGE新增的no_change视作缺口，并改为拒绝。B3真实菜单TL304305失败后，重新核对013及TL-271，确认显式空差异是复制默认值时的合法请求。已恢复空ADD在完整守卫后no_change；空更正/显式空恢复仍拒绝，并增加日期冲突、停用段及缺失补丁反例。此次回归不隐瞒，B1关闭结论以B3复验为准。B2首次编译发现授权映射迁出后遗留未用import；关系反例暴露仅引用版本变化被消除/未报告结构面；全Go首次发现旧取消探针仍指向旧源文件。均已修复，最终全Go重跑通过，首次输出保留。BU按日定位适配改用共用二分索引，避免整源转换。

B3首轮分类浏览器35通过/1失败（上述空ADD），2条定向复验通过；首轮vet发现Range类型迁出后旧测试无字段名文字值，已改为具名字段并通过make check。旧终态夹具首次漏算主体绑定摘要，第二次错设CREATE重放状态码，修正夹具后真实DB验证通过；生产代码未添加旧目录回退。主代理另发现更正下拉框用当前类型标签替代原标签，现同码保留服务端原快照，真实历史夹具浏览器通过。首轮完整集成：tests/integration 531.307s失败，三处均为同一空ADD回归（首条两步、122摘要原件、位置新增）；bootstrap 329.961s通过。第二轮输出两包通过但外层143，原因未确认；保留中断状态，另行完整重跑并取得明确退出0。

完整浏览器最后发现既有启停测试仍等待122已移除的“查看当次来源”按钮。失败截图与调用栈表明业务操作已完成；定位改为当前“查看当次资料”链接，未放宽断言或超时，随后整个启停文件8项通过。日常入口只读脚本首次遗漏baseURL导致Invalid URL，修正后运行通过，首次日志保留。

最终复核中R1-01原判断撤回，R1-02至R1-08均已修复并复验，无未关闭发现；逐项处理见[final-review.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1/final-review.json)。后续批次反例推翻前一批判断时，明确回开并重验，不将早期通过视作永久结论。

## 5 未覆盖范围与限制

不实施R2共用事务、R3共用前端、R4/R5部门及Hierarchy、R6标准和容量。关系字段合成类型只验证内核扩展点；原39个部门产品案例保持not_run。现有BU事务、页面及容量预算回归不表示R2/R3/R6已经提炼或验收。122、123原报告保持原结果，只有本轮实际重跑可计入本轮验证。未执行远程CI、独立评审、人工业务签收或生产验收。

## 6 结论与后续事项

三项任务及R1各批主代理复核已完成；116整体仍为in_progress，R2—R6未开始。本轮未创建提交，未推送、PR或部署。

4个自有临时浏览器环境的容器、网络和数据卷已回收，日志中识别出的96个集成测试容器均无残留；该96个不代表未输出ID的所有夹具。3个已有持久容器及卷保留。m02-current-b由wt-dev-b继续用于日常开发和用户复核，下次任务开始及模型变化时复查；未自动清空或升级持久库。项目、profile及实际资源结果见[resources.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1/resources.json)。

## 7 证据保存

长期证据位于[data/116-r1](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1)，保存首次失败、各批复验、运行源码快照、主代理评审、页面截图、历史保护及资源核验。日志目录已加入限定Git忽略例外，供后续本地提交完整纳入；不改变其他日志的忽略规则。
