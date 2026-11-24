# CHNHCM-116 R4剩余评审问题修订

登记信息：[registry.json](../governance/registry.json) `governance.verification_records`。
适用范围：上轮仍存在的拟分类引用查询、授权与安全诊断顺序两项设计问题；保留单部门关系来源简化。
上游依据：[116](../plans/116-m02-shared-dated-maintenance-roadmap.md)、[119](../specs/m02/119-department-functional-design.md)、[120](../specs/m02/120-dated-maintenance-technical-contract.md)、[121](../specs/m02/121-dated-object-admission-matrix.md)、[023](../specs/m02/023-m02-timeline-test-cases.md)。

## 1 验证对象与受测基线

2026-11-05，CHNHCM-wt-dev-b固定分支wt-dev-b，HEAD为03078144666924896077471c290c25dff6c44c5d。开始时工作区已有R4四包设计和关系版本简化增量；本轮在其上修订，不提交或操作索引。修改前4562项已跟踪及未忽略未跟踪文件摘要见[基线](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-review-fixes/baseline.json)。

用户授权复核并修复剩余问题。本轮核对120仍缺拟分类查询输入，关系Plan仍早于实际原/新范围追加授权，且锁阶段尚无具体诊断门槛，因此两项均仍存在。部门真实后端尚未实现，修复对象为当前设计合同，不虚构运行缺陷已修复。

## 2 环境与前置条件

主代理读取当前合同、BU更正授权与错误结构源码，核对公开类型及职责边界；不使用另一套模拟授权服务证明产品通过。既有BU的correctionIntent按原/新类别及完整声明字段形成要求，Error/Fault采用code/message/可选field；本轮部门合同沿用这些原则，不新增平行错误格式。

本轮只运行文档及静态核对工具，没有启动数据库、产品API、浏览器、容器或持久profile。临时编辑及核对脚本归本轮；必要证据已保存在仓内，临时目录在交付前回收并核验，既有环境不操作。make docs中的运行脚本单元测试输出不计真实profile/浏览器启动证据。

## 3 修订与验证结果

| 发现 | 合同修订 | 验收映射与当前结果 |
| --- | --- | --- |
| R4-REVIEW-CLASSIFICATION | 120第4.5节冻结create/maintain和ADD/CORRECT/REACTIVATE输入；保留classification缺省/显式语义、拟日期及关系补丁，共用期间计算；未选原因可查询，保存仍要求完整分类；R5废弃旧查询身份 | R4-V21；8组variant，含额外输入子参数；设计修订已完成，真实Go/DB/API及R5参数not_run |
| R4-REVIEW-AUTH-ERROR | 120第6.5节明确当前可见性→基础动作/字段→显式目标预授权→本部门预期/Prepare→实际差异授权→关系Plan及最终复核；隐藏依赖通用反馈，锁发现不提前返回诊断 | R4-V22—V24；12组variant并展开每个target_states/inputs；设计修订已完成，真实DB/API及锁参数not_run |
| 来源简化保持 | 本部门关系历史、准确成员、稳定BU关系守卫及路径证据保持；没有恢复整树版本字段/全树来源 | 4段原合同摘要相同；R4-V17—V20产品仍not_run |
| 文档及原件保护 | make docs、固定预期/映射、最终输入摘要和git diff --check | 73项脚本检查、240份Markdown、19项映射/保护核对及diff格式通过；4552项原文件不变，不以文档门禁代替产品验收 |

当前[验收映射](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-review-fixes/acceptance-map.json)保留原D01—D39和R4-V01—V20，新增R4-V21—V24；[固定对照预期](data/116-r4-review-fixes/review-case-oracles.json)由合同手工给定，不调用被测算法生成。每个包含多个输入/目标状态的variant必须展开，不能把20组当成20条产品通过。全部部门产品结果仍为not_run。

## 4 复核重点与修订边界

分类查询复核覆盖停用改普通变更、提议日期变化、换BU、原分类缺省/显式同值、空值拒绝、原因尚未选择、企业根null及迟到查询。查询只判断关系候选，不制造已可持久化的候选；名称/原因未完整时仍不能据查询成功保存。

授权复核覆盖原/新范围缺权时目标不存在/启用/停用的等价反馈，有范围权但目标引用缺失/隐藏的等价404；本部门不可见时过期版本不得先返回409。删除/改日隐式回退须在Plan前核实际范围；Prepare本地错误不能夹带外部资格。授权后的真实冲突仍保留，隐藏祖先/下级错误通用化，重放按当前权裁剪且不改原终态；技术失败继续走待确认，不能伪造业务拒绝。

编写复核将诊断预期与现行Fault序列化对齐：通用错误省略field，不新增details；也把成环反例明确为从合法旧关系经本次请求形成环，避免用非法初始库冒充正常API场景。父候选的保存对照明确带入所选父，企业根同值另验冗余CHANGE消除；新增技术参数保持连续表格。12项限定范围语义核对见[记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-review-fixes/semantic-review.json)。这些属于设计编写校正，未运行产品复验。

## 5 未覆盖范围与限制

未修改运行代码、迁移、OpenAPI或生成客户端，未执行make check、integration、smoke或产品容量测试。Prepare内部期间提炼、DepartmentReferenceQuery及授权阶段仍待R4真实实现；对应产品验收不能凭本文关闭。

本轮为主代理自查，不冒充独立评审或人工签收；结论限上轮三项发现中的剩余两项及其直接影响。原首轮报告、关系简化报告和所有既有证据保留原结果，不覆盖历史失败或把旧检查改成本轮通过。

## 6 结论与后续事项

两项问题已完成合同与独立预期修订，限定范围语义复核、文档门禁和静态保护通过。此前整树版本问题由上一轮简化处理，当前三项均有设计修订入口，本轮范围内无未关闭设计项。后续按116逐包实施并执行121真实DB/API；R5菜单与R6容量保持后置。本轮未提交更改。

## 7 证据保存

[修改前基线](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-review-fixes/baseline.json)、[当前验收映射](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-review-fixes/acceptance-map.json)、[固定对照预期](data/116-r4-review-fixes/review-case-oracles.json)。[语义核对及4段原合同保护](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-review-fixes/semantic-review.json)、[19项静态核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-review-fixes/mapping-and-protection.json)、[可复跑核对脚本](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-review-fixes/check-mapping-and-protection.py.txt)、[文档检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-review-fixes/docs-validation.txt)、[最终输入](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-review-fixes/final-inputs.json)。原[关系简化记录](116-r4-relation-simplification.md)及原映射保持；最终收尾结果在同目录closing-check.json保存。
