# CHNHCM-510 人员页面整改实施与实页复验

旧版从应用菜单分流至「员工」和「人员档案」。固定 f 原库中，同一人员有三个雇用记录编号，旧入口需要先选菜单、再猜查看日才能辨认。整改后从「人员」输入 P00000005 就能查到 0、1、2 号，点主岗可直接切换个人资料、职务信息和变更日志。初始问题、截图和归因保留在两份调查报告中。方案初审后修改一次，两组在同一版本批准，再开展产品实施。[原 11 项](500-personnel-page-review.md) · [补查 9 项及归因](500-personnel-entry-and-date-review.md) · [逐项整改与先后依赖](../plans/510-personnel-page-remediation.md) · [方案评审](510-personnel-page-remediation-review.md)

## 1 二十项问题的复验记录

全部姓名、业务记录和截图使用本地合成验收样本。表内「旧编号」沿用 500-UI，顺序与原调查一致。具体场景、账号、像素和文件校验值记入截图清单。[截图与测量清单](data/510-personnel-remediation/observations.json)

| 旧编号 | 原问题与实施 | 实际操作与证据 |
| --- | --- | --- |
| 01 | 跨人员更正时缺少本人身份。在详情及编辑控制区显示独立获权的姓名和人员编号。 | 在两笔同类办理之间切换，从 P00000002 到 P00000003，选择另一地点直至保存可用，两页能分别确认本人。[原件](data/510-personnel-remediation/17-action-person.png) · [更正草稿](data/510-personnel-remediation/18-correct-person.png) |
| 02 | 旧修订混用当前地点，同秒记录难以分辨。依该笔修订的来源及生效日读取任职状态、当日办理和原件。 | P00000002 的两笔修订在同一北京时间秒内，当前版为丙，切换较早修订后任职与变更后地点同为乙。[选中旧修订](data/510-personnel-remediation/07-older-revision.png) |
| 03 | 从办理原件切签会携带旧办理标识。人员、任职及办理链接分别保留自身需要的参数。 | 主岗切到个人资料与日志，返回原任职；历史只读模式只提供其可用操作。[职务信息](data/510-personnel-remediation/03-job.png) · [本人资料](data/510-personnel-remediation/04-profile.png) |
| 04 | 晋升撤销后仍与有效动作同样显示。使用 Go 的撤销事实显示办理状态。 | P00000007 的 07-01 展示「晋升（已撤销）」，可打开撤销前来源确认 P6。[当日办理](data/510-personnel-remediation/14-withdrawn.png) · [撤销前](data/510-personnel-remediation/15-before-withdraw.png) |
| 05 | 默认节点、正文和保存位置不一致。服务端在获权节点中定位当前适用或首个未来节点。 | 仅有未来任期者显示「尚未生效」，P00000002 可从实际的 04-01 切至 05-01。普通页查看日判据以补查修订为准。[未来任期](data/510-personnel-remediation/03-job.png) · [实际节点](data/510-personnel-remediation/06-timeline.png) |
| 06 | 三个页签的身份、正文及返回各自独立。接入共享详情的身份、页签、时间轴和正文区。 | 对同一主岗逐签核对人号、原行及上下文，桌面双列，窄屏改为单列。[桌面](data/510-personnel-remediation/03-job.png) · [390 宽](data/510-personnel-remediation/view-390x844.png) |
| 07 | 页签键盘与面板关联不完整。共用手动激活页签和面板标识。 | 左右键移焦，空格和 Enter 激活；重新进入职务面板后焦点仍在当前页签。七种显示条件的实测尺寸见清单。[桌面短屏](data/510-personnel-remediation/view-1440x540.png) · [原生 200%](data/510-personnel-remediation/zoom-200.png) |
| 08 | 首聘、签约、录用改期的保存操作位于表尾。三种表单的保存、取消接在页头，并保留原提交与放弃流程。 | 1440×540 下三种表单均可见页头操作。修改首聘说明再取消触发放弃确认；录用改期两个窗口同时从 05-07 起办，其中一个保存 05-09；另一个仍显示草稿 05-08，保存被禁用。[双窗口冲突](data/510-personnel-remediation/26-offer-conflict.png) [首聘](data/510-personnel-remediation/10-hire-controls.png) · [放弃](data/510-personnel-remediation/11-hire-discard.png) · [签约](data/510-personnel-remediation/12-contract-controls.png) · [录用](data/510-personnel-remediation/13-offer-controls.png) |
| 09 | 人员查询缺少主兼筛选，窄屏失去排序。前后端共用排序及筛选参数。 | 输入 P00000005，390 宽选姓名降序、兼岗，显示 2 份任职、1 人。[窄屏查询](data/510-personnel-remediation/19-mobile-filter.png) |
| 10 | 办理筛选折叠后不能判断生效条件，取消可能遗留草稿。展开取已应用值，取消和 Escape 复原并归焦。 | 应用「数据更改」，再改成「离职」后按 Escape。折叠摘要及重新展开的选项都是数据更改。再用有权部门与无该部门办理的 P00000002 组合检索，保留部门条件并显示 0 笔。[零交集](data/510-personnel-remediation/25-empty-intersection.png) [已应用摘要](data/510-personnel-remediation/16-applied-filter.png) |
| 11 | 新建姓名无必填表达。登记与更正共用必填呈现和错误定位。 | 从人员菜单打开登记，提交空表并核对错误摘要与姓名栏，电话等可选字段仍可空。[登记](data/510-personnel-remediation/08-register.png) · [空表](data/510-personnel-remediation/09-register-errors.png) |
| 12 | 错增「人员档案」列表、正常列表名称错误。菜单、名录、详情共用「人员」，整合未入职人员和获权任职。 | 管理员原库样本 24 人、20 份获权任职。reader 查询同一位三岗人员，得到 0 份任职、1 人，只呈现资料及日志。[管理员](data/510-personnel-remediation/01-directory.png) · [只读者](data/510-personnel-remediation/24-reader.png) |
| 13 | 本人必须经「工作关系与任职」中转。获权主岗在本人详情直接提供职务信息。 | 通过名录进入本人后直接切「个人资料／职务信息／变更日志」；本人办理历史仍可由独立入口查原件。[三页签](data/510-personnel-remediation/03-job.png) |
| 14 | 生效日期靠右。职务信息把日期轴放入与业务单元、职务相同的详情槽位。 | 1440×900 下，三页标题右边界均为 171.5、日期组左边界均为 187.5，相隔 16 CSS px。原人员页为 1171.8 px。[业务单元](data/510-personnel-remediation/21-bu.png) · [职务](data/510-personnel-remediation/22-job.png) · [职务信息](data/510-personnel-remediation/23-person-date.png) |
| 15 | 名录有「业务日期」、职务信息有「查看业务日」。普通名录按服务端今日，详情按真实节点。 | 从正常菜单进入，无需输入日期，选择实际生效节点可查回日状态。办理表单的真实业务日与跨人历史的日期条件各自保留。[名录](data/510-personnel-remediation/01-directory.png) · [时间轴](data/510-personnel-remediation/06-timeline.png) |
| 16 | 多任职人员需猜日期。按各自记录号显示可披露的任期、期间和主兼岗位。 | 查 P00000005 一次列出三份，职务信息内可直接切另两份。[三岗结果](data/510-personnel-remediation/02-three-terms.png) · [其他任职](data/510-personnel-remediation/03-job.png) |
| 17 | 下钻后菜单指向错误对象。详情归属「人员」，事务、招聘仍各自归属。 | 主岗详情桌面高亮「人员」，打开兼岗详情的移动抽屉得到相同当前项。[桌面](data/510-personnel-remediation/03-job.png) · [移动抽屉](data/510-personnel-remediation/20-mobile-current.png) |
| 18 | 本人历史重复日期并显示无关零计数。页面以本人办理为范围，关系级办理按其所有者呈现。 | 从主岗打开本人办理历史，能查看该人原办理，无任意查看日；进入原件按已披露的工作关系或任职识别归属。[本人历史](data/510-personnel-remediation/05-history.png) |
| 19 | 返回目标与真实来源不符。按经校验的本人、任职、办理明细或查询恢复来源。 | 从三岗人员进主岗、本人历史并返回，列表恢复原选行焦点。首聘有修改时先处理草稿。[历史入口](data/510-personnel-remediation/05-history.png) · [放弃确认](data/510-personnel-remediation/11-hire-discard.png) |
| 20 | 日期倒序。Go 对获权业务节点去重升序，日内动作仍按自身顺序展示。 | P00000002 从 04-01 选到 05-01。390、320 宽和原生放大采用收起式日期菜单。[桌面升序](data/510-personnel-remediation/06-timeline.png) · [窄屏](data/510-personnel-remediation/view-320x900.png) |

