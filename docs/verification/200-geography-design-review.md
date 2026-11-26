# CHNHCM-200 国家地区与地点方案精简及复用评审

登记信息：[registry.json](../governance/registry.json)的governance.verification_records。适用范围：[200路线图](../archive/plans/200-geography-and-location-roadmap.md)、[201方案](../specs/m02/201-country-region-geo-area-and-location-maintenance-contract.md)、[202默认日期](../adr/202-master-data-default-effective-date.md)。上游为[124接入指南](../guides/124-dated-object-admission-and-capacity.md)及[132必要校验边界](../adr/132-query-and-validation-boundary.md)。

## 1 验证对象与受测基线

日期2026-11-13，工作树CHNHCM-wt-dev-c，固定分支wt-dev-c，基线f0d358e3e79ed5c84056b3967d50c5a1110d95d6；开始工作区、索引及stash为空。用户要求方案评审，避免过度设计和过度防御，关注对象通用能力复用。本轮评审并修订文档，没有开始产品实施。

评审由主代理执行，未启动独立代理，不冒充独立审查或人工业务签收。[原始输入摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/200-geography-review/inputs-before.json)记录21份文档/源码，原200/201/202全文保存为同目录before文本，首轮问题保留。

## 2 环境与前置条件

只读取本仓文件并修改本轮文档。不重新访问Odoo、CubeBox、互联网或运行服务，不启动profile、容器、卷、浏览器或数据库，也不变更固定环境。前批来源核对和访问失败继续由[原编制报告](200-geography-plan.md)持有；本轮不把旧数据观察改记为最新。

## 3 评审过程与结果

| 检查 | 方法和范围 | 实际结果 | 证据 |
| --- | --- | --- | --- |
| 首轮语义评审 | 对照200/201/202、124/132及真实共用源码，按具体任务检查必要性 | 要求修改：1项P1、6项P2 | [首轮问题](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/200-geography-review/review-first.json) |
| 修订后复核 | 对照真实端口逐项检查能力、日期、引用、路径、装载、依赖与验收 | 七项在文档层闭合；接受本轮方案范围，数据准入与产品仍未完成 | [最终复核](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/200-geography-review/review-final.json) |
| make docs与git diff --check | 登记、格式、链接、来源及差异检查 | 首轮通过：74项脚本回归、310份Markdown、231份资产；最终收尾重验另记实际结果 | [检查结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/200-geography-review/checks.json)、[首轮日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/200-geography-review/docs-first.txt) |
| 原源码/来源/报告及修改范围 | 核对授权路径、受测源码摘要、索引和stash | 通过；仅本轮文档变化，原来源和报告保持 | [范围核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/200-geography-review/scope-check.json) |
| Go/DB/API/浏览器、容量、远程CI | 本轮无产品改动 | 未运行 | 本报告第5节 |

### 3.1 复用核对依据

| 事实 | 评审结论 |
| --- | --- |
| dated.Policy已经隔离资料，Kernel已拥有时间线和维护算法 | 保留一份算法；新增对象只提供真实Data/Patch/Policy及目录，不复制状态机 |
| Create、Prepare/Calculate、更正、位置新增、启停窗口都消费enterpriseStart | 统一对象日期边界，不能只改创建；1900默认不是全局合法下界 |
| submission.Repository/Run及恢复代码已经负责确认和原结果 | 三对象增加仓储/结果适配，不另建事务或导入终态服务 |
| presentation的DataFields、RecordView、ContextView/Result依赖BU/部门，Definition强制pair/selection | 改实际差异的类型及可选端口，由真实国家切片验证；不做万能表单或假实现 |
| DatedPages新建与重置使用currentDate，身份区固定显示企业 | 两处消费Go默认；目录身份不假挂企业，企业初始化不扩成新管理模块 |
| 现有固定类型生成器、选择/导航及X03 guard可扩展 | 继续强类型和独立SQL/事实；不建第三选择协议、动态EAV或第二授权框架 |

### 3.2 必须保持的保护

本次不删当前授权、字段权、目标来源冲突、必要关系资格、同日唯一、不可变历史、独立物理事实、锁后检查、原子提交及未知结果恢复。检查是否必要，以实际业务影响及责任层判断；不能因为“共用”让CountryRegion/GeoArea继承部门BU归属、整个地点生命期的目录启用约束或全期间路径分段。

## 4 问题、修订与复核

