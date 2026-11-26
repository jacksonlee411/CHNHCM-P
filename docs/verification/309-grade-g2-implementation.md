# CHNHCM-309 G2阶梯与职务职级实施记录

登记信息：[registry.json](../governance/registry.json)。范围为[309 T03／T04](../archive/plans/309-m03-grade-ladder.md)及[310](../specs/m03/310-grade-ladder-contract.md)的G2适用部分。初次交付的本地验收见第1至8节，交付后评审发现的8项问题及复验见第9节。

## 1 开工基线与环境保护

工作树`/home/jason/projects/CHNHCM-wt-dev-d`，固定分支`wt-dev-d`，HEAD `d9a702fd852aaa66eccf39e73c9b87d61bb81603`。开工工作区及stash均为空。基线见[开工核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/baseline.json)，迁移核对见[全部本地引用](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/migration-all-refs.json)。新增00019及00020，不改00017／00018；00020追加前再次核对[15个本地／跟踪引用](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/migration-00020-reservation.json)，不改已有00019。

独立profile `m03-grade-g2-d`，Web4382／API8382／DB54372；配置前确认端口未占用。用途为开发与用户验收，责任wt-dev-d，用户验收结束或下次更新时复查。4380、4381及其他已登记持久环境不用于本批写入、迁移或测试清理。随机数据库凭据由既有工具保存仓外。

## 2 实施范围与当前落点

GradeMembers及逐成员GradeAdoption独立于文字字段；Job按ID归一且无序，Ladder保留请求顺序。来源记录逐条封存成员清单、数量与摘要；旧Job来源补零成员清单，原来源、修订、采用及回执字节保持。当前／历史均按同一清单读取，损坏拒绝。

共同守卫按提交、企业、Family、Grade、Job／Ladder顺序取得；锁后重读，依赖变化回滚再规划，累计锁预算不重置。Grade资格收缩验证当前头完整启用期间；更名保留消费者采用。正常菜单及集合表单已接入Staffing；Ladder有序成员支持键盘上移／下移，Job集合无序。当前标签按业务日读取，准确来源读取逐成员采用；权限不足整体受限，503清理旧标签。

## 3 首次失败与定向复验

[首次真实DB/API](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/integration-first.txt)因运行时迁移准入仍只允许1—18，拒绝1—19布局。补齐准确账本集合及00019表存在性后，[第二轮](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/integration-second.txt)三个测试根通过：集合与封口、完整期间反向保护及来源、开放候选及提交摘要。未以重复重跑替代修复。

