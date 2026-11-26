# CHNHCM-002 继承 CubeBox 技术方向

登记信息：[registry.json](../governance/registry.json)。适用范围：本仓技术职责与交付方向；开发锁定位置见第 2 节，生产支持范围见第 6 节。来源为 CubeBox `2deaf9276086f68e07aaa0c2cf7f35025046a9ee` 的 AGENTS、package.json 及 `docs/dev-plans/tooling/216-database-technology-selection.md`，原字节保存在[离线来源包](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/reference/sources/cubebox-m02-2deaf927.zip)，摘要见[基线](../reference/baselines.json)。

## 1 背景与问题

本仓采用全新 Go 权威领域内核，保留已有体验、业务与验证资产，目标覆盖客户独立实例和离线交付。需要明确页面、业务、存储、迁移与厂商版本平台的职责，并使工程骨架与后续真实业务切片衔接。

本决策继承来源中已记录的技术方向及数据库取舍，不把来源原型当作已接通的正式后台。本节之后的比较是固定来源决策的本仓适用说明，未对各工具最新版本重新评测，也不补造一次新的选型历史。

## 2 决策与职责

| 层 | 选择 | 当前落点 |
| --- | --- | --- |
| 页面 | React 19.2.4、TypeScript 5.9.3、MUI 7.3.7、Vite 7.3.1 | 前端负责页面交互、输入辅助和 API 调用 |
| 工作区 | Node 24.19.0、pnpm 10.24.0 | 版本与锁文件可复现 |
| 内核 | Go、模块化单体、标准库 HTTP 起步 | go.mod 锁定 Go 1.26.0。Go 处理业务命令、授权和事务 |
| 持久化 | PostgreSQL、pgx/v5、sqlc、goose | 开发库使用 PostgreSQL 18.6-alpine 固定镜像，pgx v5.11.0、sqlc v1.31.1、goose v3.29.0。查询按域管理，由 upgrader 统一迁移 |
| 客户端与会话 | OpenAPI、openapi-typescript 7.13.0、openapi-fetch 0.17.0、X03 本地会话 | 从 api/openapi.yaml 生成 TypeScript 类型，前端通过客户端发起请求。服务端校验会话、写请求及业务权限 |
| 交付 | 客户独立实例、Docker Compose、本地 upgrader、轻量厂商版本平台 | 本地 Compose 和 upgrader 已用于开发与验证，厂商平台与生产交付范围按阶段确定 |
| 备份 | pgBackRest | 恢复目标、存储和演练方案待明确 |
| 验证 | Go testing、真实 PostgreSQL/Testcontainers、Playwright Chromium | 历史建仓证据见[建仓验证](../verification/bootstrap.md)。首个业务切片的真实 DB/API 与浏览器证据见[首批记录](../verification/010-first-vertical-slice.md) |

主库采用精确金额及 daterange [start,end)；业务不变量由领域维护。sqlc 只生成数据访问类型，不向领域/API 泄露存储模型。特殊动态查询采用参数化 pgx，不默认引入 ORM。目录、唯一结构来源与迁移调度由 [004](../architecture/004-repository-layout.md) 持有。

三种交付方式为厂商托管独立实例、客户环境部署与厂商版本管理、完全离线实例。客户员工/工资事实与厂商版本平台分离；离线运行不依赖厂商实时服务。已发布迁移不可修改；结构迁移、数据库引擎升级、业务规则版本分别管理。

## 3 来源方案比较与取舍

以下仅整理源 216 已有候选及项目取舍，本仓开发组合及其验证入口见 013 第 7.1 节。

| 来源候选 | 本仓采用方式与理由 | 接受的代价或重新评估条件 |
| --- | --- | --- |
| sqlc＋pgx | 核心固定查询的主方案；查询及更新范围直接受审查，减少参数和结果类型的手工同步 | 维护 SQL 与生成流程；复杂类型、动态查询及真实数据库行为仍须测试 |
| GORM | 保留比较记录，不作为默认访问层；当前优先明确 SQL、历史条件及事务边界 | 若引入，需评估关联/钩子行为及与原生 SQL 的维护边界 |
| Ent | 保留来源中的主要备选 | 若关系遍历或动态组合成为主要维护成本，再以实际证据评估建模和迁移影响 |
| sqlx | 保留原生 SQL 映射方案的比较记录 | 当前选择 sqlc 减少结构、查询和结果类型的手工同步；仍需验证业务语义 |
| 直接使用 pgx | 特殊查询或批量操作的局部补充 | 手工参数/结果映射需要明确维护责任，不能绕过权限、事务与验证 |

