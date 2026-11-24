# CHNHCM-021 main页面与T04整合验证

登记信息：[registry.json](../governance/registry.json)。依据为[010](../plans/010-m02-scope-and-plan.md)、[012](../specs/m02/012-m02-functional-and-interaction-design.md)与[021](../adr/021-bu-lifecycle-and-record-maintenance.md)，仅记录a吸收main后已接通批次的整合与复测。

## 1 基线与范围

a基线c5ea156d69770993398ac94362d63160a0806524，开始时工作区干净且stash为空；吸收main的f59bfb869480f56cd78c164b1a5d6631a6c2d531。该main来自[PR3](https://github.com/jacksonlee411/CHNHCM/pull/3)，最新头ec6612b的quality/integration/browser均成功后按merge保留历史合入，见[远程结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-main-ui-integration/pr3-ci.json)和[工作流](https://github.com/jacksonlee411/CHNHCM/actions/runs/41303342504)。本地main、b及远程b同步到该合并提交。

受测对象包含12处文本冲突的解决、启停菜单/标准表单接线、测试定位和文档修订。实际源码及测试输入159份逐文件摘要见[最终输入](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-main-ui-integration/product-inputs.json)；首轮输入另存。Go、SQL、OpenAPI与a基线相同。基线外输入、来源脚本和证据摘要见[执行清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-main-ui-integration/execution.json)，不以报告自己的最终提交号代替受测对象。

## 2 整合取舍

保留a的Go名称/NFC、T04、撤权与继承修复；UI使用b的更多操作、查看日期弹窗、标准表单及简洁表达。启停接入同一菜单和控制栏；不恢复字段勾选、常驻机制说明或默认前序详情。名称确认通过实际输入及简洁预览完成；预览显示记录生效后状态，其余时间线按需展开。日期资格、可办动作、字段权及实际保存仍由Go决定。

403重读继续清理编辑值、预览、旧请求及父详情/日志缓存，仍可读输入保留；旧200预览迟到不能恢复已撤权英文。两边文档登记、资产采用记录与历史证据合并保留；012/023修正早期UI基线中仍写“启停未实施”的当前状态说明。

## 3 环境与旅程

使用独立m02-main-ui-integration空库，运行标识9a5d83370a78，Web4456/API8456/DB54536、当前m02-timeline-3和桌面Chromium。正常启动完成迁移、运行角色及两类登录身份初始化；仓外凭据文件0600。原a的4436、b的4382及CubeBox4180监听保持，未清空旧持久库。运行归属见[核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-main-ui-integration/ownership.txt)，启动见[日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-main-ui-integration/runtime.txt)。

管理员及查询者从“设置 → 组织配置 → 业务单元”进入列表/详情，创建后经更多操作办理普通更正、停用和重新启用。覆盖三类日期、多轮启停、未保存、准确继承、独立快照、原来源、丢响应查回、并发、撤权及名称/NFC。浏览器权限夹具在finally恢复原授权，视口验证在完整浏览器结束后独立执行。

## 4 实际检查

| 检查 | 实际结果 | 证据 |
| --- | --- | --- |
| make install / make generate / make check / make build | 通过；生成器无额外差异，65项Python/4项Node、Go边界/测试/vet及前端类型通过，应用构建完成 | [安装](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-main-ui-integration/install.txt)、[生成](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-main-ui-integration/generate.txt)、[检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-main-ui-integration/check.txt)、[构建](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-main-ui-integration/build.txt) |
| 真实DB/API完整回归 | 31个根测试通过，容量1项显式跳过；integration包128.861s、bootstrap包24.222s | [集成](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-main-ui-integration/integration.txt) |
| 首轮Chromium | 48/51，三处旧定位/精确文案断言失败，保留原结果 | [首轮](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-main-ui-integration/smoke-first.txt) |
| 修正专项 / 最终完整Chromium | 3/3（29.6s）；最终51/51（约5.4分钟），包含六项启停、名称/NFC及原UI回归 | [专项](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-main-ui-integration/browser-focused.txt)、[完整](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-main-ui-integration/smoke.txt) |
| 七组视口 | 桌面、短屏、390、320、320×256及原生200%/400%全部通过；包含启停表单、标准控件对照、正常菜单、键盘、滚轮及实际更正 | [矩阵](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-main-ui-integration/matrix.json)、[51张图面包](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-main-ui-integration/viewports.zip)、[桌面重新启用](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-main-ui-integration/after.png) |
| 来源及历史保护 | a的214份和main的599份既有非导航验证/参考文件字节保持；211资产来源/整项状态及原证据/采用记录不丢失；100原计划保持 | [历史](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-main-ui-integration/history-protection.json)、[资产](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-main-ui-integration/asset-protection.json)、执行清单 |

200%/400%在固定1280×900内容外框使用原生浏览器缩放，分别得到640×450/DPR2与320×225/DPR4，visualScale=1；不是把小视口当作缩放。视口脚本复用原012并新增启停表单，菜单键盘改用End到查看项；原脚本原样保留。视口中停用验证预览/取消，重新启用验证字段/取消；真实重新启用保存与资格由完整51项旅程验证。

两轮[自查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-main-ui-integration/review.json)分别核对合同/冲突/证据和修正后实际旅程/图面。范围内没有新增主要问题；不冒充独立人工评审或业务批准。

## 5 失败与复验

首次文档检查因本报告草稿缺少results而失败，补记未运行条目后make check通过，见[原失败](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-main-ui-integration/check-first.txt)。

首轮浏览器的三项失败为：样式循环仍找“保存”而重新启用为“确认重新启用”；查询者仍以button定位已改为链接的名称；时间线精确字符串缺少本轮预览增加的生效后状态。分别改为当前按钮名、真实链接及包含“启用”的准确预期，未改产品、超时、重试或断言强度。三项专项与完整51项重新执行通过。最终输入相对初轮只变这三份测试；构建和DB对应的产品输入保持一致。

PR3此前的滚动采样失败及修正已单列[012记录](012-pr-scroll-test.md)，其原证据保持。

## 6 限制

不关闭T01/T02跨后续动作、T05/T06/T08及T07全容量；不代表全部TL、Firefox、真实下游消费者或生产交付。两个返回入口的设计调整继续暂缓，100不重开。构建仍有主块大于500KB的建议提示，不将构建成功当作性能验收。

本轮整合在本地完成。a未推送或创建PR，其远程CI尚未运行；后续发布仍须按最新提交执行三项CI，不复用PR3当作a的发布结果。

## 7 证据

最终[文档检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-main-ui-integration/docs-final.txt)与差异检查通过；业务/测试输入在最终执行后保持不变。

本地原输出在.local/verification/main-ui-integration，非敏感日志、矩阵、受测输入及摘要归档到本报告data目录；仅去ANSI控制码、行末空白及多余末尾空行。图面遮罩账户，浏览器临时profile在仓外创建并正常清理。凭据、cookie及CSRF未归档；既有定稿报告和来源包保持原字节与原结论。