| 轮次／发现 | 修复或处置 | 实际证据及状态 |
| --- | --- | --- |
| R1 领域与事务：已选候选的期间失效未提示 | 明确selectedUnavailable；不改变拟成员，保存仍独立校验 | [定向复验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/integration-candidate-review-second.txt)通过 |
| R1：原集合读权曾扫描不相关历史 | 仅核实际受影响原集合／目标集合，纯继承不增历史权 | [今日集合／缺失封口](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/integration-review-round1-first.txt)通过 |
| R1：缺失清单可能从反向索引消失 | 索引查询前从当前头核对清单及成员数量，缺失拒绝 | 同上；缺封口读取与Grade反向均503 |
| R1：Ladder重规划未保留最初依赖锁组 | 回滚后取得原始、拟定及锁后新头的排序并集，预算不重置 | [首次反例](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/integration-review-round1-first.txt)及[复验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/integration-replan-second.txt)；Job、Ladder及原Family通过 |
| R1：原子性、恢复与日期 | 真实COPY写入同一事务；取消／迟到／未知COMMIT按原号恢复；固定期间只改停用端保留恢复采用 | [R1分组](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/integration-review-round1-first.txt)四根通过；首次重规划根失败另列 |
| R1再次核查：成员数量不变但内容损坏时可能从反向索引消失 | 先批量验证所有当前来源的完整清单及摘要，再使用反向索引；不把未命中视为无关系 | [首次反例](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/integration-corrupt-index-first.txt)：Grade停用错误接受；[修复复验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/integration-seal-enterprise-second.txt)四根通过；含Ladder企业双向守卫与同物理事务 |
| R2 页面与权限：新增浏览器入口未登记CI影响表 | 登记GradeMembers及310测试入口，不扩大测试排除 | [首次文档检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/docs-first.txt)、[影响表复验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/ci-impact-second.txt) |
| R2：持久G2已有BU，企业停用前置拒绝 | 首轮V01后复用环境不满足企业空库前置；保留G2数据，改新临时profile | [权限专项首次](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/browser-permissions-first.txt)：1前置失败、3通过、6未运行；不称专项通过 |
| R2：仓外X11运行器的设备挂载缺失 | 补显式/dev挂载，验证Go、Docker及X11可用；首个临时profile已回收 | [首次启动](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/browser-full-first.txt)、[清理](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/browser-first-cleanup.txt)；后续原生缩放已实际运行 |
| R2取证复查：保存后截图仍在加载，成员编辑行字号密度不一致 | 截图等待详情与预期标题恢复；编辑行继承表单字号。当前详情的计算字体已与标签一致，不将旧截图观感直接判为当前字体缺陷 | [当前字体实测](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/member-font-first.json)；[42单元汇总](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/browser-42-cells-final.json)及[实际截图复查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/browser-visual-review-final.json)完成 |
| R2再次核查：候选撤权只清了编辑器，取消可能重新显示父详情的旧成员 | 同时清理父详情所选记录、时间线和固定期间缓存，保留仍获权的标量草稿；专项增加撤权状态下取消及保存前撤集合写权断言 | 代码审查发现；[最终32项回归](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/browser-regression-final.json)全部通过，其中G2精确Grant／故障／CSV七项 |
| R2覆盖核查：Job为无序集合，夹具第20个Grade未必是查询者页面实际末项 | 按实际渲染的20行成员取最后一行，当前及准确历史都以正常滚轮到达；附末项名称及数量 | 管理员已有第20个移除按钮可达检查；[Job查询者补强复验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/browser-job-reader-last-second.json)六测试／七尺寸单元全部通过 |
| R2专项首轮：迟到响应测试的新名称包含旧名称，负向子串断言误报 | 改用可明确区分的新旧名称；仍要求迟到旧响应不能覆盖新标签，503清标签并保留输入 | [首轮完整浏览器](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/browser-full-second.txt)两项失败及截图保留；已启动矩阵批次缓存旧模块，运行期间改动未进入该批次，[夹具核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/browser-candidate-fixture-audit.json)证明仍用旧名称；冻结后另起专项进程复验 |
| R3准备：OpenAPI缩小格式差异时重复components键 | 删除重复键并以实际生成器校验；原语义不变 | [首次生成](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/generate-first.txt)、[复验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/generate-second.txt) |
| R3：首次分页对比失败尚无具体差异，曾怀疑成员摘要；定向通过未证明根因消失 | 显式排除摘要成员字段，避免Ladder CSV重复解析；该调整记为契约整理。追加具体差异诊断，保留原比较断言，继续整片复验 | [第四片首次失败](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/integration-staffing-shard-4-final.txt)及[定向复验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/integration-list-parity-second.txt)两根通过；下一次失败才明确定位外层排序，不把成员摘要误记为已证实的根因 |
| R3再次复验：页内顺序依赖SQL连接计划，Family倒序返回被重排 | 在分页外层显式保留顺序；四对象使用不同连接策略核分页一致性及无编码权ID顺序；Grade／Ladder仍按310 D02拒绝编码倒序 | [第四片第二次失败](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/integration-staffing-shard-4-second.txt)；补强测试首次缺strings导入、第二次误把Grade编码倒序当合法，分别保留[编译失败](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/integration-page-order-final.txt)、[预期错误](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/integration-page-order-second.txt)；[修正复验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/integration-page-order-third.txt)通过 |
| R3：Family复用的changes形状随Job增加了不适用的gradeIds | 为Family五种维护输入使用单独的JobFamilyChangesInput，后端仍拒绝不适用字段 | [OpenAPI语义审查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/openapi-semantic-review.json)及[最终生成](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/generate-final.txt)通过 |
| R3全仓回归：存储准入故障测试恢复到旧G1合同，污染后续子场景 | 恢复到当前staffing-grade-ladder-1；保留所有损坏布局必须拒绝的断言，同步迁移README当前／历史版本边界 | [第七片首次失败](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/integration-shard-7-first.txt)；[存储准入定向复验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/integration-storage-admission-second.txt)通过 |
| R3覆盖复核：原投影重建用例未实际建立Ladder和Job成员 | 以真实关系删除四类派生读投影再重建；比较全部事实、清单、成员字节摘要及当前／历史不同顺序，并恢复原Job回执 | [G2投影重建及API重启](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/integration-g2-rebuild-final.txt)通过；大容量库真实DB重启另验 |
| R3暂存评审：Grade候选在编码不可读时仍按隐藏编码排序 | 无编码读权按ID排序；显式覆盖Job／Ladder两页候选、编码搜索与字段投影，保持获权编码顺序 | [首次反例](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/integration-hidden-candidate-order-first.txt)实际失败；[完整候选根复验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/integration-hidden-candidate-order-second.txt)通过；310补明候选的隐藏编码边界 |

