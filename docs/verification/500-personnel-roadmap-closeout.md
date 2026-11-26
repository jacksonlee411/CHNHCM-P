# CHNHCM-500 人员模块实施路线图关闭与场景交接

## 1 关闭范围与依据

2026-11-26，在 `wt-dev-f@a54481991d7c2d0c3e4d81d113faf2d7512b37af` 上复核后，500 执行登记为 `completed`，保留方案的 `active` 采纳状态及原评审。归档范围是 T00—T06 约定的人员、任职、合同与录用交接。500 继续按原编号保存各批使用的条件和原锚点。[归档原文](../archive/plans/500-personnel-roadmap.md) · [现行场景导航](../governance/README.md#personnel-acceptance-scenarios)

511 盘点时，把合同和招聘场景的交接列为待 T05、T06 条件落实。本次对照原验收记录，确认当时遗漏了两份已交付的证据。工作关系合同合并于 `77095a96`，录用转入职合并于 `91950796`，两者均在本次基线历史中。原报告记录了实库、双方菜单和固定 4386 的验收。本记录更正该项在办判断，511 的原结论仍可查。[511 当时的判断](511-project-standards-implementation.md#4-两份归档试点与在办场景) · [合同与招聘的主线交付](500-job-information-delivery-review.md#stage-delivery-results)

| 计划批次 | 本次采信的交付依据 |
| --- | --- |
| T00、T01 | 首批规格通过评审。人员登记、查回、更正、历史及权限已有真实办理。防重依据取后续确认的主要证号，九栏资料另由 505 持有。[首批规格](500-t00-personnel-specifications.md) · [登记试办](500-personnel-registration-trial.md) · [九栏交接](504-personal-profile-fourth-batch.md) |
| T02、T03 | 用人主体资格及绑定在提交时重核。仅按职务、按岗免占、按岗占编分别验收，最后一编竞争和未来撤销有独立结果。[真实首单及 L3](400-first-hire-integration.md#first-job-only-hire) · [三种模式及实测](500-job-information-delivery-review.md#221-从合同查到业务证据) |
| T04 | 兼岗、调动、转正、经理、退出再聘及同日动作已有分阶段结果。508 的 6 项、509 的 27 项及 510 的 20 项页面整改按对应修复版本查看。[508 修复复验](508-job-information-implementation.md) · [509 原库复验](509-t04-remediation-implementation.md#fixed-environment) · [510 逐项实测](510-personnel-page-remediation-implementation.md#1-二十项问题的复验记录) |
| T05、T06 | 两段工作关系分别签约和处置。录用在改期后按重新接受的版本转入职，核实预留转换、未来撤销及重送。两项已有正常菜单与原库验收，第 4 节另记本次定向集成复验。[合同](500-job-information-delivery-review.md#18-阶段-10-的工作关系合同) · [录用转入职](500-job-information-delivery-review.md#19-阶段-10-的招聘录用转入职) |

<a id="scenario-handoffs"></a>

## 2 八个验收场景的原文与现行定义

按用户本次采纳的归属复核原场景。每行第一列回到 500 当时的样本，第二列定位今后修订及评审使用的同号条款。业务日期、作用的关系或任职、拒绝条件和独立预期随原场景承接。

| 500 原场景 | 现行定义与维护角色 | 核对的预期及既有证据 |
| --- | --- | --- |
| [M01-SCN-001 登记与编号](../archive/plans/500-personnel-roadmap.md#m01-scn-001) | [501 人员登记](../specs/m01/501-person-registration-requirements.md#m01-scn-001)，Personnel 与人事业务 | 同名不同证可分别建档，同证并发只建一人。原提交号查回原人，改号后旧号仍定位本人。[登记试办](500-personnel-registration-trial.md#3-人事试办) · [编号与恢复](500-t01-personnel-registration.md#2-十项业务条件) |
| [M01-SCN-002 个人资料与权限](../archive/plans/500-personnel-roadmap.md#m01-scn-002) | [501 人员权限](../specs/m01/501-person-registration-requirements.md#m01-scn-002)，Personnel 与 Access。九栏字段归 505 | 更正一项保留另一项，撤权后旧来源和回执也不披露联系方式。仅有部门权时查不到范围外人员。[资料与撤权](500-t01-personnel-registration.md#2-十项业务条件) · [变更日志实测](500-personnel-change-log.md#3-实施与验收) |
| [M04-SCN-001 直接与未来入职](../archive/plans/500-personnel-roadmap.md#m04-scn-001) | [401 首单与未来撤销](../specs/m04/401-direct-hire-requirements.md#m04-scn-001)，Staffing 与人事业务 | 生效前在职人数为零，最后一编只有胜者产生任职和占用。撤销后原日占用归零，原办理可查。[首单](400-first-hire-integration.md#first-job-only-hire) · [分模式结果](500-job-information-delivery-review.md#221-从合同查到业务证据) |
| [M04-SCN-002 兼岗与人员计数](../archive/plans/500-personnel-roadmap.md#m04-scn-002) | [506 兼岗计数](../specs/m04/506-job-information-records.md#m04-scn-002)，Staffing 与 Access | 三份任职计一人，部门、人员识别与任职授权共同限制行和人数。两岗按政策合计占一编或两编。[兼岗与并行关系](500-job-information-delivery-review.md#7-阶段-4-兼岗与并行工作关系) · [按岗兼岗](500-job-information-delivery-review.md#20-阶段-4-与-9-的按岗位兼岗) |
| [M04-SCN-003 调动、离职与再聘](../archive/plans/500-personnel-roadmap.md#m04-scn-003) | [506 调动与退出](../specs/m04/506-job-information-records.md#m04-scn-003)，Staffing。占用归 313，合同处置归 512 | 两个满编岗可整组对调、改期。退出保留间隔，再聘沿原 0 号，既发兼岗号不回收。[对调与再聘实测](500-job-information-delivery-review.md#stage-delivery-results) · [退出复验](509-t04-remediation-implementation.md#fixed-environment) |
| [M04-SCN-004 合同续订](../archive/plans/500-personnel-roadmap.md#m04-scn-004) | [512 工作关系合同](../specs/m04/512-employment-contracts.md#m04-scn-004)，Staffing 与合同业务维护者 | A 续订后的前后继和签订日可查，B 合同、任职及编制保持。B 离职时须提交 B 的准确合同版本。[订立、续订与离职](500-job-information-delivery-review.md#18-阶段-10-的工作关系合同) |
| [M04-SCN-005 招聘交接](../archive/plans/500-personnel-roadmap.md#m04-scn-005) | [401 录用转入职](../specs/m04/401-direct-hire-requirements.md#m04-scn-005)，Staffing 持有端到端场景，Recruiting 持有录用及绑定，M03 持有名额 | 同一份预留从 1 转为实占 1，未来撤销后均为零。重新预留后重送旧办理仍留新预留，换人或失配被拒。[双方菜单和实库](500-job-information-delivery-review.md#19-阶段-10-的招聘录用转入职) · [313 份额边界](../specs/m03/313-position-capacity-contract.md#m03-ac-097) |
| [M04-SCN-006 同日动作与经理](../archive/plans/500-personnel-roadmap.md#m04-scn-006) | [506 同日办理与经理](../specs/m04/506-job-information-records.md#m04-scn-006)，Staffing 与人事业务 | 撤销独立晋升后调动仍有效，缺少必要后继处置时整笔拒绝。两个经理互设请求最多一笔成立。[同日及经理样本](500-job-information-delivery-review.md#stage-delivery-results) · [同日复验](509-t04-remediation-implementation.md#22-同日办理及准确来源) |

## 3 其他长期规则与专项接续

| 原文范围 | 承接位置与复查时机 |
| --- | --- |
| 第 1—3 节与第 6 节的领域归属、采用与保存 | Personnel 维护人员身份、编号与授权投影。Staffing 写工作关系、任职、合同及两层主选择。Recruiting 提供录用与交接，各方从公开端口取得准确来源。首次入职在同一事务核权重验并保存，按岗占用遵循 313。历史的分期次序和旧提交继续从 500 查阅。[人员端口](../specs/m01/503-person-registration-design.md#m01-ports) · [入职事务](../specs/m04/403-hire-technical-contract.md#hire-transaction) · [招聘边界](../architecture/target/modules/m05.md) |
| 第 4 节的页面继承、权限及恢复 | 012 持有表单、历史、视口与返回的共享规则。502、505、507 持有正常人员入口、按权列行、资料、主兼岗页签和业务日期差异。普通名录用服务端今日，历史查看和有业务日的办理按任务取日。[共享视口验收](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-viewport-matrix) · [人员页面继承](../specs/m01/502-person-directory-interactions.md#m01-page-map) · [个人资料](../specs/m01/505-personal-profile-contract.md#profile-maintenance) · [职务信息与返回](../specs/m04/507-job-information-tab.md#job-tab-layout) |
| 第 4 节的人员批量导出 | 业务明确用途、字段、导出成员及下载时的授权要求后，由 Personnel 与 Access 接续。涉及报表导出时由 X06 保存导出记录并在再次下载时重核范围。[九栏资料范围](../specs/m01/505-personal-profile-contract.md) · [导出授权](../architecture/target/modules/x03.md) · [报表导出](../architecture/target/modules/x06.md) |
| 第 2、5 节的批次状态、关闭与证据 | 维护者依 014 分别判断方案采纳、执行关闭、实际验收和物理归档。各批验证者按 003 记录受测构建、通过、首败、复验及限制。500 的原评审保留。固定 f 的 4386 由本树维护者在下次更新或用户验收结束时复查。[014 归档规则](../governance/014-document-management.md#82-归档) · [003 完成与证据规则](../adr/003-implementation-stages.md#4-完成与证据规则) · [4386 交接](510-personnel-page-remediation-implementation.md#fixed-environment) |
| 主兼岗转换、非工作关联 | 业务明确需要既有主兼岗互转时，由人事与 Staffing 按 506 的编号、关系和期间接续。客户需单办非工作关联时，M04 维护者按关联性质核对身份、组织角色与是否需要任职。[主兼岗编号](../specs/m04/506-job-information-records.md#m04-br-009) · [M04 工作关系](../architecture/target/modules/m04.md) |
| 资格、支付、薪资、年资及审批待办 | 教育或证书用于任职资格时由 M16 核验，账户用于支付时由 M14 核定用途。正式引入调薪、计薪、参缴时分别由 M09、M10、M12 承接。发布具体年资指标时由人事与 X06 定义口径。业务要求审批待办时由 M20 接入。[资格](../architecture/target/modules/m16.md) · [支付](../architecture/target/modules/m14.md) · [薪酬](../architecture/target/modules/m09.md) · [工资](../architecture/target/modules/m10.md) · [参缴](../architecture/target/modules/m12.md) · [年资待决事项](../adr/106-same-day-personnel-actions-and-daily-state.md#6-待决事项) · [待办](../architecture/target/modules/m20.md) |

509 旧库测试样本中缺少准确前版或业务依据的记录仍按原判定保留。实际升级遇到同类记录时，由该实例有权经办人逐笔核对，原件和回执按原来源读取。[旧库待核与重跑结果](509-t04-remediation-implementation.md#3-保留数据与升级)

## 4 本次复核

以第 1 节的提交为产品基线，在独立 PostgreSQL 测试容器及真实 API 上重跑四项集成用例，均通过，用时 84.551 秒。合同侧验关系隔离、准确版本和退出当日的处置权限，招聘侧验并发转换、无预留模式、资料不齐及撤销后的重送。固定环境、浏览器和容量的结果沿用各次报告所载范围。[两项合同用例](../../internal/bootstrap/employment_contract_integration_test.go) · [两项录用用例](../../internal/bootstrap/offer_hire_integration_test.go)

```sh
go test -count=1 -tags=integration -timeout=10m \
  -run '^(TestRelationshipContractsRenewAndCloseOnlyTheSelectedRelationship|TestLeaveImpactContractsFollowNoticeBoundaryAndDepartment|TestAcceptedOfferConvertsItsOwnReservationOnce|TestOfferHireWithoutReservedSeatsAndIncompleteIdentity)$' \
  ./internal/bootstrap
```

将归档前第 5 节八行的独立预期逐项与现行规格比对，8 项均完整，每个编号只在一份现行规格中定义。核对时发现 501 第 4 节仍有两处指向历史样本，已切换到本规格的同号条款。逐项从规范导航和交接表复查，均可打开对应定义。归档原文的 13 个标题、14 个显式锚点与原计划一致。

逐份比对原有 296 份定稿报告。涉及迁移的 15 份只调整了 35 处链接目的地，4 份既有归档文档的关联链接、7 条登记结果的证据路径也已更新。资产台账有 1 个现行目标随正文迁移，3 项固定提交来源保留原路径。分别从提交取出原文件与留存摘录，对照登记的 SHA-256 全部相符。[验证登记](../governance/registry.json) · [资产台账](../assets/inventory.json)

`make docs` 的 185 项回归和仓库检查通过。`make check` 的领域边界、32 项运行脚本测试、生成结果、Go 测试与 vet、前端类型检查及 `git diff --check` 均通过。
