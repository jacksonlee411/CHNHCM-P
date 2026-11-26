# CHNHCM-111 Community列表T03真实业务验证

登记信息：[registry.json](../governance/registry.json)的governance.verification_records。依据[111计划](../archive/plans/111-bu-list-experience-upgrade.md)、[012列表交互](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-list-pro)、[013接线](../specs/m02/013-m02-technical-design.md#m02-tech-list-pro)及[023第21节](../specs/m02/023-m02-timeline-test-cases.md#m02-list-pro-cases)，按[验证模板](../templates/verification.md)编写。

## 1 对象与受测基线

2026-11-03，固定wt-dev-b，基线`c68ddf064595060c15d2cdfae6dce8184597cc1d`，起始工作区和stash干净。范围为111的T03：真实菜单、Go/API/PostgreSQL下的TL-323—334、受影响维护代表例及七组布局条件；不启动T04列筛选评估或T05整体交付。

本轮修正返回定位、点击对象位置快照、共用测试入口，并新增真实业务、原生链接及视口检查。Go、OpenAPI、SQL、生成客户端、供应商源码及39px样式未改。旧T01/T02、52px和CubeBox39px报告及原始证据不覆盖；最终输入及检查见[检查记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t03/checks.json)。12例当前业务参数已通过；10组列表测试、4组维护代表回归、3项API/DB检查、七组条件及两个原生复制地址布局分别留证，不相互折算。

## 2 环境与前置条件

Node24.19.0、pnpm10.24.0、Go1.26.0、Chromium140.0.7339.186。自有profile为m02-community-t03-b，验证时Web4393/API8393/DB54393；从登录及正常菜单“设置→组织配置→业务单元”进入。管理员、查询者、实际字段裁剪与撤权均由当前服务/授权库承接。业务样本每批使用唯一前缀的73个真实API创建BU，包含未来首条、停用、长名称和37位长编码；排序预期独立于服务端实现计算，未在浏览器伪造正常列表结果。

首次隔离环境启动因Docker自动地址池耗尽失败，保留[首次输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t03/runtime-up-first.txt)。核对既有网络和宿主路由后，仅给本任务网络chnhcm-bu-45af7952bb98_default分配10.253.93.0/28；未清理他人网络或复用既有数据库。初始化及验证使用当前空库模型，真实连接密码始终留在仓外。测试不保存trace、会话或CSRF，截图遮罩身份；原生缩放截图在输入凭据前采集。

## 3 场景、步骤与实际结果

逐项结果及参数见[案例映射](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t03/cases.json)。浏览器测试组数、业务案例数、API检查数和视口条件数分别统计，不互相折算。

| 案例 | 实际检查及预期 |
| --- | --- |
| TL-323 | 管理员/查询者正常菜单；新建权、三段控制区、按需面板及#066b68 |
| TL-324 | 独立预期核对四页名称升降序/清除回编码；摘要排序入口使用同一Go白名单 |
| TL-325 | 0/1/20/21/73条、20/50/100大小、页边界和加载/失败范围表达 |
| TL-326 | 390px及700px摘要、长名称/编码、未来首条及真实字段裁剪；断点草稿保持 |
| TL-327 | 标准/摘要名称Enter、Control、中键、复制地址后打开及普通单元格不导航；原生菜单与真实剪贴板另测 |
| TL-328 | 第二页视口外对象、首次浏览器后退、显式返回、跨页对象导航、慢响应用户优先、撤权及更正后移出筛选 |
| TL-329 | A/D草稿、加载、改大小、失败重试、排序应用、刷新及前进后退重建 |
| TL-330 | 日期/状态局部草稿、取消/Escape、不完整日期、应用和服务端今天；实际启用/停用/未生效结果 |
| TL-331 | 实际字段授权和撤读权；延迟真实响应、退出/重新登录及真实当前会话失效；不让旧结果覆盖新输入 |
| TL-332 | 下表七组条件；真实滚轮、Tab及页签方向键、字段校验、未保存保护、详情/日志/来源及同对象返回 |
| TL-333 | 原生显隐、改宽度和全部隐藏恢复；展示操作无业务请求，A/D及摘要保持；取消拖动/固定，不开放未采纳能力 |
| TL-334 | 六张业务事实/结果表计数保持，查询无业务写入；实际新增、更正、旧来源及未知提交查回代表回归 |

真实业务使用API创建/读取和数据库授权变更。网络失败、延迟与丢失响应通过传输拦截注入；成功响应先由真实Go/API/DB生成，未用合成正常结果替代业务验收。

| 条件 | 实际内容视口 | 验证结果位置 |
| --- | --- | --- |
| 桌面100% | 1440×900 | matrix.json的desktop |
| 短屏100% | 1440×540 | short |
| 窄屏100% | 390×844 | narrow |
| 320px宽100% | 320×900 | width320 |
| 压力视口100% | 320×256 | stress |
| 原生200% | 640×450，外框1280×1039不变、DPR2 | zoom200 |
| 原生400% | 320×225，同一外框、DPR4 | zoom400 |

[七组结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t03/matrix.json)记录浏览器、内外尺寸、滚轮位置、操作与收尾；图片名位于matrix子目录。[200%倍率](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t03/matrix/zoom200-browser-menu.png)及[400%倍率](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t03/matrix/zoom400-browser-menu.png)已核对原生浏览器显示，结束恢复100%。使用独立Xvfb及原生快捷键，不以CSS zoom、缩小视口或deviceScaleFactor替代。焦点链接几何检查允许1 CSS px取整误差，原失败的整行移出视口不属于此容差。

[原生链接结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t03/native-links.json)另覆盖桌面/摘要的真实右键“Copy link address”、剪贴板与受控href一致、打开真实详情及原草稿保持。Linux使用Control；不冒充macOS Command或其他浏览器验收。

## 4 发现、修复与复验

实际发现一：短屏自然文档流中，名称链接首次获焦后自动行高再次测量，目标可能整个移出视口。返回钩子改为订阅实际网格stateChange及DOM渲染，只首次设置焦点，后续测量校正可见性；自身程序聚焦不误判为用户接管。用户输入、滚动或转移焦点立即停止，查询/页面变化及卸载清理订阅，不增加固定等待帧数或超时。

实际发现二：手机首次从列表进入详情后直接浏览器后退，原列表URL没有selected，焦点退到标题。名称链接现在将点击对象ID、位置和当前历史条目标识存入原有位置快照；返回时只在同一条目及原选择条件匹配时使用，恢复受控选择及链接焦点。当前API结果和读权仍优先，不保存旧正文/授权，不给纯复制地址或新建动作增加业务写入。

失败图面、原始输出和中断状态见[复测历史](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t03/review-history.json)。一次合并14组复测以143中断，未计整轮通过；后续按顺序分组完成，结果分别保存在business-a/b/c及representatives输出。

测试修订包括：共用入口从更多菜单读取日期；长编码样本遵守40位合同；对象导航使用真实按钮角色；部分选中列先恢复全选再验证全部隐藏；页签使用方向键；必填错误检查原生校验；等待实际响应后撤除拦截。严格100%交叉比例因不足1px取整失败，改为几何1px容差。首次失败与后续通过分开保留，测试修订不记为产品缺陷。

## 5 检查范围与限制

[真实API/DB输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t03/integration.txt)运行3项当前检查：字面搜索/稳定分页/字段、未来首条/确切历史，以及鉴权/撤权锁序。[make check](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t03/make-check.txt)和[应用类型/源码构建](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t03/build.txt)按实际输出记录，[实际网格返回11项](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t03/grid-return.json)及[合成延迟钩子7项](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t03/hook-return.json)另行通过，后者只证明钩子适配，不能提升为真实业务。定稿后的资料及差异检查另见最终检查。

不声明全334编号、全部290现行案例、全M02、全维护动作/故障组合、真实外域消费者、Firefox、macOS或生产能力通过。T01冻结安装及供应商来源证据按未变输入保留，不把本次业务通过提升为完整来源资产全部验证。列条件/Excel形态仍属T04，整体交付仍属T05。执行与复核均为主代理，不冒充独立评审或具名业务人员验收。

## 6 结论与资源收尾

T03按本报告12例当前参数完成，发现的两项返回问题已修复并复验；T04列筛选评估及T05整体交付仍未关闭。任务自有Web/API已停止，数据库容器exited，4393/8393/54393均无监听，临时浏览器profile与Xvfb219/221已关闭；持久卷chnhcm-bu-45af7952bb98_database及本任务网络保留，供相同profile复现。既有4180和其他BU环境未启停或写库，不接管其生命周期。实际核对见[收尾记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t03/cleanup.json)。不重建、清卷或改写已有业务历史，本轮未提交或推送。

## 7 证据保存

长期证据位于[data/111-community-t03](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t03)，含逐例映射、当前检查、失败/中断记录和必要图面；非敏感原始输出位于.local/111-community-t03。所有被引用的最终输出在仓内保存，历史报告保持原字节。任务源码和最终资料输入由最终检查关联，不能仅用HEAD代表脏工作区。

integration.txt的Testcontainers诊断含原生尾随空白，保持原字节；仅该证据路径使用既有Git属性方式保留，应用及其余文本空白检查不变。