评审方式为主代理按合同进行的顺序代码／事务／页面自查及自动化实测，不是独立代理或人工业务验收。三轮发现的问题均修复并完成定向复验；最终受影响分片、页面矩阵、完整容量及重启出口通过。

## 4 容量首次结果与修复

[全部采样](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/capacity-first.json)为49组，每组100正式HTTP，共4900正式、1010预热，全部尝试保留，语义错误0。首次有7组P95超标：Ladder候选20会话522ms；Ladder普通／成员保存1934／1369ms；Job普通／成员保存1298／1053ms；100成员Ladder详情双角色1514／1353ms。目标保持300／500ms，不援引旧非阻断决定。

修复为同事务COPY写入强类型成员、消除一次重复头读取、批量加载当前Grade，以及按已排序ID一次往返取得全部Grade共享锁；资格仍由Go判定、没有跨请求缓存。后续复验还覆盖100成员候选及保存，不以普通10成员替代专项。

[首次运行](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/capacity-first.txt)在全部正式采样及Grade1643份、Ladder985份原回执／准确来源重启核验后，达到20分钟整体运行预算。Job／Family重启未完成，整个测试未通过。[自有容器清理](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/capacity-timeout-cleanup.json)完成。复验增加流程运行预算，业务P95判据不变；资源数据在重启前保存，避免整体超时丢失已完成的测量。

第二轮在其他集成、浏览器及构建结束后独立采样，[53组原始数据](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/capacity-second.json)含5300正式、1094预热，语义错误0；[资源记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/capacity-second.json.resources.json)已保存。六组仍不达标：Ladder普通／成员保存20会话P95为1148／1185ms，Job普通／成员保存为1177／1007ms，100成员Ladder候选／保存20会话为515／960ms。目标不变，第二轮不能关闭容量出口。该轮四次真实DB／API重启核验Grade1643、Ladder1227、Job1885、Family1份原回执及准确来源，共4756份；每次比较全部24张Staffing不可变表摘要，均保持，未新增来源或终态。整个测试仍因六组P95失败退出1。后续增加仅显式启用的写入诊断，采集SQL模板耗时及Go CPU栈，不记录参数；带诊断开销的结果不替代正式容量验收。

写入诊断确认pgx COPY的UUID字符串编码回退占用CPU，锁前完整成员读取与Grade完整JSON解码也重复消耗资源。优化为：锁前只取得同语句快照的依赖ID，锁后仍完整解码／验封口并检查依赖并集；未变更成员从已锁定验证的不可变前源在同事务复制强类型行，变化成员用二进制UUID COPY；当前Grade关系读取完整按日投影及准确修订，资格与采用仍由Go判定，历史继续读取不可变原源；候选仍扫描完整期间，只为当前页构造输出映射。上述读取及复制优化没有变更格式、历史、锁序或容量阈值。后续追加00020，仅调整数据库结构引用检查方式，见下。

第二次诊断夹具因实现加速后20次新登录与原管理员登录落在同一分钟，实际触发429；[失败日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/write-diagnostic-second.txt)保留，不计性能结果。夹具改为复用原管理员会话加19个新会话，保持20个独立会话，不放宽产品限流。

