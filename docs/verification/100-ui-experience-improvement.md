# CHNHCM-100 页面体验优化与多轮评审验证

登记信息：[registry.json](../governance/registry.json) 的 `governance.verification_records`。适用范围：100 T03—T06，BU 创建/查询页面及共享展示组件。本报告只记录本批实现与实际复验；执行状态、文档采纳和完整 M02 验收分别维护。

上游依据：[100计划](../archive/plans/100-ui-experience-improvement.md)、[012页面合同](../specs/m02/012-m02-functional-and-interaction-design.md)、[011验收锚点](../specs/m02/011-m02-business-requirements.md)及[修复前基线](100-ui-baseline.md)。[T01](100-b-runtime-isolation.md)、[运行修复](100-runtime-review-fixes.md)、[T02截图补验](100-ui-baseline-capture-fix.md)及其原件保持。

## 1 验证对象与受测基线

工作树 `/home/vivobook/projects/CHNHCM-wt-dev-b`，固定分支 `wt-dev-b`，起点 `ab6e73594fc41d4be38b412ddc6f8612588720c7`；开工时工作区、暂存区和 stash 干净。用户授权完成后续建议任务及多轮评审，本次实施 T03—T06；未授权发布，不推送、建 PR、合并或部署。

本次受测对象为上述提交加本批工作区改动：[最终产品输入摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-delivery/final-product-inputs.json)、[源代码快照](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-delivery/final-source.zip)。新增日期显示函数、页面回归及验证探针也纳入输入，不能只用报告所在提交代替受测代码。最终矩阵另保存开始时的产品摘要及实际探针。

主要修改为 `BusinessUnits.tsx`、`shared/navigation.ts`、共享 Lists/RecordControl/ViewingDateControl/HistoryNotice/UnsavedChangesDialog/EffectiveDates、主题与布局、页面标题及相关 Chromium 回归。复用现有 React 状态和 API，不新增状态框架、依赖或第二套领域算法；日期/权限/业务状态和提交结果仍由 Go 判定。

### 1.1 其他工作树及集成边界

开工时 main 为 `57ac388`、a 为 `889bcd9`；a 已有并行文档修改。本任务期间 a 自身提交 `2687743`（M02 时间轴与健壮性文档）；[并行快照](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-delivery/parallel-worktrees.json)记录前后差异，本任务没有覆盖、暂存或同步这些文件。

T05 采用 b 的现行创建/查询合同，与 a 的 `26877432b115d5c78e58f74fb87271658aedcb1b` 核对 `internal`、`api`、`packages/api-client`、`deploy`、`cmd` 均无差异，见[合同比较](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-delivery/shared-contracts.json)。这是现有切片与 UI 的兼容验证，不是将 a 的新文档或未来功能合入 b；共享前端及 012 以后仍须按实际新提交逐段合并。

## 2 环境与前置条件

Linux/WSL2，Node 24.19.0、pnpm 10.24.0、Playwright 1.55.1、Chromium 140.0.7339.186。b 使用 Web 4381/API 8381/DB 54371，仓外独立配置及凭据；每次登录前检查目标归属。真实 DB 集成使用仓内锁定的 PostgreSQL 镜像与 Testcontainers。

管理员、查询者均为本机学习账号；业务数据为 `UI100_*`、`UI100_ACCEPT_*`、`E2E_*` 虚构 BU。正常菜单入口为“设置 → 组织配置 → 业务单元”，先有创建/查询权限、原因目录和服务端当前日期。21 行矩阵样本用于真实服务端分页，浏览器正常表单负责新增一条；异常响应夹具仅验证 UI 恢复，不替代真实写入。

原生缩放在有界面 Chromium、Xvfb `:199` 内执行，100% 内容先校准为 1280×900，外框 1280×1039；原生快捷键分别设置 200% 和 400%，内容分别为 640×450、320×225，DPR 2/4，visualViewport.scale 为 1，结束恢复 100%。不是 CSS zoom、设备像素比模拟或窄视口代替浏览器缩放。临时 profile 位于仓外 0700 目录并在结束删除。

