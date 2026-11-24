# CHNHCM-111 不必要限制与防御性设计清理

登记信息：[registry.json](../governance/registry.json)的`governance.verification_records`。依据用户“进一步全面去掉不必要的限制和防御性设计”，承接[111列表计划](../plans/111-bu-list-experience-upgrade.md)、[112源码入口](../adr/112-datagrid-pro-source-integration.md)及[025全新实施](../adr/025-greenfield-implementation-policy.md)。本次按实际作用排查限制，不将所有业务、权限和恢复检查都视为冗余。

## 1 对象、基线与排查范围

2026-11-03，CHNHCM-wt-dev-b、wt-dev-b，基线`fce806cd1d8d3691166969b92d5300ff6116a4e3`；开始时工作区及stash为空。最终输入见[清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-defensive-design-cleanup/input-manifest.json)。历史实施前文档、T01、T02及缺密钥阻断移除报告原件保持。

检查覆盖自有前端App/业务单元列表/新建/详情/维护/启停期间、共享表单/日期/导航/列表组件、API客户端代际保护、Vite和来源插件、本地运行与测试入口；对HTTP/组织领域入口限制按011/013/025及当前Go代码核对作用。未逐行重审全部领域算法、生成代码或第三方库，不声明所有代码均无冗余。本次不修改Go/API/SQL、鉴权、业务模型或第三方源码。

## 2 角色、操作与实施结果

管理员/查询者从设置 → 组织配置 → 业务单元进入；列设置只调整当前已获权结果的展示，无新业务写入。管理员进入新建后，即使原因目录较慢或失败，也可继续填写获权的独立字段；保存仍须合法原因及服务端校验。取消保留既有未保存确认，未知提交和历史证据保护保持。

| 清理项 | 原额外限制 | 当前处理及位置 |
| --- | --- | --- |
| 纯构建与本地运行耦合 | pnpm build读取profile/端口，运行覆盖值冲突会阻断静态前端构建 | Vite仅在serve/preview读取localConnection，build只使用源码插件 |
| 每次构建的来源核对 | buildStart核对来源归档及119项摘要，成为额外构建门禁 | 核对保留为显式pnpm verify:mui-source；构建只做必要releaseInfo转换 |
| 纯展示列能力 | 列菜单、显隐、拖动及固定被全部关闭 | 开放原生列能力；更多操作的列设置可恢复全部隐藏。列定义稳定，输入或应用查询不重置布局 |
| 旧日期入口 | 已无调用的ViewingDateControl仍作为第二个直接触发查询的共享入口导出 | 删除旧组件/导出/样式，只保留应用/取消的日期面板 |
| 无关目录加载锁 | 新建因原因尚未载入而禁用编码、名称及日期 | 字段只依据当前权限和保存/结果状态；原因未确定/目录失败仍不得保存 |
| 查看日期依赖过大 | 服务端今天尚未返回时整个查看日期入口不可用 | 手工日期面板始终可打开；只将尚无依据的今天按钮禁用，不猜今天 |
| 固定返回等待上限 | 60帧后放弃尚未渲染的目标行 | 按DOM渲染观察定位，无固定帧数上限；找到目标、失效、用户操作或卸载时停止 |
| 手机分页隐去能力 | 原生组件在小屏隐藏每页大小选择 | 保留选择器，允许控制区换行；20/50/100合同保持 |
| 测试环境前置 | 通用控制栏脚本要求VITE_MUI_X_LICENSE_KEY必须为空 | 移除空密钥断言；按实际构建运行，官方许可消息原样记录 |

## 3 保留限制及其依据

| 当前保留项 | 依据与实际作用 |
| --- | --- |
| 授权、读写字段裁剪、会话及请求代际检查 | 011授权及013当前授权；阻止旧会话响应和撤权后正文恢复，不是环境配置门槛 |
| 日期/状态/首条/启停转换、并发版本及引用检查 | Go唯一业务权威及011/013；保护真实业务历史，不在本轮改写领域规则 |
| 原提交绑定、幂等、未知结果查回及未保存确认 | 011/013可靠保存；防止原意图重复或未保存输入丢失。前端保存前的pending引用持久化继续保留 |
| 当前排序白名单、页大小及未实施业务控件关闭 | 当前API仅支持code/name/name_desc及20/50/100，无列过滤、分组/导出合同；不将纯展示开放扩大为新查询/写入能力 |
| 稳定分页、未知数量表达、合法返回地址及焦点取消 | 012查询/返回合同；普通错误不冒充新结果，用户操作优先，地址不造成外跳 |
| 本地服务、进程/容器/持久库归属检查 | AGENTS及003/017；避免向其他实例发送凭据或误停其他任务，实际serve/preview仍核对配置 |
| HTTP会话/Origin/CSRF、JSON合同及执行容量界限 | 当前HTTP/业务入口的授权、格式与已测容量合同，不为简化界面取消服务端准入 |
| 锁文件、类型/文档检查、不可变来源与官方许可 | 本仓自足及来源追溯要求；来源核对改显式，不删除原件，不修改官方水印/校验或过滤日志 |