优化后的写入诊断CPU由11.47s降到5.05s，但成员INSERT及COMMIT仍占主要SQL耗时。临时库先试逐语句批量检查，再对重复采用依据去重；保留全部成员行及原封口，重复的存在性查询只执行一次。追加00020将这项结构约束落地；Grade来源／修订及两类成员清单不可变，应用仅可SELECT／INSERT，其他事务未提交的来源不可见，因此不依赖逐成员KEY SHARE行锁防止父行删除。迁移及运行准入复核不可变守卫；资格、排序、权限及生命周期仍由Go判定。新增真实SQL／COPY反例验证缺清单、缺来源、缺记录／修订、跨Grade／实例、混合有效无效批次全回滚及父事实禁止更新／删除。API池仍12，不调整容量阈值。

读投影同时改为复用未变化修订并切换当前source_id，仅替换变更／删除记录；核对受影响行数，缺投影拒绝。完整重建仍从不可变来源重建全部索引。四个性能修复分片通过后，00020、投影变化及新增结构引用反例又完成[50根四分片复验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/integration-constraints-status.json)，四片命令退出0，其中Staffing308ClockBrowser因未启用专门浏览器环境条件跳过；跨既有Grade采用来源另做[定向复验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/integration-reference-integrity-final.txt)通过。

第三轮[53组正式采样](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/capacity-third.json)完成5300个正式及1094个预热请求，语义错误0；[资源记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/capacity-third.json.resources.json)完整。仍有四组20会话P95超标：Ladder候选754.954ms、普通保存558.946ms、成员保存836.328ms、100成员候选641.031ms。Job全部组及Ladder100成员保存已达标。四次真实DB／API重启再次核对4756份原回执及准确来源与全部24张不可变表，均保持；测试仍因四组P95失败退出1，未关闭出口。

候选专项诊断显示完整Grade读取及重复分配占主要CPU。进一步缩减查询传输列，预分配有界记录切片；成员读取按每份清单返回强类型数组，存储仍为原强类型行，数量、逐成员依据、顺序及摘要继续完整验证。摘要复用必须先逐字段比较实际成员完全相等，缓存仅限本次事务读取。写入将已决定的头、来源、读投影、未变成员复制及Outbox合并为同事务批次，检查投影行数后再写变更成员；任何失败仍回滚全部事实与Outbox。[传输及批次优化后的50根四分片](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/integration-transport-summary.json)已完成：49根PASS、1根条件SKIP，四片退出0；同摘要不同实际成员反例同时覆盖Job及Ladder。未以诊断数据替代容量验收。

候选插桩诊断由100次请求P95 613.732ms／CPU采样7.91s降为521.688ms／5.98s，均无语义错误；[两轮原始成本](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/candidate-diagnostic-second.json)及CPU栈另存，冷连接和插桩数据不作为正式通过证据。第四轮完整容量在所有构建、浏览器、DB/API分片和诊断进程结束后独立运行。

第四轮[53组完整HTTP数据](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/capacity-fourth.json)已完成5300个正式及1094个预热样本，全部尝试保留，语义错误0，52个有延迟目标组均达标；500Job＋100Ladder反向拒绝组的100次请求全部按语义拒绝，该组没有延迟目标。以下列各类最高P95对应组，全部组的P50／P95／P99及逐样本数据见原始文件。随后四次真实DB／API重启核对Grade1643、Ladder1227、Job1885、Family1份原回执及准确来源，共4756份；全部24张不可变表摘要保持，无新增来源或终态。[完整运行](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/capacity-fourth.txt)974.612秒，退出0。

| 组别 | 会话 | P50 ms | P95 ms | P99 ms | P95目标 ms |
| --- | --- | --- | --- | --- | --- |
| Ladder普通候选（单会话读组最高P95） | 1 | 49.046 | 73.901 | 80.945 | 300 |
| Ladder100成员候选（20会话读组最高P95） | 20 | 263.929 | 481.724 | 502.155 | 500 |
| Job普通保存（保存组最高P95） | 20 | 342.170 | 493.996 | 529.334 | 500 |

[资源成本](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/capacity-fourth.json.resources.json)在重启前保存：包含准备、预热及采样共498.091秒，Go HTTP服务及测试客户端进程用户／系统CPU为150.595／90.278秒，最大RSS为166384KiB，累计分配11902720808字节；Staffing关系存储213360640字节。18逻辑CPU，API连接池12，PostgreSQL的fsync、synchronous_commit、full_page_writes保持开启。各组正式采样CPU／分配／耗时另列原始数据；没有把上述进程指标称作数据库或整机CPU。Job普通保存P95接近500ms阈值，结果是本机本轮完整实测，不表述为线上SLA。

