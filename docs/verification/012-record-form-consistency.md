# CHNHCM-012 记录表单一致性调查与修复验证

登记信息：[registry.json](../governance/registry.json)。对应010的T03-UI-FORM及[012对象表单合同](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-object-form)。本批检查已完成，结论仅覆盖以下范围。

## 1 对象与基线

用户要求先提交现有变更，再调查并修复记录新增/更正与新建业务单元的控件样式不一致。原页面优化已本地提交为db2b5f8，本批以该提交后的干净b工作树为基线；固定wt-dev-b，使用m02-more-actions既有隔离环境（Web4382/API8382/DB54372）。密码只保存在仓外，未进入提交；没有推送或改动a/main。

正常入口为设置 → 组织配置 → 业务单元。管理员分别从列表新建、详情新增生效记录、生效日期轴选择记录 → 更多操作 → 更正此记录进入。资料只使用合成BU；沿用当前Go/API、权限、字段显式修改、来源与结果恢复。

## 2 原因与证据

结论是实施/整合遗漏为直接原因，同时存在规则适用范围与评审覆盖不足，不能归结为“没有样式组件或任何规范”。[源文件摘要和关键行](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/012-record-form-consistency/root-cause.json)保留各基线证据。

| 层次 | 实际证据 | 判断 |
| --- | --- | --- |
| 既有组件 | ObjectForm提供样式上下文、局部主题与.object-form作用域；ObjectTextField在该上下文中生成左侧标签和标准输入，离开时有意保留MUI默认行为 | 组件已有可用标准，单独复用输入组件不等于接入完整表单 |
| 实施及整合 | 新建使用ObjectForm；a的7c89dc2及db2b5f8中的TimelineEditor仅导入ObjectTextField，没有ObjectForm与字段列布局；旧b的f5cf6ec维护页原本使用完整组合 | a/b整合采用当前维护实现时遗漏了原表单组合，造成描边/浮动标签、字号/高度及复选框密度回退 |
| 文档 | 012原2.4明确“创建表单”的14px、32px、左侧标签、下划线及两列规则，未明确新增/更正必须沿用，也未说明完整组件组合依赖 | 有标准但适用范围和接线要求不充分，本批在原规格澄清，不新建重复设计规范 |
| 验收 | 前轮43项及七视口主要验证菜单、行为、焦点及可达性，未比较维护字段与新建的实际外观 | 上轮评审遗漏；旧报告和图面原件保留，本报告补充当前交付结论 |

基线浏览器从正常菜单测得：新建名称输入为14px/32px、行高20px、标签在左；新增输入为16px/56px、行高23px、浮动标签。新增的独立样式对照在未修复产品上失败（6.1s），与用户问题一致。更早一次探针因必填标记导致标签定位不匹配，仅计脚本准备失败，不计产品复现。

新增和更正共用同一名称字段映射，因此都受这一上下文缺失影响；基线的浏览器数值来自新增，未将代码推断伪称为两种页面都单独测量。修复后分别对两种入口重新验证。

## 3 修复内容

记录新增和普通更正复用ObjectForm、ObjectDetailSection/ObjectDetailField、ObjectTextField和ObjectFormError，沿用现有样式，不新增一套CSS或全局覆盖。资料/记录信息使用同一两列网格，窄屏改单列；日期、原因、只读原分类、必填帮助及字段错误接入同一外观，复选框和控制栏按钮沿用紧凑密度。

编辑时主体只保留一份字段区域；原日期和分类仍只读，新增不默认原因，更正不增加原因输入。输入后自动标记、显式清空英文、预览/保存、撤权/历史与未保存保护均保留。控制栏仍使用原portal和同一状态，不复制表单或改变业务规则。

012明确统一规则的适用范围和可观测外观；013说明组件组合依赖，023补充对照断言，010跟踪本次修正。100保持原完成记录，新增体验采用只登记本批范围。

## 4 预期与实际检查