## 2 实施中发现的回归与复验

原库副本的真实浏览曾在列表和详情同时读取同一搜索引用时返回 503。名录现将查询身份、授权与保留时点固定在一次事务内，搜索续期独立完成。调出侧只能读原办理时，也只向该侧返回可读任职节点；指定旧来源另逐节点检查历史权。真实 PostgreSQL 用例覆盖了两侧单独授权、查回原件及在同一请求内跨过资料到期时点。[单侧调动](../../internal/bootstrap/assignment_history_integration_test.go) · [截止时点与混合万人名录](../../internal/bootstrap/personnel_directory_capacity_integration_test.go)

首测出现过并发改日后原草稿的比较区消失，以及筛选脚本试图操作已折叠的表单。前者保留授权比较面板，后者改为从已应用摘要重新展开。录用草稿依据打开时的版本及操作资格，读权或办理资格变化时清除原草稿。变更日志页签、本人再聘成功和已改日原件分别保留进入时的返回目标。

## 3 验收范围与环境

修复前产品为 `b67cb72`，方案复审通过于 `6a196cb`。产品实施和复审修订依次见 `003e1aa`、`705ef28`、`56e59cd`，普通人员行分页复用见 `b03a20b`。本次保留 34 张实操截图。27 张来自更新后的固定 f 原库，包括标准桌面、短屏、原生 200% 及编辑取消。另 7 张来自一次性原库副本，包含双窗保存冲突及旧矩阵的窄屏、400% 操作。每张图单独标注环境及摘要。五种 100% 视口为 1440×900、1440×540、390×844、320×900、320×256。另从 1280×900 以 Chromium 原生 200%、400% 得到 640×450、320×225，实测页面根节点横向溢出均为 0。截图包含选择节点、切签、焦点恢复、200% 打开和取消表单以及 400% 只读历史返回。原始登记保留旧构建的结论，整体验收状态见实施计划登记。[视口与实测数据](data/510-personnel-remediation/observations.json) · [原生 200% 编辑](data/510-personnel-remediation/zoom-200-edit.png) · [原生 400%](data/510-personnel-remediation/zoom-400.png)


