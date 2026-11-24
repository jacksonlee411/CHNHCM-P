# CHNHCM-101 全场景直接保存验证

登记信息：[registry.json](../governance/registry.json)。适用范围：全部已接BU维护动作取消手动预览及其保存门槛。上游依据：[101决策](../adr/101-direct-save-and-form-assistance.md)、[012交互](../specs/m02/012-m02-functional-and-interaction-design.md)、[013技术合同](../specs/m02/013-m02-technical-design.md)。

## 1 验证对象与受测基线

2026-11-02，固定wt-dev-b，基线95ab6e9c03319dc505799917d93e721a3caee0b9及本批未提交差异。受测代码清单和逐文件摘要见[输入清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/101-direct-save/source-inputs.json)，完整差异文件内容见[代码快照](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/101-direct-save/source-inputs.tar.gz)。清单包含产品、公开合同、生成客户端和当前测试；报告、登记和新证据另随本次工作区交付。

单条新增/更正/启停/删除/改日，以及联合、首条补全、分类纠错均取消预览按钮。普通更正不调用辅助接口；必要默认和状态影响自动读取，输入/来源/会话变化丢弃迟到结果。保存函数不以辅助accepted/canSave作为许可；Go命令重新核验全部守卫。首条保留选择不可默认，新增与重新启用原因来自实际选择。未增加迁移、兼容格式、前端领域算法或持久草稿。

## 2 环境与前置条件

Go1.26.0、Node24.19.0、pnpm10.24.0、PostgreSQL18.6镜像、Chromium140.0.7339.186。独立profile m02-direct-save，页面4532/API8532/DB54632，新建当前模型合成测试库；Testcontainers使用各例独立库。管理员从设置→组织配置→业务单元正常菜单办理，查询者核对只读边界；凭据仓外。

用户原环境m02-current-b仍为页面4382/API8382/DB54373。仅更新源码和重启该profile的API，未初始化或清库；重启前后全部组织表行数/摘要一致，见[原环境验证](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/101-direct-save/current-runtime.json)、[前](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/101-direct-save/persistent-before.json)/[后](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/101-direct-save/persistent-after.json)。浏览器业务样本全部在独立4532环境；验证后已停止该测试profile并保留卷，原4382环境保持可用。

## 3 验证过程与结果

| 场景或检查 | 操作与预期 | 实际结果 | 证据 |
| --- | --- | --- | --- |
| 生成、静态、单元及类型 | 当前OpenAPI生成唯一客户端；文档、边界、脚本、Go单元/vet和TS检查 | 通过；65项仓库脚本单测、4项运行脚本及边界检查包含在make check | [生成](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/101-direct-save/generate.log)、[检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/101-direct-save/check-closeout.log) |
| 正式构建 | 编译Go服务/upgrader及Web | 通过；保留既有前端分包大小提示，不改变门槛 | [构建](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/101-direct-save/build-closeout.log) |
| 真实DB/API | make integration完整两包；授权、版本、原子写入、历史和结果恢复 | 通过；integration包515.044s，bootstrap包259.531s；此后Go、SQL及存储输入未变 | [集成](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/101-direct-save/integration.log) |
| 普通更正及首条/原因专项 | 直接提交非法名称不落库；合法名称及英文清空不依赖辅助；必选项不能绕过 | 定向8项通过；首条辅助503仍可按最终命令核验保存亦有独立实测 | [定向复验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/101-direct-save/fixes-target.log)、[前轮专项](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/101-direct-save/direct-target.log) |
| 最终完整Chromium | 全部正常菜单及现行回归；没有预览按钮，无中间手动核对步骤 | 100/100通过，15.0m；全部使用真实API/DB | [完整回归](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/101-direct-save/smoke-final.log) |
| 视口与实际缩放 | 两套七组矩阵，含原生200%/400%；键盘/滚轮、取消、真实保存 | 两套各7/7通过，命令正常退出；普通更正七组真实保存，联合400%真实保存，其余组验证编辑/滚轮/键盘及取消 | [图面入口](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/101-direct-save/index.html)、[普通矩阵](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/101-direct-save/matrix/results.json)、[联合矩阵](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/101-direct-save/joint-matrix/results.json) |
| 现行专项脚本 | 午夜、重启与故障脚本更新直接保存动作 | 语法检查通过；未重跑这些独立专项 | [脚本语法](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/101-direct-save/scenario-syntax.json) |
| 文档定稿 | make docs及git diff --check；登记、来源及本批长期证据可解析 | 通过 | [文档](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/101-direct-save/docs-final.log)、[差异](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/101-direct-save/diff-check.log) |
| 原环境与历史保护 | 校验原4382环境归属、更新程序及持久事实；来源资产和旧报告不变 | 通过，211项来源身份/保留位置保持 | [运行](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/101-direct-save/current-runtime.log)、[历史](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/101-direct-save/history-protection.json) |

## 4 失败、修复与复验

[复核清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/101-direct-save/review.json)记录主代理发现及处置。主要修正为：同时去除按钮和保存函数门槛；移出只在预览执行的首条选择检查；发送实际原因，避免固定目录值绕过必填；区分版本409和普通业务409；自动辅助按输入/会话丢弃旧结果，并按当前读权裁剪缓存。

前轮处于实现/测试调整阶段，日志分别保留：[首轮](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/101-direct-save/smoke-first.log)、[初步复测](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/101-direct-save/smoke-review.log)、[专项](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/101-direct-save/direct-target.log)、[删除复测](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/101-direct-save/delete-target.log)、[维护复测](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/101-direct-save/maintenance-review.log)。其中首轮和初步复测中断，不记整轮通过；维护复测39通过7失败，包含旧禁用断言、只在首个文件安装的观测夹具、异步响应取样及热更新期间的导航竞争。修复后定向8项通过，最终以冻结输入的完整回归为准。原专项空英文断言误取null，已按实际响应省略字段修正，原失败日志保留。

## 5 未覆盖范围与限制

本批未重跑完整容量、257案例全部主补层、特定T07午夜/物理网络COMMIT故障矩阵、真实跨域消费者、生产恢复、Firefox/WebKit。受控浏览器故障只证明相应交互和恢复，不宣称基础设施故障完整演练。旧通过结果保留原范围，不推导全M02完成。

## 6 结论与后续事项

生成、静态、构建、DB/API、定向8项、冻结完整100项Chromium及两套各七组矩阵全部通过；两轮实施后复核已关闭已发现问题，无遗留发现。此处记录主代理实施后复核，非独立代理评审或人工业务批准。不提交、推送、PR或部署。

## 7 证据保存

长期证据位于[data/101-direct-save](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/101-direct-save)，非敏感原始输出保留于.local/direct-save。输入摘要与代码快照标识脏工作区的实际受测代码；未复制凭据、数据库正文或会话令牌。当前运行验证仅含归属和组织表计数/摘要。既有验证报告、失败原件、离线来源包保持原件，不用新结果覆盖历史。