## 3 验证过程与结果

### 3.1 交互合同与实际行为

| 100 项目 | 本轮结果与业务含义 | 证据及边界 |
| --- | --- | --- |
| R01 布局与滚动 | 小视口改为文档滚动；桌面保留独立正文；控制栏随日期帮助伸展，正文不再零高 | 七组矩阵、实际滚轮/键盘及已有返回回归；不只测 DOM 存在 |
| R02 查询一致性 | 草稿与已应用查询分离；筛选/排序/完整日期/查询应用草稿；分页、改大小、失败重试保留草稿；清空只有应用后才清 URL | 新回归核对输入、URL、请求和状态；慢成功/失败、旧响应晚到、刷新和前后退分别验证 |
| R03 焦点及加载 | 分页控件在读取期间保留；不可翻页用 aria-disabled 与事件守卫；错误重试将焦点转到稳定的查询按钮 | 延迟分页时及结束均保留焦点；403 清空草稿、行和数量，无旧数据占位 |
| R04 可读性 | 主题及链接采用深青色；未生效状态采用深蓝色；测量 16 个前景与实际合成背景 | [颜色与日期记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-delivery/display-round1.json)，最低 4.606:1；不是全站无障碍合规结论 |
| R05 历史模式 | 紧凑历史短条含只读说明与返回入口；短屏滚动时贴顶，切换日志仍可见 | 退出保留查看日、重新读取当前资料；无效来源明确失败，不回退最新 |
| R06 日期与日志 | 纯业务日期显示年/月/日；记录时间转北京时间并保留原始精度；创建日志展示获授权的中英文名称、生效日、状态及保存时分类 | 实际 API 日志和洛杉矶浏览器一致；UTC 跨年样本单列显示验证；actor 明确称主体标识，不造姓名或未来动作差异 |
| R07 表单反馈 | 编码/原因帮助、保存禁用说明、字段定位、依赖重载、未保存与待确认分别表达 | 目录 503 原地重试；字段拒绝无误导重载；真实提交成功但丢响应后按原标识重试，最终仍一身份一记录 |
| R08 页面层次 | 新建仅一个主标题，移除无切换意义的单页签；窄屏简化身份/控制区和字段布局 | 保留编码/名称/状态/日期及标签关联；图面对照与正常办理路径核对 |
| R09 链接与返回 | 名称为真实 href 链接；标题区分列表/新建/授权对象；正常返回、后退和新标签按已应用 URL 重建并定位链接 | Ctrl 新标签不改原草稿；目标不在结果时聚焦说明；权限拒绝清标题及受限正文 |
| R10 追踪与交接 | 100/012、导航、017 和三类资产记录本批范围 | 不把组件夹具、来源测试或现有切片结果扩成完整 M02 验收 |

### 3.2 七组页面矩阵

每组均从登录及失败反馈、正常菜单开始，使用滚轮、Tab/Shift+Tab、Enter/Escape；MUI 页签按方向键切换。逐组执行查询/筛选/分页/名称打开与返回、全部表单字段/字段拒绝/取消保护、实际创建丢响应后同次重试、详情/日志/来源历史与退出。原生缩放另核对输入保留和恢复 100%；没有强制点击或脚本改 scrollTop 来证明正常路径。

| 条件 | CSS 内容尺寸 | 结果 |
| --- | --- | --- |
| 桌面 100% | 1440×900 | passed |
| 短屏 100% | 1440×540 | passed |
| 窄屏 100% | 390×844 | passed |
| 320 px 宽 100% | 320×900 | passed |
| 压力视口 100% | 320×256 | passed |
| 原生 200% | 640×450；外框 1280×1039；DPR 2 | passed，已恢复 100% |
| 原生 400% | 320×225；外框 1280×1039；DPR 4 | passed，已恢复 100% |

