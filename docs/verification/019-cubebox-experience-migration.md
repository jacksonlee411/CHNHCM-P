# CHNHCM-019 CubeBox共享体验迁移与页面对比验证

登记信息：[registry.json](../governance/registry.json)的`governance.verification_records`。适用范围：十组展示组件及现有BU创建/查询。上游为[019计划](../archive/plans/019-cubebox-experience-migration.md)、[012交互合同](../specs/m02/012-m02-functional-and-interaction-design.md)、[006资产保留](../assets/006-preservation-contract.md)。

## 1 验证对象与受测基线

本仓固定`wt-dev-a`，从干净基线`57ac388a3a2c8001d8e335ebb35298e47f70e126`开始，2026-10-31至2026-11-01实施。受测内容是该基线加本地未提交的公共UI、页面接线、体验用例、补充来源包和登记变更。后端、OpenAPI、SQL及已发布迁移没有改动。逐文件受测摘要、检查输出及产物见第7节；不能只用HEAD标识本批修改。

来源固定`2deaf9276086f68e07aaa0c2cf7f35025046a9ee`，在仓外独立检出并安装锁定依赖、构建，原CubeBox固定工作树未切换或修改。来源[构建信息](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/019-ui/source-build-info.json)记录`sourceDirty=false`、源码树和HTML摘要；原离线包保留，新11项UI依赖及3项字体/许可资产分别追加来源包和逐项摘要，不覆盖原件。

## 2 环境与前置条件

WSL2；Go1.26.0、Node24.19.0、pnpm10.24.0；真实PostgreSQL18.6锁定镜像；Playwright1.55.1桌面Chromium。管理员和查询者使用本机仓外凭据，数据为独立前缀的虚构学习BU。目标4380经同源代理访问8380 Go API，开发DB为54370；来源预览4381仅供显式对比。浏览器套件与图面采集串行，遵守真实登录限流，不保存trace或凭据。

正常入口为“切换应用 → 设置 → 组织配置 → 业务单元”，窄屏通过导航抽屉。实际比较桌面1440×900、短屏1440×540、窄屏390×844。源BU列表/详情/旧表单9张，源公共紧凑表单3张，目标列表/详情/新建/只读历史12张，共24张。

## 3 验证过程与结果

### 3.1 十组迁移与实际使用

| 组 | 当前公共组件或适配 | 接线及验收边界 |
| --- | --- | --- |
| 主题及应用外壳 | AppShell、theme、layout、离线字体 | 青绿顶栏、受控消息区及独立正文滚动；保留正式登录/退出 |
| 应用和分组导航 | AppNavigation | 桌面菜单、窄屏抽屉、键盘关闭与焦点恢复；只列已接通入口 |
| 页面控制区 | RecordControl、RecordBackButton；保留PageHeader基础组件 | 固定返回/保存/取消/查看日，对象身份不挤走操作；PageHeader为可复用基础，BU使用RecordControl |
| 列表与筛选 | ListControl、FilterBar、SearchControl、ListPager、DataGridPage | 当前使用MUI Table，服务端排序分页；高级搜索、分组和收藏未采用 |
| 对象详情 | ObjectDetailSheet/Identity/Section/Field、StatusChip | 居中详情、双列/单列、状态、来源说明 |
| 紧凑表单 | ObjectForm、ObjectTextField、ObjectFormError | 创建表单及服务端错误保输入/定位；夹具另验跨页签折叠组定位 |
| 日期展示 | EffectiveDates、ViewingDateControl | 查看日与记录选择分离；多节点/单节点/空集合及键盘为组件夹具证据 |
| 对象和返回导航 | RecordPager、useBusinessUnitNavigation、useListReturn、usePageScrollRestoration | 当前授权列表、跨页移动、边界停用、选中/焦点和滚动恢复；无本地授权缓存 |
| 未保存与提交反馈 | UnsavedChangesDialog及原提交恢复适配 | 继续/放弃、保存反馈、响应丢失查回、会话交错和输入清理 |
| 来源、历史及页签 | SourceExplanation、HistoryNotice、ChangeLogEntry、ObjectTabPanel | 真实API来源与变更日志、历史只读及返回当前；展示Hook保留输入 |

