# CHNHCM-100 页面问题基线与 012 修订验证

> 2026-11-01 后续勘误：评审确认下列首次 50 张图中，200%/400% 的 12 张页面截图仅覆盖左上区域，不能作为完整视口图面证据。原始图片、数据及原结论保留；倍率和 DOM 测量未因此失效。补采及修复结果见[截图补验](100-ui-baseline-capture-fix.md)，当前图面入口改用[校正后的索引](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-capture-fix/index.html)。原采集脚本作为历史原件保留，后续补采使用新报告的入口。

登记信息：[registry.json](../governance/registry.json) 的 `governance.verification_records`。适用范围：100 T02 的修复前问题基线、012 修订及 M02 交接。**T02 的材料已形成，R01—R09 产品问题尚未修复，T03/T04 未实施。**

上游依据：[100 计划](../archive/plans/100-ui-experience-improvement.md)、[012 页面合同](../specs/m02/012-m02-functional-and-interaction-design.md)、[011 查询](../specs/m02/011-m02-business-requirements.md#m02-ac-001)、[创建](../specs/m02/011-m02-business-requirements.md#m02-ac-002)、[返回](../specs/m02/011-m02-business-requirements.md#m02-ac-007)及[历史](../specs/m02/011-m02-business-requirements.md#m02-ac-008)。[打开修复前图面](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-baseline/index.html)。

## 1 验证对象与受测基线

2026-11-01，在固定工作树 `CHNHCM-wt-dev-b`、分支 `wt-dev-b`，从干净提交 `718ef7afcc629ae0ddd2e26a6b63e13b37644d05` 开始；stash 为空。先登记本报告，再建立正文和证据。当前差异只有 100、012、登记/导航及本报告/附属证据，产品、API、Go、迁移和测试输入保持。

[采集前快照](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-baseline/inputs-before.json)记录 302 份产品、脚本、测试、原验证/来源及导航文件的摘要，以及其他固定工作树的分支、HEAD、索引摘要和状态。收尾核对见[保护结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-baseline/protection.json)。本报告没有覆盖旧 018/019/T01 报告或修改来源资产的采用状态。

## 2 环境与前置条件

本机 WSL2 Linux 6.6.114.1、Node 24.19.0、Python 3.12.3、Playwright 1.55.1 / Chromium 140.0.7339.186。b 独立 Web/API/DB 为 4381/8381/54371；采集前及收尾执行 `python3 -B scripts/local-bu.py verify` 核对本工作树归属。真实 Go API 和本地 PostgreSQL 承接请求，没有使用相邻仓库服务。

本地管理员正常入口：“登录 → 设置 → 组织配置 → 业务单元 → 新建/查询 → 详情 → 变更日志 → 查看当次来源 → 返回当前资料”。完成一次正常表单的无效编码 400 拒绝和修正后 201 保存，真实查询/详情/日志读取；另经同一已认证 API 补 20 条虚构 BU 形成 21 条分页数据。本轮完整采集前缀为 `UI100T02_0DE876`，第一次中断采集的 `UI100T02_450A39` 数据仍保留，不清库。矩阵中的重复表单只输入后放弃，没有逐组新增业务写入。

100% 五组视口使用独立无界面 Chromium context；真实缩放使用独立 Xvfb 显示和有界面 Chromium profile。profile 位于仓外临时目录，权限 0700，关闭后移除；凭据仅从 0600 仓外文件读入内存。未保存 cookie、CSRF、storageState、trace 或 HAR；图面遮蔽登录输入和账号身份。原生倍率截图在未输入凭据的登录页面截取。

## 3 验证过程与结果

### 3.1 R01—R09 问题基线

以下“失败”表示当前产品不满足相应体验合同，不表示 T02 未完成取证。结果详见[实测数据](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-baseline/results.json)；接口字段名称保留，主体标识值未写入证据。

| 项目 | 前置与实际操作 | 预期 | 实际结果及影响 | 结论与证据 |
| --- | --- | --- | --- | --- |
| R01 · P1 布局 | 正常菜单进入列表；七组条件用滚轮读取并尝试打开首条；空登录页滚轮/Tab 定位 | 正文、登录按钮及操作可达 | 320×256 和真实 400% 列表顶部控制区延至约 y=311，正文高度为 0，名称点击超时；两种条件登录按钮滚轮后仍完全不可见，Tab 才可带入视口 | 失败；[压力列表](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-baseline/stress-list.png)、[400% 列表](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-baseline/zoom400-list.png)，JSON `matrix` |
| R02 · P1 查询一致性 | 应用虚构前缀，输入未应用的“研发”，切换状态为启用；再输入“研发”并翻页 | 筛选原子应用草稿；翻页保留草稿且沿用已应用词 | 筛选请求仍用旧前缀，输入框被重置；翻页也丢弃草稿，未应用提示不存在。输入意图与请求不一致 | 失败；[筛选后图面](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-baseline/search-filter-draft-loss.png)，JSON `R02-filter`、`R02-R03-paging` |
| R03 · P1 分页焦点 | 21 条结果，聚焦下一页后按 Enter；仅延迟真实第 2 页请求以观察加载阶段 | 加载及完成后焦点位于有意义的控件或状态区 | 加载时分页节点数变 0，焦点到 `BODY`；真实响应结束后仍在 `BODY`。延迟是浏览器探针条件，不代表服务器故障 | 失败；JSON `R02-R03-paging` |
| R04 · P1 文字可读性 | 正常菜单中采集顶栏“组织配置”及下拉“业务单元”的实际前景/祖先背景 | 普通文字合成对比度至少 4.5:1 | 顶栏当前选中项为 14px 粗体白字，青色底加 15% 白层后约 2.53:1；初始、悬停及聚焦采样均不足。下拉选中项约 15.01:1，此项通过，不能泛称所有菜单不合格 | 顶栏失败、下拉采样通过；[计算](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-baseline/contrast.json)、[菜单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-baseline/header-menu.png) |
| R05 · P2 历史识别 | 详情 → 日志 → 当次来源，滚动详情；使用返回当前资料 | 阅读期间持续识别只读模式并可直接退出 | 桌面 900px 高的单条内容未触发丢失；短屏、窄屏、320 宽、压力及两种真实缩放中，提示和退出入口随正文滚走，测得可见面积均为 0。返回顶部后退出可清除来源参数 | 持续标识失败，当前退出路径通过；[短屏滚动后](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-baseline/short-history-scrolled.png)，JSON 历史各项 |
| R06 · P2 日期与日志 | 创建后进入详情/日志，并核对同一真实日志 DTO 的字段集合 | 纯日期与时刻清楚；可读分类、操作者和有权字段完整 | 原生输入显示 `11/01/2026`，时间轴中文日期、正文 ISO 日期并存；日志直接显示带 Z 的原始时刻，只列中文名称变化。API 已有 actor、英文名、生效日及分类快照；可读姓名及后续维护前后差异 DTO 未提供 | 失败；[日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-baseline/creation-log.png)、[详情](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-baseline/short-history-scrolled.png)，JSON `R06-log`；缺口按第 6 节交接 |
| R07 · P2 帮助与反馈 | 首次新建、未提交就取消；再提交 `_invalid` 编码 | 格式/禁用说明提前可读，字段错误引导修正，未提交不暗示已有提交 | 编码无前置帮助；保存禁用无关联说明；原因已有必填语义，但仍缺少为何不能保存的解释。未提交离开弹窗出现“待确认提交仍可查回”；真实字段 400 错误也显示“重试加载”。字段聚焦本身正确 | 反馈失败；[首次表单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-baseline/form-initial.png)、[未提交提示](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-baseline/unsubmitted-guard.png)、[字段错误](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-baseline/field-error.png) |
| R08 · P2 信息层次 | 390×844 从菜单新建，核对标题/页签/字段和列表首屏 | 保留语义并减少重复层次 | “新建业务单元”出现 2 次，“基本资料”出现 2 次且只有一个页签；窄屏列表控制区约 247px 高。关键字段仍在，但重复层次占用空间 | 对本轮优化目标为失败；[窄屏表单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-baseline/narrow-create.png)、[窄屏列表](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-baseline/narrow-list.png)，JSON `R08-form-hierarchy` |
| R09 · P2 导航语义 | 正常菜单查询后检查对象名称及列表/新建/详情标题 | 真实链接支持新标签/复制，标题能区分页面 | 名称为 `BUTTON` 且无 href；三个页面的 document.title 都是 `CHNHCM`。当前按钮返回定位保持既有实现，本轮未把它说成新链接验收通过 | 失败；JSON `R09-navigation`；链接/目标失效的完整回归留 T04 |

源码定位：[布局与滚动](../../packages/ui/src/layout.css)、[列表/分页结构](../../packages/ui/src/components/Lists.tsx)、[主题](../../packages/ui/src/theme.ts)、[提示组件](../../packages/ui/src/components/Feedback.tsx)、[BU 页面接线](../../apps/web/src/features/organization/BusinessUnits.tsx)、[返回适配](../../apps/web/src/shared/navigation.ts)及[实际 DTO](../../api/openapi.yaml)。这些文件本批没有修改。

### 3.2 七组矩阵的实测边界

| 条件 | 实际内容视口 CSS px | 列表正文高度 / 从列表打开首条 | 历史提示及退出滚动后 | 其他实际结果 |
| --- | --- | --- | --- | --- |
| 桌面 100% | 1440×900 | 716 / 通过 | 本条记录内容下仍可见 | 登录按钮可达；表单离开 Escape 保留输入、放弃返回已执行 |
| 短屏 100% | 1440×540 | 356 / 通过 | 不可见，失败 | 同上，返回顶部后可退出历史 |
| 窄屏 100% | 390×844 | 约 532.56 / 通过 | 不可见，失败 | 抽屉菜单及表单继续/放弃路径可操作 |
| 320 宽 100% | 320×900 | 约 588.56 / 通过 | 不可见，失败 | 表单及历史返回可操作，但历史模式不常驻 |
| 压力视口 100% | 320×256 | 0 / 失败 | 不可见，失败 | 登录按钮滚轮不可达、Tab 可达；表单正文约 100px 高，实际滚轮可移动 |
| 真实 200% | 640×450 | 约 230.31 / 通过 | 不可见，失败 | 表单正文约 294px 高、实际滚动约 256.5px；放弃/返回路径完成 |
| 真实 400% | 320×225 | 0 / 失败 | 不可见，失败 | 登录按钮滚轮不可达、Tab 可达；表单正文约 69px 高、实际滚动约 400px |

真实缩放先校准内容 **1280×900**，窗口外框 **1280×1039** 不变，通过任务专用 X11 显示向 Chromium 发送原生 Ctrl+0、Ctrl++，未设置 CSS zoom、设备倍率或捏合。浏览器自身显示[200%](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-baseline/zoom200-browser.png)和[400%](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-baseline/zoom400-browser.png)；测得 DPR 分别 2/4、visualViewport.scale 始终 1。结束恢复 100%，内容回到 1280×900、DPR 1，详见 JSON `zoomCalibration`、`zoomRestored` 和登录矩阵。

各条件均从菜单检查列表及新建，记录滚轮、Tab/Shift+Tab、Enter/Escape 与离开保护。为了在列表打不开的条件下继续采集详情/日志/历史，矩阵详情统一采用已创建对象的真实直达地址，JSON 明确标为诊断访问；**这些图面不证明该条件下正常菜单全链路通过**。完整的菜单创建/拒绝/保存/日志路径在桌面执行，登录实际提交在 100% 执行；没有逐视口重做所有认证错误、提交未知、权限变化和完整搜索状态事件。

### 3.3 文档与保护检查

012 修订覆盖 R01—R10 的长期页面要求，保留 011 稳定业务锚点。`make docs` 的 63 项脚本测试及仓库文档检查通过，见[初次输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-baseline/docs-first.txt)和[定稿检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-baseline/docs-final.txt)。`git diff --check` 与新增文本附件的等价空白检查通过；b 运行归属再次核对通过。302 份原文件全部对照：301 份未变，1 份为本批授权的验证导航更新；010/011/013/API 另与受测提交核对无差异。主工作树及 a 的 HEAD、索引和状态在本轮前后均保持；没有写入其他工作树。这些检查不证明页面问题已修复。

## 4 失败、修复与复验

首次主采集在返回已有同一查询条件后，错误地等待一次必然出现的新响应而超时；当时已经完成真实创建和部分表单/日志采集。[首次原始结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-baseline/attempt1-interrupted.json)保留为中断，不登记为页面测试通过。探针改为识别已应用的相同查询并等当前结果就绪，随后完整采集见 `results.json`，50 张图面独立保存。另在原生缩放环境准备时修正了 X11 隐藏窗口焦点识别，并根据实际 Chromium 工具栏高度校准内容视口；它们属于探针准备，不是产品修复。

收尾首次保护脚本把本批需要更新的 `docs/verification/README.md` 也归为不可变文件，因该预期导航修改而失败；[首次保护结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-baseline/protection-first.json)保留。随后明确这一项为授权交付物，仍对照原有全部 302 项，未删除任何对照项；其余 301 项未变。

产品 R01—R09 没有修复或复验。下拉菜单对比度、字段错误焦点、Escape 保留输入和已有历史退出等局部表现保留为通过，不掩盖同一场景中的失败。对比度计算基于实际颜色的 sRGB alpha 合成与相对亮度；悬停/聚焦为采样时刻，未声称覆盖过渡动画每帧或所有文字状态。

## 5 未覆盖范围与限制

- T03/T04 修复未实施；搜索事件表全量、新链接/复制/新标签及返回目标失效、历史跨页签持续标识的完整增强验收留后续，不以旧 27 项浏览器通过代替。
- 本轮针对当前创建/查询切片。未来新增记录、更正、删除、停用、多个业务来源维护、消费者引用及完整日志差异未实施；多个独立 BU 只为分页，不冒充一个 BU 的多条生效记录。
- a/b 同时在线、两分支最新合同集成及远程 CI 未运行，留 T05；a 的并行文档只核对状态，没有读取并采纳其未合入业务决定。
- 未重跑 `make check/build/integration` 或完整浏览器套件；产品输入未改，本次使用真实浏览器定向取证和文档检查。没有新增覆盖率目标、政策阈值或低信号单元测试。
- 没有 Firefox、WebKit、真实移动设备、屏幕阅读器、全站无障碍或生产验证。原生日期的含糊显示是本次 Chromium 环境观察，不推断所有语言/系统都相同。

## 6 结论与后续事项

### 6.1 012 修订及审阅范围

本轮按用户继续实施授权完成 T02，审阅人为实施代理 `/root`，属于自检，不冒充独立人工或其他维护者批准。逐项核对 100 清单、012 原合同、当前源码/DTO 和实测结果：

| 修订 | 确定内容 | 依赖/边界 |
| --- | --- | --- |
| 查询与返回 | A/D 逐事件表、失败/迟到/会话/刷新/POP、新标签及真实链接返回焦点 | 不持久化搜索草稿，不把查询改动变为表单离开保护；Go/API 查询权威不变 |
| 布局与密度 | 小视口允许文档滚动，适配实际滚动容器；保留语义、去重复标题/单页签 | 原桌面基线保留；不为固定结构压没正文 |
| 历史位置 | 紧凑只读短条及退出入口；桌面控制区常驻，小视口文档滚动时贴靠可见顶部 | 长说明留正文；不引入写入/恢复历史入口，退出仍重读当前授权事实 |
| 日期/时刻 | 纯日期中文年/月/日，API 仍 YYYY-MM-DD；记录时刻按 Asia/Shanghai，明确 UTC+08:00，展开保留原精度 | 不做第二套日期资格、时区业务换算或日志排序 |
| 日志及表单 | 利用已有获授权字段和保存分类；actor 明确为主体标识；错误/禁用/未保存/未明结果分开表达 | 不将 UUID 当姓名，不从当前详情拼不同来源事实 |
| 验收 | 七组视口/真实缩放及各事件映射写入 012；旧证据范围保持 | 文档采纳不提升产品通过、完整 M02 或资产状态 |

### 6.2 M02 交接及下一批

[100 具体交接清单](../archive/plans/100-ui-experience-improvement.md)列明 `packages/ui`、`BusinessUnits.tsx`、`shared/navigation.ts`、012/浏览器用例、013/OpenAPI 和登记的协作边界。a 采集开始时为 `889bcd9`，有包含 010/011/012/013、ADR 和登记等的并行修改；b 不导入这些未合入规则。双方实际提交到位后再逐段核对合同与实现，不整文件覆盖。

当前日志 DTO 已有 actor 主体标识、action、时间、记录字段及分类快照；T04 可使用这些真实字段。**可读操作者名称、后续维护的前后值和目标/关系**由 Organization/API 维护者在 013 定义来源、当前授权及一致性，再由 010 实施；本轮没有擅自新增姓名或差异 API，也不将这一缺口描述为所有日志数据都缺失。

下一批 T03 先修复压力视口/400% 列表零高并适配返回滚动，再处理查询草稿、分页焦点及主题对比度，运行 012 规定的受影响行为检查。T04 处理历史、日期/日志、表单层次与链接；T05 保留集成和 a/b 联运行。T02 材料完成，整体 100 仍为 draft/in_progress，未提交或发布本轮更改。

## 7 证据保存

长期入口：[图面索引](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-baseline/index.html)、[结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-baseline/results.json)、[对比度计算](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-baseline/contrast.json)、[采集前摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-baseline/inputs-before.json)、[收尾保护](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-baseline/protection.json)及[证据摘要清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-baseline/manifest.json)。完整采集原输出在 `.local/t02/run2/`，首次中断在 `run1/`，仓内另存结果，不依赖临时目录作为唯一证据。

[浏览器探针](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-baseline/baseline.mjs)、[原生缩放助手](data/100-ui-baseline/native-zoom.py)和[显示截图助手](data/100-ui-baseline/x11-shot.py)保存本次采集实现，属于验证附件，不接入产品或常规测试门禁。复现时先核对受测提交及产品输入仍与本报告一致；探针中的基线标识为本次固定值，不能在改变产品后沿用为新基线证据。从本仓根目录准备新的非敏感输出目录，将两个 Python 助手放到 `.local/t02/`，在独立 Xvfb 显示运行 `DISPLAY=:<任务显示号> node docs/verification/data/100-ui-baseline/baseline.mjs <新输出目录>`；助手通过原生键盘设置浏览器缩放。需要本仓依赖、Xvfb/X11/XTest 和已核对归属的 b 服务，不访问相邻仓库。脚本会创建新的虚构 BU，不覆盖本报告已定稿证据。
