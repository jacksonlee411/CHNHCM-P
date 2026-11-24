# CHNHCM-309 职级与职级阶梯方案评审

登记信息：[registry.json](../governance/registry.json)。对象为[309方案](../plans/309-m03-grade-ladder.md)及[310接入合同](../specs/m03/310-grade-ladder-contract.md)。本记录只证明文档语义复核和文档检查，不是独立代理评审、人工批准或产品验收。

## 1 基线、授权与保护

2026-11-17用户要求进入M03第二批，先完成实施方案、对象接入合同及评审，保留4380及现有数据，下一批使用独立环境。实际工作树`/home/jason/projects/CHNHCM-wt-dev-d`、分支`wt-dev-d`、HEAD为`9050fa8c3040d54dc3bbc08605d90d651a77341f`；开始时工作区干净，三份stash存在。文档登记、既有文件摘要和开始状态见[基线](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-ladder-plan-review/baseline.json)。

用户随后要求优先评估并彻底处理三份stash，已完成核对、仓外归档、三份真实恢复和删除，见[005处置记录](005-stash-disposition.md)。这一明确授权取代本轮最初“保持三份stash”的临时安排；源码、已有证据和运行环境保护不变。新文档不是从旧stash恢复的未交付实现。

本轮只读核对012／121／124、308及最新发布交接、M03目标／术语、Staffing领域和公开接口、存储锁与企业端口、页面与日志、CI及资产。未运行或变更产品服务、数据库、容器与持久卷，没有读数据库秘密或在4380写入样本。

## 2 形成的合同与切分

Grade独立维护；Job有零至多个可用职级，空集合明确未配置，不等于全部可用；GradeLadder至少一个成员，按低至高排列，同Grade可在多个阶梯处于不同位置。适用范围限同企业，Job不绑定阶梯，JobFamily不传播职级，薪酬金额归M09，级内档次后置。全期间正反向资格、准确采用、历史和当前授权分别定义。

先G1交付Grade完整维护，再G2交付Ladder及Job关系、Grade反向保护。两批各有领域、API、权限、页面、历史和独立验收；新集合不塞入现有字符串字段，来源关系统一显式清单，旧Job补零成员索引但原事实字节不改。后续保持岗位定义及层级，再容量与真实任职。

310定义10条新规则、12个场景、18项验收、11项逐页继承和5项业务差异；G1安排14个尺寸角色单元，G2安排Ladder／Job28个及Grade14个回归单元，共56个分批单元。日志直接维护、筛选后退重查、窄屏／原生缩放与返回焦点均有独立预期；这些全部是计划，当前实际执行为`not_run`。

## 3 三轮主代理语义评审

| 轮次 | 输入及实际关注 | 结论／修订 |
| --- | --- | --- |
| 1 | draft-0；原来源保护、唯一当前模型、集合空值及权限 | 要求修改：R309-01／02；改为全部来源显式成员清单，补创建及显式集合字段授权；输出draft-1 |
| 2 | draft-1；对照真实HTTP路由、CSV引用权限、列表排序 | 要求修改：R309-03／04／05；固定selection/resolve、精确CSV字段权和排序白名单；输出draft-2 |
| 3 | draft-2；业务反例、日期与采用、共同守卫、页面及分批出口交叉核对 | 文档接受；30项反例与合同一致，无未关闭P1／P2；不代替实现验证 |

评审者均为Codex主代理/root顺序复核，没有委派独立代理。各轮实际稿件全文、摘要和问题见[评审清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-ladder-plan-review/review.json)、[draft-0](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-ladder-plan-review/draft-0.json)、[draft-1](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-ladder-plan-review/draft-1.json)、[draft-2](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-ladder-plan-review/draft-2.json)。

| 问题 | 级别 | 文档关闭依据 |
| --- | --- | --- |
| R309-01 | P1 | 310第7.1节：全部Job来源／记录补明确零清单，新旧统一封口读取；缺行／损坏拒绝，历史原字节保持；AC047 |
| R309-02 | P2 | 310第6.1节：必填字段缺权不开放创建，显式集合先授权、最终隐式差异再授权；AC039 |
| R309-03 | P2 | 310第7.2节：沿实际selection/resolve办理排除修订，不新增未定义写端点；记录与固定期间路径明确；AC045 |
| R309-04 | P2 | 310第8.2节：Ladder CSV需所有Grade编码／名称权及必要历史权，不能空值降级；AC045 |
| R309-05 | P2 | 310第8.2节：编码仅升序、名称／英文名双向、状态／日期仅筛选、ID平局；关系不参与搜索排序；AC043 |

5项问题为1项P1及4项P2，均只在文档层关闭。[30项反例](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-ladder-plan-review/counterexamples.json)覆盖空集合、跨阶梯排序、有限候选、边界同日、未来引用、删除回退、恢复、受限集合、并发、未知提交、页面回归、CSV和旧源封口；它们不是测试运行结果。

## 4 检查与证据

| 项目 | 实际状态 | 证据／边界 |
| --- | --- | --- |
| 文档语义评审 | 通过 | 三轮记录及30项反例；仅接受方案 |
| 登记、导航、稳定ID及契约追踪 | 通过 | 本批新增与原范围接续，不改旧报告 |
| make docs | 通过：最终128项脚本测试、388份Markdown及12440条本地链接检查 | [最终日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-ladder-plan-review/make-docs-final.log) |
| git diff --check、受控改动及非文档保护 | 通过：2024份非文档文件保持 | [最终核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-ladder-plan-review/final-check.json) |
| 产品Go／DB/API／浏览器／容量／部署 | 未运行 | 本轮仅方案／合同及stash处置，T01—T04未开始 |

本轮资产仅登记未来采用：现有公共控制／列表体验及本仓308验证资产均新增`implemented=false/verified=false`的310采用范围，不提高原资产整体完成状态。所查CubeBox旧staffing-structure及控制入口没有可直接采纳的Grade／Ladder实现，不据此冒称保留了不存在的职级引擎。

## 5 交接

309／310文档经三轮主代理复核接受；最终文档检查通过，登记active，T00文档完成，产品计划仍为`not_started`，所有新验收未运行。下一产品任务为T01及G1 Grade完整维护，G2不得在反向守卫未接通时开放关系写入。

4380的`m03-manual-d`及既有验收数据继续由wt-dev-d保留，用户验收结束或下次更新时复查；本轮没有启动`m03-grade-d`，故没有实际新端口或产品运行资源可交接。stash归档与临时恢复目录收尾由005报告持有。未提交、推送、触发Actions或更新固定环境；方案不关闭125及其他既有容量遗留。
