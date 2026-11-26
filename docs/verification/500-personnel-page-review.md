# CHNHCM-500 人员页面布局与交互问题登记

2026-11-26，500 页面专项实测确认 **11 项待修复问题，其中高优先级 2 项、中优先级 8 项、低优先级 1 项**。建议暂缓本次页面专项验收。两项高优先级分别影响经办人确认正在更正哪位员工，以及核对旧办理当时的任职地点。[跨员工更正](#500-ui-01) · [旧修订与当次任职](#500-ui-02) · [治理登记](../governance/registry.json)

## 1 受测范围与依据

受测工作树为 `wt-dev-f`，提交为 `b67cb724f759473a761e7153796a5a1acd4b2bdf`，实测开始时工作区干净。固定环境 `m01-profile-f` 使用 Web 4386、API 8386、DB 54376。工作树的 1,157 个前端和共享包源文件与固定构建清单一致，在线返回的 28 个前端文件也逐个核对了摘要。管理员、只读账号均从正常菜单进入，数据为本地合成样本。509 已在其受测版本关闭前轮 27 项，本次 11 项关联上述提交。[500 页面继承要求](../archive/plans/500-personnel-roadmap.md#personnel-pages) · [509 固定环境交接](509-t04-remediation-implementation.md#fixed-environment)

使用有界面 Chromium 140.0.7339.186，对照同一构建的业务单元和职务详情。员工、档案及办理记录按场景实测五种 100% 内容视口，分别为 1440×900、1440×540、390×844、320×900、320×256。原生缩放以 1280×900 内容、1280×987 外框校准，200% 下内容为 640×450，核对了草稿、来源和返回焦点。400% 下内容为 320×225，专项核对了只读证件明细及 Escape 归焦。首聘、合同和录用改期另从各自业务入口走查。[012 视口矩阵](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-viewport-matrix) · [121 接入矩阵](../specs/m02/121-dated-object-admission-matrix.md) · [400% 证件明细原图](data/500-personnel-page-review/57-reader-native400-dialog.png)

## 2 高优先级问题

<a id="500-ui-01"></a>
### 2.1 500-UI-01 跨员工更正表单缺少员工身份

管理员从「人员 → 人事事务 → 办理记录」筛选「数据更改」，得到三笔。打开第 3 笔的 P00000002，再用「上一个对象」切到第 2 笔的 P00000003。两笔同为 2027-05-01 生效，两个详情标题都是「数据更改 · 0 号主岗」。选择「更多操作 → 更正记录」并改选地点后，「保存」已可用，表单仍无姓名或人员编号。实测最后取消了草稿。[列表](data/500-personnel-page-review/26-register-applied-collapsed.png) · [第 3 笔](data/500-personnel-page-review/28-register-person-two.png) · [第 2 笔](data/500-personnel-page-review/29-register-person-three.png) · [可保存的更正表单](data/500-personnel-page-review/65-correction-ready-without-person.png)

**关闭条件**　在同日、同类、同为 0 号主岗的两人间往返，经办人在详情及可保存的更正表单都能直接核对获权的员工姓名、人员编号和办理对象，切换后身份随原笔更新。[507 跨员工办理](../specs/m04/507-job-information-tab.md#personnel-action-register) · [办理详情标题与表单](../../apps/web/src/features/hire/RegisterActionPage.tsx#L232)

<a id="500-ui-02"></a>
### 2.2 500-UI-02 旧修订混用了当前任职与原件地点

P00000002 的 2027-05-01 地点原笔为甲→乙，后来更正为甲→丙。从 0 号主岗展开「数据更改」，选早一版「查看修订」。页面提示「正在查看只读历史资料」，上方「该日任职」显示丙，下方原件却是甲→乙。点「查看当次任职」后，上方才显示乙。两次修订发生在同一秒，两个修订按钮的可见文字也完全相同。[旧修订的上下两处地点](data/500-personnel-page-review/42-native200-old-revision.png) · [同笔当次任职](data/500-personnel-page-review/43-native200-exact-source.png)

**关闭条件**　早一版能明确核对当次地点乙，当前版能核对丙，原件各自显示相应变化。如果同时展示当前与历史值，两处均标明各自来源。同秒修订的入口也能区分。复验须同时读取「该日任职」与原件差异。[506 地点更正样本](../specs/m04/506-job-information-records.md#m04-ac-010) · [012 历史阅读](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-history-access) · [任职来源读取](../../apps/web/src/features/hire/AssignmentPage.tsx#L78) · [办理修订读取](../../apps/web/src/features/hire/JobInformation.tsx#L132)

## 3 中优先级问题

<a id="500-ui-03"></a>
### 3.1 500-UI-03 展开办理原件后切页签或任期会误报

管理员从 P00000002 的地点原件切到「个人资料」，仍能看到「更正资料」，页面却提示「没有本次办理权限」，原有「职务信息」页签消失。从 P00000008 的 2027-10-01 再聘原件切到「同号任期 → 2027-04-01 起」，旧任期已打开，却提示「该办理或来源当前不可用」。同账号未展开原件就切换时，两条路径都正常。[切个人资料](data/500-personnel-page-review/16-tab-from-original.png) · [切旧任期](data/500-personnel-page-review/59-old-term-from-rehire.png)

**关闭条件**　分别从原件切到资料、日志和另一任期，目标只按自己的对象、任期和当前权限读取，页签与提示正确。代码中任职办理号和档案编辑动作都读取了路由参数 `action`，复验需保留「先展开原件」这一前置。[502 人员页签](../specs/m01/502-person-directory-interactions.md#m01-change-log) · [507 任期入口](../specs/m04/507-job-information-tab.md#job-tab-layout) · [跨页参数](../../apps/web/src/features/hire/AssignmentPage.tsx#L135) · [档案动作解析](../../apps/web/src/features/personnel/Detail.tsx#L81)

<a id="500-ui-04"></a>
### 3.2 500-UI-04 当日办理没有区分已撤销晋升

在办理记录查询 P00000007，晋升明确标为「已撤销」。从员工列表进入其 0 号主岗，选 2027-07-01，「当日办理」把转正、调动、晋升并排列为相同样式的入口。展开晋升才可见撤销依据。实际日状态为 P5，撤销前来源为 P6，当日有效办理为转正和调动两笔。[办理记录的撤销标识](data/500-personnel-page-review/61-same-day-register.png) · [任职的当日办理](data/500-personnel-page-review/62-same-day-timeline.png)

**关闭条件**　从当日办理即可读出晋升已撤销及另外两笔仍有效。晋升原件、撤销依据和撤销前来源仍可按权打开。[507 同日办理验收](../specs/m04/507-job-information-tab.md#m04-ac-018) · [当日办理按钮](../../apps/web/src/features/hire/JobInformation.tsx#L590)

<a id="500-ui-05"></a>
### 3.3 500-UI-05 生效时间轴把查询日当成所选记录

在员工列表以业务日 2027-04-15 打开 P00000002 的主岗。实际生效节点为 2027-04-01、2027-05-01。桌面两者都未选中。390 像素宽时，收起的选择器显示 2027-04-15，展开后只有两个真实节点，也都未选中。[桌面时间轴](data/500-personnel-page-review/06-primary-between-dates.png) · [窄屏展开菜单](data/500-personnel-page-review/12-compact-date-options.png)

**关闭条件**　查看业务日保留 2027-04-15，桌面和窄屏都标出当日适用的 2027-04-01 记录。切到 2027-05-01 时，选中节点和日任职同步更新。[507 生效日期与日任职](../specs/m04/507-job-information-tab.md#job-tab-fields) · [日期轴选中值](../../apps/web/src/features/hire/JobInformation.tsx#L550)

<a id="500-ui-06"></a>
### 3.4 500-UI-06 同一主岗跨页签时丢失身份、布局和返回位置

在 1440×900 下，职务参考页和个人资料使用宽 1260 px、从 x=83 开始的居中详情区。任职标题从 x=0 起，正文宽 1425 px，字段集中在左侧，缺少「任职安排／记录信息」两列。切至个人资料或日志后，原工作关系、0 号主岗及按任职列表前后定位的控件消失。「返回列表」第一次回到任职详情，第二次才回到原行。1 号兼岗上下重复显示唯一的「职务信息」页签和标题。[职务参考](data/500-personnel-page-review/03-job-reference.png) · [主岗任职](data/500-personnel-page-review/06-primary-between-dates.png) · [同人资料](data/500-personnel-page-review/08-primary-profile.png) · [兼岗](data/500-personnel-page-review/18-secondary.png)

**关闭条件**　主岗三页签共享身份、查询内导航和详情布局，切换只替换获权面板。直接按「返回列表」能恢复原条件和任职行焦点。单面板兼岗清楚标识当前职务信息，无重复的切换入口。[500 详情与返回](../archive/plans/500-personnel-roadmap.md#personnel-pages) · [507 页签布局](../specs/m04/507-job-information-tab.md#job-tab-layout) · [任职布局](../../apps/web/src/features/hire/AssignmentPage.tsx#L181) · [资料返回目的地](../../apps/web/src/features/personnel/Detail.tsx#L187)

<a id="500-ui-07"></a>
### 3.5 500-UI-07 主岗页签用空格无法激活，回车切换后失焦

在「职务信息」上按左方向键移至「个人资料」，按空格后焦点在个人资料，选中项仍是职务信息。回车可进入资料，焦点却落到 `body`。任职面的三个页签均缺 `aria-controls`，面板缺 `aria-labelledby`。个人档案内的资料和日志可以用方向键及空格切换。[空格后的焦点与选中态](data/500-personnel-page-review/07-tab-space.png) · [按键前后 DOM 读数](data/500-personnel-page-review/measurements.json)

**关闭条件**　逐个使用方向键移动，以空格和回车激活。激活后焦点仍在选中页签，每个页签及对应面板通过可访问名称和控件标识相互关联。[502 页签键盘合同](../specs/m01/502-person-directory-interactions.md#m01-change-log) · [507 面板验收](../specs/m04/507-job-information-tab.md#m04-ac-016) · [任职页签实现](../../apps/web/src/features/hire/AssignmentPage.tsx#L206)

<a id="500-ui-08"></a>
### 3.6 500-UI-08 首聘、合同和录用改期的整单操作落在正文底部

从 P00000001 档案选「办理入职」，桌面十项拟任字段排成一列。「保存／取消」在表尾，1440×540 时取消按钮顶端约在 y=666，滚动后才可见。从 P00000022 任职进入「订立合同」，以及在「招聘 → 招聘录用」打开原单调整录用日期，整单操作也在正文末端。任职的「新增记录」已经把保存和取消放在页头控制区。[首聘桌面](data/500-personnel-page-review/39-first-hire.png) · [短屏滚动前](data/500-personnel-page-review/44-first-hire-short-top.png) · [滚动后](data/500-personnel-page-review/45-first-hire-short-bottom.png) · [订立合同](data/500-personnel-page-review/46-contract-draft.png) · [录用改期](data/500-personnel-page-review/66-offer-reschedule.png)

**关闭条件**　三条路径的整单保存、取消均可在页头控制区操作。桌面字段按业务分组双列，窄屏按组单列。用键盘验证末字段、两个动作和离开草稿的往返。[500 表单继承](../archive/plans/500-personnel-roadmap.md#personnel-pages) · [402 首聘表单](../specs/m04/402-hire-interactions.md) · [首聘提交区](../../apps/web/src/features/hire/HirePage.tsx#L485) · [合同提交区](../../apps/web/src/features/hire/RelationshipContracts.tsx#L58) · [录用提交区](../../apps/web/src/features/recruiting/Offers.tsx#L655)

<a id="500-ui-09"></a>
### 3.7 500-UI-09 员工列表缺主兼岗筛选，窄屏缺排序入口

查询 P00000005 并选研发部门，正确显示「2 份任职，1 人」。桌面「主兼岗」列菜单只有「隐藏列／管理列」，控制区也无筛选主岗、兼岗的条件。请求合同未声明该条件。切为 390 像素宽后，还缺少桌面支持的姓名、人员编号排序入口。[桌面列菜单](data/500-personnel-page-review/53-primary-filter-menu.png) · [窄屏控制区](data/500-personnel-page-review/55-employee-mobile-controls.png)

**关闭条件**　同部门同人的 0、1 号可分别按主岗、兼岗或全部查询，分页及两类计数反映获权的筛选结果。窄屏能选姓名或编号排序，进入任职后返回保留原条件和行。[502 员工查询](../specs/m01/502-person-directory-interactions.md#m01-page-map) · [012 窄屏列表](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-list-pro) · [员工控制区](../../apps/web/src/features/hire/EmployeeDirectory.tsx#L280) · [任职查询接口](../../api/openapi.yaml#L13916)

<a id="500-ui-10"></a>
### 3.8 500-UI-10 办理记录的已应用筛选与草稿难以区分

应用「数据更改」得到三笔办理，收起筛选后只看到「筛选条件」和办理数。重新展开，改选「离职」但不应用，收起再展开时控件仍为「离职」，列表仍是三笔数据更改，已应用参数仍为 `DataChange`。面板有「应用筛选／清除筛选」，没有取消本次编辑的入口。[已应用后收起](data/500-personnel-page-review/26-register-applied-collapsed.png) · [草稿重开与实际结果](data/500-personnel-page-review/60-filter-draft-mismatch.png)

**关闭条件**　收起时直接显示已应用条件。编辑、取消、应用、清除分别验证，取消后恢复已应用值，条件摘要与结果一致。用 Escape 退出时焦点回到触发点。[012 筛选草稿](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-query-state) · [507 办理列表](../specs/m04/507-job-information-tab.md#personnel-action-register) · [筛选面板](../../apps/web/src/features/hire/EmploymentRegister.tsx#L99)

## 4 低优先级问题

<a id="500-ui-11"></a>
### 4.1 500-UI-11 登记姓名缺少必填标识

从「人员档案 → 登记人员」打开空表，唯一必填的姓名没有星号，输入框的 `required` 为 false，`aria-required` 未设置。直接保存才提示「请填写姓名」，错误摘要正确获得焦点。打开已有档案的「更正资料」，同一字段已标为「姓名 *」。[登记空表](data/500-personnel-page-review/35-registration.png) · [空白保存后](data/500-personnel-page-review/36-register-error.png)

**关闭条件**　输入前即可从可见标签及辅助技术识别姓名必填。空表提交仍定位现有错误摘要，姓名以外的可选项保持可选。[501 登记字段](../specs/m01/501-person-registration-requirements.md#m01-fields) · [012 共享字段](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-detail-sections) · [登记姓名实现](../../apps/web/src/features/personnel/Editor.tsx#L528)

## 5 对照结果、证据与后续复验

| 核对场景 | 已观察的结果 |
| --- | --- |
| 固定库的正向对照 | 同部门按任职列行、按人去重正确。个人资料九栏可打开，联系方式行应用到草稿后放弃整页，重读仍为零条。合同 A 可打开后继 B，B 可查看前任 A。只读者可从日志打开当次资料并返回原链接。 |
| 一次性环境首次联跑 | 304 企业维护前置 4 项通过、1 项在 `checkPairPeriod` 请求报 `Failed to fetch`。该轮 3 项人员用例均未运行。 |
| 一次性环境定向复跑 | 3 项人员 Chromium 用例均通过，耗时 26 秒。实测地点保存甲→乙、原笔更正到丙并返回原行，受限读者只见 2 号兼岗，以及主岗、日志和撤销原单随当前授权重新读取。地点用例核对了旧原件差异，缺少对同屏「该日任职」的断言。外层进程返回 143，Playwright 记录为通过，两轮临时库、网络及容器均已回收。[首败与复跑记录](data/500-personnel-page-review/browser-regression.txt) · [现有地点用例](../../tests/e2e/first-hire.spec.ts#L540) |

本轮在固定库中对写表单只做校验、草稿应用和取消。4386 由固定 f 保留供用户验收，下次更新或验收结束时复查。本次范围为实际页面及所列菜单路径，数据库不变量、并发和全套业务回归由各业务专项承接。

68 张原始截图中，随本登记保存了 27 张合成样本原图。清单逐图列出像素尺寸与 SHA-256，并保留了相关操作的焦点、日期选中值和布局读数。完整原包位于原工作树的 `.local/500-ui-review-b67cb72/evidence.zip`，其摘要也已留存。[截图清单与操作读数](data/500-personnel-page-review/measurements.json) · [受测版本的页面验收依据](data/500-personnel-page-review/page-criteria.txt) · [验证资产登记](../assets/inventory.json)

登记日上述 11 项均待修复，修复后复验尚未执行。后续按原问题编号追加受测提交、角色、入口、实页结果和证据。优先用两名员工核对可保存的更正表单，再用同一原笔的前后修订核对上下两处地点。