按日有效期、已采用修订、金额及历史保护需要可审查的取数和写入条件；本仓接受多维护显式 SQL 的成本。选择数据访问工具不证明性能、事务或计算正确，结论须来自实际查询与测试。

## 4 影响与验证边界

业务层、存储模型和公开 API 分别维护，通过组合根与端口接线；前端不复制第二套金额、日期、状态或权限引擎。来源资产需按 [006](../assets/006-preservation-contract.md) 重新接线并验证。

CubeBox 来源没有锁定正式 Go/PostgreSQL/sqlc/goose/pgx 的完整版本组合。Go 1.26.0 是建仓可用工具链，不能据此宣称生产支持。建仓时未引入 Go 外部依赖；现创建/查询批次已落 go.mod/go.sum、sqlc 配置、真实查询与生成代码，锁定和运行范围见[017](../guides/017-local-bu-development.md)及[实施验证](../verification/010-first-vertical-slice.md)。前端生成命令和运行依赖分别登记于[工作区配置](../../package.json)及[客户端配置](../../packages/api-client/package.json)。镜像摘要见[开发镜像锁](../../deploy/local/postgres-image.txt)。

生产组合的核对至少应覆盖精确金额和日期区间映射、实际查询、权限拒绝、事务回滚/并发，以及本次范围内的安装或升级路径。是否包含部署、备份恢复与离线演练由选定交付范围确定；未实施项保持可见，不能用构建通过代替。

## 5 适用与维护位置

2026-11-03，BU列表按[113](113-bu-list-community-adoption.md)采用Community DataGrid8.29.3原字节TS/TSX与响应式摘要列，保留110剩余交互、原生分页及Go权威。T01/T02源码/依赖/页面已接线，定向检查见[111报告](../verification/111-community-t01-t02.md)；T03真实业务与七组条件另见[本轮报告](../verification/111-community-t03.md)。同版发布声明仅供TypeScript，运行入口仍原始源码，不引入发布JavaScript；本仓自足构建和显式来源核对保持。

本决策持有技术选择与取舍；004 持有仓库依赖及迁移归属，切片的具体模型、接口、配置和验证由 013 承接，实施阶段按 [003](003-implementation-stages.md)。013 的正文入口见[规格入口](../specs/README.md)。本次没有反转技术选择或替代其他 ADR。

Go module 使用 `chnhcm.local/chnhcm`，仓库现已托管在 GitHub。固定分支和 PR 集成方式见[分支协作规范](../governance/005-branches-and-documents.md)。

## 6 开发接入与生产交付范围

工具或身份机制首次接入、替换时，在同一变更中更新锁定位置、实际调用方和本表状态。生产支持范围按客户交付及演练证据另行确认。

| 事项 | 当前状态 | 负责角色与后续条件 |
| --- | --- | --- |
| 开发工具、数据库与验证 | Go、pgx、sqlc、goose、PostgreSQL 镜像与 Testcontainers 已锁定并接入 | 技术设计维护者随实际升级更新锁文件、生成产物及实库验证。[开发组合](../specs/m02/013-m02-technical-design.md#71-版本候选和锁定责任) |
| API 客户端与本地身份 | OpenAPI 为唯一 HTTP 合同，类型生成和运行客户端已接通。X03 校验本地会话、写请求 CSRF 及业务授权 | 合同维护者同步更新声明、客户端和服务端验证。外部身份源待具体交付立项。[本地身份合同](../specs/m02/013-m02-technical-design.md#61-首批真实身份) |
| 生产支持、备份恢复、离线与升级 | 待确定客户交付范围 | 交付维护者明确支持版本、恢复目标、维护窗口及离线条件，完成对应安装、升级和恢复演练后记录结论 |
