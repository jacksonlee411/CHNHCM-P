# CHNHCM-115 全查询结果选择与导出实施验证

登记信息：[registry.json](../governance/registry.json)的governance.verification_records。关联[114](../adr/114-bu-query-selection-and-export.md)、[115](../plans/115-bu-query-selection-and-export.md)、011—013及023。本报告承接用户后续实施指令，原[文档报告](115-query-selection-docs.md)保留未实施时点，不回写原结果。

## 1 基线与范围

固定wt-dev-b，HEAD为`0e3a0920875741263e7ae8f085e50deca82b5ceb`；进入实施时已有上一轮未提交文档。已按原输入摘要逐项验证并保存[原资料快照](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/115-query-selection-implementation/document-baseline.tar.gz)，防止文档阶段与实施阶段混记。当前差异包含这些文档及本轮Go/API/SQL、授权迁移、客户端、Community页面和验证输入，最终输入摘要与结果见第7节。

## 2 实施与环境

全量选择通过同源授权SQL生成完整集合依据、count及页/全量投影；只向浏览器返回集合摘要。逐项/全量减排除、跨页保持、独立返回高亮、工具条、当前授权重验及完整CSV已接线。使用原生Community行选择和公开表头适配，未修改第三方包。新00003仅扩充当前模型的bu.export授权动作；现有组织历史不改，不支持其他旧开发模型。

只读业务一致性采用单条SQL快照和当前事务内策略共享守卫，不提升会话TouchSession事务为REPEATABLE READ；校验/生成失败零CSV响应，网络中断则由浏览器完整接收后才创建下载。固定四类字段按当前读权裁剪，独立bu.export不扩大可读集合。

真实DB为仓内锁定PostgreSQL18.6/Testcontainers，浏览器使用本仓Playwright1.55.1/Chromium和自动回收临时profile。[环境记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/115-query-selection-implementation/environment.json)持有实际Go/Node、CPU、内存及镜像。真实缩放使用任务自有Xvfb及原生Ctrl+加号，固定外窗1280×987，对应200%为640×450 CSS px/DPR2、400%为320×225/DPR4；不以CSS zoom或viewport缩小冒充。无clip的CDP可视面截图与像素尺寸核验见[缩放记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/115-query-selection-implementation/zoom.json)及[200%](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/115-query-selection-implementation/zoom200.png)/[400%](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/115-query-selection-implementation/zoom400.png)。

## 3 已执行检查

| 检查与业务路径 | 实际结果 | 证据及边界 |
| --- | --- | --- |
| 最后make check | 通过：73项Python、6项运行工具检查、Go单元/vet、边界、名称表及TypeScript、diff检查 | [输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/115-query-selection-implementation/check-final.txt.gz)；不是整产品验收 |
| make build | 通过，Go/server/upgrader与本仓独立Web构建 | [输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/115-query-selection-implementation/build-final.txt.gz)；仍有既有大于500kB分块提示，不作为性能SLA |
| 完整make integration | 两包通过，tests/integration约483秒、bootstrap约315秒 | [完整输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/115-query-selection-implementation/integration-all.txt.gz)；后续选择错误边界及新增断言由下一行补验 |
| 选择真实DB/API及失败保护定向复验 | 3个选择集成测试通过，含字段裁剪、同数成员替换/来源变化、分权、恶意ID/模式/null/超正文预算、九类组织表快照零变化、在途取消后权限锁重新取得 | [定向输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/115-query-selection-implementation/integration-selection.txt.gz)；不把夹具准备或普通读取当作业务写入流程 |
| 从正常菜单开始的Chromium | 同轮18项全通过：10项既有列表/导航/权限/只读回归，8项选择/CSV/迟到/键盘/视口/真实缩放检查 | [最后18项](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/115-query-selection-implementation/browser-final18.txt.gz)；1440×900、1440×540、390×844、320×900、320×256及200%/400%均有实际操作，另含800px响应式分支 |
| 100条页大小补验 | 1项通过，137全选排1条，20→50→100及排序后仍136，实际CSV136行 | [补验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/115-query-selection-implementation/browser-page100.txt.gz)；后续仅增加测试参数，产品未改 |
| 2021BU/102页容量及内存 | 完整选择约78ms，CSV2021行约2.05s/90989字节；测试进程采样堆峰值6538144字节、基线2174816字节、累计分配增量101553608字节 | [采样](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/115-query-selection-implementation/capacity.json)及[原输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/115-query-selection-implementation/capacity-memory.txt.gz)；API与HTTP客户端在同一Go测试进程，PG内存不计；不是p95/生产SLA |
| 日常4382正常菜单 | admin全选并导出原有7条；reader全选7条且导出禁用；两角色无BU业务写请求；另验7条显式选满、排除后重选及表头状态 | [两角色](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/115-query-selection-implementation/daily-verification.json)、[边界](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/115-query-selection-implementation/daily-boundaries.json)及[同步日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/115-query-selection-implementation/daily-up.txt.gz) |
| 实现复核与资源 | 主代理复核通过，无剩余阻塞项；临时资源均回收，日常保留 | [复核](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/115-query-selection-implementation/implementation-review.json)、[资源](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/115-query-selection-implementation/resources.json)；非独立评审或人工签收 |
| 文档定稿与Git引用 | make docs的73项脚本检查、登记/格式/本地引用及Git引用检查通过，完整新增文本空白检查通过 | [文档](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/115-query-selection-implementation/docs-closeout.txt.gz)、[引用](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/115-query-selection-implementation/refs-closeout.txt.gz)、[空白](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/115-query-selection-implementation/diff-closeout.txt)；结构检查不代替业务验证 |