### 3.2 业务与交互检查

| 场景或检查 | 前置与操作 | 预期与实际 | 结论 | 证据位置 |
| --- | --- | --- | --- | --- |
| 图面对比 | 来源从正常设置菜单打开；目标真实创建、刷新、来源、历史及返回；三个视口采集 | 24张已逐张查看，布局和必要差异符合第3.3节 | 通过 | [对照页](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/019-ui/index.html)、[几何](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/019-ui/metrics.json) |
| 构建 | `make build` | Go两可执行文件、前端生产构建成功；字体及OFL随产物分发，生产产物无展示夹具 | 通过 | [构建日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/019-ui/checks/final-build.log)、[产物摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/019-ui/target-build.json) |
| 真实DB | `make integration` | 独立Testcontainers完成11项现有业务/鉴权/历史/原子性回归；容量项按开关跳过 | 通过 | [DB日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/019-ui/checks/final-integration.log) |
| 浏览器首次完整 | `make smoke`，27项、无自动重试 | 26通过、1菜单测试流程失败，见第4节 | 失败 | [首次完整日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/019-ui/checks/first-full-smoke.log) |
| 浏览器最终 | `make smoke`，单worker、无重试 | 27/27通过：25项BU及异常交互，2项展示夹具；含最后滚动恢复和字体修复 | 通过 | [最终浏览器日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/019-ui/checks/final-smoke.log) |
| 静态及资料 | `make check` | 46项治理回归、登记/来源、Go导入边界、单元、vet、TypeScript及diff检查通过 | 通过 | [检查日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/019-ui/checks/final-check.log) |

