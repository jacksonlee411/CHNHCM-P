# CHNHCM-140 地点与法律实体删除回执验证及计划交接

## 1 受测问题与基线

2026-11-27，公共仓固定工作树 `wt-dev-b@1284f50e32c16d7191d6ea8f375c2e4b67aa7941`。地点和法律实体各建一份独立资料，四条记录的生效日依次为 1900、2000、2010、2020 年的 1 月 1 日。管理员删除第三条。独立预期是详情回到从 2000 年起接管该期间的第二条，办理回执指向 2010 年的被删原件，2020 年的独立后继保留。

原分支上，两个对象的删除响应和原提交号查回均返回第一条的选中身份、修订及 1900 年的日期。被删目标的记录身份正确，第四条修订也保持。基线实库复现耗时 208.10 秒，两个对象均失败。这一处同时涉及两个对象真实写入和原号查回，触发 140/T04。相同输入在修复候选上耗时 218.80 秒，两个对象均通过。[原计划与验收条件](../archive/plans/140-module-cohesion-roadmap.md) · [永久实库回归](../../internal/bootstrap/record_history140_integration_test.go)

## 2 业务结果与历史保护

| 条件 | 操作及实测结果 |
| --- | --- |
| A06 准确继承与独立后继 | 每个对象在第四条后停用并恢复，再更正第四条。停用继承该前序的资料、记录身份及新修订，独立恢复记录的资料与修订相同。随后删除第三条，第二条扩至 2019-12-31 并获新修订，第一、第四、停用和恢复四条逐项相等。八次成功办理形成连续的八份来源、八条操作和八个不同的对象事件。 |
| A06 删除原件 | 首次响应、同号同内容重送及原号查回均指向第三条准确的删除前修订和 2010-01-01。删除前来源可读回原记录，办理来源可按被删身份读回只读前像。独立测试另验岗位占用的地点期间受到保护、登记国家改名后法律实体原采用的准确名称和当前权限。 |
| A07 无写入及拒绝 | 查询者发起删除返回 403，独立连接检查业务头、历史中的本次提交及独立终态。无变化更正只留一份 `no_change`，删除首条只留一份 `rejected`。两种情况业务头均维持在 7。 |
| A07 提交与恢复 | 在真实事务内于提交前制造失败，先用独立连接确认零事实、零终态，再用原号查回得到 `not_applied`，重送返回 `M02_REQUEST_CLOSED`。另一份提交在数据库成功后丢失响应，查回前已可见一份事实与来源，同号重送仍为原结果。重启后能按原号查回。 |
| A07 同号及当前权限 | 同一管理员用同一个提交号分别删除地点和法律实体，各自成功并查回自己的来源。原号异内容返回 409 `M02_REQUEST_MISMATCH`。撤生效日读权后查回省略日期，撤历史读权后返回 403。 |

对象命令各自负责权限、原采用、业务约束和提交。Go 共用受影响记录的修订、停用继承以及回执所指的目标与选中记录。历史存储格式继续使用两类对象原有的修订类型。[共用实现](../../internal/contexts/organization/application/recordhistory/records.go) · [地点故障恢复](../../internal/bootstrap/location204_acceptance_integration_test.go) · [法律实体故障恢复](../../internal/bootstrap/legal_entity206_integration_test.go) · [岗位引用](../../internal/bootstrap/staffing312_position_integration_test.go) · [登记国家采用](../../internal/bootstrap/legal_entity206_acceptance_integration_test.go)

## 3 规则与归档交接