[完整矩阵数据](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-delivery/matrix-final/results.json)及[执行日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-delivery/matrix-final.txt)保存每组操作、真实保存结果及截图尺寸/遮蔽检查。共 42 张页面图和 2 张倍率证明；另补 3 张首屏/控制栏图，见[图面入口](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-delivery/index.html)与[首屏测量](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-delivery/entry-layout/results.json)。390×844 新建保留全部五个输入字段、一个主标题且无单页签；桌面日期帮助完全在控制栏内。所有受测页面无为完成办理所需的页面横向滚动。

### 3.3 自动检查与当前切片保护

| 检查 | 实际结果及范围 | 证据 |
| --- | --- | --- |
| `make check`、`make build` | passed；Python 63 项、Node 4 项、文档/边界、Go 单测/vet、TS 和生产构建 | [检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-delivery/check-final.txt)、[构建](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-delivery/build-final.txt) |
| `make integration` | passed，真实 Testcontainers DB，45.648s | [日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-delivery/integration.txt) |
| `make smoke` | passed，32/32，桌面 Chromium 及已有组件/窄屏回归 | [日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-delivery/smoke-final2.txt) |
| 七组完整页面矩阵 | passed，7/7；原生 200%/400% 分别通过 | [结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-delivery/matrix-final/results.json) |
| 返回时列表撤权补验 | passed，1/1，焦点到说明，草稿/受限行清空 | [日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-delivery/return-denied.txt) |
| 文字状态及日期显示 | passed，16 个颜色状态、纯日期/UTC 跨年及实际日志 | [结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-delivery/display-round1.json) |
| a/b 联运行 | passed，实际两个工作树的会话、数据、启停和错误目标 | [结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-delivery/parallel-runtime.json) |

32 项完整回归和七组矩阵后，只在测试文件追加返回列表 403 断言；该项单独通过，产品代码没有变化，见[输入差异](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-delivery/post-suite-delta.json)。完整回归原输入另见[摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-delivery/full-suite-inputs.json)及[快照](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-delivery/full-suite-source.zip)。本批未修改 Go、OpenAPI、迁移或 SQL，后续纯 UI 修订不改变 DB 受测范围。

现有浏览器保护涵盖未来首条、来源不存在/重复参数/旧历史链接、响应丢失后查回与原提交重试、只读账号、当前撤权清理、跨会话迟到响应、退出失败及正常取消。多日期节点、隐藏页签错误定位属于展示组件夹具，不代表新增生效记录、更正、删除或停用已实现。

### 3.4 a/b 同时在线

[实际联运行记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-delivery/parallel-runtime.json)五项通过：各自正常菜单及实例匹配；b 的 T02 21 条样本在 a 为 0；b 预检拒绝 a 地址；退出 b 不影响 a 会话；停止/重启 b 时 a 会话与读取继续成功，b 的 21 条原事实保留。

a 原先停止，本次仅为 T05 用 a 自己的脚本启动，未在 a 写 BU 业务数据；完成后核对记录 PID/启动时间/工作目录，再恢复停止并保留数据卷。b 保持运行。独立 browser context 是前提，同一浏览器 profile 的跨端口 Cookie 隔离不在本次结论内。

## 4 多轮评审、问题修复与复验

评审均为本实施代理的分轮源码、合同和实际行为复核，未声称独立人工评审。P1/P2 作为本轮主要问题；测试探针错误与产品缺陷分别记录。

