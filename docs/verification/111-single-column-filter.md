# CHNHCM-111 单列条件实施与验收

登记信息：[registry.json](../governance/registry.json)。用户在T04后另行采纳“单列条件＋统一应用/取消”。原[T04评估](111-column-filter-evaluation.md)和[T03](111-community-t03.md)保留原结果，不改写其历史未实施/已通过范围。当前规则由[110第2.6节](../adr/110-bu-list-and-datagrid.md)、011 BR009、012第3.1.5节及013第4.1.2节持有。

## 1 范围与输入

基线`b63a2de`，固定wt-dev-b；此前T04评估文档已按用户要求先本地提交。本增量含前后端、合同、案例及证据，交付时未另提交/推送。管理员/查询者从设置→组织配置→业务单元进入；详情和名称真实链接、20/50/100及#066b68保持。

| 列 | 已实施条件 | 语义 |
| --- | --- | --- |
| 编码/名称 | 包含、等于 | trim及不区分大小写的字面比较，最多400码点；独立于统一q，不检索英文名 |
| 业务状态 | 等于、属于任一 | 单值或1—3个不同枚举，同列OR；一个item，不是多列条件 |
| 生效日期 | 等于、早于、晚于、不晚于、不早于 | 比较businessAt解析的展示记录；未生效时为未来首条；纯日期，不扫描任意历史记录 |

每次至多一个列条件，与q/status按AND组合；切列替换原条件。取消/Escape/外部关闭不改A/D、不查询；清除只改草稿，应用才移除条件。应用合并当时搜索D、回第一页。Excel候选/联动、跨列组合、空值/否定、日期范围及高级查询后置；T05整体交付未关闭。

## 2 实现及关键取舍

公开GridFilterForm承接列/操作符及原生枚举输入，应用承接按需Dialog与独立草稿；文字和纯日期使用同步MUI输入，避免原生防抖尚未提交最后输入即应用及Date对象时区转换。桌面列菜单/已应用图标和更多操作、移动更多操作/摘要共用同一个编辑器；主网格只投影A，filterMode=server。列宽/显隐保持原生，不引入Pro或修改供应商源码。

公开API新增成组filterField/filterOperator/filterValue，独立于组件模型；Go白名单及值校验，SQL在当前对象/字段授权、日期解析、全结果筛选、总数及页码纠正的一份statement snapshot中执行。不可读编码/名称/日期不命中该字段谓词，也不进入数量；状态沿用现行对象读权。未知、重复、缺项、错配及非法值由API拒绝。无新迁移、缓存或兼容格式。详情对象导航和URL/刷新/历史/返回携带同一已应用条件，失败/重试与请求代际沿用当前合同。

## 3 真实API和数据库验证

[最终集成日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-single-column-filter/integration-final.txt)的7个真实Testcontainers根测试通过，包含新增TestSingleColumnFilter的4组及6组受影响旧查询/来源/权限回归。每个根测试使用隔离当前空库及真实HTTP/Go/PG18.6，容器在cleanup终止；没有mock列表结果。

新增参数包含25个合成BU（23个编码组、编码/名称交叉命中及未来首条）、一次资料新增及停用：独立编码/名称、字面%_、大小写/空白、空集、跨页23条和页99纠正、q/status相交；状态集合不等于all；五种日期边界、未来首条和非任意历史命中；缺项/重复/错配/超长、无权字段、对象范围和撤权。查询前后的业务表快照相同。旧测试另覆盖Unicode查询、合并字段授权排序、当前/旧来源、字面检索和稳定分页。

[首次集成失败](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-single-column-filter/integration-first.txt)使用了名称规则不允许的反斜杠样本，400符合原合同；改成合法%_名称后[复验通过](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-single-column-filter/integration-second.txt)。未放宽名称规则。

## 4 浏览器业务验收与修复

[执行表](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-single-column-filter/cases.json)记录18个业务组最终通过：新增TL-335—342的8组＋受影响既有列表10组，不按运行次数累加。正常结果均来自本任务真实API/DB；503/延迟是受控传输注入。新增8组覆盖桌面列菜单及页外对象命中、应用/取消/清除、状态单/多值及日期、A/D与分页/排序/历史、详情跨页及返回、字段/对象权和非法直达、失败重试/迟到响应、390/320px与短屏及断点焦点。Go集成补足更多边界。