## 5 当前验收覆盖与出口跟踪

| 合同出口 | 独立预期与证据定位 | 当前出口 |
| --- | --- | --- |
| AC031 | `TestStaffing309DefaultFloorAndInstanceIsolation`增加同物理库双实例Ladder身份／来源与Job成员隔离；新建默认1900、同类型重复、跨类型同码；浏览器正常菜单 | G1受影响DB/API与42单元正常菜单已通过；00020及性能修复后的50根分片通过 |
| AC032 | `TestStaffing310LifecycleCoverageAndAdoption`；矩阵管理员实际完成新建、新增、更正、改日、单条启停／删除、固定两条插入／维护／删除 | 生命周期DB/API及42单元完整维护已通过；00020及性能修复后的50根分片通过 |
| AC033 | `TestStaffing310CollectionsAndSeals`、领域members反例：缺省／null／[]／重复／深克隆，Job换序无变化及保留Family；空集合样本G2-ACCEPT-EMPTY | 领域、DB/API及Job14单元通过；00020及性能修复后的50根分片通过 |
| AC034 | 同根Ladder重排新来源、共享Grade及旧顺序；20成员各尺寸正常键盘操作；两个共享职级验收阶梯 | 领域、DB/API及Ladder14单元通过；00020及性能修复后的50根分片通过 |
| AC035 | `TestStaffing310CandidateAndDigest`：开放尾段排除、已选期间失效、改日扩大前序拒绝；日期边界与完整拟成员绑定；2000／2001字符边界 | 完整候选根（含隐藏编码）及冻结夹具后的日期／迟到／503专项通过；00020及性能修复后的候选根通过 |
| AC036 | `TestStaffing310WholePeriodReverseAndEvidence`：Grade停用、首日及未来恢复改日、删除回退、扩大停用期，对两类隐藏未来引用拒绝且无新来源；更名不改采用 | [源动作专项](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/integration-source-actions-final.txt)通过 |
| AC037 | `TestStaffing310LifecycleCoverageAndAdoption`：恢复新采用、仅改停用端保持恢复采用、删除固定段回退原采用、已移除历史不阻塞源停用 | 生命周期及准确采用通过；00020及性能修复后的50根分片通过 |
| AC038 | WholePeriod／LadderExportAndLog：当前标签与准确源分离，成员及顺序日志，Grade／Family所选摘要与原件回归 | DB/API、日志及准确历史页面通过；00020及性能修复后的50根分片通过 |
| AC039 | MemberPermissionsAndCorruption／SelectedPermissionAndMissingSeals及六个权限专项；完整集合读取、隐式写权、源读／历史权，503／迟到不泄漏；取消不恢复被撤权成员 | DB/API及最终精确Grant专项通过，含撤权取消不恢复旧成员；00020及性能修复后的50根分片通过 |
| AC040 | GradeWriteRaces八种先后／空Job竞争；Ladder企业双方先后；物理事务探针；GradeReplanAndSwaps锁后新头／回滚排序并集及双Grade互换 | [批量锁复验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/integration-batch-guards-first.txt)、[封口及企业复验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/integration-seal-enterprise-second.txt)通过 |
| AC041 | RecoveryAndLockBudget：原号、跨类型同号、取消、迟到／resolve、未知COMMIT、换账号隔离；浏览器响应丢失；完整容量库真实重启 | 恢复故障注入、页面专项及第四轮4756份原回执／准确来源重启全部通过 |
| AC042 | 日志内直接新增／更正，取消恢复日志位置与焦点、保存定位，准确历史只读；G2、Grade、Family／Job同入口回归 | 42单元及Family19／BU6回归通过 |
| AC043 | 查询草稿与已应用条件、21项跨页、筛选后退重查／关闭旧面板／继续查询，原Staffing导航和BU日志回归 | 42单元及Family19／BU6回归通过 |
| AC044 | 表单直接字段、原值和自动变化；20成员末项、键盘排序、错误焦点、取消／Escape／未保存及未知提交离开 | 42单元通过，含20成员实际末项和原生缩放；未知提交离开专项通过 |
| AC045 | LadderExportAndLog逐值七列及JSON成员顺序／逗号、北京时间跨日；Job八列保持，Grade六列回归；完整集合／源字段撤权整笔拒绝；Ladder页面21项固定选择排除、新项不补入 | 逐值CSV、跨日及固定选择页面专项通过 |
| AC046 | [最终47测试清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/browser-matrix-final-manifest.txt)：36个矩阵测试代表42单元，另六个Grant／故障测试、一个固定选择CSV测试及四个企业前置；同构建BU参照 | [42单元逐项证据](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/browser-42-cells-final.json)、[32项回归](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/browser-regression-final-summary.json)、[同构建对照](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/browser-visual-review-final.json)通过 |
| AC047 | 00018到00019旧Job逐来源补零封口、字节不改；缺失／损坏及同数成员损坏均拒绝；全仓八片＋最终Staffing四片；sqlc／OpenAPI／客户端 | [生成及五项静态检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/checks-closure-fourth.json)、[最终50根分片](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/integration-transport-summary.json)及[新增引用完整性反例](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/integration-reference-integrity-final.txt)通过；最终生成物及完整暂存差异另行复核，见收尾记录 |
| AC048 | 1000Grade／100Ladder／1000Job、20记录、100成员、500＋100反向；逐组100正式、预热单列及资源成本；所有原源／成员采用／回执重启保持 | 第三轮四组P95未过；传输及批次优化的49根通过、1根条件跳过，第四轮53组、5300正式及1094预热请求语义错误0；52个有目标组达标；4756份原回执／准确来源四次重启全部保持，测试退出0 |