| 轮次 | 发现与处理 | 实际复验及保存 |
| --- | --- | --- |
| 第 1 轮：状态与布局 | 首次进入列表时同 URL 导航覆盖新输入；MUI 默认滚动覆盖短屏规则；返回测试取样早于焦点引发的真实滚动 | own-query/URL 比较保留输入；提高实际滚动容器规则优先级；按用户操作完成后取样。smoke-round1 为 24/27，round2 为 3/7，round3 为 5/8，均保留失败 |
| 第 2 轮：历史与矩阵 | 无效来源不应提前显示已加载历史；隐藏页签时间文本导致选择器歧义；矩阵错用 Tab 寻找非选中页签，并未等待离开弹窗结束 | 历史标识要求授权数据已到达；修正日志定位和页签/弹窗同步。round4 为 7/8；matrix-round1 七组失败，round2 五组通过，round3 补验两组通过；均不替换为最终矩阵 |
| 第 3 轮：可读性与图面 | 选中行链接/信息状态颜色不足；桌面日期帮助溢出；目录失败仍提示选择原因；失败重试丢焦点 | 修正颜色、控制栏自适应、禁用说明和焦点落点；16 状态对比度通过。smoke-final1 为 31/32，剩余为页签切换未等待选中状态；review5 为 3/4，剩余为在加载前后分别测量几何造成错配；均纠正同步后复验 |
| 第 4 轮：完整状态表与最终复核 | 增加读取中的分页焦点、迟到失败草稿、点击查询清空关键词、错误重试焦点及目录恢复检查；核对改动没有增加领域算法或通用框架 | [定向回归](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-delivery/smoke-review6.txt)4/4、完整 32/32、矩阵 7/7 及返回撤权补验通过；核对首屏、400% 历史短条、控制栏与接口边界后，未发现新增 P1/P2 |

所有较早失败日志、可用遮蔽截图、探针及各轮输入保存在[早期轮次归档](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-delivery/earlier-rounds.zip)。首次静态检查因新报告缺导航失败，补导航后通过；记录保留，未改门禁规则或将失败写成通过。

## 5 未覆盖范围与限制

- 仅 BU 创建/查询及已有来源回读。新增生效记录、更正、删除、停用、跨域采用/事件投递仍由 010 承接；未把 a 的新规格当成本分支实现。
- 可读操作者姓名、后续维护完整前后值/目标/关系仍待 013/API；本次只展示同一授权日志记录已有字段。
- Chromium 指定七组条件及若干组件夹具；未运行 Firefox、WebKit、移动端真机、屏幕阅读器或全站 WCAG 审计。
- API 故障/延迟夹具用于 UI 例外保护，真实 DB 验证单列；没有以 mock、健康接口或旧来源测试证明业务完成。
- 本机学习运行；不涉及生产数据、生产备份恢复、发布或远程 CI。构建现有依赖指令及包体大小提示如实保留，不为本轮页面任务引入打包重构。

## 6 结论与后续事项

T03—T06 及 R01—R10 的当前创建/查询 UI 范围已完成，最终一轮评审未发现新增 P1/P2。32 项完整浏览器回归、返回撤权补验、七组矩阵、真实 DB、静态/类型/构建及实际 a/b 联运行均通过；计划执行在此限定范围内关闭。

[输入保护](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-delivery/protection.json)确认 157 份前序证据/来源文件无变化，main 原文件保持，b 分支/HEAD/stash/暂存区保持；a 自身并行提交另记。[运行清理](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-delivery/cleanup.json)确认临时有界面浏览器环境及 profile 已清理、a 恢复停止、b 归属预检通过。改动保留在 b 工作区，未提交或发布。后续 M02 工作按 010 推进，并基于本批共享 UI 与 012 合同适配，不移植未授权业务实现。

## 7 证据保存

持久材料位于 `data/100-ui-delivery/`：最终源代码及输入摘要、各轮失败归档、矩阵与图面、颜色/日期实测、真实 DB/浏览器/构建日志、联运行及工作树保护记录。原 T01/T02/019 数据和来源包不回写。

截图由既有校正采集器生成完整可视表面，登录字段、账号名和展开的操作者标识遮蔽；原生倍率证明在输入凭据前采集。关闭 trace、storageState 和 Playwright 明文页面失败快照。最终[敏感值扫描](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-delivery/safety.json)只报告是否命中，不输出凭据值；仓外临时 profile 清理结果另存。文件摘要见[证据清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-delivery/manifest.json)。
