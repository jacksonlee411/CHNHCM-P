# CHNHCM-508 人员职务信息修复方案评审

## 1 对象与结论

2026-11-25，Codex 主代理在固定 f 起草并评审 508。以 `d981fed840d374837f1797001af444fe94c2db6f` 的代码、迁移、合同和本次工作区中的方案为依据。首次走查要求修订历史成员更新、旧数据处理、回执定位和详情日期。修订后逐项复审，**方案评审通过，可以开始实施**。六项产品修复的执行状态为未开始，500 第 6.3 节整体交付继续暂缓接受。[实施方案](../plans/508-job-information-remediation.md) · [500 第 6.3 节](../plans/500-personnel-roadmap.md#job-information-delivery) · [登记表](../governance/registry.json)

## 2 六项产品反例与版本

下表复核上一轮独立评审留下的实库或菜单观测。固定 f 的 4386 运行产品合并 `dfd495542ac60a610dacb0d71dc30340976a3951`，与受审基线的 Git 差异为五份文档。4386 原库用于只读复查，改变业务状态的反例来自上轮专属库。本轮逐项核对原始观测、复现输入与基线代码。[产品 PR](https://github.com/jacksonlee411/CHNHCM-P/pull/25) · [原验收记录](500-job-information-delivery-review.md#stage-delivery-results)

| 发现、引入批次 | 上轮实际业务结果 | 本次复核的实现依据 |
| --- | --- | --- |
| F1　P1，#12 | 明确引用同日调动后删去该调动，返回 200，任职回研发，当前转正仍引已删修订，产品部仍匹配 1 笔。先办 7 月 1 日转正再补 6 月 1 日调动，也得到日状态已调走、当前转正仍归原部门。 | 历史重算只选择任职变更类型，关系成员从旧来源复制。单纯扩充类型清单还会漏掉从兼岗登记的关系动作。[历史重算](../../internal/contexts/staffing/infra/postgres/assignment_history.go) · [跨任职修订归属](../../db/migrations/00046_relationship_probation.sql) |
| F2　P1，#12、#13 | 选择「先调动后转正」，菜单实际显示「转正、调动」。完全反序登记时，在晋升前补转正不能指定已有晋升，转正显示「未确定」，晋升修订未变。 | 转正的前序位于其采用证据中，时间轴只传通用前序。现有「反序」回归先登记了转正。复审对照了 507 要求的第二轮操作。[时间轴](../../internal/contexts/staffing/infra/postgres/job_information.go) · [原回归](../../internal/bootstrap/assignment_promotion_integration_test.go) · [双顺序合同](../specs/m04/507-job-information-tab.md#m04-ac-018) |
| F3　P2，#14 | 4386 的「离职再聘验收bf3bb9ed」在 2027-08-31 有一笔离职，发生部门为空。选实际部门「研发bf3bb9ed」后只见入职。另在独立库复现离职未筛 1、按原部门 0。 | 离职写入原件根部的部门，查询却从试用证据取值。退出逻辑已将两种输入日期归一为首个不在职日。[退出写入](../../internal/contexts/staffing/infra/postgres/employment_exit.go) · [事务列表](../../internal/contexts/staffing/infra/postgres/employment_register.go) |
| F4　P2，#8 | 单任期在职员工的 API 于 2026-11-25 返回产品部，从档案进入却是 4 月 1 日的研发。另一个已经结束的 4386 样本在明确选择 4 月 5 日后显示核 3 实 4 超 1，刷新跳回 4 月 1 日。两人分别验证默认日与刷新。 | 单任期入口用起日作默认，详情选日只在组件状态中。任职直达页和名录已有不同用途的日期参数。[任职页面](../../apps/web/src/features/hire/AssignmentPage.tsx) · [Go 日期读取](../../internal/contexts/staffing/infra/postgres/job_information.go) |
| F5　P2，#12 | 0 号办转正，1 号用新读的来源、新提交号重办同日期及同通知，真实 API 返回 `409 ALREADY_CONFIRMED`，没有原笔位置。数据库仍只有一笔。 | 领域已有关系唯一判定，现有回执封装有分别返回动作身份、准确修订的通路，可保留原提交协议。[转正判定](../../internal/contexts/staffing/application/hireprep/probation.go) · [回执事务](../../internal/contexts/staffing/infra/postgres/hire_history.go) |
| F6　P3，#23 | 在 390×844 从发出历史返回已接受录用，第一次点击改期后 5 秒内未出现到岗日。同条件单项四次通过。控制真实交接查询的送达时机，两次测得按钮上沿从 581.03 移至 617.53 像素，松开命中按钮外，表单数为 0。再点一次可打开。首轮的具体时序仍未确定。 | 交接资格查询完成时会在已有操作行上方增加按钮。验收保留受控真应答、原时限和操作结果三个观察点。[操作行](../../apps/web/src/features/recruiting/Offers.tsx) · [交接读取](../../apps/web/src/features/recruiting/OfferHireHandoff.tsx) |

原始观测 `observations.txt` 的 SHA-256 为 `b64eb3b6406b339406f39bc87ce746bd533fcea9384aeb4f8e4fd2d6ef98d943`，六文件复现包 `review-repro.tar.gz` 为 `8a07201b0bcb32c6e13a9f506a79f1388d0c537e0c06019b6d6d5d16d4b77b6b`。本表和计划第 5 节保留了可在其他检出中重建的输入及预期，运行时不依赖该仓外附件。[复验清单](../plans/508-job-information-remediation.md#acceptance-cases)

## 3 首轮意见与修订复核

| 首轮发现的方案问题 | 修订后接受的条件与复核依据 |
| --- | --- |
| “所有任职来源指向新修订”可能被实施为回写历史，且旧数据对账缺少可执行入口。 | 限定在受影响任职**新发布**的来源替换成员，旧来源固定旧答案。原通知、采用依据、回执逐一对账。在统一升级器提供按客户的预检和应用入口，调用同一 M04 Go 逻辑，按已授权主体写入，逐关系可重跑。待补依据记录可定位的原因。[关系重算](../plans/508-job-information-remediation.md#relation-revisions) · [旧数据处置](../plans/508-job-information-remediation.md#existing-data) |
| 仅检查时间轴显示的三行，无法证明后项引用跟随新修订，也容易把受限前项暴露给兼岗读者。 | 补入完全反序的逐笔前后值与引用断言。先按源内事实排序，再裁剪读者可见行、链接和数量。用仍可独立成立与资格必须依赖两组样本分别验证删除和整笔拒绝。[业务先后与资格](../adr/106-same-day-personnel-actions-and-daily-state.md) · [六组验收](../plans/508-job-information-remediation.md#acceptance-cases) |
| 原号、新号都“返回原笔”仍可能误取首次修订。新增可选字段也可能改变升级前请求的摘要。 | 已加入后继修订后的双号对照。原号恢复当次修订，新号固定核对时的准确修订。保持旧请求省略新字段时的摘要。以当前来源同案换号、旧来源首次竞争、同号异内容三组判据复核。[关系回执设计](../plans/508-job-information-remediation.md#relation-revisions) · [提交恢复合同](../specs/m04/403-hire-technical-contract.md#hire-recovery) |
| 把详情选日写入名录的业务日会改变原查询，单任期入口也可能在重读时换目标。 | 采用已绑定任职的详情日期。明确详情日、入口日和服务端默认有明确优先次序。用在职人检验初入，用已结束样本检验指定日刷新，并复核名录返回、源版本、焦点和撤权。[菜单设计](../plans/508-job-information-remediation.md#register-date-offer) |

## 4 复算与验证边界

以 506、507 和 403 为预期，用合成关系在内存独立枚举三笔的六种登记排列。给定「转正→晋升→调动」的同一通知，六组的当前引用都顺着准确前项。完全反序时，先记调动，补晋升后调动改引晋升，再补转正后晋升改引转正、调动随之改引新版晋升。三个时点的旧来源均能复述各自的链。这里的次数是样本中的修订次数，产品仍用准确修订身份。[业务样本](../specs/m04/506-job-information-records.md#m04-scn-018) · [同日写入](../specs/m04/403-hire-technical-contract.md#same-day-action-write)

用「总数、研发、产品」复算转正，归研发为 `1/1/0`，归产品为 `1/0/1`。同日调动但通知明确载明有效的研发部门时为 `1/1/0`，前序未知且仅影响统计时为 `1/0/0`，部门决定权限且无法核实时应拒绝。离职的「未筛、原部门、另一部门」在全权、仅有行权且无部门披露权、无行权三种样本中分别为 `1/1/0`、`1/0/0`、`0/0/0`。两种退出日期换算均落到 8 月 31 日取部门。[发生部门规则](../specs/m04/506-job-information-records.md#m04-br-014) · [办理查询合同](../specs/m04/403-hire-technical-contract.md#personnel-action-query)

本轮评审对象为方案文档和上述验收预期。产品修复、真实库升级与并发、修复版正常菜单、原生缩放和 4386 更新均留到实施验收。固定 f 继续负责原验收环境。[验收与交接](../plans/508-job-information-remediation.md#acceptance-cases)

## 5 文档检查与现场

受审定稿的方案正文 SHA-256 为 `b32544955ca6bbe524e14fce228d2d5541c315d92a39244fe862e0325e969ca2`。原 500 交付验证记录及其登记条目与基线逐项相同。工作区增量为 508 方案、附属评审记录、登记表，以及文档索引和 500 第 6.3 节的导航。

- `make docs` 通过 180 项治理回归，资料检查覆盖 470 份 Markdown 和 10,102 处仓内链接。
- 对两份新文稿另核 85 处仓内链接、其中 31 处显式锚点，以及六项问题与验收行的一一对应、两处入口、末行和空白。全部通过。
- `make boundaries` 通过 765 份 Go 文件的依赖和 137 份 SQL 源文件的归属检查。`git diff --check` 通过。
- 本轮只读核对 `m01-profile-f` 的 Web/API 归属与就绪状态。4386 保留原构建和原库。此次文档评审通过，修复版产品验收尚待实施。