页面矩阵及原消费者回归已完成。性能修复后的受影响DB/API分片及规定容量均已完成。独立验收环境固定构建、样本及数据保护复核通过。完整暂存差异与交付检查见收尾记录。

没有推送、PR或合并；本地验证不替代正式quality／integration／browser CI。用户验收环境保留；[651个自有临时测试容器及最后容量卷核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/temporary-resources-final.json)、先前浏览器／X11清理均完成，不按全机资源数量判断。

全仓集成第六片的外层运行进程收到SIGTERM，日志无完成结尾，第八片尚未开始；中断不计通过。[剩余根清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/integration-interrupted-selection.json)按原分片清单扣除已有PASS／SKIP，继续使用现有分片器拆为八个较小片执行；[遗留自有容器清理](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/integration-interrupted-cleanup.json)保留归属及结果。最终覆盖以逐根结果核对，不以外层命令启动或部分片通过代替全量覆盖。

首轮完整浏览器报告为105项：103通过、2失败、0跳过及0重试；两项失败均为候选专项名称夹具的子串断言，完整报告保留。冻结修正后的专项和最终API后，[32项定向回归](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/browser-regression-final-summary.json)全部通过：Family19、实际共享日志消费者BU6、G2精确Grant／异常／CSV7。原105项的外层进程中断，子进程完成的JSON报告与外层退出状态分别记录，不将其说成一次命令全部通过。

[构建比对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/browser-final-build-equivalence.json)确认42单元矩阵与上述32项回归的index.html及引用的全部前端资产字节相同。回归目录多一个未被引用的旧JS文件，不能声称整个目录相同。后批API包含分页外层排序修复，四对象强制连接计划的DB/API复验及Family页面回归覆盖该差异；不是声称两批API二进制相同。补强Job查询者实际末项的首次进程中断、零测试完成，另留[中断状态](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/browser-job-reader-last-interrupted-status.json)后重新执行，六测试／七单元全部通过。

## 6 用户验收样本准备

独立G2环境已准备[25个合成业务样本](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/acceptance-samples.json)，统一搜索`G2-ACCEPT`。[准备脚本](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/seed-acceptance.mjs)仅访问4382，未连接4380／4381。20个Grade、两个共享Grade的Ladder、一个Family、已配置与空集合Job均有正常菜单入口；主阶梯有2030独立重排记录，共享阶梯有2027—2028停用段。Grade01更名后，当前成员标签改变，历史采用名称保持，已由真实API核对。

G2已刷新到包含00020的固定构建，见[构建及运行交接](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/runtime-handoff-final.json)。[32张Staffing表](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/runtime-facts-after.json)在刷新前后数量和逐表摘要一致；[100次双角色当前／准确历史查询](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/runtime-samples-after.json)及[8项正常菜单只读检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/runtime-menu-final.json)全部通过，业务写入0。前端index及全部引用资产与42单元矩阵相同；候选传输及写入批次优化后再次刷新，[第四次刷新核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/runtime-refresh-fourth-status.json)四步均通过；固定构建的全部源码摘要与当前候选一致，32表／100次查询／8项菜单检查再次通过。

