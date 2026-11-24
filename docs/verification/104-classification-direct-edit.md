# CHNHCM-104 分类直接编辑整改验证

登记信息：[registry.json](../governance/registry.json)的governance.verification_records。适用范围：更正记录内操作类型和原因直接显示、原因候选加载及受影响维护流程。上游为用户采纳方案、[012页面表达原则](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-expression)及[统一更正验收](../specs/m02/011-m02-business-requirements.md#m02-ac-031)。

## 1 验证对象与受测基线

2026-11-04，工作树`/home/vivobook/projects/CHNHCM-wt-dev-b`，固定分支wt-dev-b，基线`cba247d3181fd993b30bfe9b0326e7032d3d6d5e`，开始时工作区及index干净、无stash。受测内容为本次未提交的TimelineEditor、相应用例及011—013/023合同修改；不改Go领域、API、原因目录、迁移或持久库。

最终受测文件摘要见[inputs.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-classification-direct-edit/inputs.json)，与基线的代码、用例及当前规格差异见[受测补丁](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-classification-direct-edit/tested-inputs.patch.gz)。报告、登记、导航和证据本身不进入递归摘要。[104原报告](104-unified-record-correction.md)保留原结果与折叠设计的历史证据。

## 2 环境与前置条件

管理员由设置→组织配置→业务单元正常菜单创建合成样本，选择记录进入更正记录。使用当前Go、PostgreSQL及Chromium 140.0.7339.186，版本见[environment.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-classification-direct-edit/environment.json)。普通查询者、单一修改面、动态读写权撤回使用本轮隔离样本和真实授权判定；不写入既有持久开发库。

业务批次使用独占临时profile `task-classification-direct-b`，末批Web57736/API55448/DB56666；视口profile `task-classification-viewport-b`，Web54686/API58200/DB54610；单项复验profile `task-classification-reason-b`，Web54266/API56510/DB57044。端口为本轮临时分配，均已回收，不能作为在线入口。早期批次另分配端口，保留于各批原始输出。

## 3 验证过程与结果

| 检查 | 实际步骤及结果 | 证据与边界 |
| --- | --- | --- |
| 类型检查及构建 | `corepack pnpm build`，tsc及Vite构建退出0 | [最终构建](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-classification-direct-edit/build-scoped-final.log.gz)；保留既有依赖指令/源码映射及包体积警告，不将构建记为业务验收 |
| 分类直接编辑及差异 | 正常菜单进入即显示原因及可编辑类型，固定类型只读；初值保存无请求、取消无确认；类型/原因恢复原值后仅保存名称，请求不含classification、日志无分类差异；用户明确选类型和原因 | [批次及逐项结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-classification-direct-edit/browser-runs.json)，关联[TL-282](../specs/m02/023-m02-timeline-test-cases.md#m02-tl-282)、[TL-289](../specs/m02/023-m02-timeline-test-cases.md#m02-tl-289) |
| 分类权限、目录及历史 | 单一名称/日期/分类权、动态撤读/写权、冲突重读、目录缺原值/503、转停用名称保护、非法链拒绝、组合保存、旧来源与丢响应查回通过；新增原因选择保护另行复验通过 | 受控目录HTTP只验证页面保护；真实保存和来源读取使用Go/PostgreSQL。类型和原因不从名称或日期推断 |
| 启停及改日 | 正常菜单、非法顺序、字段错误、准确快照、取消/撤权/迟到辅助及原结果恢复通过 | 末批38项中37项通过、1项新测试的空值文案断言失败；修正该断言后单项通过，不能写末批38项一次通过 |
| 其他受影响回归 | 本轮第二批删除10、直接保存9、首条维护4项实际通过，共23项；与末批闭合参数去重共61项 | [复用依据](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-classification-direct-edit/unaffected-regression-reuse.json)：末次仅将分类候选清理限定到CORRECT，未改变这些参数的代码路径、断言和业务预期；不是重新跑完整套件 |
| 表单及视口 | 桌面1440×900、短屏1440×540、390×844、320×900、320×256及200%/400%真实缩放七组通过；键盘直接到达原因/保存、错误定位、32px控件与字号、滚轮和无横向溢出均核对 | [七组结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-classification-direct-edit/viewports.json)；矩阵对象为首条名称/日期/原因组合。保留[桌面](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-classification-direct-edit/desktop-edit.png)、[窄屏](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-classification-direct-edit/narrow-edit.png)、[400%缩放](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-classification-direct-edit/zoom400-edit.png)代表图，完整原始图留本地 |
| 文档及差异 | `make docs`的73项Python及文档检查通过，`git diff --check`通过，见[验证摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-classification-direct-edit/validation.json) | 文档结构通过不替代语义及产品验收 |
| 资源及日常入口 | 三个临时profile的容器/网络/卷、仓外配置及登记均已清除；原三个运行持久环境保持 | [resources.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-classification-direct-edit/resources.json)，日常入口只读核验通过，无本轮业务写入 |

## 4 失败、修复与复验

首轮浏览器8通过、1失败，其余51项中断：分类写权撤回后，停止目录加载时清空候选，影响仍有读权的原因草稿核对。已改为撤写权保留可读候选，撤读权清理分类候选和输入；清理限定在更正动作，普通新增仍可明确重选原因。最终分类读写撤权参数均通过。

第二批37通过、3失败，其余20项中断。两项改日失败来自旧用例要求更正表单没有下拉框，现已改为原因直接显示及不存在折叠按钮；同批启停撤字段读权用例30秒超时，该用例未修改，末批13秒复验通过，不据一次超时或复验推断根因。

末批37通过、1失败；失败是新增原因选择保护用例把MUI选择器的空值误断言为“请选择”。修正为空值不包含原选中原因后，独占单项复验10.6秒通过，真实新增及保存分类正确。保留全部批次结果，不用最终闭合覆盖历史失败。

视口环境首次因PostgreSQL初始化期间SQLSTATE 57P03失败，未进入业务检查，已回收。之后先后发现旧脚本依赖列表常驻查看日期、详情加载前操作菜单，以及窄屏日期按钮的桌面名称假设；改为当前列表标题、详情标题/基本资料页签就绪与实际键盘触发。三次脚本定位失败及已有部分通过均见[早期视口记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-classification-direct-edit/viewport-earlier-attempts.json)，最终七组全部通过。

末批业务smoke在资源回收阶段退出143；随后显式执行该临时profile的destroy成功，再逐项目核对无遗留。单项复验及视口环境自动回收成功；不把业务通过与回收结果混为一项。

## 5 未覆盖范围与限制

未运行完整make check、独立全量Go/integration套件、完整浏览器套件、Firefox、真实消费者、生产容量或远程CI；未启动116的R1—R6。真实DB/API证据来自本轮浏览器正常提交，不表示完整后端集成套件重跑。

视口矩阵为首条组合更正，非首条类型切换的真实业务验证为桌面Chromium。原因目录异常为受控HTTP响应，不推导真实依赖故障演练；原因目录迟到响应沿用effect取消及sessionGuard，并作源码复核，未另做该目录的迟到响应故障注入。主代理实施后自查，非独立评审或人工业务签收。

## 6 结论与后续事项

本次取消分类编辑前的多余点击，类型和原因按权限直接呈现，默认保留原值，固定类型只读；用户明确选择分类，系统只识别实际差异。当前分类准入、启停链、名称输入保护、旧来源及提交恢复规则保持，未引入第二套业务计算。

61项去重定向业务参数分批通过，七组首条更正视口通过。原23项本轮实跑回归及七组矩阵的复用边界已说明，不宣称一次完整套件或全M02验收。

三个临时profile均已回收，原三个运行持久环境保持。日常`m02-current-b`保留用于用户体验，由当前工作树实现和验证维护者负责，下次使用或交接时复查；实际Web入口为[业务单元](http://127.0.0.1:4382/settings/organization/business-units)，API8382。仅核验归属和就绪，不用健康接口替代业务验收。修改保持本地未提交，不推送、PR、合并或部署。

## 7 证据保存

长期证据在[data/104-classification-direct-edit](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-classification-direct-edit)：受测文件摘要和压缩补丁、环境版本、逐批浏览器结果及原始输出、23项复用依据、七组视口结果及代表图、早期定位失败、资源及最终文档检查。原始完整图面和本地辅助输出在`.local/classification-direct-edit/`，不作为唯一长期验收依据。原始凭据、cookie、CSRF及数据库密码不写入仓库。