| 原计划 | 现行责任与复核入口 |
| --- | --- |
| T01 / A01、A02 | X03 持有策略版本及守卫，Organization 经公开端口翻页。策略变化、失权、并发和守卫故障沿原验收用例追踪。跨域 SQL 正反例由架构边界检查承接。[201 候选合同](../specs/m02/201-country-region-geo-area-and-location-maintenance-contract.md) · [013](../specs/m02/013-m02-technical-design.md) · [004](../architecture/004-repository-layout.md) · [策略版本端口](../../internal/contexts/organization/application/geoarea/candidates.go) |
| T02 / A03、A04 | Go 提供最后适用日，详情、单条删除、首条改日缺口和成对删除共用显示语义。受限日期和准确采用标签由对象读取时裁剪。[026](../adr/026-date-display-and-input-format.md) · [012 历史交互](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-history) · [013](../specs/m02/013-m02-technical-design.md) · [阶段反例](140-module-cohesion-review-fixes.md) |
| T03 / A05 | 模块和已实施切片按各自入口和验收登记，架构维护者随真实能力更新登记。原批三个登记反例保留当时的事实，后续 M03、M04 进展按各自批次更新。[上下文登记](../architecture/contexts.json) · [004](../architecture/004-repository-layout.md) · [T01–T03 分项验收](140-module-cohesion-implementation.md) |
| T04 / A06、A07 | Organization 维护两对象共同的修订及回执规则，地点与法律实体各自维护地址、法律身份、准确采用与访问权限。后续历史修改从两个对象的独立回归进入。[120 共用约定](../specs/m02/120-dated-maintenance-technical-contract.md#location-legalentity-history-receipt) · [201](../specs/m02/201-country-region-geo-area-and-location-maintenance-contract.md) · [207](../specs/m02/207-legal-entity-design.md) |

## 4 管理员、查询者与保留环境

在本树专用的 `m02-manual-public-b`，Web 4490、API 8490、DB 54490，使用全新合成资料。地点和法律实体各建 2020、2021、2022、2023 四条，在 1440×540 打开 2022 年记录的删除确认，看到“2022-01-01 至 2022-12-31”。取消后删除请求数为零，来源头相同。管理员在 Chromium 原生 200%（1280×900 的窗口）用键盘确认，日期菜单显示 2021-01-01，2023 年独立后继仍可选。查询者在 1440×900 从设置 → 组织配置进入相应列表，按编码查找，在变更日志中点“查看删除前原件”，可读到 2022 年的准确记录、来源和原名称。[浏览器场景](../../tests/e2e/delete-history-support.ts)

用户可从 [4490 验收入口](http://127.0.0.1:4490) 搜索地点 `L204-1795770919999`、法律实体 `LE206-1795770899666` 复查上述日志。该环境由公共仓 wt-dev-b 保留至用户验收结束或下次本树更新。4390 属于本机原私有仓的另一棵工作树，未操作其服务或数据。

在旧程序中另外建立 `T04-OLD-LOC` 和 `T04-OLD-LE` 并实际删除第三条，两者使用同一个提交号。升级前在仓外私有目录保存数据库备份、目录清单及原构建。升级后从 4490 以同一管理员取回下列三类业务投影，对按键排序的 JSON 计算 SHA-256。六次比较全部一致，包括已有回执原先错误的选中身份和日期。

| 样本 | 删除前来源的选中记录 | 删除来源的只读前像 | 原号查回 |
| --- | --- | --- | --- |
| T04-OLD-LOC | `1413f91a9a3100587a5698a9f23a8e75e77cf4a1f809e2d424531b1e3fa3d13e` | `1cad3141df928f7332eecdd980cfa5fcc383caf1409616635f852b8c19c28263` | `89e4d18c75b1a54b610294bf1736b61cb4acdb7d0da1a7ecb42aa09cd8230a7f` |
| T04-OLD-LE | `5fcf8d5d7a263bcaf99cf79a906e992fb721f54354a5fbdebfa61dbd67bdd48d` | `6e3119ef5fffc9a1c219e1adb92826e945410f4d9a59c8134b29518b3530d275` | `29739fbea944920c45aa4cf9a84562eb84e7bc829b38f1b1076109c8cb153a17` |

## 5 首次失败、评审和验证范围

回归用例首轮从详情顶层取选中身份而失败。读合同并改为 `selectedRecord.recordId` 后，两类对象通过。浏览器首轮在原生 200% 下等待桌面日期按钮超时。按现行紧凑视口的“选择生效日期”菜单复验，保留原期间及选中身份断言。基线复现的错误回执和这两项取证脚本问题分别登记。

本次由 Codex 主代理沿两条对象命令、按日计算、提交与恢复、存储读回及共享页面复评。确认共用实现只接收已通过领域校验的候选，固定首条受保护并且始终最早。单条删除后可按原日定位接管记录。成对操作继续走各自的入口。新增反例还在完成删除后把接管的第二条另行删除，原号仍返回首次删除的准确来源和选中身份，提示当前资料已变。旧回执从已存操作读取，未执行历史回填。[按日校验](../../internal/contexts/organization/domain/dated/candidate.go) · [共用组装](../../internal/contexts/organization/application/recordhistory/records.go) · [原号恢复](../../internal/contexts/organization/application/submission/recovery.go)

临时环境 `smoke-140-receipts-b` 的生产构建运行地点和法律实体两个 spec，各覆盖单条删除回到适用记录、丢响应恢复、完整启停时间线、成对插入维护及删除原来源，8/8 通过。单独核对已回收该 Compose 项目的容器、网络、数据卷和临时配置。固定入口另以同样的双角色路径复验两个对象，交付构建 2/2 通过。容量及全量套件、远端 CI、发布和人工验收不在本次本地签收范围。

## 6 候选与复核入口

- 本地保留环境原 API 的 SHA-256 为 `35b050e60f91b606769829e78b52809689c3539920113664ecce1e2fce0fedaf`，交付 API 为 `f4777766fb38537548608b8267ac726f68d8d25e32c8d5b551b9b567d5327cdd`。生产 Web 文件逐个摘要相同。按本树固定构建清单比对 1,971 个源码及构建输入，交付时逐项相符。基线与交付之间的三个 Go 产品文件为两条对象命令和共用修订组装，公开结果描述与生成的客户端同步更新。
- `make check` 通过，含 189 项 Python 测试、文档及链接检查、796 个 Go 文件的导入边界、162 份静态 SQL 来源的归属、32 项运行脚本测试、Go 单测与 vet、前端类型及差异空白检查。`corepack pnpm generate:api` 在两个结果类型生成 8 条字段说明。归档前的 8 个标题、60 处原相对链接及其片段逐项复核，归档登记保留编号、原路径和原方案采纳记录。
- 最终候选按 Makefile 的真实 DB/API 入口执行下列定向命令。`internal/bootstrap` 的 7 组全部通过（含子用例共 23 个通过项，557.004 秒），其中地点完整验收的 10 个子用例均运行。`tests/integration` 包没有匹配该筛选的用例。专用 PostgreSQL 容器在各组结束后回收。

  ```sh
  GOFLAGS='-run=^(TestHistory140DeleteAndRecover|TestLegalEntity206Recovery|TestLegalEntity206ExactCountry|TestLegalEntity206PairsAndPermissions|TestLegalEntity206Core|TestStaffing312Position|TestLocation204Acceptance)$ -v' make integration
  ```

- 浏览器临时环境使用 `python3 -B scripts/local-bu.py smoke --profile smoke-140-receipts-b --fixed-web`，限定项目 `chromium --no-deps` 和 `location204.spec.ts`、`legal-entity206.spec.ts` 中上述 8 项。固定环境通过 `CHNHCM_LOCAL_PROFILE=m02-manual-public-b` 的同仓参数启动、验证所有权及复验业务流程。合成环境的备份由本树保留在仓外。

另行执行 `go vet -tags=integration ./internal/bootstrap`，15 个既有集成测试文件各有一处复制含锁存储结构的告警，命令返回 1。以 Git 基线替换本次改动再次运行，完整诊断逐字相同（日志 SHA-256 `6e8d1cbac96159ff53fc61d90c1a27d714ab08ec1860d4a68bac4a9574e3b659`）。本次修改的文件无该诊断，Organization 应用包带相同标签的 vet 通过，项目规定的 `make check` 通过。
