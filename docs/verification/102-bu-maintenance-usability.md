# CHNHCM-102 BU维护日志与专用表单验证

登记信息：[registry.json](../governance/registry.json)。适用范围：用户采纳的四项维护体验修订。上游依据：[102计划](../plans/102-bu-maintenance-usability.md)、[012现行交互](../specs/m02/012-m02-functional-and-interaction-design.md)、[101直接保存](../adr/101-direct-save-and-form-assistance.md)。

## 1 验证对象与受测基线

2026-11-02，CHNHCM固定wt-dev-b，基线83b4a903ee836a0f53ca0dd406560d7c3289e3a5及本批未提交差异。[输入清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/102-bu-maintenance-usability/source-inputs.json)及[代码快照](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/102-bu-maintenance-usability/source-inputs.tar.gz)标识实际产品/测试输入：前四项为完整浏览器回归的冻结差异，最后一项为随后完善的视口脚本。完整回归及最终视口结束后摘要复核一致。仅修改前端表达、编辑保护和验证资产；Go、SQL、迁移、OpenAPI及生成客户端未变。

## 2 环境与前置条件

Go1.26.0、Node24.19.0、pnpm10.24.0、Chromium140.0.7339.186。隔离profile m02-direct-save，页面4532/API8532/DB54632；沿用当前模型开发测试库，新增合成BU，不删除原对象。管理员从设置→组织配置→业务单元办理，查询者及来源历史按当前授权只读。测试后停止该profile并保留卷，见[停止记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/102-bu-maintenance-usability/test-runtime-down.log)。

用户环境m02-current-b为页面4382/API8382/DB54373，源样本环境4527。本批只读复核这两处样本；4382沿用本工作树Web源码，未重启API或修改业务数据，最终[归属及就绪检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/102-bu-maintenance-usability/current-runtime.log)通过。

## 3 验证过程与结果

| 场景或检查 | 前置与操作 | 实际结果 | 结论与证据 |
| --- | --- | --- | --- |
| 联合摘要与定位 | 新增停用/恢复，查看联合日志及逐项来源 | 摘要显示真实分类与数量；默认及各项历史定位准确，旧来源保持 | 通过，[专项](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/102-bu-maintenance-usability/target-final.log) |
| 新增日志 | 新增名称及英文，再查看日志 | 直接呈现登记内容，不将无原记录误报为读取失败 | 通过，见同一专项 |
| 未修改退出 | 两种专用入口，桌面/390宽，等待自动辅助并展开关联操作 | 取消及返回列表不确认；实际修改仍保护，恢复原名称可直接取消 | 通过，见同一专项 |
| 关联分类纠错 | 独立分类更正非法后加入后继纠错项 | 非法409不落库；关联项整笔保存，旧来源保持，主项固定首位 | 通过，见同一专项 |
| 联合删除与更正 | 同笔更正中段、删除末段，逐项查看 | 删除原件及更正资料可读，目标与来源准确 | 通过，见同一专项；专项共6/6 |
| 完整Chromium | 全部现行正常菜单及异常、权限、恢复、历史回归，真实API/DB | 106/106，16.3分钟；包含六项新增专项及TL-050/TL-249完整流程 | 通过，[完整日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/102-bu-maintenance-usability/smoke-final.log) |
| 视口与实际缩放 | 桌面1440×900、短屏1440×540、390×844、320×900、320×256及原生200%/400% | 7/7组、两种表单共14项；滚轮读取末项、键盘到达关联/保存/取消；未修改退出和真实修改保护；400%两种表单各真实保存一次 | 通过，[结果及图面](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/102-bu-maintenance-usability/matrix-verified/results.json) |
| 静态、单元及类型 | make check、make build | 文档、边界、运行脚本、Go测试/vet、TS与构建通过；保留既有分包大小提示 | 通过，[检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/102-bu-maintenance-usability/check-complete.log)、[构建](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/102-bu-maintenance-usability/build.log) |
| 原环境及六样本 | 4382全部对象来源/时间线摘要前后比对；六编码正常菜单UI；源4527核对 | 7个原对象来源与时间线不变；六样本UI、删除原件、分类旧来源及闰日边界通过；4527六个来源保持 | 通过，[前](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/102-bu-maintenance-usability/current-data-before.json)/[后](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/102-bu-maintenance-usability/current-data-after.json)、[UI](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/102-bu-maintenance-usability/current-samples/ui-result.json) |
| 定稿与差异 | make docs、git diff --check；新文本及输入摘要复核 | 定稿文档检查与差异检查通过 | 通过，[定稿检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/102-bu-maintenance-usability/docs-final.log) |

## 4 失败、修复与复验

首轮六项专项为4通过、2失败：两个用例已完成桌面/窄屏退出核对，返回列表后未查询新编码，误等不在首屏的新样本。补入正常搜索步骤后六项全部通过。[首轮](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/102-bu-maintenance-usability/target-first.log)与最终日志分别保留，不把首轮记为通过。

视口脚本前四轮未完成：初轮和诊断轮在日期切换后键盘开菜单失败；其后两轮桌面/短屏通过，窄屏因未等待详情加载并未使用紧凑日期菜单而失败。补齐实际详情等待、定位控件上的键盘触发与窄屏菜单分支后，最终七组全部通过。失败与中间输出保留于[matrix](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/102-bu-maintenance-usability/matrix/results.json)、[诊断](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/102-bu-maintenance-usability/matrix-detail/results.json)、[中间轮](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/102-bu-maintenance-usability/matrix-final/results.json)、[加载诊断轮](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/102-bu-maintenance-usability/matrix-complete/results.json)；相关脚本版本和遮罩失败截图同目录保留。修正只涉及新增验证脚本，完整106项回归后的产品代码保持冻结。

文档编写期间的检查因新编号登记早于文件及审阅元数据齐备而失败，[原输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/102-bu-maintenance-usability/check-document-incomplete.log)保留。文档齐备后make check通过；定稿检查曾因两项待落盘的证据链接失败，见[定稿原输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/102-bu-maintenance-usability/docs-before-evidence.log)，证据齐备后另运行make docs。主代理[复核记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/102-bu-maintenance-usability/review.json)逐项对应四项修复，无遗留发现。

## 5 未覆盖范围与限制

本批不重复运行独立Go Testcontainers集成、全量容量、午夜/物理COMMIT故障矩阵、真实消费者、生产恢复或Firefox/WebKit。真实API/DB的浏览器验证覆盖本次页面行为；七组中仅400%分别真实保存，其余组验证编辑、取消与操作可达，不扩大为全M02或整体无障碍合规验收。主代理实施后复核，非独立代理评审或人工业务批准。

## 6 结论与后续事项

四项修订完成，六项专项、完整106项Chromium、七组视口/真实缩放、静态与构建、原样本保护及定稿检查通过。用户可继续在原4382验证六个既有编码。长期页面规则归012，102计划按此增量关闭；未提交、推送、PR、合并或部署。

## 7 证据保存

长期证据位于[data/102-bu-maintenance-usability](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/102-bu-maintenance-usability)，非敏感原始输出位于.local/maintenance-usability。只保存合成对象标识、摘要、遮罩截图、代码及检查输出，不复制登录凭据、cookie或CSRF。新文件与证据的[摘要清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/102-bu-maintenance-usability/artifact-manifest.json)用于复核；既有验证报告、4527来源与4382样本保持。
