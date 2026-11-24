# CHNHCM-313 岗位容量本地化与兼岗评审

2026-11-19，Codex 主代理复核了岗位、招聘和人事三个接入批次。结论为接受本次合同修订。正式内部兼岗进入当前合同，员工办理在 M04 真实任职接入时交付。M03 首批可独立验收核定名额和 FTE。[现行合同](../specs/m03/313-position-capacity-contract.md) · [首版五轮评审](313-position-capacity-contract-review.md)

## 1 修订范围和交付取舍

复核基线为 wt-dev-d 的 `51db97531e6c6d56115f6801bd000083ce4081b6`。开始时工作区已有 313 首版合同、复算、评审和登记导航，stash 为空。两批岗位定义与层级已有各自签收记录。首版合同尚未提交，修订前保存了原文，解压后的 SHA-256 为 `dd1bfeb35ddedad494ca057ae0f380e6ba26b088539dfbbb4862f05b9d161709`，对应首版评审。原评审及其复算脚本沿用各自原件。[首版合同原件](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/313-position-capacity/contract-before-localization.md.gz) · [311 交付顺序](../plans/311-m03-positions.md)

| 方案 | 对业务和交付的影响 | 决定 |
| --- | --- | --- |
| 到 M04 再定义兼岗口径 | 首批若把核定名额标成在岗人数，或让一份任职隐含一整份工时，招聘会先采用这些口径。后续改口径会影响既有预留和按日历史 | 现在就明确名额、人员、任职和工时份额的含义 |
| 在 M03 首批交付员工兼岗入口 | 新增或结束兼岗要同时处理员工工作关系、主选择、工时来源、任命依据、权限与两岗占用。仅凭岗位设置无法写出完整业务事实 | M03 先独立设额，M05 接真实招聘，M04 首批真实任职接入一并验收正式跨岗兼任 |
| 扩展到集团借调、关联单位同时用工、代签及待遇 | 用人或用工主体认定、授权、合同变更和津贴结算各有责任方，岗位身份不足以决定这些事实 | 本次只接同一用人单位、同一有效劳动关系下的正式跨岗任命。纯代办交相应用途，跨单位情形交关系核查，待遇交责任域 |

