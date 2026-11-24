# CHNHCM-308 独立职类合同评审

2026-11-19，三位独立 Codex 评审代理接受同一份终稿。职类列入既有职务基础合同第14节。新职务按工作性质和专业领域分别归类，旧职务按有据的业务日期补录。职类与岗位容量首批可按任意顺序实施、分别签收。[职类合同与验收表](../specs/m03/308-job-foundation-contract.md#m03-job-function) · [岗位容量首批](../specs/m03/313-position-capacity-contract.md)

## 1 基线与业务依据

工作树为 `wt-dev-d`，HEAD 为 `51db97531e6c6d56115f6801bd000083ce4081b6`，开始时 stash 为空。此前已有 313 合同、两份评审、复算资料和对应导航登记。308 原文 SHA-256 为 `bed8852cc8b8a45bf7b8a6619fb813b7cb020641fd633e3a404ae65cca8559ba`。307 实施、308 页面合同和页面验收各有已完成的交付记录。[307 实施](307-m03-job-foundation-implementation.md) · [308 原页面合同评审](308-job-page-admission-contract.md) · [308 页面验收](308-job-page-standard-implementation.md) · [岗位签收](311-position-definition-signoff.md)

| 核对来源 | 对本合同的影响 |
| --- | --- |
| Oracle Fusion 的 Job 业务对象分别列出 `JobFunctionCode`、`JobFamilyId`、`ManagerLevel` 和 `ApprovalAuthority` | 这些属性支持区分职能分类、专业归属、管理层次和审批权限。CHNHCM 自行决定企业级目录、双维必填、存量例外和按日修订方式。[Oracle Attributes for Job](https://docs.oracle.com/en/cloud/saas/human-resources/faiau/attributes-for-job.html) |
| CubeBox 的 303《职务体系功能说明书》第2节和 JA02。提交 `2deaf9276086f68e07aaa0c2cf7f35025046a9ee` | 采用职类表示工作性质，沿用 CHNHCM 的“职务系列”表示专业领域。人力资源经理与人力资源专员共用系列，增加财务经理即可验出两个方向的交叉组合。保留两个候选框的独立性。[来源路径、摘要与采用登记](../assets/inventory.json) |
| 本仓职务、职级、岗位及术语材料 | 既有 Job、系列、职级及岗位保持各自稳定身份。企业本地名称相同或相似时逐项确认归类。核定名额和 FTE 由真实岗位的容量合同接管。[术语对照](../architecture/target/17-terminology-glossary.md#domain-m03) · [310 职级](../specs/m03/310-grade-ladder-contract.md) · [312 岗位](../specs/m03/312-position-contract.md) |

## 2 独立评审与修订

主代理编写和修订合同。独立评审代理只读核对固定候选、现行合同和必要源码，分别给出结论。每个意见按其能否导致不同业务结果或错判验收处理。

| 轮次 | 评审者与结论 | 所评合同 SHA-256 |
| --- | --- | --- |
| 一 | `job_function_review1` 要求修订，提出下列五项 | `843cee01411140b2675c0fb9a57fab1e2122c5407054fc457f520e3106b09002` |
| 二 | `job_function_review1` 逐项复验后接受。独立加入的 `job_function_review2` 核对权限和正常入口，另提三项，要求修订 | `fc8280fdb1f2126b0756d81542f4492e4c5419a85f3dfe7aef81bbf85211d6ac` |
| 三 | `job_function_review1` 复核首轮结论，`job_function_review2` 关闭第二轮三项，`job_function_review3` 从业务样本、历史、反向保护及分批交付独立终审。三方均接受本行固定稿 | `139cf85eed0a356acc27675f0298e79c4380acd1f04c1c781d6d37de018e5eb6` |

| 首轮发现 | 修订与可复验预期 |
| --- | --- |
| 一份原未登记职类的启用首条改日，新日无职类可重新采用 | 区分纯改日与新增启用资料，明确字段省略、显式清空及写权的结果。用 AC084、AC091 复验 |
| 空白新建表单的系列候选使用完整 Job 构造，另一维必填会堵住首选 | 两个候选分别按本维所需期间和权限求解，在保存时验证全部维度。AC089 要求两种起手顺序均能完成。[现有系列候选](../../internal/contexts/staffing/application/reads.go) |
| 只检职类首条的企业下界会漏掉企业从另一方向后移起日 | 把已停用及无引用职类纳入 Staffing 最早日期公开端口。AC092 交换两种锁序并核对已停用职类。[现有反向端口](../../internal/contexts/staffing/infra/postgres/store.go) |
| 草案假定旧八列 CSV 有服务端历史下载入口 | 以已下载原件和再次导出为两个操作。有效的旧固定选择重新授权并生成九列。AC093 同时覆盖跨升级的提交查回。[现有即时导出](../../internal/contexts/staffing/application/selection.go) |
| 单独验 313 时无法在职类页面操作，首批也没有真实用量可查 | AC090 分别验证仅有 308、仅有 313 和两者并存。已有用量按 313 随真实消费者另行验收 |

| 第二轮新增发现 | 修订与可复验预期 |
| --- | --- |
| 移动未登记的旧节点，能让前一段的管理类多覆盖整月 | 第14.2节要求逐段比对。AC086 的五月因此要求职类写权，AC091 的未登记首条改日只缩短同一段，获权者可办理 |
| 回送原职类 ID 与省略字段在旧职责更正中可能给出相反的授权结果 | 按现行显式字段规则，回送同值和异值都要职类写权。AC091 另验省略字段只改职责可成功。[Go 字段裁决](../../internal/contexts/staffing/domain/timeline.go) |
| 现行列表应用第二个分类筛选会替换第一个，桌面与窄屏都不能走出合同里的交集 | 第14.3节规定逐维应用、取消、清除和返回行为。桌面、窄屏按 AC083 逐步核对匹配数 2、1、1、2。[现行职务列表](../../apps/web/src/features/staffing/List.tsx) · [012 查询与筛选](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-query-state) |

每轮补丁均以 UTF-8 保存。取第1节基线提交的原始 308，按成员编号从小到大在临时副本上依次应用补丁，每步的 SHA-256 应等于上表该轮摘要。仓内保存这些评审稿的差异，可独立还原每位评审者所见的文本。[评审候选逐版补丁](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/308-job-function-review/reviewed-revisions.zip)

## 3 检查与产品验收范围

用独立样本复算 AC083 与 AC086。三份职务按管理类、人力资源这两个集合取交集，唯一成员为人力资源经理，应用、追加、取消、清除的计数依次为 2、1、1、2。将未登记边界从 2027-05-01 移至 2027-06-01，逐日比对可见仅 2027-05-01 至 2027-05-31 的 31 天发生归类变化。第一位评审者首次调用本机缺少的 `python` 别名，随后使用 `python3` 复算通过。主代理也复算了这两项预期。

在本次文档基线上运行 Go 既有领域用例，覆盖 `TestStaffing307Admission`、`TestStaffing307IndependentSuccessors`、`TestStaffing307PairAndAdoption`、`TestStaffing307LifecycleInheritanceAndHiddenFields`、`TestStaffing307FullCoverage`、`TestStaffing307ExplicitAndImplicitFieldSet`、`TestMembershipDatedDecisions` 和 `TestPositionFutureJobGradeBoundary`。八项均通过，成员测试中的 Job 和 GradeLadder 子例亦通过。它们验证已有系列、独立后继、字段权、职级及岗位边界。[领域验收样本](../../internal/contexts/staffing/domain/timeline_test.go) · [职级集合](../../internal/contexts/staffing/domain/members_test.go) · [岗位资格](../../internal/contexts/staffing/domain/position_test.go)

仓库文档检查 `make docs` 与 `git diff --check` 均通过。逐版应用三个补丁均还原对应固定稿。新增四条规则及十二项验收在 M03 的锚点唯一。收尾首次比对发现 307 与 308 在登记表中原范围文字相同，新范围误落入 307 条目。按文档 ID 归正并复验，三个导航文件和登记表去除本次 308 差异后均与开工时逐字节一致。313 原有合同、两份评审及三份复算／快照文件的六个摘要也全部匹配。

新职类的 Go、迁移、API、页面及相关真实业务验收由 308 后续实施批次执行。岗位容量首批按 313 的现有验收范围另行交付。人工产品签收和 Oracle 租户运行均未执行。