[4380／4381只读保护核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/protected-runtime-final.json)确认固定API和构建清单与本轮中途指纹相同，端口仍开放；没有读取或写入其数据库，不声称有开工数据库全量摘要。G2固定环境继续保留，责任wt-dev-d，用户验收结束或下次更新时复查。

运行环境截图复查发现，管理员Ladder首批菜单脚本在查询路由更新前点击旧列表行，详情返回条件缺少q；补充查询完成等待及back查询条件断言后，八项复验全部保持查询条件，未改页面实现。[首次证据与复验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/runtime-menu-fourth-review.json)分别保留。

## 7 验证口径与限制

[逐根累计覆盖](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/integration-coverage-final.json)发现337个测试根，最终累计326个PASS、11个条件SKIP、0个FAIL／未完成；覆盖来自原全仓分片、补齐中断的余项及修复后的定向分片；新存储约束及性能修复后的50根受影响测试另行完整执行。不是声称最终候选在一次命令中跑过全部337根。11个条件跳过为Country200ListCapacity、DepartmentR5BusinessDayBrowser、JointMidnightBrowser、LegalEntity206Capacity、Location204Capacity、Staffing307Capacity、Staffing308ClockBrowser、Staffing309Capacity、T07BusinessDayBoundaryBrowser、T07CurrentTimeline20Records、BUSliceLoad（各名前缀Test）；跳过不计通过。写入／候选诊断曾默认跳过，后续显式运行通过，仅作成本定位。G2容量单独显式运行，不能用默认跳过关闭AC048。

未知COMMIT用真实Store.Within完成提交后注入确认丢失，验证原号查回和重放不新增来源；HTTP响应丢失由浏览器route.fetch完成真实请求后中断响应验证。它们是确定性故障注入，不是随机断网或人工验收。真实数据库与API重启另以全部原来源、成员采用、日志／Outbox及回执逐份核对。

运行成本中的Go CPU、RSS和分配量包括测试HTTP客户端／服务进程；PostgreSQL同步配置、连接池和关系存储占用另列，不将该进程CPU说成整机或PostgreSQL CPU。正式采样不启用pprof／SQL追踪；诊断只用于定位，不替代验收。没有执行正式quality／integration／browser CI，没有推送或合并。

## 8 收尾与本地交付

00017／00018保持原字节，追加00019／00020；[最终本地编号核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/migration-final-reservation.json)覆盖15个分支／跟踪引用及3个实际工作树，没有远端fetch。三轮评审方式为主代理顺序审查及自动化实测，首次失败均保留，不冒称独立人工验收。

T03／T04的领域、事务、权限、页面、历史、恢复、导出、容量及资源出口已通过；完整暂存差异、最终文档检查和本地提交按本次用户授权收尾，不推送或合并。保留环境为m03-grade-g2-d：Web4382／API8382／DB54372，账号admin／reader、公开开发密码admin1234567；设置→职务配置→职级／职级阶梯／职务，搜索G2-ACCEPT。4380／4381及其他已登记持久环境不变。

包含新增证据的首次完整暂存检查发现工具日志的行尾空白／末尾空行，详见[首次检查及原字节封存](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/raw-log-normalization.json)。原始日志以gzip逐字节保留；可读副本只规范空白，失败、次序与数值保持。310正文遗留的验收中状态同时与实际出口对齐。

最终产品、生成物、完整暂存清单、原始日志封存和文档检查由[交付复核](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/delivery-review-final.json)逐项记录；本地提交包含新增实现、测试、文档与全部本批证据。

<a id="review-fixes"></a>

## 9 T03／T04 交付后评审修复

本轮从固定分支 `wt-dev-d` 的 `8e529a8a205b376399dbb657abb4e9fe65da05f4` 开始。第1至8节记录初次交付。4382 仍运行第6节登记的构建，本轮验证使用临时实例。