业务关联现有[查询](../specs/m02/011-m02-business-requirements.md#m02-ac-001)、[创建](../specs/m02/011-m02-business-requirements.md#m02-ac-002)、[取消返回](../specs/m02/011-m02-business-requirements.md#m02-ac-007)、[历史](../specs/m02/011-m02-business-requirements.md#m02-ac-008)、[当前授权](../specs/m02/011-m02-business-requirements.md#m02-ac-009)、[重试](../specs/m02/011-m02-business-requirements.md#m02-ac-010)和[可靠保存](../specs/m02/011-m02-business-requirements.md#m02-ac-012)的创建/查询子集。未覆盖的记录维护反例仍归010，不将这些完整验收项整体置为完成。

### 3.3 实际图面结论与差异

实施代理通过图像工具逐张查看24张原图，非仅判断脚本退出码。两侧顶栏计算高度均65px（64px主体加1px边框），控制区起点64px，青绿背景均`rgb(9,167,163)`。桌面详情容器均宽1260px、左边82.5px、顶部144px；目标历史提示使内容向下移，属于明确历史标记。紧凑表单两侧输入均14px字体/32px高，字段正文起点按业务内容不同；目标保存/取消在三种视口均可见。各页文档无横向溢出，浏览器窗口scrollY为0；真实滚轮另验证内部区域可滚。

目标详情采用来源公共组件的紧凑身份区，源BU旧页的图标/标题更大。目标表单采用同一来源的公共紧凑标准；“设置 → 地理区域 → 国家/地区组 → 新建配置记录”是该标准的实际参考页，同时保留源BU旧描边表单截图。没有将旧表单差异隐藏为逐像素一致。

目标使用本项目“业务单元”术语、所属企业、查看日状态、操作类型和原因；不搬入源原型的上级、适用范围、办理依据、状态开关或未实施动作。列表额外显式显示查看日/状态/排序；窄屏显示列标签和选中卡片，因此目标列表信息密度与源不同。顶栏只显示已接入组织入口及真实退出，账号名截图遮蔽。这些差异遵循012及本批范围；主题、结构和关键交互的迁移对比通过。

## 4 失败、修复与复验

初次定向创建/日志验证发现：切换日志页签时详情重复请求，后续日志请求可能覆盖首次503错误。修正详情请求键，排除只影响展示的tab/back/saved，保留来源/日期参数；原地切换日志仍可重试。对应创建/日志用例复验通过。

首次完整套件26/27通过。“默认列表查回未来首条”测试调用公共开菜单助手，选择应用“设置”已导航并触发未保存弹窗，助手继续点击被遮住的组织菜单导致超时。页面保护正常；该用例改为从当前设置内点击组织配置末级入口，再断言弹窗和放弃，保留默认查看日、未来未生效、查回与返回原断言。未放宽授权、限流或失败重试。

修复后该用例单独通过，随后完整27/27通过。收尾静态复核又发现日期轴引用旧字体别名，以及异步详情重新加载时浏览器可能在空内容阶段把返回滚动位置压为0；分别改用本仓离线字体、等待当前路由内容及日志就绪后恢复。新增“滚动日志 → 查看当次来源 → 浏览器返回 → 日志位置恢复”真实流程断言，连同多节点日期轴定向2/2通过，再冻结输入执行最终图面和完整回归。

来源参考首次构建缺少Git提交/来源ref，补齐任务独立目录的固定refs后构建成功。图面采集先后因顶栏含边框1px、国家菜单匹配歧义、国家本身没有新建入口而失败；按源码核对几何并使用真实可创建的国家/地区组入口后完成24张。早期截图还发现查询按钮换行和列表列宽偏差，修正后以本次原图为准。失败不是通过记录，相关过程和最终结果分别保留。

失败截图改为统一遮蔽登录名、密码输入及顶栏账号；原自动失败截图含本机账号标识，按仓库敏感资料禁存规则移除，保留非敏感文本失败证据。

## 5 未覆盖范围与限制

新增生效记录、更正、删除、停用、审批、工作台、消费者采用、事件投递和生产备份恢复未实施；没有扩大后端业务范围。多日期和跨页签折叠组使用展示夹具，不作为多记录维护验收。浏览器拒绝/断网/迟到响应采用受控注入；真实授权与事务另由DB用例核验，不冒充生产演练。

浏览器限Chromium，未运行Firefox、WebKit、全产品或外部HCM验收。截图来自本地开发页面；生产构建另行验证，并未部署。字体原包8.33MB；前端JS压缩约649kB触发Vite默认500kB分块提示，构建成功，未修改阈值或宣称网络性能验收。

## 6 结论与后续事项

本批十组体验迁移、BU创建/查询接线、三种视口图面对比和完整检查通过，019执行范围完成。原197项资产身份/摘要及整项状态、259份后端/API/历史资料和来源原件逐项未变；34项记录本批部分采用，PageHeader仅保留基础组件并通过类型/构建检查，未作BU页面验收。来源旧通过记录和其他未采用资产保持原边界；高级搜索等需未来服务端合同，其他BU维护继续由010推进。本次不提交、推送、PR或部署。

## 7 证据保存

长期图面入口为[019对照页](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/019-ui/index.html)，同目录保存24张PNG、[逐图复核](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/019-ui/visual-review.json)、几何测量、来源构建信息与[受测输入摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/019-ui/test-inputs.json)。最终检查输出、差异及截图/构建摘要见[证据清单](019-cubebox-experience-migration.evidence.json)。127个运行/验证输入在最终采集、构建、静态及浏览器前后摘要一致；DB用到的后端输入另由[259份原件保护清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/019-ui/protected-baseline.json)证明本批未变，不因前端收尾重复运行DB。

对照页在本机[4382浏览入口](http://127.0.0.1:4382/data/019-ui/)验证可打开，15种页面/视口组合、原始像素模式和窄屏排版检查见[查看器结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/019-ui/viewer-check.json)。重启查看器可在仓库运行`python3 -m http.server 4382 --bind 127.0.0.1 --directory docs/verification`；产品入口仍为4380。本地原始输出保存在`.local/ui-migration-019/`；持久结论不依赖该临时目录或参考服务。旧验证报告、旧包及原登记来源摘要均须保持。