### 3.1 审查与运行结果

两组分别审查名录权限与页面操作。对 705ef28，第一组要求修正名录计数跨到期和旧来源的历史日期裁剪；第二组要求修正合法筛选零交集、改日返回、日志及再聘来源、工作关系身份及录用草稿失权。两组复核 56e59cd 后逐项通过。另在原库副本的两个真实窗口保存录用改期，验证过期草稿仍显示自己的日期且不能提交。固定环境交接时仅做浏览和取消。

独立固定前端构建的 19 项 Chromium 用例全部通过，包含企业前置、真实首聘及撤销、地点更正与并发改日、跨人办理更正、兼岗与并行关系、合同及离职、招聘录用、三页签和第 21 行跨页返回。首轮 181 项组合运行因发现重复的旧筛选步骤而在第 25 项中断；后续针对受影响的实际流程重跑。19 项结果对应本次选定范围。两套 Go 集成包按影响选取 12 项通过。此前仓库默认全量两包的 20 分钟时限均用尽，保留中断记录。[浏览器步骤](../../tests/e2e/first-hire.spec.ts) · [人员资料集成](../../tests/integration/personnel_test.go) · [任职与真实日期](../../internal/bootstrap/assignment_history_integration_test.go)

### 3.2 一万人名录的容量复验

首轮在整库及浏览器验证同时运行时，20 会话混合名录、任职行定位和人员行定位的 P95 分别为 1206、1727、1669 ms，超过 500 ms。修订授权集合、分页及行定位后，最终产品与构建及数据库测试同时运行时，普通名录和受限同名末页仍分别测得 866、625 ms。在相同产品和门槛下单独运行两套容量试验，结果如下。数值按毫秒四舍五入，各组完整分位数及错误数可逐项查阅。[容量与环境核验数据](data/510-personnel-remediation/validation.json)

| 独立实库试验 | 样本与结果 | 20 会话 P95 |
| --- | --- | --- |
| 一万人名录、受限同名末页、旧号、详情、来源、保存 | 1 和 20 会话各 6 组，每组 200 次，共 2400 次，业务及 HTTP 错误 0，各组通过 | 普通名录 387 ms，受限同名末页 258 ms |
| 一万人、10002 行，含同一人三份任职 | 1 和 20 会话各 3 组，每组 200 次，共 1200 次，匹配数、人员与任职行定位逐次校验，错误 0 | 混合名录 366 ms，任职行定位 336 ms，人员行定位 350 ms |

<a id="fixed-environment"></a>
### 3.3 固定原库交接

固定环境为 `m01-profile-f`，Web 4386、API 8386、数据库 54376。2026-11-26 更新前已在仓外私有位置备份原库和旧构建并校验摘要。前向更新后从真实菜单以 admin 和 reader 进入，完成逐签切换、日期节点和旧修订选择、返回原行、筛选取消、表单放弃，并以同一构建的业务单元和职务页对照日期轴。标准桌面、桌面短屏及原生 200% 均完成实操。双窗口竞争保存使用一次性副本，副本已回收。固定环境由 wt-dev-f 保持运行，用户验收结束或下次更新时复查。[人员验收入口](http://127.0.0.1:4386/personnel/employees) · [截图与角色](data/510-personnel-remediation/observations.json)

固定构建取自 `eb44651`，与受测产品 `b03a20b` 之间仅增补文档与证据。已核对 1963 个构建输入、运行目录的 API 文件摘要，以及从 4386 逐个读取的全部 29 个前端文件。升级前与浏览完毕后，在只读事务中比对 127 张业务表及 4 张授权表的行数和逐行内容摘要，131 张均一致。本次原库有 24 人、49 笔办理，临时搜索引用不计入业务数据比对。[构建、持久数据与正式验证结果](data/510-personnel-remediation/validation.json)
