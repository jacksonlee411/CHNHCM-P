# CHNHCM-113 Community采用文档多轮复核

登记信息：[registry.json](../governance/registry.json)的governance.verification_records。适用范围：113替代决定及相关文档；依据[113](../adr/113-bu-list-community-adoption.md)、[111](../plans/111-bu-list-experience-upgrade.md)和[014](../governance/014-document-management.md)，从[验证模板](../templates/verification.md)起草。

## 1 验证对象与受测基线

2026-11-03，CHNHCM-wt-dev-b，固定分支wt-dev-b，HEAD为`5748df9a5882865002df563fc0615b19ca312901`；开始时工作区及stash均为空。本轮只修改文档、NOTICE及登记/来源/采用记录，当前运行代码仍为Pro。合同、登记和本报告等最终受测输入的SHA-256见[输入摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/113-community-adoption-docs/final-inputs.json)，检查输出另存以避免自引用。

新增113替代决定、Community依据记录和发布包MIT许可原件；修订002/004/010/011/012/013/017/023/100/110/111/112、文档入口、NOTICE及资产清单。110保持active并登记部分替代，112整体superseded，111保持in_progress；原Pro许可未验项随选型退出撤销，不记通过。既有验证登记、报告及采用批次原样保留。

## 2 环境与前置条件

本仓Python文档工具、MUI官方公开资料及已安装Community8.29.3发布基础包静态读取；CubeBox固定提交的摘要列实现只读观察。未启动profile、容器、Web/API、DB或浏览器，未读取或注入密钥，无资源启停/清理操作。

官网v8滚动页实际显示8.29.5，列拖动页为9.17.0，不能当作8.29.3运行证明。精确基础包静态检查见[包观察](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/113-community-adoption-docs/community-package-inspection.json)，CubeBox观察见[摘要列依据](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/113-community-adoption-docs/cubebox-summary-inspection.json)。ui.tsx不在当前来源归档，仅记录固定提交和摘要；两份实际归档成员另列，不补造提取或验收状态。

## 3 验证过程与结果

| 检查 | 操作及预期 | 实际结果与结论 | 证据 |
| --- | --- | --- | --- |
| R1资料结构 | make docs检查登记、链接、格式、来源及工具回归 | 65项回归及资料检查通过 | [R1日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/113-community-adoption-docs/round-1-docs.log) |
| R1语义复核 | 正向核对新裁决与受影响文件 | 发现3项：100后续目标残留Pro、TL-326名称重复、新依据误列归档成员；本轮失败，后修复 | [R1发现](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/113-community-adoption-docs/round-1-semantic.json) |
| R2辅助检查首轮/重试 | 用辅助断言支持合同与状态复核 | 两次因精确措辞匹配误报，分别2项和1项；修正检查条件，未因此改变产品合同 | [首次](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/113-community-adoption-docs/round-2-semantic-initial.json)、[重试](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/113-community-adoption-docs/round-2-semantic-retry.json) |
| R2修订后复核 | 检查源码方式、替代关系、A/D、链接、分页、列能力、后置项及历史保护；make docs和check-repository.py --check-refs | 34项辅助检查、65项回归、资料及固定分支引用检查通过；无新增文档发现 | [R2复核](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/113-community-adoption-docs/round-2-semantic.json)、[命令结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/113-community-adoption-docs/round-2-checks.json) |
| R3从验收反查 | TL-323—334及源码/无水印专项反查合同、计划、现状和未验项，再复核结构 | 9组追踪、34项辅助检查及make docs通过；无新增主要发现 | [追踪](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/113-community-adoption-docs/round-3-traceability.json)、[R3复核](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/113-community-adoption-docs/round-3-semantic.json)、[命令结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/113-community-adoption-docs/round-3-checks.json) |
| 原件及实现保护 | 对比受测基线的运行/测试输入、旧来源及验证原件，比较旧登记/采用条目 | 2310个既有文件原字节保持；357个运行/测试输入、17个来源文件、1936个验证文件；旧批次登记/采用未改写 | [保护摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/113-community-adoption-docs/protected-inputs.json)、R2/R3复核 |
| 最终差异检查 | git diff --check | R3通过；报告定稿后的重复检查另保存实际结果 | [R3日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/113-community-adoption-docs/round-3-diff.log)、[最终检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/113-community-adoption-docs/final-checks.json) |
| Community安装/类型/构建及真实业务 | 不在本轮文档任务范围 | 未运行，不计T01/T02/T03或TL-323—334产品通过 | 本报告第5节 |

## 4 失败、修复与复验

R1-01只修订100的后续目标说明，原执行和截图/报告保留。R1-02改为“窄屏摘要列”，不重编稳定ID。R1-03区分实际归档成员与外部固定提交静态观察，并记录未归档限制；不为消除发现伪造来源文件或修改旧归档。

R2辅助断言两次误把等价措辞判为失败，保留原结果；修正为核对113登记范围及正文中的其余规则保持、计划中的许可项撤销表述，后34项通过。辅助断言只能支持一致性核对，不能代替语义评审或人工批准。后续R2/R3复核覆盖所有修复，无未闭合文档发现。

## 5 未覆盖范围与限制

本轮不实施Community源码/依赖或页面替换，不运行类型/构建、真实Go/API/DB、浏览器或七组视口验收。当前仍含Pro运行源码、许可入口和夹具，实际页面可能出现官方水印/提示；文档采用不等于运行已切换。无Pro运行依赖、无许可水印/提示、摘要列及非Pro拖动/固定仍待111实施验收。

只读核对了精确8.29.3基础发布包的MIT声明、GridPagination、useGridApiRef、GridApiCommunity及列宽挂钩，未导入或验证该版本未转译Community源码。新源码纳入范围/适配细节、非Pro拖动/固定实现结构及列筛选评估按113第6节处理；既定能力不默默取消。主代理多轮自查，非独立评审或人工产品验收，未抬升任何业务通过数。

## 6 结论与后续事项

修订后的文档准备范围通过R2/R3复核，保留仅撤销Pro、其他已定规则继续的裁决；报告定稿后的资料/差异和输入摘要复核以最终检查文件的实际结果为准。产品任务继续111 T01—T05，不再等待商业密钥。未提交、推送或操作远端。

本任务未启动运行资源，无需停止或回收profile、容器、Web/API或卷；不操作既有环境。不以未访问的历史localhost入口宣称服务在线。

## 7 证据保存

本轮证据在[data/113-community-adoption-docs](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/113-community-adoption-docs)；原始失败、复验、命令日志、保护摘要和最终受测输入分别保存。既有报告、来源包、许可原件及旧验证登记保持，补验不覆盖旧结果。.local仅保存本轮非敏感编辑/检查辅助脚本，不作为唯一长期结论依据。
