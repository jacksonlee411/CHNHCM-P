# CHNHCM-012 详情更多操作与查看日期验证

登记信息：[registry.json](../governance/registry.json)。受测对象：[010](../archive/plans/010-m02-scope-and-plan.md)的T03-UI、[012](../specs/m02/012-m02-functional-and-interaction-design.md)及[023](../specs/m02/023-m02-timeline-test-cases.md)相关浏览器断言。本批验证已完成；结果仅适用于下述范围。

## 1 基线与范围

工作树为CHNHCM-wt-dev-b、固定分支wt-dev-b。以f5cf6ec和a的7c89dc2整合为a45eb94；采用a的单一当前Go/存储/API模型，保留b的100布局、导航、中文日期、历史短条和本地运行归属检查。b原有旧模型DB停止但保留；未改动a或main。

当前UI基于a45eb94的未提交差异。[执行清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/012-detail-more-actions/execution.json)保存最终源码摘要、命令和范围，不能仅以HEAD识别受测内容。旧010、018、019、100、024/025报告和来源包保持原证据。a在本批基线冻结后提交的a82d85b（组织名称准入与NFC校验）未纳入；没有推送、PR、远程合并或部署。

## 2 环境与正常入口

使用本机Go、Node、Python、锁定PostgreSQL与Chromium。独立profile为m02-more-actions，key为686ae1f9f756；Web4382、API8382、DB54372，使用新的当前模型空库00001/00002，API角色为chnhcm_api。旧b卷chnhcm-bu-39bc5023cef2_database保留，未执行升级或清库。凭据只在仓外0700/0600路径；浏览器临时profile在仓外创建并在结束时删除，不保存cookie、CSRF、trace或登录正文。

管理员正常入口为“设置 → 组织配置 → 业务单元 → 列表 → 新建/目标详情”。创建合成BU后，从标题旁更多操作进入更正或查看日期。查询者使用真实只读主体；历史来源通过变更日志进入，日期改变不能解除只读或改用当前来源。

## 3 检查与结果

| 项目 | 独立预期 | 当前结果 |
| --- | --- | --- |
| 基线真实DB集成 | 当前空库、事务/历史/读写授权、恢复及初始化合同通过 | 通过；tests/integration 93.195s、internal/bootstrap 9.671s |
| 菜单与日期定向浏览器 | 草稿/取消不查询，应用使用当前GET，焦点回触发处；只读无写入口；编辑与未保存保护保持 | 通过；最终43项，244.486s，其中本批菜单交互6项 |
| 桌面、短屏、窄屏 | 1440×900、1440×540、390×844、320×900、320×256中菜单/弹窗和编辑操作可达，无页面水平溢出 | 通过；最终七组均完成正常菜单、键盘、滚轮及真实保存核对 |
| 真实200%/400% | 外框1280×1039保持不变；100%校准1280×900，CSS视口分别640×450、320×225，DPR分别2、4，visualScale=1；实际键盘/滚轮/保存 | 通过；最终七组均完成正常菜单、键盘、滚轮及真实保存核对 |
| make generate/check/build | 生成无差异；65项Python、4项Node、Go测试/vet、边界/类型/文档及构建通过 | 通过；生成无差异，make check与make build均exit 0 |

浏览器通过正常菜单创建合成BU，使用真实API与DB；新UI六项检查另观测请求数、URL、焦点、只读主体/确切来源、日志进入编辑及真实预览保存。503等界面异常使用明确的浏览器故障注入，不声称真实基础设施故障演练。集成后internal/cmd/db/API未漂移，复用本轮集成结果；UI通过不等于023全部253例现行案例通过。

图面入口见[七组截图](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/012-detail-more-actions/index.html)，每组包含菜单、日期弹窗及编辑；[矩阵数据](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/012-detail-more-actions/matrix.json)保存实际像素、遮罩、浏览器版本、键盘和滚轮结果。原生倍率另保存浏览器证明，不以viewport/deviceScaleFactor或裁切大图代替真实缩放。短屏及缩放沿用100的文档滚动策略，不要求所有正文同时出现在首屏。本轮只复验受影响详情，不将100全部页面记为重新验收。

