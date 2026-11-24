# CHNHCM-020 T06全类别改日实施验证

登记信息：[registry.json](../governance/registry.json)。范围：T06单条首条、普通、停用和重新启用记录改日。依据[010](../plans/010-m02-scope-and-plan.md)、[011](../specs/m02/011-m02-business-requirements.md)、[012](../specs/m02/012-m02-functional-and-interaction-design.md)、[013](../specs/m02/013-m02-technical-design.md)、[023](../specs/m02/023-m02-timeline-test-cases.md)及024/025。T08联合、分类纠错、首条补全和T07全容量仍后置。

## 1 对象与基线

2026-11-01启动、2026-11-02完成本轮技术验证，固定工作树CHNHCM-wt-dev-a、分支wt-dev-a，基线3e724797dc788cdaf4c97e4624949991b020f60a。开工工作区干净、stash为空；基线外为本批改日及验证。本报告冻结实际技术验证；单独评审代理尚未授权，独立评审未运行，不能据此将T06整阶段标为完成。受测差异见[源码摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-redate-implementation/tested-inputs.json)及source.patch.gz。

## 2 环境及正常入口

Go1.26.0、Node24.19.0、PostgreSQL18.6；DB/API使用Testcontainers。浏览器使用独立profile m02-t06-current，Web4486/API8486/DB54566和m02-timeline-5空库，00001/00002当前基线。旧4476/4466及更早环境保留原程序和库。凭据只在仓外受保护目录。

管理员从“设置 → 组织配置 → 业务单元 → 列表 → BU详情 → 生效日期轴选中记录 → 更多操作 → 修改生效日期”办理。仅编辑新日期，原资料及分类只读；先预览，再确认改日。保留查看日和返回列表上下文，定位同一记录的新日期。

## 3 当前实现与已观测结果

| 场景 | 独立预期及观测 | 技术验证证据 |
| --- | --- | --- |
| 四类九格 | D固定2026-06-15，首条/CHANGE/S/R各9格；独立核对快照、身份分类、前段期间、今日状态及准确影响，只有T→T无变化 | [Go](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-redate-implementation/domain-delivery.log) |
| 首条及重排 | 固定首条前后移、唯一首条到未来NO_FACT、越过后继/同日/企业外拒绝；F9/F2前后重排保留独立快照 | 同上；DB/API及正常菜单代表 |
| 生命周期及结构 | S/R完整后继合法性、无自动级联；同值前序改日仍产生期间与准确继承新修订，不能当无变化 | 同上及[DB/API](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-redate-implementation/db-expanded.log) |
| 权限及历史 | 四种类别专用权16格、日期字段权、同值仍需权限版本；原来源/修订/原结果不改，撤写权可读，撤日期权裁剪、撤历史权拒绝 | 同上 |
| 正常菜单 | 普通跨后继、首条到未来、S/R改日、同值、查询者、短屏未保存保护、丢响应重试/刷新、迟到预览、来源冲突及撤日期缓存清理 | [专项8/8](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-redate-implementation/browser-second.log) |

## 4 首次失败、修复及复审

首轮浏览器6条通过，一条因夹具在延迟route.fulfill结束前移除路由而报Route is already handled。改为等待响应交付完成，再移除拦截；补撤日期读权用例后8条通过，保留[首次结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-redate-implementation/browser-first.log)。这不是后端业务拒绝。

实现复查补齐预览的原件、目标前序状态及首条后移新增NO_FACT期间，由Go返回；前端不计算第二套时间线。提交继续使用已修复的外层确认边界，未确认时保留原身份。独立阶段的执行者复核发现：目标原分类代码或名称快照缺失时仍可能接受改日。用-overlay仅移除新保护的隔离反例证实原路径200（预期503），见[修复前反例](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-redate-implementation/db-review-before.log)。补LoadTimeline证据完整性检查，停止维护且不从当前目录补造；四类目标×三个缺失字段共12组，预览/提交均503、无伪终态，受控损坏恢复后业务表与原快照一致，见[复验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-redate-implementation/db-review-final.log)。损坏只在可丢弃owner夹具中注入，不计正常业务历史维护。

