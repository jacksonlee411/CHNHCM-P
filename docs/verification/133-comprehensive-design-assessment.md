# CHNHCM-133 过度防御与高成本设计再调查

登记信息：[registry.json](../governance/registry.json)的`governance.verification_records`中本文件条目。适用范围：按用户“重新评估和调查、应纳尽纳”要求，对当前已实现链路再次核查，扩展[133计划](../plans/133-overdefensive-processing-remediation.md#r133-comprehensive-assessment)并登记[132待决边界](../adr/132-query-and-validation-boundary.md)。原[方案编制](133-remediation-plan.md)及[追加九项](133-additional-scope-assessment.md)定稿记录保持。

## 1 对象与基线

2026-11-12，工作树`/home/vivobook/projects/CHNHCM-wt-dev-b`，分支`wt-dev-b`，HEAD为`b82a1027c35b2591aa8a9b86f10246c25f5f3e09`。开始已有未提交132/133、131固定环境更新及相关资料，索引与stash为空。本轮编辑132/133、AGENTS、两个资料入口和登记表，新增本报告及必要证据，不修改产品。

开始时记录1619个非docs且非AGENTS的已跟踪文件摘要，以及32个不属本次编辑的已有变更/未跟踪文件摘要。摘要保护用于核对未改动，不表示逐行审查1619个文件。132/133尚未提交，保存本轮修改前原文以便复核前轮结论；输入见[inputs.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/133-comprehensive-design-assessment/inputs.json)。

## 2 调查方法与环境

先将原OD133-01—19作为去重基线，再按用户任务追踪入口→领域计算→存储/锁→响应→前端消费者，并检查生成物、构建标签和诊断调用者。当前`internal/contexts`只有organization和access实现，调查同时覆盖共用历史codec、客户端读取/提交、CI和资料入口；未实现上下文不按目录或架构图推定存在运行成本。

本轮只进行本地静态源码、合同和既有证据核对，没有发起新API/数据库/浏览器请求，没有启动profile、容量控制器或修改固定环境。时间线复杂度、SQL调用数和摘要载荷是代码推导；真实成本和收益须后续候选对照。没有独立代理评审，本文记录主代理调查及自查。

## 3 调查结论与结果

### 3.1 新增、合并和保留

新增十四项OD133-20—33，分别覆盖详情全量读取、单来源查回、读取段重建、查询封口残留、DB追加守卫、历史定位目录、全快照/派生物保留、缓存预算与装载、BU普通列表、部门普通列表计数排序、字段查询否决范围、过期读取取消、授权读取往返和CI装配。各项的源码链接、成本机制、改造成本、必要保护与责任阶段统一维护在133第2.4节，不在本报告复制另一份工作清单。

九组既有事项补充到原ID：01/02的选择请求/CSV内存边界；03的BU候选；06的隐含能力/上下文准备；07的克隆、影响重算及路径证据；11的其他记录交叉扫描；15的旧诊断消费者；16的资料/检查职责；17的全局localStorage依赖；19的投递字段/索引。新增项对应V24—V37，补充项仍使用原V组，清单总计33项、验收参数37组。

优先形成可测候选的是20/21/22/27/31；23先清点生产与诊断依赖。其余按契约和净成本决定，28随BU交付。所有项均有明确评估去向义务，不能以“可选”隐去未处理问题；也不要求所有候选都实施或成为部门503整改前置。

### 3.2 有证据的成本机制

| 静态核对 | 支持的判断 | 不支持的推论 |
| --- | --- | --- |
| 详情遍历全部Records投影；部门snapshot读取历史关系身份对应的完整当前事实 | 当前任务与宽数据加载耦合，可评估任务化读取 | 尚未量化每个字段/祖先对首屏耗时占比 |
| populateResultItems和删除原件从History结果中再找唯一source | 单来源任务存在可去除的前序历史遍历 | 不等于全部日志都可以只读取一个source |
| departmentSegments对每个边界遍历所有edge；saveReadSegments全部删除/写回 | 有B×E遍历和整对象派生写入成本 | source/dictionary绑定未改前，不能直接假定未变段可以原地复用 |
| 五个查询封口列各32字节；Go生成四列、SQL生成第五列；全量proof只在诊断构建 | 原始摘要载荷160字节/段且有计算成本，可评估改为离线核对 | 160字节不是实测行大小或可保证的整库节省；不是当前每请求全量验损 |
| append guard遍历旧目录；对象head、依赖数组、投影分多次更新 | 有重复已封存事实检查/头写入候选 | 目录最多64项，不属于无限目录循环；不能据此删除DB历史守卫 |
| source定位unnest目录；submission采用16位tag后完整身份核对 | 存在扫描与空间折衷，须在长线/碰撞形状下比较 | tag没有被当作完整身份；未证明当前真实查询发生大量碰撞 |
| Catalog只在active=0时超过64MiB全清；BU等待单一loading时已在事务内 | 存在持续负载预算和装载竞争问题，须分别测量 | 64MiB不是实际RSS上限；不等于已证实OOM或2682次503根因 |
| Guard成功路径五个独立数据库调用；CI有12个浏览器分片各自装配 | 有可量化的串行往返和重复装配候选 | 调用数/分片数不等于耗时，不能承诺合并后加速倍数 |

详细源码位置和摘要见[source-findings.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/133-comprehensive-design-assessment/source-findings.json)。本轮没有修改当前SQL、增加索引、调整权限、删除摘要或改变CI。

### 3.3 覆盖中的反证

当前生产查询已经退出全企业完整性聚合；旧proof受`integration && query131`构建标签隔离。新建/叶部门写入已有共享范围守卫，不能称全部全局排他。128授权档案已删除，130已有离线字典回收；本轮26针对其他仍存在的派生结构。选择和父候选已经支持请求取消，31仅覆盖仍缺失取消的读取。

认证、撤权、CSRF、幂等/未知结果、真实历史和必要DB约束有明确业务保护责任，本轮评估减少其不必要工作。迁移模型准入、密码散列和健康检查没有本轮高成本证据，不把安全边界机械列为应删除项。完整矩阵按最终构建复用适用证据的规则已经存在，不再建议建立另一套调度/审计平台。

### 3.4 检查记录

| 检查 | 操作及预期 | 实际结果 | 结论 | 证据 |
| --- | --- | --- | --- | --- |
| 调查、去重与阶段映射 | 当前源码核实十四项、九组合并项及已整改/保留边界，阶段与验收逐项关联 | 33项及37组验收连续无重复，8个产品阶段仍未开始，十四项各有责任与验收 | 通过；静态调查及文档初评，非独立评审 | 133第2.4—2.6、3.6、4及6.6节；[源码指针](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/133-comprehensive-design-assessment/source-findings.json) |
| 文档门禁 | make docs，核登记、引用、结构及资料脚本回归 | 首轮74项脚本测试和293份Markdown结构检查通过，HEAD保持 | 通过 | [首轮日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/133-comprehensive-design-assessment/docs-first.txt)；结果登记后另做最终结构检查 |
| 范围和历史保护 | git diff --check；摘要、编号、分支/索引/stash及无关登记核对 | 1619个产品/工具等文件及32个已有非本次文件摘要不变，无关登记保持，分支/HEAD及空索引/stash保持 | 通过 | [输入](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/133-comprehensive-design-assessment/inputs.json)、[范围检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/133-comprehensive-design-assessment/scope-check.json) |
| 新机制/产品验证 | 原型、DB/API、浏览器、新容量、CI和固定服务检查 | 本轮未执行 | 未运行 | 133产品T00—T07未开始 |

## 4 编制修订与自查

本轮纠正了三个容易误判的前提：旧完整性SQL不在普通产品构建；关系锁并非全部排他；字典已支持受控离线回收。由此将调查聚焦到仍有生产写入成本的封口、实际关系范围以及尚待量化的shape/快照，而非重复认领已完成整改。

补入的方法约束包括：先减少单来源遍历再考虑索引；先改读取段扫描算法再考虑增量模型；先统计无引用shape再开发回收；授权查询合并须保持锁后新状态；缓存淘汰不能使在途请求丢失必需材料；前端取消不能被解释为写入未生效；三门禁及历史故障反例保留。

一次补丁因第4节标题与预期不符而原子拒绝，按实际标题重新定位后应用；这属于编辑过程，不是产品或门禁失败。检查结果只在实际执行后填写，不覆盖原报告和并存任务记录。

文档检查首轮通过。自查保留了B×E和64项上界等量纲，区分Go当前投影与诊断专用五列摘要，明确字段权限改变和派生shape回收需另行取舍；没有把存储压缩、缓存、锁或安全规则整体标为应删除。文档脚本运行隔离外层Git路径覆盖变量，未修改仓库HEAD或索引。

## 5 限制与待证事项

没有新性能测量，不能证明任何候选的净收益、容量改善或与原503的因果关系。source定位碰撞率、无引用shape数量、缓存持续负载高水位和CI最优分片数均待测；14项是有源码依据的评估候选，并非全部已判定必须替换。

调查覆盖已实现业务和横向设施的主要调用链，不是全仓逐行审计或未实现模块成本审计。现行字段权限、详情、分页、会话和存储合同继续有效，采纳替代方案后再同步产品/规格/验证。125容量P1、原失败分母和后置范围保持。

## 6 交接与资源

132/133保持in_review，133产品not_started。本次完成进一步调查和方案纳入，具体原型、最终采用和产品交付依T00—T07分项推进；新增候选不扩大P1开工依赖。原两份133定稿报告、131运行资料及其证据保持。

本任务没有新增运行资源，无需创建环境执行清理；没有变更固定程序、容器、卷或服务，也未核验其当前在线状态。未提交、推送、合并或部署。文档脚本测试如出现夹具profile输出，只代表测试夹具，不是实际业务环境启动。

## 7 证据保存

- [输入、产品与原件保护摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/133-comprehensive-design-assessment/inputs.json)。
- 本次修改前未提交草稿：[132](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/133-comprehensive-design-assessment/132-before.txt)、[133](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/133-comprehensive-design-assessment/133-before.txt)。
- [源码调查指针及摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/133-comprehensive-design-assessment/source-findings.json)。
- [文档检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/133-comprehensive-design-assessment/docs-first.txt)、[范围检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/133-comprehensive-design-assessment/scope-check.json)。
- 实际文档检查、最终结构检查及范围检查保存在[本次证据目录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/133-comprehensive-design-assessment)，不修改此前定稿证据。