| 问题 | 等级 | 原风险及修订结果 |
| --- | --- | --- |
| R200-03 引用来源变化与目标冲突混淆 | P1 | 原文可能使上级仅更名也拒绝地点保存，改日引用资格又不够明确。201 3.2/7.1区分目标冲突和锁内新采用，改日按新日核验，纯名称/门牌保留旧采用；SCN206/207和AC205/206同步。文档关闭。 |
| R200-01 三对象默认接齐全部入口 | P2 | 201 4.1增加按任务能力表：目录保留核心维护，普通高级时间线入口与选择/CSV后置；Location完整维护及选择保留。共用算法与接口可选性分开。文档关闭。 |
| R200-02 共用适配过宽或漏掉实际耦合 | P2 | 201 4.2/4.3列出直接复用、最小修改和地理独有逻辑；日期边界覆盖所有入口，Go用常量/纯函数提供默认；202不扩建企业管理模块。文档关闭。 |
| R200-04 路径“业务日/期间”引入无界证据责任 | P2 | 201 3.3明确单日路径及采用时有限链，不复制部门全期间path_evidence、独立关系来源或反向刷新任务；旧引用证据保持。文档关闭。 |
| R200-05 全包就绪边界与运行校验时点含混 | P2 | 按国家目录/地区发布完成标记，固定离线命令复用应用与提交机制；全包检查在装载执行，普通请求仅取必要对象/引用。中断行续做，已受理行不重复。文档关闭。 |
| R200-06 全部来源冻结阻断无关接入 | P2 | 200 4.1将默认日期独立，公共类型随国家真实切片接入；大陆/澳门待决只约束相应数据发布，最终四地出口仍保留。文档关闭。 |
| R200-07 机械搬用组织长线负载及重复测试 | P2 | 保留10000地点检索规模，普通形状改为1—3当前记录/1—5来源，单独1000来源样本；六条关键链路测量，完整公共规则与对象适配分层验证。文档关闭。 |

二次阅读补齐三点：同摘要包零增长不妨碍未受理行续做；记录改日按新日重新采用，但无关名称/门牌新增保留原采用；预览/保存共享函数不替代锁后新事实校验。全部为方案语义修订，没有执行产品修复或性能对比。

### 4.1 反例复核

| 反例 | 修订后的独立预期 |
| --- | --- |
| 目录只有251条却被共用接口要求提供pair/选择草稿 | 国家首批没有这些入口和请求；核心维护继续使用共用算法，不能用假函数满足类型 |
| 明确提供1899-01-01且真实资格覆盖 | 不因初始默认1900而拒绝；晚起企业仍按自身边界拒绝 |
| 地点更名时原区域已停止新选 | 沿用原采用证据，可办理无关修改；不枚举全国或全部消费者 |
| 地点创建前区域仅更名，身份及资格不变 | 锁内读取最新合格资料并采用，不能仅因引用sourceVersion变化拒绝 |
| 先创建地点成功，随后区域停止新选 | 两次可先后成功，旧地点仍保留；相反顺序新采用须拒绝；不套部门全期覆盖预期 |
| 地点记录改日，新日不满足区域资格 | 按新日核验并拒绝；不能复制旧日许可 |
| 祖先多次更名，地点只改邮编 | 地点形成新来源，保留未变的区域采用链；不重建祖先全部历史或全期间路径 |
| 澳门条目待核对，大陆目录已完整 | 已完成范围继续可用；澳门未核对区域不成为候选，基础地址依原范围交付 |
| 包在第N项后中断，再跑同一包 | 原已接受行按原身份恢复且零新增，未受理行续做；完整范围结束后才发布其候选 |
| 国家首次接入暴露前端资料类型差异 | 与真实适配器一起最小扩展，不先完成全域泛型平台再交付国家功能 |

## 5 未覆盖范围与限制

源版次选择、澳门八条处置及国家代码/显示字段冻结仍由T00分项执行；评审没有确认全部来源已可发布。新的表、API、类型、权限映射、默认日期和性能均未实施或测试。所有反例是可执行验收预期，不是产品测试结果。

本轮后置目录普通高级维护和选择/导出，保留Location完整维护；这不是删除共用内核能力。需要后续开放的目录动作应直接接既有机制，并按其实际业务规则验证。125容量P1、冷态/10000记录非阻断和原目标保持。

## 6 结论与后续事项

200/201经本轮主代理方案评审修订，七项问题在文档层关闭；接受本次能力范围、复用边界、必要检查及实施顺序。文档登记active，200 execution_status仍为not_started；202用户默认规则保持active。此结论不代表产品批准发布或具体数据来源准入。

后续可独立推进默认日期适配，以国家真实切片带出其余共用扩展；各地区来源与功能按自己的依赖接入。原编制报告、离线来源、源码和既有运行环境保持。本轮无新增运行资源，资源收尾不适用，不为评审启动或清理容器；没有提交、推送、PR、合并或部署。

## 7 证据保存

- [原输入及源码摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/200-geography-review/inputs-before.json)、[200原方案](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/200-geography-review/200-before.txt)、[201原方案](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/200-geography-review/201-before.txt)、[202原决定](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/200-geography-review/202-before.txt)。
- [首轮问题](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/200-geography-review/review-first.json)及[最终复核](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/200-geography-review/review-final.json)：首轮要求修改保留，文档闭合不计产品通过。
- [实际检查结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/200-geography-review/checks.json)、[首轮日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/200-geography-review/docs-first.txt)与[范围核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/200-geography-review/scope-check.json)：实际返回码及保护边界。
- [最终输入摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/200-geography-review/inputs-final.json)：本轮文档及证据SHA-256，排除该清单自身和执行输出，避免自引用。
