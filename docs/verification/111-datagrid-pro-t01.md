# CHNHCM-111 T01源码依赖与许可接线检查

登记信息：[registry.json](../governance/registry.json)的`governance.verification_records`。适用范围：[111 T01](../plans/111-bu-list-experience-upgrade.md)、[112源码决定](../adr/112-datagrid-pro-source-integration.md)及110体验目标的依赖准备。T02页面替换未开始；本记录不宣称已挂载Pro或完成无水印许可验收。

## 1 验证对象与基线

2026-11-03，wt-dev-b，从文档提交`3aac1c238d01c4a59621793c0a1025409fbef87f`开始。用户授权提交原文档并实施T01；随后要求无水印/无控制台提示，并改为未转译源码接入。用户确认已有Pro许可，但未配置或提供密钥。

原实施前文档30份文件及证据已单独提交，原报告保持。当前差异包含本地源码工作区、依赖/锁、许可初始化入口、开发夹具、Vite源码插件、112/进度及本报告；源码审计见[来源核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-datagrid-pro-t01/source-audit.json)，最终受测输入和检查见[检查汇总](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-datagrid-pro-t01/checks.json)。

## 2 环境与前置

Node24.19.0、pnpm10.24.0、React19.2.4、MUI7.3.7、TypeScript5.9.3、Vite7.3.1，Pro固定官方v8.29.3及112登记提交。118份运行TS/TSX及LICENSE原字节保留，45份上游测试不引入；Pro原始包清单及所选源码子集另存原件。基础DataGrid/x-license/x-internals为官方8.29.3发布依赖，React/MUI基础版本没有升级。

未启动BU profile、API或数据库。浏览器检查只使用任务自有临时静态服务和Chromium，finally中关闭浏览器及服务；不触碰既有持久容器/卷。本轮未配置真实密钥，不保存或生成替代密钥。

## 3 过程与结果

| 检查 | 实际结果及范围 | 证据 |
| --- | --- | --- |
| 原发布包安装准备 | 安装8.29.3；后来按用户要求改为源码入口，保留原记录 | [原安装](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-datagrid-pro-t01/install.log) |
| 首次冻结安装 | 失败：新增UI peer尚未更新锁；同步后冻结离线安装通过 | [首次结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-datagrid-pro-t01/first-frozen-install.json)、[源码冻结安装](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-datagrid-pro-t01/source-frozen-install.log) |
| 发布包分页类型接线 | 首次失败：GridPagination不接收直接文案属性；改用DataGrid.localeText后类型通过 | [首次类型结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-datagrid-pro-t01/typecheck.log)、[修正结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-datagrid-pro-t01/typecheck-fixed.log) |
| 源码类型与构建 | 首次缺源码包自身React类型依赖；补精确dev依赖后typecheck/build通过 | [首次源码类型](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-datagrid-pro-t01/source-typecheck.log)、[复验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-datagrid-pro-t01/source-typecheck-fixed.log)、[构建](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-datagrid-pro-t01/source-build.log) |
| 源码确实进入构建 | 通过：实际Rollup输入含70个本地Pro源码模块，无已发布Pro包模块；release占位按官方同版本替换 | [构建输入](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-datagrid-pro-t01/source-fixture-build-fixed.json) |
| 首次夹具浏览器 | 失败：源码包缺自动JSX编译上下文导致React is not defined；补本地tsconfig，未改上游源码 | [首次错误](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-datagrid-pro-t01/browser-guard-first-failure.json) |
| 无密钥挂载保护 | 通过：桌面/390px均展示未挂载状态，console warning/error为空；无外部请求 | [浏览器结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-datagrid-pro-t01/browser-guard.json)、[截图](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-datagrid-pro-t01/no-key-guard.png) |
| 上游源码/许可保护 | 通过：119项源码及LICENSE摘要一致，未移除水印、修改校验或过滤日志 | [来源审计](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-datagrid-pro-t01/source-audit.json) |
| 有效密钥及真实Pro无水印运行 | 未运行：有效密钥未注入，Pro未挂载；缺配置保护不能替代此验收 | [112待决](../adr/112-datagrid-pro-source-integration.md#6-待决事项) |

构建日志保留上游use client指令及默认分块体积提示，未屏蔽构建警告或改变阈值。用户的无许可控制台提示要求按真实浏览器结果核验，不以构建退出码推导。发布包和源码的原始失败记录均保留，没有把第一轮失败登记为通过。

## 4 修复与复验边界

包清单/锁、GridPagination文案配置、源码包React类型、自动JSX上下文分别修复；没有修改118份上游运行源码。Vite只给上游releaseInfo占位提供官方同版值，未进入或关闭license verifier分支。未使用伪造密钥、CSS隐藏水印、替换console或社区版本回退。

## 5 未覆盖范围

无真实有效密钥，不能核验授权计划/版本期限，也不能确认挂载后的无水印和无许可提示。Pro实际桌面/List view/分页运行、BU正常菜单、TL-323—334、真实DB/API、七组缩放及T04列筛选均未执行。现有产品仍使用Table；夹具是编译/挂载保护证据，不是BU业务验收。

## 6 当前结论及后续

T01的源码依赖、类型、构建及许可初始化入口已接通，正式Pro无水印运行验收待有效密钥。111执行状态为in_progress，T01保持部分完成，T02未开始；待注入有效密钥并运行夹具后才能整体关闭T01。源码接入不会消除官方许可要求。

密钥仅通过仓外环境VITE_MUI_X_LICENSE_KEY注入，官方LicenseInfo.setLicenseKey在任何Pro挂载前执行；含真实密钥的交付构建输出到仓外目录。无密钥的夹具保持未挂载，不让用户误认为Pro运行已通过。本轮临时浏览器和服务已关闭，没有持久数据或容器操作。

## 7 证据保存

本报告data/111-datagrid-pro-t01保存日志、失败、源码审计、缺配置保护截图及最终检查；本地原始输出位于.local/111-t01。原源码包、清单、许可证和逐文件摘要由112关联。后续真实密钥验证另增记录，不覆盖本次未运行边界。
