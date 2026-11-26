# CHNHCM-133 用户可见提示语排查

登记信息：[registry.json](../governance/registry.json)的附属验证记录。
适用范围：当前BU、部门、部门树、登录及共享提交恢复中的提示、错误和操作文案。
上游依据：[012页面表达原则](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-expression)、[119部门页面](../specs/m02/119-department-functional-design.md)、[133去过度处理路线图](../archive/plans/133-overdefensive-processing-remediation.md)。

## 1 范围与基线

2026-11-14按用户要求排查“已提交请求请保留原标识查回”及类似增加理解和操作负担的表达。工作树为Jason主机wt-dev-d，HEAD为dfc4ba17fe9cdc3a9a28c556788ee82f6f71325c；包含前两轮未提交的关键词清空和旧对象提示修复。本轮只检查和登记，不修改产品文案、恢复流程、API或运行环境。

首次枚举apps/web/src、packages/ui/src、packages/api-client/src、internal、cmd、api下407个文件；按源码类型及排除条件实际扫描235份Go／TS／TSX文件，抽出694处含中文行，再核对错误的产生、捕获、透传和渲染。694包含标签、目录值及注释，不是694条问题。首次归纳21组问题、63个匹配表达；[首次位置与源码摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/133-user-facing-message-audit.json)保持原样。用户随后要求多轮补查，本次4轮新增17组、补充或限定原9组，累计38组，另列2类相邻表达问题；详见第7节和[补查证据](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/133-user-facing-message-audit-review.json)。

测试注入文案、生成文件、供应商源码及历史报告不当作现行产品提示。以下区分直接显示、API可透传和正常页面已遮蔽的原文；静态可达不表示所有故障均已在真实页面复现。文案来源是否由AI生成不作为判断依据，问题依据用户能否理解、能否执行及是否准确。

## 2 用户所问句子的含义与问题

“原标识”是系统给某次保存生成的submissionId，用来确认这次操作最终是否保存，并在重试时防止重复办理。正常用户不需要抄写、保存或输入它。