[执行者复核记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-redate-implementation/implementation-review.json)未发现其他遗留技术问题。单独评审代理的明确授权尚未收到，独立评审未运行；执行者自查不能冒充独立代理或人工批准。

## 5 未覆盖范围

不启动T08联合、首条补全、分类纠错，T01/T02对应部分继续保留；不运行T07全容量、真实外部消费者或生产恢复。COMMIT网络未知的完整矩阵不由本批代表故障推定通过。旧版本兼容和已退役TL不重做。

## 6 检查与结论

| 检查 | 实际结果及证据 |
| --- | --- |
| make generate | 通过，[当前生成](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-redate-implementation/generate-final.log) |
| make check / make build | 通过；分类缺失保护后重跑，[检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-redate-implementation/check-final.log)、[构建](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-redate-implementation/build-final.log)。前端既有主块大小提示保持 |
| make integration | 两个真实DB包全部通过，分类缺失保护后再次完整通过，[最终集成](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-redate-implementation/integration-final.log)。后来仅补强旧修订及边界断言，分别定向通过，不改业务实现 |
| 独立环境make smoke | 桌面Chromium完整69/69通过，[全套](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-redate-implementation/smoke.log)。此结果早于最后的分类缺失保护；正常快照均有完整分类，该保护仅增加缺证据拒绝，UI未改，复用正常路径证据。最终新程序的T06八条另[8/8通过](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-redate-implementation/browser-review-final.log)，新断言的首条NO_FACT预览[补验通过](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-redate-implementation/browser-first-gap-final.log) |
| 七组视口及原生缩放 | 通过1440×900、1440×540、390×844、320×900、320×256，以及固定窗口1280×1039的真实200%/400%缩放。CSS分别640×450/DPR2与320×225/DPR4，visualScale=1；真实滚轮、键盘取消与400%实际保存通过，[矩阵](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-redate-implementation/matrix/results.json)。桌面和400%截图另人工式图面检查；执行者检查不代表人工批准 |
| 执行及清理预算 | 120000条当前线、改日取消5次，最大本地停止57.065655ms；20条各0.6秒SQL累计超时10.00284659秒；真实取消后BU锁约5.08025ms释放，无残留、原来源保持。见Go输出及[DB测量](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-redate-implementation/db-final-focused.log)，[环境](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-redate-implementation/environment.json)。不是T07容量验收 |
| 重启及历史保护 | 原/新来源两种查看日、原改日目标修订/日期、日志和终态摘要在API重启前后一致，[摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-redate-implementation/restart-readback.json)。782份原报告/证据/来源字节不变；211项资产保留原状态，仅8项追加本批partial采用，[历史](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-redate-implementation/history-protection.json)、[资产](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-redate-implementation/asset-protection.json) |
| 独立代理评审 | 尚未授权，未运行；T06保持待评审，不启动T08或T07 |

[TL及实际参数](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-redate-implementation/tl-mapping.json)逐项区分单条闭合与公共案例未完成部分。当前技术验收通过，整阶段完成结论等待独立评审。

## 7 证据保存

非敏感原始输出保留.local/verification/t06；必要复核材料已保存本报告data目录。源码摘要覆盖最终实现和补强后的测试，source.patch.gz包含当前差异；原失败和修复复验分别保存。旧定稿报告及来源资产保持原结果。日志入库副本仅整理行尾空白，git暂存前核验所有日志和摘要均被纳入。封存后的文档与差异检查见[最终文档检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-redate-implementation/docs-final.log)。

可复核普通改日样本：在4486菜单列表搜索`REDATE_e2c119a4`，BU为`b619194a-2e52-4766-a210-c12f749bc3ca`，同一目标`78e2544d-c0ef-43e6-803f-ed344ebac2e1`从6月1日移到11月1日，原来源`0db71475-cf98-4ce5-b9ab-05f13833ec61`仍可读旧日。完整新来源和修订定位见重启摘要。
