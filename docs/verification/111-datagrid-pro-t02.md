# CHNHCM-111 T02基础页面接线与定向检查

登记信息：[registry.json](../governance/registry.json)的`governance.verification_records`。适用范围：[111 T02](../archive/plans/111-bu-list-experience-upgrade.md)、[110体验决定](../adr/110-bu-list-and-datagrid.md)、[112源码入口](../archive/adr/112-datagrid-pro-source-integration.md)及[012/013接线合同](../specs/m02/013-m02-technical-design.md#m02-tech-list-pro)。本轮完成基础页面代码，不把合成响应、适配夹具或缺密钥保护计作真实Pro/BU业务验收。

## 1 验证对象与基线

2026-11-03，CHNHCM-wt-dev-b、wt-dev-b，从`5d390e6fd9258b78c058b4c1680762741eb60bc1`开始，初始工作区及stash为空。用户明确要求“继续推进和完成T02基础改造”，因此在T01源码/类型/构建可用、正式许可运行项未关闭的条件下先完成T02代码，不改记T01结果。最终受测差异及摘要见[输入清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-datagrid-pro-t02/input-manifest.json)，实际检查保存在本报告data目录。

本次变更为Lists.tsx、列表控制组件、BusinessUnitList、导航返回钩子、样式、缺配置保护和定向UI检查程序，以及当前合同/指南/登记/资产进度。没有Go、API、SQL、领域规则、依赖版本或上游Pro源码变更；历史文档准备及T01报告/证据原件保持。

## 2 环境、入口与资源归属

使用本仓Node24.19.0、pnpm10.24.0、TypeScript5.9.3、Vite7.3.1、React19.2.4、MUI7.3.7、Pro8.29.3及Playwright1.55.1的Chromium。未注入有效许可密钥，也未生成或伪造密钥。

控制栏检查使用生产构建及任务自有127.0.0.1随机端口静态服务，从应用“组织配置 → 业务单元”进入；会话、上下文和列表HTTP响应均为明确合成值，不是真实鉴权/API/DB验证。返回夹具使用应用真实返回钩子和延迟行渲染适配，模拟第三页第八行；没有挂载第三方DataGrid，不能证明其实际虚拟化。390px和短屏切换只检查控制适配，不算七组真实缩放验收。

两个程序均在finally关闭浏览器及静态服务，结果文件记录stopped；没有启动、停止或清理BU profile、API、数据库、容器及持久卷。测试不发送真实登录、CSRF或许可凭据。

## 3 实施内容与实际检查

### 3.1 基础代码

| 内容 | 当前接线及保护 |
| --- | --- |
| 控制区 | 新建、标题、更多操作；保留文本输入/查询及A/D。日期/状态按需打开，局部草稿只在应用时并入A，非默认条件显示可操作摘要 |
| DataGrid及窄屏 | Table退出，由同一Pro受控实例承载四列或低于900px的原生List view；名称主行及授权编码/状态/中文日期次行，自动行高及长文换行；短屏/手机使用自然高度 |
| 分页/排序 | GridPagination位于稳定toolbar插槽；server分页/排序，0/1基转换及20/50/100，加载/失败不保留旧数量；名称升降/清除回code，编码只升序，其他列/多列关闭 |
| 链接与返回 | 仅名称是真实Router Link并接入tabIndex；无行导航。当前授权页完成后用Pro滚动/定位API等待名称链接，第三页位置包含页偏移；用户输入/滚轮/焦点优先取消迟到恢复 |
| 权限与未承接功能 | 会话和请求代际保护，当前拒绝清草稿/正文/计数；列菜单/过滤/选择器/固定/重排、多列排序、行写入、导出不开放，无新API或本地业务引擎 |
| 许可边界 | 官方源码、releaseInfo、水印及校验保持；无配置时业务列表不挂载Pro，显示简短不可用反馈，保留查询控制，不回退Table或过滤console |

### 3.2 已运行的检查

| 检查 | 结果与范围 | 证据 |
| --- | --- | --- |
| 首轮类型检查 | 通过；本地应用/共享组件及固定Pro源码类型接线 | [首轮日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-datagrid-pro-t02/typecheck-first.log) |
| 控制栏首轮 | 失败：检查程序的日期定位匹配到同名dialog；限定日期input后继续 | [首轮结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-datagrid-pro-t02/controls-first.json) |
| 控制栏第二/三轮 | 失败：跨断点弹窗关闭焦点丢失；改为DOM提交后恢复；随后“今天”移除摘要又发现焦点失效 | [第二轮](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-datagrid-pro-t02/controls-second.json)、[第三轮](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-datagrid-pro-t02/controls-third.json) |
| 控制栏第四轮 | 失败：摘要在路由提交后移除，过早聚焦旧触发点；等待已应用日期/状态/排序提交再恢复 | [第四轮](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-datagrid-pro-t02/controls-fourth.json) |
| 修复后控制栏 | 通过12项：菜单、草稿/取消/应用、无效日期、今天、断点、重试、迟到拒绝、当前拒绝、只读与无写请求；只含合成HTTP及缺配置保护 | [修复后结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-datagrid-pro-t02/controls-fixed.json) |
| 返回适配 | 通过6项：第三页40+7索引、延迟目标、显式返回/后退、输入/滚轮取消及目标失效焦点；只验证应用钩子 | [返回结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-datagrid-pro-t02/return-fixed.json) |
| 分页焦点接线类型复验 | 失败：8.29.3不接受直接onFocusCapture；移入公开root插槽后复验 | [类型失败](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-datagrid-pro-t02/typecheck-pager-first.log) |
| 最终检查 | 源码摘要、typecheck/build、make docs及diff按最终冻结输入执行，具体结果由汇总持有 | [检查汇总](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-datagrid-pro-t02/checks.json) |

控制栏检查保留预期503/403合成请求导致的浏览器网络错误日志；没有过滤或替换console。SDK许可提示、额外warning、pageerror均未出现，因为Pro未挂载。这不等于真实Pro已达成无水印/无提示。

## 4 修复及复核

首轮只修复检查程序的日期input定位；第二至四轮修复真实应用的焦点时序，分别处理弹窗卸载与URL提交后原摘要消失的情形。最终恢复仍优先实际可用触发点，失效才回更多操作，不在body结束。首次失败结果保留。分页焦点补强时发现根组件不接收直接DOM焦点属性，改用公开root插槽；类型和构建失败日志另存，不计通过。

静态复核还确认8.29.3的scrollToIndexes会减去page乘pageSize，即使server分页也要传全局行索引，已增加当前实际页偏移并由第三页独立预期47验证。分页控件因加载/边界变成不可用时，保留可聚焦状态反馈作为安全替代；具体原生控件的键盘行为仍须真实Pro补验。

## 5 未覆盖及限制

有效密钥尚未配置，真实Pro没有挂载。原生分页范围/边界、实际表头排序、List view行高/长文本/字段权限、真实虚拟行及名称修饰键、原生分页焦点、正常鉴权/真实DB/API、七组缩放和新建/维护/历史回归均未在本轮验收。现有浏览器套件中依赖常驻查看日期、排序下拉或Table行定位的选择器也需在T03按新菜单及DataGrid实际DOM适配并执行，本轮不把旧套件列为通过。

TL-323—334的整例业务结果保持未运行；控制栏和钩子夹具只提供其中应用层风险的局部证据，不改变旧T07/019/100/109的历史结果。T04列筛选评估及T05整体交付没有完成。无密钥时正常列表显示不可用，而非可读取的Pro列表；不能将此状态当作业务页面正式就绪。

## 6 结论与后续

T02基础代码接线完成，类型/构建与定向UI检查按本报告实际结果记录；这项代码完成与T01正式许可验收、T03真实Pro/业务验收分开。111继续in_progress，不声明整个列表项目、T01或M02完成。有效密钥须从仓外注入，后续实际运行及业务检查另增报告，不覆写本次范围。

任务自有静态服务和浏览器已结束，未触碰既有profile或持久库；没有保留需要用户访问的运行入口。本次脚本仅在本机回环临时端口运行，不宣称默认端口已有可浏览新版。

## 7 证据保存

长期证据位于data/111-datagrid-pro-t02，包含各轮失败/通过JSON、代表截图、最终日志、冻结输入和检查汇总。本地完整构建及夹具产物位于.local/111-t02；日志只整理末尾冗余空行以满足diff检查，不改变诊断内容。固定上游源码逐字节核验仍由112及packages/mui-x-data-grid-pro/upstream.json持有。