现有[SubmissionFlow](https://github.com/jacksonlee411/CHNHCM/blob/dfc4ba17fe9cdc3a9a28c556788ee82f6f71325c/apps/web/src/features/organization/dated/submission.ts)在发送前生成标识并保存待确认引用；引用只含实例、用户及提交身份。当前页重试复用已保存的attempt.id和attempt.command，用户不负责重建这两者。[BU适配器](../../apps/web/src/features/organization/businessUnitAdapter.ts)与[部门适配器](../../apps/web/src/features/organization/departmentAdapter.ts)把标识自动带到结果查询接口。保存引用失败只能证明当前调用尚未执行send；若之前已经发送过同一attempt或有旧待确认引用，不能据此推断先前请求未保存。刷新后只恢复引用，不恢复原请求内容，因此“重试保存”还须以本页实际持有原attempt为前提。

这句通用错误在[writeError](../../internal/bootstrap/api.go)中产生，未分类错误都可能使用它，函数并不区分查询和保存。于是单纯读取失败也可能暗示“你已经提交了一笔请求”，并要求保留页面没有要求用户管理的标识。前端[unwrap](../../packages/api-client/src/index.ts)把API message交给各页面；普通列表、详情等读取错误会直接呈现其message。

保存结果未知是真实状态，不能随意改成“保存失败，请重新保存”，否则用户可能重新发起同一业务操作。适合页面的表达是“暂时无法确认是否保存”，配合“查看保存结果”等实际按钮；标识保留、原内容复用由系统承担。普通读取失败只应说明当前内容暂时无法加载，并提供重试。

当前[恢复组件](../../apps/web/src/features/organization/dated/SubmissionRecovery.tsx)读取本地引用后显示按钮，由点击调用resolve，并没有在进入页面时自动查询所有待确认结果。因此，单改文案为“系统正在自动确认”会作出当前实现没有履行的承诺。是否增加自动确认是另一项交互实现，不属于本次排查已完成内容。

## 3 问题清单

### 3.1 优先处理的责任外推和状态误导

| 编号 | 当前表达与场景 | 用户负担／误导 | 建议及源码入口 |
| --- | --- | --- | --- |
| MSG-01 | 服务暂不可用；已提交请求请保留原标识查回 | 查询也被套用保存指令；用户找不到要保留什么、如何查回 | 读取与保存分场景表达，通用错误不要求管理标识；[API](../../internal/bootstrap/api.go) |
| MSG-02 | 原提交结果暂不可读，请保留提交标识；原分类证据不完整，请保留提交标识并核对来源 | 将结果恢复和数据完整性诊断转交业务用户 | 说明保存结果／历史资料无法显示；标识与内部诊断由系统处理；[结果读取](../../internal/contexts/organization/infra/postgres/timeline.go)、[分类检查](../../internal/contexts/organization/infra/postgres/history_read.go) |
| MSG-03 | 请按原提交标识查回；请查回原提交；按原标识与内容重试；按原内容重试 | “查回”“原标识”不是页面业务动作；内容本来已由系统固定 | 页面用“查看保存结果／重试保存”，由现有流程复用原请求；[事务](../../internal/contexts/organization/infra/postgres/transaction/scope.go)、[新建](../../apps/web/src/features/organization/DatedPages.tsx)、[编辑](../../apps/web/src/features/organization/TimelineEditor.tsx)、[固定期间](../../apps/web/src/features/organization/LifecyclePairDialog.tsx) |
| MSG-04 | 请先查回待确认提交；有一笔提交结果待确认，请先查回 | 新建禁用帮助、页面警告和全局恢复条可能重复讲同一件事 | 合并为一处保存结果确认区，保留阻止重复办理的必要状态；[新建](../../apps/web/src/features/organization/DatedPages.tsx)、[恢复](../../apps/web/src/features/organization/dated/SubmissionRecovery.tsx) |
| MSG-05 | 无法保存待确认引用，请启用浏览器本地存储后重试；无法读取待确认引用，请检查浏览器存储 | 用户不知道引用是什么；“检查存储”不可操作，还可能误导清缓存而丢掉恢复依据 | 区分本次未发送、先前结果仍未知和无法读取上次保存进度，不以当前存储错误断言旧请求未保存；技术原因供诊断，不让用户找引用或清存储；[提交](https://github.com/jacksonlee411/CHNHCM/blob/dfc4ba17fe9cdc3a9a28c556788ee82f6f71325c/apps/web/src/features/organization/dated/submission.ts)、[恢复](../../apps/web/src/features/organization/dated/SubmissionRecovery.tsx) |
| MSG-08 | 已重读当前资料，请核对后保存 | 成功重读后被写进error，再以红色error Alert显示，造成操作仍失败的错觉 | 清除已解决的错误；如确需提示资料已变化，用中性反馈及具体变化；[普通编辑](../../apps/web/src/features/organization/TimelineEditor.tsx)、[固定期间](../../apps/web/src/features/organization/LifecyclePairDialog.tsx)、[错误组件](../../packages/ui/src/components/ObjectForm.tsx) |
| MSG-15 | 部门读取计划不可用；部门查询依赖不完整；部门路径来源证据不完整；候选上下文不完整 | 展示的是取数实现和内部数据诊断，用户不能据此修复操作 | 按实际入口表达列表、路径或上级选项无法加载；细节留给维护诊断，不能伪装成空结果；[查询](../../internal/contexts/organization/application/department/query.go)、[路径](../../internal/contexts/organization/infra/postgres/history_read.go)、[候选](../../internal/contexts/organization/application/department/parents.go) |

MSG-05必须分别处理存储被禁止、引用格式损坏等原因，不能把所有故障都归因为浏览器权限。MSG-15也不能删掉必要的数据损坏拒绝，只处理用户看到的表达。审计日志仍须保留诊断所需事实。

### 3.2 可以缩短或取消的说明

| 编号 | 当前表达 | 建议及源码入口 |
| --- | --- | --- |
| MSG-06 | 原提交已保存。原结果只读，后续办理须返回当前资料；已确认原提交未保存，可核对后重新办理 | 成功只说明“已保存”；确认未保存才说明可重新保存；历史只读在实际历史视图表达；[恢复](../../apps/web/src/features/organization/dated/SubmissionRecovery.tsx) |
| MSG-07 | 待确认提交仍可查回 | 必要时用“稍后仍可查看保存结果”，不重复叙述存储保证；[离开保护](../../packages/ui/src/components/Feedback.tsx) |
| MSG-09 | 已清除日期输入；仍可更正有权字段；已保留仍可读输入；输入已保留供核对；移除无权修改 | 保留真实权限变化反馈，说明受影响字段与可做动作；按钮可用“撤销这些字段的修改”，去掉系统保留输入的长解释；[编辑](../../apps/web/src/features/organization/TimelineEditor.tsx) |
| MSG-10 | 搜索词尚未应用；查询后生效，离开列表或刷新后不保留。已应用关键词…… | 保留必要的草稿／生效条件区分，缩为“输入已修改，点击查询后生效”；不常驻介绍内存生命周期；[列表控件](../../apps/web/src/features/organization/BusinessUnitListControls.tsx) |
| MSG-11 | 原页码已失效，已返回有效页 | 当前分页控件已反映实际页码，可取消独立提示条；[列表](../../apps/web/src/features/organization/DatedPages.tsx) |
| MSG-12 | 原对象已不在当前查询中，请返回列表重新查询；对象导航暂时不可用，请返回列表重新查询 | 自动辅助导航失败不应让用户误以为当前详情不可用；保留当前资料，按需局部提示无法切换并提供重试；[详情导航](../../apps/web/src/features/organization/dated/useObjectNavigation.ts) |
| MSG-13 | 路径上下文；请重读本层；请重新定位或选择全部 | 节点不可选择时说明“不可选择”；树变化对应刷新树，不要求理解缓存层，也不把扩大范围当成默认修复；[树面板](../../apps/web/src/features/organization/DepartmentTreePanel.tsx)、[树读取](../../apps/web/src/features/organization/DepartmentTree.tsx) |
| MSG-14 | 来源已变化，请重新核对资料；所选记录修订已变化，请重新核对 | 保留并发冲突，表达为“资料已更新，请核对最新内容后保存”；使用获权名称／日期定位，不要求理解来源或修订；[命令](../../internal/contexts/organization/application/commands.go)、[候选](../../internal/contexts/organization/domain/dated/candidate.go) |
| MSG-16 | 导出内容超过本次处理预算，请缩小查询范围；选择暂存额度不足；导出暂存空间不足 | 这里至少三种原因，不应统一改成数量过多。实际文件超过大小限制时建议减少所选项或分批导出；改变查询范围不等于改变已有固定选择。临时资源不足说明暂时不可用；[导出](../../internal/contexts/organization/application/selection.go)、[选择](../../internal/contexts/organization/application/department/draft.go)、[暂存](../../internal/contexts/organization/infra/exportfile/file.go) |
| MSG-20 | 开放终点；至某日（不含当日） | 使用Go提供的包含结束日或“自某日起”。真实期间影响仍要展示，不在前端新增业务日历；[删除及日期影响](../../apps/web/src/features/organization/TimelineEditor.tsx) |

此前删除的“原选中对象不在当前页或筛选结果中，查询条件已保留”不再计为本轮现存产品问题；实际修订和未通过的原生链接综合用例仍由[300记录第6节](300-department-tree-implementation.md#6-清空关键词后末级部门仍为空修复)持有。

### 3.3 协议层和已遮蔽文案

| 编号 | 原文／问题 | 当前边界及建议 |
| --- | --- | --- |
| MSG-17 | 同一提交标识不能改变内容；提交标识或内容摘要无效；JSON重复或多余字段；路径后续页需要游标；管理BU筛选须使用完整BU身份及等于条件 | 多数属于异常请求或客户端组包缺陷，不能声称普通用户一定遇到。保留协议错误码及拒绝条件，页面按场景提供可执行恢复，不让用户修改标识、摘要、游标或JSON。位置详见逐项清单 |
| MSG-18 | 至少明确一个资料字段；明确修改时至少提供一个资料字段；请选择修改内容 | 前端已自动识别修改，这些参数合同不应变成额外勾选要求。真实无修改继续“没有修改内容”，客户端错误另作诊断；[资料规则](../../internal/contexts/organization/domain/bu_data.go)、[部门命令](../../internal/contexts/organization/application/department/commands.go) |
| MSG-19 | 本批未开放此动作；通用联合办理已退出 | 开发批次／旧实现沿革不面向业务用户；仅必要时说明“此操作不可用”。当前多为未支持协议分支，不能据源码存在推断正常菜单有这个步骤；[候选](../../internal/contexts/organization/domain/dated/candidate.go)、[期间命令](../../internal/contexts/organization/application/lifecycle_pair.go) |
| MSG-21 | 预览超出执行预算，请重新预览；读取变更日志超出执行预算，请重试 | 预览是自动辅助，普通用户没有“重新预览”入口。但当前TimelineEditor已将该503转成“辅助资料读取失败，请重试加载”，不能把原API文案冒称页面实见；仍应清理API术语。日志错误可经读取流程显示，应改成加载失败／重试；[预览错误](../../internal/bootstrap/maintenance.go)、[日志错误](../../internal/bootstrap/changes.go)、[实际捕获](../../apps/web/src/features/organization/TimelineEditor.tsx) |

## 4 需要保留的真实信息

不能把所有“失败”“权限”“确认”都当作过度防御并删除。以下信息确实影响业务判断，只应简洁表达并匹配实际按钮。

| 信息 | 保留理由 |
| --- | --- |
| 保存结果未知 | 不等于未保存，防止用户误发新操作；标识处理是系统职责 |
| 未保存编辑的离开保护 | 用户真实输入可能丢失，需要明确选择 |
| 同日已有记录、日期范围、启停顺序、部门关系冲突 | 属于Go业务不变量，用户需要据此修改实际字段 |
| 当前无权限、会话过期、范围不可用 | 不能显示受限数据或把拒绝伪装成成功／空列表；不给出隐藏对象的内部拒绝原因 |
| 资料已被修改 | 覆盖他人变更前需要核对；不要求用户管理来源版本 |
| 历史资料只读、未来记录尚未生效 | 区别当前可办理资料，影响用户下一步选择；只在相关视图说明一次 |

变更日志按需展开的来源版本、原始时刻和操作者标识属于获权历史证据，不能与普通流程中要求用户“保留标识”混同。它们不应常驻普通表单，但不在本次建议删除范围。

## 5 整改顺序与验证边界

优先顺序为MSG-01／02／03／04／05／08／15，再处理其他普通页面说明，最后整理协议层和已被遮蔽的文本。落实时同步前端与API文案，保持现有错误码、未知结果、授权、业务历史和幂等保护；不能靠统一改成“操作失败”掩盖真实状态，也不建立第二套业务判定。

恢复区应由一个组件明确呈现保存状态和对应动作；同一未知结果不同时堆叠通用503、新建阻断说明、表单警告和恢复条。若增加自动查询结果，必须先实现并验证，不能只更换措辞。存储错误不能引导清空恢复引用；结果校验不一致不能静默采用返回内容。

后续实施需验证真实读取失败、发送前失败、发送后丢响应、保存成功／未保存／仍未知、刷新恢复、同内容重试、撤权、迟到响应、冲突后重读及选择导出。自动辅助和错误显示还应核对按钮存在、错误已解除后的呈现及重复警告数量。这里列的是待实施验证，不是本轮已通过结果。

## 6 首次检查与交接

首次完成源码扫描、页面／接口传播路径核对、21组分类及替换建议，未新建API、数据库、浏览器或测试profile；没有业务写入，也没有更新45300、其他固定环境或前批已保留的运行资源。原容量和原生链接检查失败保持各自记录。本报告不表示文案整改或自动恢复已交付。

首次仅维护报告、索引和登记，按仓库要求运行make docs及git diff --check；结果由[首次证据](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/133-user-facing-message-audit.json)记录，保留登记缺少results的首失败及补齐后通过。产品构建、DB/API与浏览器异常复现未运行。

## 7 用户要求的多轮补查与评审

### 7.1 轮次与方法

2026-11-14在同一HEAD及前批未提交工作上，由主代理连续进行4轮检查，未使用独立评审代理。检查以源码、调用链及显示条件为依据，不冒称真实页面逐条故障复现。新增问题位置、匹配片段、40份引用文件摘要及锁定的React Router依赖核对见[本次证据](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/133-user-facing-message-audit-review.json)。首次235份产品源码摘要仍一致。

| 轮次 | 检查内容 | 结果 |
| --- | --- | --- |
| 1 | 补查页面、字段帮助、Tooltip、选择与导出、恢复条、离开弹窗及动态字符串 | 中文检索会漏掉错误码／字段路径拼接、原生网络异常；形成补查候选 |
| 2 | 从重试、重置、重新读取和离开按钮反查请求参数及状态变化 | 形成MSG-22—37共16组；明确哪些已清空输入、哪些按钮仍重发旧参数 |
| 3 | 反向审查原21组建议，核对保存未知、权限遮蔽及历史含义 | 补充或限定9组原问题；7类候选排除或限定；相邻问题单列，不为凑数并入防御性提示 |
| 4 | 复查未捕获异常、框架兜底及来源证据 | 新增MSG-38；核对源码位置、摘要、清单ID和文档，实际检查结果由本次证据持有 |

### 7.2 新增问题

以下均为静态确认的条件与代码路径，浏览器异常复现本轮未运行。优先项不仅需要改字，还要同步按钮、状态清理或错误分流。

| 编号 | 遗漏的情况及实际触发 | 建议与位置 |
| --- | --- | --- |
| MSG-22 | 恢复结果直接拼接issue.code、字段路径及“原项目”编号。带failureDetails的结果通过view.lines直接显示，普通中文扫描会漏过 | 用获权业务字段名和实际拒绝原因，内部码留作诊断；[BU](../../apps/web/src/features/organization/businessUnitAdapter.ts)、[部门](../../apps/web/src/features/organization/departmentAdapter.ts)、[恢复区](../../apps/web/src/features/organization/dated/SubmissionRecovery.tsx) |
| MSG-23 | “选择核对失败，请重试”“选择已被修改，请重新核对”，但相关catch已经clear选择，提示条没有核对／重试按钮 | 有原选择才提供重试；已清空则明确重新选择，不能按最新查询擅自重建固定集合；[选择流程](../../apps/web/src/features/organization/dated/useQuerySelection.ts)、[选择提示](../../apps/web/src/features/organization/DatedPages.tsx) |
| MSG-24 | 导出成功、列表变化和选择失败共用info提示；清除选择、改查询及成功重读并不清旧notice | 通知绑定实际操作，区分失败和成功，已解除或已过时就结束；[选择流程](../../apps/web/src/features/organization/dated/useQuerySelection.ts) |
| MSG-25 | “选择内容过大，请使用全选后排除少量记录”可能改变原稀疏选择；部门“请分批选择”把客户端已有的分块又交给用户，而且读取请求体失败也套用过大提示 | 区分传输失败和真实容量限制，系统分块不改变用户集合；不默认引导全选；[接口](../../internal/bootstrap/selection.go)、[客户端分块](../../apps/web/src/features/organization/dated/useQuerySelection.ts) |
| MSG-26 | “查询位置已失效，请重新查询”：部门路径列表错误框的重试仍携带旧cursor/page，顶部查询才清游标；父候选retry已有重置，不同入口不能混写 | 保留筛选条件，重新建立合法分页位置；错误动作须实际恢复，而非无限重发失效位置；[游标](../../internal/contexts/organization/application/department/cursor.go)、[列表](../../apps/web/src/features/organization/DatedPages.tsx) |
| MSG-27 | 日志所有错误都把“重置日志条件”作为首个动作，临时失败也引导丢筛选；重复logReturn会被拒绝，重置却不删除它，重试仍失败 | 临时故障保留条件重试，条件无效才定向重置，并覆盖所声称处理的参数；重复参数属于异常链接场景；[日志](../../apps/web/src/features/organization/BusinessUnitChangeLog.tsx) |
| MSG-28 | 编辑、固定期间和恢复的部分404／不可见路径统一变成“当前权限不足，请重新读取资料” | 不确定原因时说明当前资料不可访问，不武断要求用户处理权限；继续保护隐藏对象的存在性；[编辑](../../apps/web/src/features/organization/TimelineEditor.tsx)、[父页面](../../apps/web/src/features/organization/DatedPages.tsx) |
| MSG-29 | “会话校验失败，请重新载入”：CSRF不一致与业务撤权同为403，表单可能只重读资料，保存仍使用原session.csrf | 区分请求校验、登录失效和业务权限；提供实际恢复入口并保留输入，不能让用户猜测该刷新哪里；[鉴权](../../internal/contexts/access/infra/postgres/access.go)、[编辑](../../apps/web/src/features/organization/TimelineEditor.tsx) |
| MSG-30 | 普通编辑的操作原因加载失败只写入error，无局部重试；部门BU筛选的session读取失败同样没有重试入口 | 提供具体选项重试，保留编辑。新建和固定期间已有重试，不计为该问题；[普通编辑](../../apps/web/src/features/organization/TimelineEditor.tsx)、[筛选](../../apps/web/src/features/organization/DepartmentRelations.tsx) |
| MSG-31 | 保存未知时沿用“尚有未保存输入／继续编辑／放弃并离开”：字段已禁用，离开仅丢本页输入，不撤销已发请求 | 未知状态使用“留在此页／离开此页”等准确动作，必要时说明离开不撤销保存；普通草稿保护保持；[弹窗](../../packages/ui/src/components/Feedback.tsx)、[离开流程](https://github.com/jacksonlee411/CHNHCM/blob/dfc4ba17fe9cdc3a9a28c556788ee82f6f71325c/apps/web/src/features/organization/dated/useDraftExit.ts) |
| MSG-32 | 每个名称字段重复“最多200个字符……不可用表情、装饰图形或控制字符” | 长度帮助简洁保留，完整规则按需显示，错误定位具体字段；Go准入不变；[名称规则](../../internal/contexts/organization/domain/name.go)、[字段渲染](../../apps/web/src/features/organization/dated/DataEditor.tsx) |
| MSG-33 | “正在读取辅助资料／辅助资料读取失败”不说明具体加载内容及对保存的影响 | 按实际内容说明操作影响等，保持技术失败与业务阻断分流；不能统一说“无法保存”或“可直接保存”；[自动预览](../../apps/web/src/features/organization/TimelineEditor.tsx) |
| MSG-34 | 无日期读权时提示“候选时间线存在启停依赖冲突，请核对可读资料”，却不给也不应给隐藏日期 | 保留安全拒绝，用获权范围内的可执行动作说明当前变更无法完成；不要要求反复核对看不到的依据；[安全错误投影](../../internal/contexts/organization/application/commands.go) |
| MSG-35 | 管理BU筛选摘要仅显示“已选择的业务单元”，看不出选的是谁；用户清空搜索仍无结果时难以辨认交集 | 展示当前获权的BU编码／名称，无名称读权时准确说明；继续保留用户要求的筛选交集；[筛选摘要](../../apps/web/src/features/organization/BusinessUnitColumnFilter.tsx) |
| MSG-36 | 上次“已保存。生效日期……”依赖URL的saved值，打开新一轮更正不清它；旧成功提示可与当前编辑或新错误共存 | 新办理开始时结束上一轮反馈，或明确它属于上一笔操作；不要暗示当前输入已保存；[详情](../../apps/web/src/features/organization/DatedPages.tsx) |
| MSG-37 | 多处直接展示Error.message，网络或JSON解析异常可变成浏览器英文原文，保存场景还会叠加未知结果提示 | 对传输／解析错误按加载、登录、保存分别表达；旧会话、主动取消仍按现有守卫忽略。具体英文随浏览器而异，本轮未复现；[客户端](../../packages/api-client/src/index.ts)、[登录](../../apps/web/src/app/App.tsx)、[列表详情](../../apps/web/src/features/organization/DatedPages.tsx) |
| MSG-38 | 唯一路由未配置页面错误兜底；渲染异常可落入React Router默认页，显示Unexpected Application Error、message及stack，开发警告启用时还有ErrorBoundary/errorElement建议 | 为页面异常提供中文说明和可执行恢复，不让业务用户处理框架配置；不将未知保存说成已取消；[路由](../../apps/web/src/main.tsx)、[锁定依赖](../../apps/web/package.json)。本次只核对本地依赖实现，不声称开发者提示在生产必然显示 |

### 7.3 原清单修订及反向评审

| 原编号 | 本次补充／限定 |
| --- | --- |
| MSG-03 | “重试保存”必须有原attempt；刷新后只有提交身份，不能承诺恢复原请求内容 |
| MSG-05 | 当前存储失败不等于旧业务未保存；第2节及原建议已纠正，不将首次发送前失败与原请求重试混同 |
| MSG-13 | 路径上下文属于可显示但不可选择的祖先类型，不能猜测成完全无权限；说明不可选择的用途即可 |
| MSG-14 | 补入记录不在当前来源、停用配对变化及详情的“所选记录在此来源中不存在”。主动选择历史记录缺失与列表返回高亮不同，必要反馈应保留并简化术语 |
| MSG-15 | 补入来源路径证据、删除前来源、部门查询路径、操作身份等不可用提示，按真实入口表达 |
| MSG-16 | “选择暂存额度不足，请稍后重试”也用于单次编辑超过1024和成员超过131072的硬上限。等待不会修复相同超限输入，必须区分临时不足与硬限制 |
| MSG-19 | “该历史查询方式已停用”在详情knownAt分支直接显示，不能概括为仅API文案；正常菜单不生成该参数 |
| MSG-20 | 补入固定期间删除的“至某日之前”；日期含义本身正确，属于表达统一，不认定为业务错误 |
| MSG-21 | 补入“日志上界不能超过所查看的来源”。上界由客户端维护，用户不应调试；自动预览技术错误已被页面替换的事实保持 |

反向评审没有把以下候选列为应删除的防御性提示：登录／退出真实未知状态；停用前路径、未生效及历史只读；转为停用时撤销资料修改的真实取舍；被异步守卫拦下的取消／旧会话异常；只有供应商词条而未证明页面可达的覆盖层；API文档和开发CLI输出；不存在与不可见合并后的安全拒绝。尤其不能为了让提示“具体”而泄露隐藏对象或日期。

相邻问题另列，不计入新增17组：共享列表的复选框无障碍名称固定为“选择业务单元”，用于部门时对象名不对；列设置搜索空结果等仍可使用供应商英文词条。没有据此声称供应商全部默认文案都在产品页面显示，也没有将纯本地化项混算为防御性表达。

### 7.4 优先次序与交接

优先处理原MSG-01／02／03／04／05／08／15，以及新增MSG-22／23／25／26／27／28／29／30／31／34／37。核心是内部技术信息外露、提示与动作不匹配、状态或原因误导。其他项随后处理文字长度、通知生命周期及辅助说明；框架异常按异常页面另验。具体建议仍待实施，不能只统一替换为“操作失败，请重试”。

本次只补充报告、索引、登记与静态证据，没有产品代码修改、部署或业务写入；未创建API、数据库、浏览器或测试profile，原45300及其他保留环境不作操作。首次证据与原检查失败原样保留。源码位置／摘要核对、make docs和git diff --check的实际结果由本次证据登记；产品构建、DB/API、浏览器故障注入和后续整改验收本轮未运行。
