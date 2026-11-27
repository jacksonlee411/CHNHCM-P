# CHNHCM-140 授权、日期与实施登记的分项验证

## 1 候选与范围

2026-11-22，T01（A01、A02）、T02（A03、A04）和 T03（A05）分别通过本地验收。固定工作树 wt-dev-b 从 `2a2317b4669ca3a0517185d86460f829f979efcb` 的干净状态开始，受测候选包含本批工作区变更。140 保持进行中，T04 待下一次 M02 历史流程变更触及地点与法律实体的共同规则。[140 路线图](../archive/plans/140-module-cohesion-roadmap.md) · [候选摘要、命令与运行结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/140-module-cohesion-implementation/results.txt)

## 2 T01 行政区划授权和翻页

X03 的 `LockPolicyShared` 在当前事务取得主体的 `policy_no`。同一个 guard 提供判权、授权集合及只读版本。M02 继续使用原游标签名、绑定与短事务，删除重复 SQL。管理员从设置的组织配置进入地点，新建地址，选择中国并搜索吉林省。[X03 守卫](../../internal/contexts/access/infra/postgres/access.go) · [候选服务](../../internal/contexts/organization/application/geoarea/candidates.go) · [地点页用例](../../tests/e2e/location204.spec.ts)

| 验收 | 观察 |
| --- | --- |
| A01 稳定权限 | 独立预期取自仓内固定区划包中中国代码以 22 起始的 70 项。返回 50＋20，合并后的编码按顺序逐一相等，末页无后续，页面选项与两页路径逐项相等。 |
| A01 两种策略变化 | 分别撤销非必需的性质读权，以及撤销后恢复到原授权集合。两次旧游标均得 409 `M02_CURSOR_CHANGED`。拦住自动重查的网络发送时页面旧选项为零；放行后按原条件无游标取得 50 项，性质字段分别按新授权省略或保留。 |
| A02 必要权限与锁 | 撤销 `geoarea.read` 后续页返回 403 `ACCESS_DENIED`，页面清空。两条并发连接分别持有实际候选事务的 X03 读锁、更新同一主体的策略行；观察连接用 `pg_blocking_pids` 确认写方进入等待。读方结束后写方提交，新 guard 读到原版本加一。旧 HTTP 游标返回 409。 |
| A02 会话与守卫故障 | 撤回测试数据库角色对守卫表的读取权，真实 HTTP 返回 503 `M02_DEPENDENCY_UNAVAILABLE`，数据库报告 SQLSTATE 42501。恢复表权限后使会话失效，返回 401 `AUTH_REQUIRED`。两种错误均无候选和游标。 |

撤权后取消新建并放弃草稿，页面返回地点列表。浏览器发出的新建地点请求数为零，列表总数与新建前相同。

`make boundaries` 覆盖三个域的 sqlc 源、infra 中 Go 字符串与嵌入 SQL。以基线的完整 M02 文件运行检查器，在原第 383 行报 `organization SQL references access.policy_guard` 并退出 1。本域查询、Go 类型引用和 SQL 注释退出 0。限定表名换行、常量拼接和嵌入 SQL 的跨域 JOIN 均被发现。动态表名另查执行点，Staffing 的 `table(k)` 枚举限定 staffing 对象，Organization 的历史查询固定 organization schema，由 kind 选对象前缀。静态结果只覆盖可识别的显式关系引用。[检查器](../../scripts/check-boundaries/sql.go) · [正反例](../../scripts/check-boundaries/main_test.go) · [锁与故障实测](../../internal/bootstrap/location_region_picker_integration_test.go)

## 3 T02 组织记录期间

| 验收入口 | 观察 |
| --- | --- |
| A03 详情与单条删除 | 管理员建 2026-01-01、06-01、10-01 三条 BU 记录。六月记录的详情和删除确认均为“2026-06-01 至 2026-09-30”，末条为“自 2026-10-01 起”。取消后头来源相同；另行确认后按删除前来源读回原记录。 |
| A04 首条后移 | 企业单例在下游建数前单独验收，随后还原首条供其他样本使用。BU 与企业分别从 2024-02-28 拟改到 03-01，检查预览的排他终点为 03-01、最后适用日为 02-29、页面显示“2024-02-28 至 2024-02-29”，取消后来源相同。 |
| A04 成对删除 | BU 的恢复资料变化到 2026-07-31，后继 08-01 独立。企业单独设有限段到 1899-12-31。部门、地点和法律实体分别更正恢复端名称，使用开放段。逐对象检查当前 API 与实际弹窗；没有资料变化时返回空集。企业和部门撤日期权后只保留名称变化；地点和法律实体失去所需日期权后撤去成对动作。 |