## 4 发现、修复与复验

整合时发现b旧终态授权、双代格式与前向升级设施不符合024/025，统一采用a当前实现；旧代码和升级证据保留于Git及原报告。运行隔离保留b端口、进程、实例检查，并加入按profile分开的仓外状态、日志及程序目录，避免复用旧开发模型。

首轮定向浏览器为3失败、2通过。日期应用的路由更新分两阶段，原焦点在旧数据仍存在时提前恢复，随后因加载禁用再次丢失；改为等待目标日期响应再恢复。只读测试以重名链接定位存在歧义，修正为按当前合成编码定位目标行；未放宽产品断言。第二轮5例通过。原失败截图保留。

首轮全回归32通过、10失败，修正整合后的旧按钮角色、日期表示及滚动容器假设，并恢复a已有的历史退出保留所选记录和无变化提示。写入拒绝场景通过实际重试入口触发上下文403，不要求所有写入错误都显示无关的重试加载按钮。随后40通过、2失败，修正短屏键盘到达新建链接后的实际离开位置断言，补齐按profile仅重启API且保留前端进程记录；受影响30项及完整43项后续通过。原失败日志和截图不改写。

原生200%初轮因验证脚本在登录后导航尚未呈现时选错分支，未进入目标页；等待导航和字体后，原生200%/400%通过。末轮自查又消除编辑区重复标题，编辑标题和目标日期使用中文表示，修正从变更日志进入编辑时基本资料不可见的问题：进入显示表单，取消恢复原页签，保存定位实际记录。补充真实预览/保存和焦点断言后冻结源码，最终完整回归按固定产品源码摘要重新执行。归档时发现普通headless context关闭后遗漏浏览器关闭，七组完成后需人工终结；该轮不计正常命令完成。补齐仅关闭本脚本所持browser的清理后，七组重新执行并在50.853s内正常退出；产品及全回归输入未变，最终图面取正常退出一轮。

上述为实施代理的多轮静态、行为及图面自查，非独立人工批准。

## 5 未覆盖与暂缓

两个返回入口的区分及新增未实施，页面不增加返回今天或返回查看日适用记录；弹窗的今天只是CubeBox既有日期应用快捷操作。已有返回当前资料和历史只读保护继续回归，不重构其职责。

启停、删除、改日、有限联合、分类纠错、真实消费者、全范围容量、Firefox及生产/对外交付不在本批；UI通过不代表全部TL或完整M02完成。旧b数据库不做当前模型兼容升级。

## 6 结论与后续

本批T03-UI已完成；多轮自查修复并复验后，没有新增主要问题。100保持原completed及关闭记录，文档仍active；010整体仍为in_progress。只有ASSET-0133/0158/0203本批采用范围可提升验证状态，整项资产的implemented/verified保持false；a后续a82d85b名称策略不在本报告内。

## 7 证据保存

非敏感证据为[执行清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/012-detail-more-actions/execution.json)、[集成日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/012-detail-more-actions/integration.log)、[生成](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/012-detail-more-actions/generate-final.log)、[静态检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/012-detail-more-actions/check-final.log)、[构建](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/012-detail-more-actions/build-final.log)、[最终浏览器](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/012-detail-more-actions/smoke-final.log)和[图面](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/012-detail-more-actions/index.html)。旧失败见[回归一](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/012-detail-more-actions/smoke-round3.log)、[回归二](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/012-detail-more-actions/smoke-round4.log)及[早期图面归档](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/012-detail-more-actions/earlier-rounds.zip)；原始非敏感输出位于.local/verification/012-detail-more-actions。凭据和浏览器持久profile不进入仓库。

运行方法见[017](../guides/017-local-bu-development.md)，独立入口为http://127.0.0.1:4382，在线状态以verify为准。基线整合已经本地提交，本批UI/文档增量尚未提交；发布或main同步须另有用户指令。