首版规定每份计入的任职扣一个名额。本次将这条规则落实到两种不同的申请。员工跨岗且各自占编时两岗各扣一个名额，同关系同岗期间重叠时先拒绝新任职。岗位使用受限时仍校验全部真实占用，只返回当前范围可读的信息。[兼岗合同](../specs/m03/313-position-capacity-contract.md#m03-br-028)

## 2 公开材料与采用理由

以下来源均于 2026-11-19 核对。Oracle 部分采用在线产品指南以及标明版本的 26D 功能说明。中国大陆法规记录适用版本与生效时间，统计部门答复按其调查对象解释。

| 来源及观察 | 本合同采用的边界 |
| --- | --- |
| Oracle Fusion Human Resources 的工作关系可按配置容纳多任职，每段关系选一份主任职。正式新增任职可默认次任，任职的 Headcount 工作量指标一般默认主 1、次 0。[Assignments](https://docs.oracle.com/en/cloud/saas/human-resources/fawhr/assignments.html) · [Transfer Actions](https://docs.oracle.com/en/cloud/saas/human-resources/fawhr/transfer-actions.html) · [Headcount](https://docs.oracle.com/en/cloud/saas/human-resources/fawhr/headcount.html) | 借用关系与任职分层。HR 用途的主选择由 M04 维护。岗位名额采用具体任命和企业政策，允许兼岗分担工时或占独立名额 |
| Oracle 任职 FTE 通常按任职工时除以标准工时，可分层取基准及按任职覆盖。岗位核定 FTE 支持手输或依配置自动计算。[Assignment FTE](https://docs.oracle.com/en/cloud/saas/human-resources/fawhr/full-time-equivalent-fte.html) · [Position FTE](https://docs.oracle.com/en/cloud/saas/human-resources/faucf/how-fte-is-calculated-in-positions.html) | 岗位 FTE 显式核定。用量固定有依据的基准版本、工作量来源和单位周期，按份四位小数。缺可用基准拒绝计量 |
| Oracle 招聘状态可与岗位启停分设，任职人数上限检查及 Allow Overlap 可以配置。录用可按预计开始日查额度，26D Redwood 的岗位需求新建也可配置人数校验。[岗位录用限制](https://docs.oracle.com/en/cloud/saas/talent-management/faimh/configure-job-offers-to-respect-positions.html) · [26D 需求校验](https://docs.oracle.com/en/cloud/saas/readiness/hcm/26d/recr-26d/26D-recruiting-wn-f50322.htm) | M03 对计入的维度执行硬限额。M05 接入独立的按日暂停补员和按需求份额预留。暂停前可识别的原承诺可在原期间原额度内履行。该转换规则为本项目决定 |
| Oracle 的岗位预算可按部门、地点及预算期间汇总并校验岗位。[Position Budgeting](https://docs.oracle.com/en/cloud/saas/human-resources/faspb/overview-of-position-budgeting.html) | WorkforcePlan 维护计划与预算，通过明确采用写入单岗运行限额，暂停补员也单独保存 |
| 全国人大常委会《劳动合同法》2012 年修正，修正决定自 2013-07-01 施行。第十七、三十五条规定合同事项及协商书面变更。第六十八、六十九条规定非全日制用工的计酬工时特征及多单位订约条件。[劳动合同法](https://fgk.chinatax.gov.cn/zcfgk/c100009/c5193025/content.html) · [修正决定施行日期](https://www.beijing.gov.cn/zhengce/zhengcefagui/qtwj/201307/t20130701_780700.html) | 对普通全日制员工按任命依据建立内部兼岗。变更约定内容时核对适用的变更依据。一个岗位占 0.2 FTE 并不据此重分类员工的用工形式 |
| 最高人民法院劳动争议司法解释（二）第三条，自 2025-09-01 施行，规定关联单位交替或同时用工时如何结合书面合同或实际用工事实确认关系。[法释〔2025〕12号](https://www.court.gov.cn/fabu/xiangqing/472691.html) | 即使同集团，另一用人或实际用工主体也需要核实。内部兼岗入口不能替办理人认定第二劳动关系或改变缴费主体 |
| 国务院《关于职工工作时间的规定》1995 年修订，自 1995-05-01 施行。第三至五条规定标准工时及特殊情形。国家统计局人口和就业统计司 2024-04-10 的答复说明劳动工资统计按调查单位和就业人员定义取数。[工作时间规定](https://xzfg.moj.gov.cn/front/law/detail?LawID=629) · [国家统计局答复](https://www.stats.gov.cn/hd/lyzx/zxgk/202405/t20240524_1954119.html) | 40 小时是已确认适用标准工时制时的周基准示例。工时制度、实际考勤和薪酬各自采用适当依据。单位内部去重人数、岗位名额与对外报表分别计算 |

## 3 本地项目与反例复核

CubeBox 快照 `2deaf9276086f68e07aaa0c2cf7f35025046a9ee` 的 `apps/prototype/src/domain/position-resources.test.ts` 覆盖了同人同岗的两条任职共用一份工作量、跨岗独立工作量、来源冲突和暂停补员。Bugs-And-Blossoms 快照 `e1c0291f3989b34dc1a92d7a891f62bf231d4494` 的 `docs/dev-plans/staffing/894-staffing-domain-readmission-and-rename-contract.md` 第 21.4 节拒绝同人同岗重叠，分别计算企业人数、部门参与人数、任职数和占编数。其 `docs/dev-plans/position/678-position-p0-domain-contract.md` 第 18 节当前按整数编制控制。前者提供重复工作量的反例，后者提供可解释的重叠边界。CHNHCM 保有核定名额与 FTE 两个维度，在同关系同岗阻止重复建任职，在跨岗处允许同一工作量按原工时分配。本仓的构建及复算独立于这两个快照。

| 评审输入（合同 SHA-256 前 12 位） | 反例、修订与结论 |
| --- | --- |
| `dd1bfeb35dde` | 同一人到第二部门分担 8 小时，原合同不能确定是第二个人、第二个编制还是部分工作量。补明工作关系边界、同岗重叠、独立占编与只计 FTE 的用途、原始工时约束和分批入口 |
| `4ddc30aabac8` | 同时在两个不同岗位申请同一来源余下的 8 小时，只锁岗位会超分。预设未来暂停时还须确定哪些在途承诺可以履行。复核统一锁序与关系守卫，补足按业务期间核验、两种先后顺序、明确的主任职及合同变更反例 |
| `8473c336ae1c` | 候选撤回后原需求仍可能主动释放。暂停或撤销恢复如果本来就没有额度，测试无法证明拒绝原因。分别补入显式释放、尚有空额的暂停样本、被重新分配的具体 FTE 和接入前可观察的需求申请结果 |
| `3c73357ddad6` | 仅检查生效日会漏过两周任命中途变更用人单位，拟结束日也可能被误作已释放。补充 AC081 的 2027-04-08 跨主体拒绝及确定截止日，区分适用的周基准直接采用与须另查制度和日历的换算，复核接受 |

## 4 验证及未覆盖范围

本轮合同定稿 SHA-256 为 `ba303deccc7cda97922bc13b3d520ce18285e7bd2e10a95ce4234e6fad4fff8a`。在 Python 3.14.4 运行首版 13 项固定断言，并另行逐份和逐日复算 5 组新样本。单人每周 32＋8 得 0.8000＋0.2000。两个岗位各自能接受 8 小时，合起来会比同一来源的余量多用 8 小时。三份各三分之一在原工时用尽时仍有 0.0001 的 FTE 账面余额。将原预留延后一天转换，暂停期间首个无承诺覆盖日为 2027-07-01。[首版复算](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/313-position-capacity/recalculate.py) · [修订复算](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/313-position-capacity/localization_cases.py)

在上述 CubeBox 快照、Node v24.19.0 执行 `node --test --test-reporter=spec --test-name-pattern='同人同岗两条任职|两名半职|重复来源的不同值|冻结不抹去现任|新人员的非占编兼岗' apps/prototype/src/domain/position-resources.test.ts`，5 项均通过。该命令只用于复现来源行为。Bugs-And-Blossoms 核对固定提交的合同文本。

仓库执行 `make docs`，128 项脚本检查及文档登记、链接检查通过。`git diff --check` 通过。另对未跟踪文本复核行尾和末行换行，逐一核对 26 个验收锚点、5 个规则锚点、首版原件与现行合同摘要。

现行 26 个产品验收样本为 AC056—081，其中 AC072—081 为本次新增。M03 产品功能、M05/M04 真实办理、数据库并发、页面权限验收、Oracle 租户实操及人工业务签收均未执行。文档结论限于本地方案与公开资料，不作个案劳动关系认定。[验收样本](../specs/m03/313-position-capacity-contract.md#m03-ac-072)