Go 仅在读取投影提供 `endInclusive`。BU 与企业的首日空档使用独立的预览类型。其他四种对象按自己的权限输出成对删除的区间；地点、法律实体还按可读业务资料裁剪变化。日期格式化和两种显式期间由共享 UI 负责。已存回执、影响区间、历史编码及迁移沿用原格式。数据库测试比较投影前后的业务表快照、按原号查回终态、按原来源核对修订。[技术合同](../specs/m02/013-m02-technical-design.md) · [成对删除与查回](../../tests/integration/lifecycle_pair_test.go) · [各对象页面核对](../../tests/e2e/dated-object-support.ts)

## 4 T03 实施登记

从组合根核对三个服务域，从正式菜单或会话端口反查七个 Organization、六个 Staffing 以及一个 Access 已验证切片。上下文登记保留原有 17 域和模块分配。每个已验证切片给出所属模块、限定范围、入口文本与源码、域内源码和原验收报告。原 G0 和首切登记保留阶段含义。[组合根](../../internal/bootstrap/runtime.go) · [切片登记](../architecture/contexts.json) · [目录说明](../architecture/004-repository-layout.md)

| A05 反例 | 裁定依据 |
| --- | --- |
| 将 Staffing 的进展外推给 M04 | 必须逐模块给出状态，删去 M04 单列项会使结构检查失败。M04 人事动作当前只有提案。 |
| 将 308 独验外推给 313 | 核编没有运行入口与域内实现，改标运行切片时检查失败。回读 314 签收正文，AC090 的“仅 313”和“两批并存”仍均未运行。[原分支签收](314-job-function-t05-acceptance.md) |
| 删除已验证切片的证据 | 检查要求已有入口、归属本域的源码及正式登记的验证报告。缺证据拒绝。证据正文另按对象逐项复核，法律实体只登记 L1，地点的容量遗留仍由原批次跟踪。[法律实体原报告](206-legal-entity-implementation.md) · [地理与地点原报告](205-g5-implementation.md) |

## 5 首次失败与执行范围

地点首轮已完成候选及授权断言，在新建页寻找“取消编辑”超时；该页实际入口是“取消”链接。第二轮返回列表保留 `?page=1`，只接受裸 URL 的断言失败。BU 的 390 像素预览取消后，桌面日期轴脚本在紧凑布局等待“2024-02-28”按钮超时。修订用例按业务列表路径、对象数和写请求核对取消，在窄屏核对空档后切回桌面选择后续记录。

## 6 执行结果与资源

| 检查 | 本地结果 |
| --- | --- |
| 结构与构建 | `make check`、`make build`、生成客户端及仓内引用检查通过。结构检查含 131 个脚本测试、32 个 Node 测试、629 个 Go 文件导入检查和 103 个 SQL 源或适配器文件归属检查。 |
| PostgreSQL 与 Go API | 两轮定向 `make integration` 涵盖 9 个不同测试函数，两个包均通过。末轮复跑行政区划候选与 BU 成对删除的原件比较。 |
| 实际页面 | 在临时环境 `smoke-140-signoff` 使用生产前端构建，16 项全部通过。地点布局覆盖两个角色各七组视口，含 320×500 短屏。另验 Chromium 原生 200% 与 400% 缩放的键盘操作。 |
| 清理 | 三轮临时浏览器环境均已回收。按各自 Compose 项目标记回查，容器、网络与卷均为零。 |

本批未运行仓库全量集成、全量浏览器套件、远端 CI 或人工产品验收。T04 的 A06、A07 留给触发该项的历史写入变更。[定向命令与最终结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/140-module-cohesion-implementation/results.txt)