18个新增TL编号的主/补充层及实际参数见[案例映射](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/115-query-selection-implementation/case-results.json)。格式/非法协议/服务端取消反例由Go/DB/API承担，页面不开放非法入口；不能把一个编号的所有可能参数或全M02整体算为通过。最终资料登记和引用另由文档定稿检查覆盖。

## 4 发现、修复与复验

已保留[早期输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/115-query-selection-implementation/earlier-rounds)：授权动作约束尚未纳入bu.export；首轮夹具使用错误表名或不符合名称规则的公式文本；异步表头不能及时反映用户勾选；分页测试未等待新页；返回高亮原断言与独立勾选合同需区分；列显隐需保留选择列；行勾选默认控件令紧凑行增高。分别修复迁移/夹具、视觉反馈、前置等待、独立高亮与列过滤以及勾选单元格内边距，未放宽业务断言或删掉保护。

排序后旧数量可能在新响应前短暂匹配，用户紧接导出会被后续查询代际取消。已在查询动作开始同步暂停消费和隐藏旧结果，查询/选择未核对时禁导出；复验实际等待排序响应，并增加“查询中禁用”和“已生成CSV迟到不下载”用例。权限变化后同时清除选择并重读列表，受限字段不继续显示；失败反馈不误导查回不存在的写入标识。

真实缩放的失败属于启动及取证前置：WSLg具名Unix目录不能作为Xvfb准备判据，使用任务独立抽象X11传输、xdpyinfo有界探测；viewport=null明确移除默认deviceScaleFactor。Playwright默认截图在原生缩放下裁切可视面，改用既有CDP无clip助手并验证物理像素和掩码，未改变缩放本身。上述早期失败、混合输入期间观察与最后冻结输入的通过分开，不能覆盖原失败。

## 5 限制

本报告只覆盖115增量和受影响查询/导航/权限/历史保护，不声明完整M02、来源整项资产、真实消费者、Firefox/macOS/Windows原生Edge或生产验收。2021规模夹具每BU一个来源；未复跑T07的20并发、多来源大历史容量专项。完整集成后新增的选择错误处理和测试断言以定向集成补验，不伪称原完整运行包含后来才添加的断言。采样内存不等于无采样遗漏的峰值，也不包括PG进程；不把单次耗时当作p95。

## 6 结论与资源交接

115的T01—T05按本报告限定范围完成。全选覆盖完整当前授权查询结果，不是当前页；翻页/页大小/纯排序保留，实际范围或版本变化要求重选。admin/reader的正常菜单和实际导出分权已核对，原111/T07与资料报告不提升或改写，来源资产complete_asset_verified仍false。

task-selection-b至b10的自有临时profile，以及Testcontainers和任务Xvfb/浏览器均已回收；最后仅原有4个持久容器运行，其他工作树环境未操作。m02-current-b已执行同模型增量迁移和初始化后同步，原7条BU仍在，保留Web4382/API8382/DB54373供用户体验；由当前工作树实现/验证维护者负责，用户结束本轮使用或下次服务/版本交接时复查是否停机，持久卷默认保留。现场已verify并作实际查询，不能据入口配置承诺以后一直在线。

本轮无提交、push、PR、合并或部署动作，所有变更留在固定wt-dev-b工作区。后续超出本报告的能力另定合同及证据。

## 7 长期证据

本轮长期证据放在[data/115-query-selection-implementation](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/115-query-selection-implementation)，保留原资料快照、失败及最后日志、案例映射、资源核对和输入摘要。[最终输入摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/115-query-selection-implementation/input-files.sha256)与[实现及合同快照](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/115-query-selection-implementation/implementation-inputs.tar.gz)覆盖本轮变更中的代码、测试、合同、登记与采用信息；报告和证据自身不纳入递归摘要。日常环境最后[verify](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/115-query-selection-implementation/daily-verify-final.txt)通过。仅.local输出不作为唯一长期依据；不保存Cookie、CSRF、数据库连接密码或真实客户资料。
