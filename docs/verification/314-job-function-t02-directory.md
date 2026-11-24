# CHNHCM-314 T02 职类目录实施与评审

## 1 交付与基线

2026-11-20，T02 在固定工作树 `wt-dev-d` 通过本地验收。管理员从「设置 → 职务配置 → 职类」建立“管理类”和“专业类”，维护生效记录；查询者可查、可导出获权的六列资料。[对象合同与 AC082、AC092](../specs/m03/308-job-foundation-contract.md#m03-job-function) · [执行清单与结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/314-job-function-t02/results.json)

起点为 T01 的 `9e409c5d`，工作区干净、stash 为空。写 SQL 前抓取远端并核对七个分支、开放 PR（0）及本机三个工作树；七个远端分支均指向 `94e9aefb`，本机没有在途迁移。该版已使用 00023 调整澳门地理区域，因此先合入 main；三处文档索引及登记冲突合并了双方条目。以下实测基于合入后的 `41a2fe9213d9bd946228467b4f9afd4a44242ce7` 及本批工作区。[T01 留样](314-job-function-t01-baseline.md) · [迁移登记](../../db/migrations/README.md)

## 2 实现与迁移评审

Staffing 负责 JobFunction 的身份、企业内编码、按日事实与独立修订。编码归一为大写、同企业同类型唯一，建立后不可改；名称必填，英文名称和说明可选。Go 负责首建默认日、企业下界、状态转换和字段变更判定。X03 独立判定 `jobfunction.*` 动作、企业范围及字段授权。菜单、编辑器、列表与提交查回沿用职务系列的共用实现。[领域实现](../../internal/contexts/staffing/domain/timeline.go) · [OpenAPI](../../api/openapi.yaml)

00024 追加职类权威头、不可变来源与修订、Outbox、终态、按日投影和固定选择，日期区间为左闭右开。运行时要求完整账本 00001—00024 和 `staffing-job-function-1`。核验时使用 PostgreSQL 18.6、独立临时库及与迁移匹配的 Go 程序。[迁移 SQL](../../db/migrations/00024_job_function.sql) · [复验入口](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/314-job-function-t02/upgrade.py)

| 路径 | 实际结果 |
| --- | --- |
| 空库安装 | 从空模板安装全部 24 号迁移，再执行一次。初始化后职类列表为空，从真实 API 建立第一份职类成功。 |
| T01 原备份升级 | 核对原 `baseline-23.dump` 的 SHA-256，再还原至新库。升级前、升级并重试后、建立首份职类后，逐行核对的 44 行旧事实均等于 T01 清单。原提交查回与原样重送给出原终态，同号异载荷被拒；前后两条有效记录、已删除记录的准确来源、四次变更日志和岗位原采用均可核对。 |

在已登记的临时 profile 中，将 `T01_HANDOFF` 指向 T01 报告第 4 节原交接目录，`T02_BIN` 指向本候选的 server、upgrader 所在目录，可重复执行两条路径。每次均新建分库。

```sh
python3 docs/verification/data/314-job-function-t02/upgrade.py --handoff "$T01_HANDOFF" --bin "$T02_BIN"
```

## 3 AC082 目录、权限与页面

| 核对项 | 实际结果 |
| --- | --- |
| 正常菜单与输入 | 管理员先建“管理类”，再以同码建立“专业类”得到冲突，焦点落在编码，名称仍在。改码后建立成功。该编码可同时用于同企业职务系列。首建默认 1900-01-01，也可明确指定其他合法日期。 |
| 按日维护与取消 | 测试将业务日从 2027-04-01 推进至 2027-04-02，同一编码先显示“管理类”，到更名日显示“管理职类”。覆盖记录新增、插入、更正、改日、删除非首条、末端启停、固定期间插入／维护／删除。删除固定期间后仍可读取恢复端的原修订。桌面和窄屏从日志进入更正、修改后放弃，返回原日志，来源头与操作数保持。 |
| 提交、权限与导出 | 同号重送及查回返回同一职类、同一来源。查询者菜单搜索并下载两类，逐行核对编码、名称、英文名称、说明、状态和生效日期；直接 POST 新建返回 403。撤去名称读权后详情裁剪，原选择再次导出被拒。撤去更正名称的写权后直调被拒，来源与 Outbox 行数保持。 |
| 与职务系列页面对照 | 经两个正常菜单实测新建表单的资料分栏位置、具名字段高度和字号、名称标签字号。双角色分别运行五组普通视口与 200%／400% 原生缩放，包含键盘、滚动、日期轴、菜单可达性和返回上下文。1280×900 基准在两档缩放时分别得到 640×450、320×225。 |

浏览器套件 22 项通过（8.1 分钟）。加严的 AC082 从真实下载流读取两行 CSV，并对照职务系列表单，再单独复验 1 项通过。[AC082 菜单与下载](../../tests/e2e/staffing314.spec.ts) · [普通视口与按日流程](../../tests/e2e/staffing-page-suite.ts) · [原生缩放](../../tests/e2e/staffing314-zoom.spec.ts) · [桌面实景](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/314-job-function-t02/admin-desktop.png) · [窄屏实景](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/314-job-function-t02/reader-narrow.png)

## 4 AC092 企业起日

企业拟后移起日时，Organization 在原事务通过 Staffing 公开端口读取本企业所有职类的最早首条。2001-01-01 首建、2003-01-01 已停用且始终无人引用的职类，实际阻止企业将起日移至 2005-01-01（409）。在本可行的 2000-01-01 后移上分别注入端口读取错误和端口缺失，均返回 503，企业起日、来源头及终态数量保持。[企业反向校验](../../internal/contexts/organization/infra/postgres/enterprise_references.go) · [双连接及故障断言](../../internal/bootstrap/staffing314_integration_test.go)

| 两个 PostgreSQL 连接的写入 | 职类先取得守卫 | 企业先取得守卫 |
| --- | --- | --- |
| 职类首建于 2001-01-01，与企业起日移至 2005-01-01 | 职类提交，企业后移拒绝 | 企业提交，职类首建拒绝 |
| 原首条从 2010-01-01 前移至 2001-01-01，与同一企业后移竞争 | 职类改日提交，企业后移拒绝 | 企业提交，职类前移拒绝 |

四个用例均等待 PostgreSQL 报告真实锁等待后才放行持锁连接。[竞争控制与断言](../../internal/bootstrap/staffing307_races_integration_test.go)

## 5 分层验证、首次失败与交接

生成 SQL 客户端和 OpenAPI 类型、构建、`make check`、最终差异检查通过。定向 `make integration` 覆盖本批 AC、既有 Staffing 日期与选择、原子提交、迁移与运行准入，另单跑固定期间原件回归。全部匹配的 DB/API 测试位于 `internal/bootstrap`，另一集成包本轮过滤后为零。[范围与实测摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/314-job-function-t02/results.json)

临时环境第一次核对时地理目录还在初始化，API 归属核验未通过，待就绪后复验通过。主机缺少原生缩放所需 X11 工具，临时取得依赖时一次下载返回 502，首次挂载也未能启动 Xvfb；重试下载、修正临时 X11 启动路径后，预检及双角色缩放通过。评审中收紧了重复编码的错误归属和 ASCII 编码归一化，并以字段焦点、同码保留输入及领域用例复验。

本批临时 profile 为 `t02-314-catalog`（Web 55614、API 55615、DB 55616），验后已回收。既有用户验收环境仍由原工作树按原约定保留，T01 仓外原备份留待 T03／T04。本次为本地 T02 签收；下一批 T03 实施职务双维写入与存量升级，九列职务导出在 T04 验收。远端 CI 随后续发布流程执行。[实施路线图](../plans/314-m03-job-function-roadmap.md)

## 6 分支默认端口接续

2026-11-20 按用户要求，`wt-dev-d` 默认持久环境 `primary303-current-d` 已运行 T02 提交 `0376b95b`，Web 45300、API 45301、数据库 48275。通过实际登录和「设置 → 职务配置 → 职类」验证两种角色。管理员可打开并取消新建表单，默认日为 1900-01-01；查询者可查看列表。此次更新后职类目录为零条，供用户自行建类。该环境持续运行供手工验收，责任为 wt-dev-d，用户验收结束或下次更新时复查。[登录入口](http://127.0.0.1:45300/) · [职类目录](http://127.0.0.1:45300/settings/staffing/job-functions)

更新前将 00014 原库、数据库角色、原固定构建及配置备份到仓外私有目录 `~/.local/state/chnhcm-evidence/314-default-45300-YfxJGbs8/`，并完整读取 104377126 字节的备份校验。upgrader 在原数据卷追加至 00024，Compose 按当前配置重建数据库容器。升级后核对九张原业务表的逐行内容摘要，全部一致（含 87346 个部门和 87346 份部门历史）；原账号数量仍为两份。运行归属、实际 API 二进制及全部六个 HTTP 前端文件均已核对。