新增浏览器断言以同环境真实新建为基准，比较新增/更正的字号、高度、行高、标签位置、下划线和选择器（按控件类型与同环境新建逐一对照），核对复选框及按钮密度；桌面两列、窄屏单列，尺寸切换后输入保持，取消后焦点回更多操作。首次修复后的对照通过（7.2s）；最终完整44项通过，含本次1项跨入口外观对照及原43项旅程。

| 检查 | 实际结果 | 证据 |
| --- | --- | --- |
| 修复前独立外观对照 | 失败，符合用户问题；名称输入16px/56px而新建为14px/32px | [基线日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/012-record-form-consistency/before.log)、[失败图](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/012-record-form-consistency/before-add.png) |
| make check | 通过；Go测试/vet、边界、类型、65项Python及4项Node；18.648s | [检查日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/012-record-form-consistency/check.log) |
| make build | 通过，12.887s；保留既有MUI指令和大分块提示 | [构建日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/012-record-form-consistency/build.log) |
| make smoke | 最终44/44通过，247.367s | [最终浏览器](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/012-record-form-consistency/smoke.log) |
| 七组视口及原生缩放 | 全部通过，76.045s，进程正常退出；逐组新建参照、新增预览/取消、更正预览/真实保存 | [矩阵](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/012-record-form-consistency/matrix.json)、[图面入口](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/012-record-form-consistency/index.html) |
| 原证据及来源保护 | 522个原文件、211项来源/完整资产状态和100元数据保持；Go/API/存储及共享UI原语未改 | [执行清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/012-record-form-consistency/execution.json) |

七组为1440×900、1440×540、390×844、320×900、320×256及原生200%/400%。原生100%先校准1280×900，外框1280×1039保持；200%为640×450、DPR2，400%为320×225、DPR4，visualScale均1。截图保存实际可见表面和账号遮罩；每组经正常菜单、键盘、实际滚轮及API读回核对，不用模拟DPR代替真实缩放。字段测量及截图均区分新建、新增、更正；原生日期本轮为34px且两侧一致，未硬套普通文本32px。

原业务旅程继续验证真实API/DB中的新增/更正、准确目标、来源保护、权限、未知结果、字段错误、名称边界及显式清空。Go/API/存储未修改，不因样式通过提升完整M02或全部TL案例状态；独立DB专项不在本次重跑范围。

## 5 评审与未覆盖范围

本轮由实施代理进行静态、行为及图面多轮自查，非独立人工批准。字段错误测试按标准错误摘要和aria-invalid定位；只读分类测试按字段标签及对应值核对，不再依赖原来一串拼接文案。业务预期和权限断言不放宽。

全回归首轮43项通过、1项因错误信息同时出现在摘要和字段旁而定位歧义；改为核对摘要按钮、字段错误状态和原焦点，产品代码不变，随后完整44项通过。图面探针准备阶段分别修正原生日期高度假设、重开详情后等待动作可用及原生日期内部Tab停靠：使用同控件新建参照，实际离开日期控件后才等待默认资料。上述脚本失败及原图保留，不计为产品通过；最终七组使用修正后的真实操作路径完成。

两个返回操作的区分继续暂缓；停用/重新启用、删除、改日、联合、真实消费者、综合容量、Firefox及生产交付未纳入。七视口仅复验受影响的新增/更正表单，不宣称100全部页面重新验收；旧012/100报告及资产来源不改写。

## 6 结果与证据

T03-UI-FORM已完成。修复后的静态、完整浏览器、图面及多轮自查没有新增主要问题；010仍in_progress，100保持原完成记录。受测内容是db2b5f8之后的差异，输入摘要及命令在[执行清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/012-record-form-consistency/execution.json)，修复另作本地提交；不把报告自身提交号当受测基线。

源证据见[原因摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/012-record-form-consistency/root-cause.json)，原失败全回归见[首轮日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/012-record-form-consistency/smoke-round1.log)，探针准备和失败截图见[早期归档](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/012-record-form-consistency/earlier-rounds.zip)。原始非敏感输出在.local/verification/012-form-consistency；凭据、cookie、trace和持久浏览器profile不进入仓库。没有推送、PR或main合并。