## 4 实际验证

| 检查 | 实际结果与范围 | 证据 |
| --- | --- | --- |
| 冲突运行覆盖值下生产构建 | 通过；设置与本地运行配置不一致的Web/API值，纯构建仍完成 | [构建](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-defensive-design-cleanup/build-runtime-independent.log) |
| 构建/服务配置分离 | 通过；build无运行身份插件，正常serve保留插件，错误serve端口仍拒绝；未启动服务 | [配置检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-defensive-design-cleanup/config-check.json) |
| 最终类型/生产构建 | 通过；应用及共享组件当前输入，包括稳定列定义和独立字段录入 | [最终构建](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-defensive-design-cleanup/build-final.log) |
| 显式来源核对 | 通过119项；官方源码/许可证原字节保持 | [来源核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-defensive-design-cleanup/source-audit.log) |
| 首轮控制栏/小屏复验 | 失败：手机选择器样式被原生样式优先级覆盖；提高自有布局选择器优先级后复验 | [首次结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-defensive-design-cleanup/browser-first.json) |
| 链接补验首轮 | 失败：在URL变化后立即读取了旧DOM href；改为等待实际链接更新后复验，未修改应用逻辑 | [首次链接结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-defensive-design-cleanup/browser-links-first.json) |
| 扩展控制/表单复验 | 15项；实际Pro配合合成HTTP，包含固定/显隐/恢复/拖动、布局保持、链接仅携带当前A、目录失败继续录入和今天未就绪时手工查询 | [最终结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-defensive-design-cleanup/browser-final.json) |
| 返回适配 | 7项通过；增加1400ms延迟行，超过旧60帧仍能恢复；用户输入/滚轮取消及失效回退继续通过。此后钩子/夹具未变，最终列定义/分页样式变更不作用于该夹具 | [返回结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-defensive-design-cleanup/return.json) |
| 文档及冻结输入 | make docs、diff、受保护文件及摘要核对由汇总记录 | [汇总](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-defensive-design-cleanup/checks.json) |

浏览器角色、目录及列表HTTP响应是合成值，未连接真实鉴权/API/DB；保存不发出业务写请求。SDK缺密钥消息和合成403/503的网络错误原样记录，没有console过滤。来源CLI及配置检查不读取或输出真实密钥/数据库凭据。

## 5 修复、复核与未覆盖

手机选择器首轮失败原因是同等样式优先级下原生规则后写覆盖；复验使用更具体的自有布局选择器，不修改第三方CSS或隐藏许可水印。开放拖动后，将列定义固定在模块级，通过导航上下文更新真实名称链接；避免查询草稿或应用条件改变时原生组件重建列顺序。返回定位改为渲染观察并在结束/用户操作/卸载时断开，不用无限逐帧忙轮询。链接补验先以URL变化作为DOM就绪依据而失败，随后等待实际href更新；复验确认新A可进入链接、未应用D不会混入，没有为测试增加应用等待或阻断。

本轮没有新增业务规则或测试编号；023仅修订TL-333的纯展示/未实施业务功能边界，不将本次合成UI结果计作其整例真实业务通过。未运行真实Go/API/DB业务、全维护矩阵、七组实际缩放、外部系统或生产验收。官方无水印/无许可提示专项及T03完整业务验收继续待后续；旧业务浏览器套件的新菜单/DataGrid选择器适配仍归T03。

## 6 结论及资源收尾

九类已识别的额外限制已清理，当前构建与界面不再因无关技术/目录状态阻断可独立完成的任务。保留项都有现行合同或实际数据/环境归属依据，不作“所有防御代码删除”或“全产品验收通过”的结论。

检查仅使用任务自有静态服务及Chromium，均在finally关闭并写入stopped；配置检查未绑定端口。没有启停BU profile/API/数据库、容器或卷，没有保留新的在线浏览入口。既有历史报告、来源包及真实业务/权限代码保持。

## 7 证据保存

长期证据位于data/111-defensive-design-cleanup，原始本地输出位于.local/111-simplify。保存首次失败、最终UI及钩子结果、代表截图、构建/来源/配置检查、最终输入摘要和汇总；日志仅整理末尾冗余空行，不改变诊断内容。最终输入及执行结果分别记录，文档状态不替代产品验收。