| 评审发现 | 修复与回归断言 |
| --- | --- |
| 大写职级 UUID 通过领域校验，但遗漏新成员的共享守卫 | 校验与锁计划共用同一规范化函数。Job／Ladder 分别覆盖新建、更正及与 Grade 停用的两种先后顺序。 |
| 提前生效可隐式改变区间内的职级集合 | 比较新旧两组时间边界，校验受影响集合的读取权和写权。缺少成员写权时，成员不同的边界前移、后移均返回 403 并保持原来源，成员相同的相邻期间仍可单独调整日期。 |
| 消费者派生读取行缺失时可漏判在用职级 | 从当前头及封存成员逐条核对启用期间。分别移除临时库中 Job／Ladder 的读取行，停用预览和实际提交均返回 409，原来源保持。 |
| 初始日期无职级历史权后，切到有权日期仍无法选择 | 按新期间重新查询。以仅有当日职级读取权的用户从新建默认日切到当日，可选中职级并保存。 |
| 保存前撤销成员写权会丢失名称草稿 | 重读当前权限及对象，按同一主体、实例、动作、记录及来源恢复可写输入。浏览器核对再次保存只提交名称，原职级成员保持。 |
| 同名成员换序的日志原值、新值相同 | 日志显示获权的位次、编码和名称。浏览器比较原值和新值的顺序，并验证撤销源编码读取权后编码消失。 |
| 阶梯候选接口合同拒绝空的拟选集合 | 候选更正允许 `changes.gradeIds=[]`，写入继续要求非空。真实 API 验证空草稿可返回候选，JSON Schema 分别检查空数组、null、合法 UUID、重复 UUID 和无效标识。 |
| 重启核验只枚举了成功回执 | 逐笔核对原始反向拒绝和存储终态，按原提交号在数据库及 API 重启前后核对全部终态。拒绝回执必须保留失败原因且不产生业务来源。 |

### 9.1 容量实测与本轮验收决定

功能修复阶段的首轮容量共53组、5300笔正式HTTP请求，预热单列，业务语义错误为0。重启前后按原号找回4858笔终态，包含4756笔成功及102笔反向引用拒绝（正式100、预热2）。成功终态的准确来源均可读取，拒绝终态没有对象或来源。重启前后业务事实、采用和回执摘要一致。[首轮原始样本、运行成本及重启计数（gzip JSON）](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/review-capacity.json.gz)

| 20会话组 | 正式样本 | P95 | 目标 |
| --- | ---: | ---: | ---: |
| 职级阶梯普通保存 | 100 | 772.097 ms | 500 ms |
| 职级阶梯成员保存 | 100 | 740.787 ms | 500 ms |
| 职级阶梯100成员候选（管理员） | 100 | 567.863 ms | 500 ms |

2026-11-18 用户明确上述3组在本轮 T03／T04 验收中登记为非阻断。其余50组满足各自时延或正确性要求。测试保留原阈值、原始失败标识及非零退出结果，本次决定只适用于上述实测。

另一次单来源查询优化对比中，7个并发组P95超标，5300笔正式请求的语义错误仍为0，4858笔终态再次全部找回，102笔拒绝的原始HTTP业务回执与存储逐笔一致。性能对比用的代码已移除。第三次对比在造数阶段结束，未计入验收；自有临时数据库及卷已回收。[对比样本及重启计数（gzip JSON）](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g2/review-capacity-comparison.json.gz)

### 9.2 功能复验

首次新增的日期权限用例已收到403，但断言误用了下游字段错误码；按动作授权层实际返回的 `M03_FORBIDDEN` 修正断言后，两类对象均验证了原来源保持和同集合改期成功。候选与写入合同的10项JSON Schema断言均通过。

最终前端的 Chromium 专项15项通过（4项企业前置、9项职级专项、2项320×900管理员流程）。最后一次临时 profile 为 `smoke-486d6fa3940b`，Web 45141、API 47361、DB 47975，结束后已回收。本轮没有重跑初次交付的完整42单元矩阵，没有更新固定用户验收环境或运行正式CI。

最终代码执行 `make check`、`make build` 及 `git diff --check` 通过。真实PostgreSQL／API定向复验执行 `^TestStaffing(307ImmutableRebuildAndRestart|310)`，16个测试根通过；显式容量及诊断2根按开关跳过，容量实测另见9.1。文档首次结构检查指出两个三级标题缺少编号，补为9.1及9.2后复验通过。
