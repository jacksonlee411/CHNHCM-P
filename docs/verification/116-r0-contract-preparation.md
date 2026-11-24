# CHNHCM-116 R0合同准备与状态称谓验证

登记信息：[registry.json](../governance/registry.json)的governance.verification_records。适用范围：用户授权完成116 R0，包含117—121合同、基线及独立预期；当前BU“业务状态”更名“状态”做定向产品检查。部门及共用内核提炼尚未实现。

## 1 验证对象与受测基线

日期2026-11-04；工作树`/home/vivobook/projects/CHNHCM-wt-dev-b`，固定分支wt-dev-b，HEAD `3736354855be0acb3a257c4a0fa89be5cd3e81c8`。R0开始已有docs/README、docs/specs/README、登记表和116路线图四份未提交资料；原字节已[打包保存](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r0/before-r0-documents.tar.gz)，关键源及原资料摘要见[before-r0-inputs.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r0/before-r0-inputs.json)。未清理stash或改动其他工作树。

本轮合同为[117决定](../adr/117-m02-dated-maintenance-and-department-boundary.md)、[118业务](../specs/m02/118-department-business-requirements.md)、[119交互](../specs/m02/119-department-functional-design.md)、[120技术](../specs/m02/120-dated-maintenance-technical-contract.md)和[121矩阵](../specs/m02/121-dated-object-admission-matrix.md)。同步116、当前状态标题、术语生成源/修订链、导航及三类资产；不回写既有定稿报告或来源原包。最终受测文件摘要在同目录final-inputs.json保存。

## 2 环境与前置

本工作树Python/Git、Go及仓内pnpm工具链。浏览器使用local-bu.py smoke独占临时profile `task-r0-status-b`，真实PostgreSQL当前模型及Chromium，Web实际入口`http://127.0.0.1:57958`，API54866、DB56536；均为本次临时分配，已关闭，不作为日常在线入口。管理员从既有BU正常菜单查询/筛选/选择并导出，测试仅使用合成数据。

本轮不启动持久开发环境，不读取或保存持久库内容。现有四个运行数据库容器保持，临时容器/网络/卷及仓外配置已删除，见[资源记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r0/resources.json)。登录凭据及DB维护凭据未进入证据。

## 3 合同与定向验证结果

### 3.1 R0交付

| 交付 | 实际内容 | 结论及边界 |
| --- | --- | --- |
| 业务范围 | 部门必填单值BU、BU多部门、不跨BU共用；企业内编码稳定；名称/简称，英文不启用 | 合同冻结，产品未实现 |
| 部门路径 | 最小Hierarchy唯一行政父边、同BU单父/全期间无环；当前派生路径与旧源封存证据分开 | 纳入R4/R5，无树平台和整树搬迁 |
| 操作分类 | 117记录四类及七个原因组合；数据更改显示与CHANGE技术码分开 | R1切换目标；R0未更改当前Go准入目录 |
| 权限/提交 | 动作、对象、字段、引用及导出分别控制；独立表/终态、分类型路由与锁、待确认隔离和共同守卫 | 120冻结，真实并发/分表接入留R2/R4/R5 |
| 规则与基线 | 共用规则矩阵、19个部门独立预期、22个BU保护测试/报告入口及摘要 | JSON设计资产，不声称已执行部门算法或回放持久库 |
| 阶段与容量 | 116退出条件、120交付包与混合负载样本 | R1—R6尚未启动；不预编人日或生产容量结果 |

### 3.2 已发生的检查

| 检查 | 实际结果 | 证据与范围 |
| --- | --- | --- |
| 当前BU Go领域基线及应用现有测试 | 通过 | [validation.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r0/validation.json)保存本轮实际命令/输出摘录；领域基线使用当前源码的有效Go缓存（原始输出另存go-domain-baseline.txt.gz）；CSV头改为编码、名称、状态、生效日期，启用值保持 |
| 前端typecheck | 通过 | 同validation记录，tsc --noEmit退出0 |
| 真实DB/Chromium状态列筛选 | 通过 | [原始输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r0/status-browser.txt.gz)；同列多状态、替换为日期及未来首条，使用“状态”选择器 |
| 真实DB/Chromium跨页选择及CSV | 通过 | 同原始输出；137条全结果、跨页排除、页大小/排序保持及实际四列表头断言 |
| 主代理合同语义复核 | 通过 | [semantic-review.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r0/semantic-review.json)，6项合同不一致已闭合；非独立评审、非人工业务签收 |
| 文档/生成/登记与引用 | 通过 | make docs的73项Python、生成源/登记/本地关联及最终固定分支引用检查，见validation和docs/refs输出；不计部门实现 |
| 独立预期与输入保护一致性 | 通过 | [design-input-checks.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r0/design-input-checks.json)，19个样本均未运行、4份原资料及22个基线入口摘要一致 |
| 自有资源回收 | 通过 | [resources.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r0/resources.json)，本次容器/网络/卷为空，原4个持久容器保持运行 |

## 4 修订与复核

复核关闭原路线图中关系基数未定和路径全部后置、换BU仅单层级版本、祖先变化遗漏选择版本、开放部门遇未来BU停用、分类切换影响企业初始化、合同结果混记产品验收等问题。精确修订依据和关闭位置见语义记录。独立预期D16采用实际当前NAME_CHANGE/名称变更快照，避免虚构旧目录事实。

状态变更仅显示文案/CSV头和相应断言，API键、枚举、启停、查看日未生效和授权语义保持。术语表由JSON源生成，修订链记录117的前后摘要；其他域状态描述和原报告原文保持。

## 5 未覆盖范围

R1共用Go内核、R2事务提炼、R3共用页面BU接入、R4部门/Hierarchy真实DB与共同守卫、R5双对象正常菜单和路径视口、R6容量/标准均未实施或执行。新目录未运行、19个部门样本actual_result均not_run。未运行完整make check/build/integration/smoke，不作全M02、真实消费者、生产能力或人工业务签收结论。

已有BU历史/恢复测试和旧报告只被索引并保存摘要，不是本轮全部重跑；以后每阶段按实际代码重新执行。未复制或清空持久库，不以空壳页面、目录和合同检查证明部门已接通。

## 6 阶段结论与资源交接

R0的业务/技术准入事项已闭合，可按116进入R1；本次授权止于R0和明确状态称谓变更，不自动启动R1。116整体执行保持in_progress，后续阶段仍not_started。

临时profile及对应容器、网络、数据卷和仓外运行配置已由smoke finally回收并再次核验；临时Web入口已失效。原4个持久数据库容器保持，由其原工作树/任务维护者负责，复查节点为其下次使用/收尾；本轮没有接管、重启或删除这些环境，未验证其业务入口在线。不提交、不推送、不开PR、不合并或部署。

## 7 证据保存

长期非敏感证据在[data/116-r0](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r0)：原资料归档/输入摘要、BU基线索引、独立预期、语义复核、定向浏览器输出及资源记录。validation记录检查的实际命令/退出结果；Go/typecheck为本轮工具输出摘录，未伪装成完整原始日志。最终文件摘要用于识别dirty工作区内容，不包含自身或本报告/登记表的递归摘要。