首轮发现：对话框中原生表单外层与内部宽度叠加、隐藏逻辑控件留下空位，列/条件可访问名称包含当前值导致严格标签无法定位。通过公开输入props同步外层和内部宽度、隐藏未采用的逻辑/删除包装，并为列/条件提供稳定可访问名称；390px垂直布局，不修改第三方源码。首轮中第四组在修订导致的HMR期间登录未就绪，随后人工中断，另3组未运行；不计通过，也不把该受扰运行当最终产品结论。首次类型检查发现公开声明不接受GridFilterForm.sx，改为应用Box及公开输入props承接。[首轮失败/中断](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-single-column-filter/browser-first.txt)保留；[第二轮8组通过](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-single-column-filter/browser-second.txt)。

最终组合运行[17/18通过](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-single-column-filter/browser-final.txt)，唯一失败为TL333仍执行“筛选入口关闭”的旧断言。按用户新合同改成筛选入口可见，继续验证降序/固定/导出关闭及列显隐/宽度/恢复只改变展示；[定向复测通过](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-single-column-filter/tl333-final.txt)。原失败和[截图](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-single-column-filter/failures/tl333-old-expectation.png)保留，不把组合运行记为整轮18/18。其他17组已经通过，未因这一测试预期更新重复全套。新增8组没有pageerror。

## 5 七组布局与真实缩放

[七组结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-single-column-filter/matrix/results.json)均通过：1440×900、1440×540、390×844、320×900、320×256；有界面Chromium固定外框1280×1039，从100%内容1280×900原生放大至200%（640×450）和400%（320×225），结束各自恢复100%。没有CSS zoom、DPR或小视口替代原生倍率。图像按实际设备像素保存，截图时账号与登录输入遮罩；浏览器倍率图发生于输入凭据之前。

每组真实菜单进入、输入当前页外对象条件、Tab到达应用并Enter，真实结果1条；另改列草稿后Escape，A和搜索D保留；Tab到清除及应用，恢复73条；校验对话框/文档无横溢，动作及焦点可见。低短屏对话框可滚动。截图人工查看桌面、390px、原生400%及倍率显示；[桌面](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-single-column-filter/matrix/desktop-filter.png)、[窄屏](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-single-column-filter/matrix/narrow-filter.png)、[400%](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-single-column-filter/matrix/zoom400-filter.png)分别留证。七组无pageerror或Pro许可提示。

## 6 检查、复核与复现

[make check](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-single-column-filter/make-check-first.txt)包含65治理测试、边界、运行设施测试、Go测试/vet及TypeScript检查通过；当时报告为草稿，最终文档与引用另复核。[生产构建](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-single-column-filter/build.txt)通过（Vite保留大chunk提示，不屏蔽）；[Community来源核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-single-column-filter/mui-source.txt)确认8.29.3、固定提交、501源码/500声明和MIT，源码未改。[最终检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-single-column-filter/checks.json)记录最终文件摘要、文档/引用及分轮语义复核结果。登记表、规则、案例和实际运行范围分别管理，不将结构检查计作业务通过。

复现先在本工作树选择独立profile，配置仓外端口并up；本次profile为m02-column-filter-b（Web4394/API8394/DB54394）。运行`pnpm test:smoke column-filter.spec.ts community-list.spec.ts`，通过CHNHCM_T03_OUT指定新的非敏感证据目录；该复用支持程序的名称不意味着执行旧T03批次。视口脚本为tests/scenarios/column-filter-viewport.mjs，fixture输入取指定输出目录的fixtures.json，每次生成新的合成资料。集成复现命令：`go test -tags=integration -timeout=6m ./tests/integration -run '^Test(SingleColumnFilter|T07LiteralSearchStablePaginationAndFields|T07FutureFirstAndFourExactHistoricalAnswers|BUNameSortUsesCombinedVisibleFields|BUFullUnicodeNameSearch|BUCreateQuerySourceHistory|T07ExactReadonlyScopeAndJointFieldPermissions)$' -count=1 -v`。无需外仓、Pro或密钥。

本轮主代理实施并分轮自查；不是独立代理评审或具名业务人员签收。不声明全342例、完整M02、全维护组合、跨浏览器、真实消费者、生产或容量SLA验收；旧T03原生复制链接的证据不重跑为本批新结果，名称链接及返回的受影响代表行为已重新验证。

## 7 环境收尾及结论

[收尾核验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-single-column-filter/cleanup.json)：任务自有Web/API停止，数据库容器exited，4394/8394/54394无监听；临时浏览器profile与Xvfb219关闭。持久数据卷chnhcm-bu-e444db862af7_database及任务网络10.253.94.0/28保留。Docker默认地址池耗尽的启动失败保留于日志，核对不重叠后创建本任务专用网络，没有prune或操作既有服务/库。旧T04/T03报告、证据及供应商原件保持；未在仓内保存真实凭据或令牌。

本次单列条件与统一应用/取消实施及限定验收通过。T04评估原批已完成，本增量另行获授权并交付；Excel/跨列能力和T05整体交付保持原后置边界。
