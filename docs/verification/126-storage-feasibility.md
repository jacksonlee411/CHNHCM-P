# CHNHCM-126 完整历史存储可行性实验

登记信息：[registry.json](../governance/registry.json) `governance.verification_records`。
适用范围：[126草案](../plans/126-compact-history-storage-research.md)的空间研究、隔离原型与反例。
上游依据：[125](../plans/125-r6-capacity-remediation.md)、[013](../specs/m02/013-m02-technical-design.md)、[120](../specs/m02/120-dated-maintenance-technical-contract.md)。

## 1 结论与受测基线

2026-11-07，在固定wt-dev-b、HEAD `d6a544a63e1051c74177eac8c6020843eea86ce3`上研究。开始时只有两组既有未跟踪布局证据；本轮新增126草案、报告、研究工具和结果，更新登记/导航。未修改产品、迁移、OpenAPI或旧报告，未提交、推送或运行原125容量负载。

本轮由主代理执行，不作为独立评审或人工业务签收。

最后一个独占PostgreSQL候选在正常重启后为191,502,015字节，约191.50 MB / 182.63 MiB，含其表、TOAST、实际索引与系统目录。源容量库本轮为59,139,962,559字节，约55.08 GiB。此结果支持约200 MB的案例存储可行性，不代表现有业务系统已经迁移或完成容量验收。

后续正常重启的精确结果见[重启后整库统计](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/126-storage-research/candidate-size-after-restart.json)；系统目录可能出现少量页增长，不能只挑最小一次数值。[修改前状态](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/126-storage-research/before.json)及[研究输入摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/126-storage-research/evidence-manifest.json)标识本轮差异。

## 2 数据、运行环境及口径

### 2.1 实际数据覆盖

| 类别 | 数据来源与数量 | 本轮证明的边界 |
| --- | --- | --- |
| 背景BU | 2000对象，每个50来源、当前20成员 | 从真实API原型按原种子UUID规则构造；保留全部历史字段和内容字面值 |
| 背景部门 | 20000对象，每个50来源、当前20成员 | 同上；20000个管理BU目标均指向实际重建BU |
| 附加对象 | 原库实际导出的23个BU、2104个部门 | 包含深度100、扇出2000、单对象1000来源及其它现存样本；全部对象逐值往返 |
| 总对象/来源 | 2023 BU / 22104部门；101053 BU来源 / 1003153部门来源 | 与上述背景和实际附加对象组成一致 |
| 安全审计 | 原库全部36490条，逻辑JSON共822077241字节 | 落库后全量逐值比较，包含原压缩载荷字节 |
| 其它终态 | 两类`result_*_id IS NULL`均为0 | 无该类遗漏终态；没有据此删除未来需要支持的拒绝/无变更结果 |
| 辅助数据 | 实际会话、主体、凭据、授权、企业、存储模型及迁移登记共83行 | 仅在内存中直接写入隔离库；不写仓库文件。原辅助表和索引401408字节另有空间预留 |

重要限制：没有逐行复制约55 GiB原库。22000背景对象的全部身份和当前头逐项与原库一致，8个背景完整对象图与独立递归种子映射比较；未为每个背景对象重新读取所有源表。源表的`xid8`等物理事务元数据使用原型值，不能当作原库逐字节迁移。

背景的原请求摘要本来就是合成复制，沿R6种子原说明不支持真实请求重放证明；授权证据引用也复用原型，不能把约110万来源视为约110万次独立真实操作审计。本轮没有增加“可以原请求重放”的声明。原种子代码是[seed_copy.go](https://github.com/jacksonlee411/CHNHCM/blob/d6a544a63e1051c74177eac8c6020843eea86ce3/tests/capacity/r6/seed_copy.go)，当次对象头核对及数量见[源库清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/126-storage-research/source-inventory-final.json)。

### 2.2 环境和资源归属

源profile `r6-capacity-b-1106`开始时停机；只短时启动其数据库作只读导出，未启动API/Web，未复制/删除数据库。源连接设置只读和语句时限，每次提取均在finally正常停机。最终仍停机保留。

任务容器 `chnhcm-storage126-probe` 与卷 `chnhcm-storage126-probe-data`创建时标记临时；网络为none、不发布端口，限制2 CPU / 768 MiB。镜像沿仓库锁定PG18.6，`fsync=on`、`synchronous_commit=on`、`shared_buffers=128MB`；实际版本、镜像和设置见[环境](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/126-storage-research/probe-environment.json)与重启后统计。

使用Python3无损研究编码器、系统libzstd和Go1.26压缩格式还原探针。无数据库扩展、无列存服务、无外部对象存储。索引使用PostgreSQL原生B-tree与GIN。临时库只是物理布局原型，没有加载现有产品DDL、业务角色权限及全部触发器。

`pg_database_size`只表示本数据库。实验末尾集群文件逻辑大小约735.54 MB，其中WAL约520.11 MB；见[磁盘与重启](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/126-storage-research/restart-and-disk.json)。不能据191.5 MB宣称运行实例磁盘也只有191.5 MB，也不把本次多候选准备产生的WAL当成正常业务写入速率。

## 3 候选与实际结果

### 3.1 保留失败候选

| 检查 | 结果 | 判断与证据 |
| --- | ---: | --- |
| 每对象原JSON压缩 | 名义22000背景约835.80 MB载荷 | 超目标；仅载荷估算，[第一轮](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/126-storage-research/codec-first.json) |
| 每对象列编码压缩 | 名义背景约363.97 MB载荷 | 超目标；未计数据库页/索引 |
| 再共享常量字典 | 名义背景246.11 MB载荷 | 超目标，[字典候选](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/126-storage-research/codec-shared.json) |
| 共享结构、完整UUID GIN、独立当前投影 | 342.60 MB整库 | 未达到200 MB，[第一物理候选](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/126-storage-research/probe-size-first.json) |
| 当前投影复用值池、18位标签GIN | 234.50 MB整库 | 未达到目标，[第二候选](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/126-storage-research/candidate-size.json) |
| 审计外层内容共享，仍把内层压缩载荷当作不透明值 | 234.02 MB整库 | 收益很小，[首次审计共享](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/126-storage-research/audit-shared.json) |
| 19条压缩输入展开后共享，并验证原封装可精确重建 | 203.63 MB整库 | 接近目标，[审计精确保真](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/126-storage-research/audit-exact.json) |
| 标签改16位并重新核验完整UUID | 191.47 MB整库 | 本案例空间目标通过，[最终布局](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/126-storage-research/candidate-size-final.json) |

没有以最终结果覆盖早期失败值。最终大小不是将各段压缩率相乘所得，也不是`pg_dump`或gzip文件大小。

### 3.2 内容保真与查回

| 检查 | 实际执行及结果 | 结论 |
| --- | --- | --- |
| 背景身份、头及BU归属 | 2000 BU与20000部门身份/头匹配；全部20000个BU目标存在 | 通过，范围仅身份/头及关系目标 |
| 编码前后值 | 两类原型、全部2127实际附加对象完整字段往返 | 通过；源成员及路径没有裁减 |
| 落库对象/结构摘要 | 24127个对象表示、254个历史/当前结构摘要全部核验 | 通过，[落库还原](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/126-storage-research/persisted-readback.json) |
| 真实对象图抽检 | 8个背景对象及深度100、扇出末节点/根、1000来源、部门原型，共13个图从PG恢复并逐值比较 | 通过，[最终核验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/126-storage-research/candidate-validation-16.json) |
| 全部审计 | 1801内容块、571封套块校验；从PG还原36490条，与原导出全量逐值相等 | 通过，包含旧压缩载荷原字节 |
| 19条大压缩输入 | 解压长度、原SHA-256及Go同版重新压缩字节均一致 | 通过；不是仅对“判定通过/拒绝”做汇总 |
| 16位查回索引 | 1000个已有UUID均唯一精确匹配；平均16.36候选、最多31个 | 通过，仅SQL层查找 |
| 真实哈希碰撞 | 构造不存在但标签相同的UUID，13候选、0精确结果 | 通过；不会把标签命中当提交成功 |
| 损坏与截断 | UUID字节改动导致摘要不等；截断字面值解码拒绝 | 通过；不代表所有数据库约束已替代 |

16位索引的1000次暖态SQL p95为0.126ms，18位前一候选为约0.080ms。两组为本机单连接微探针，不含HTTP、完整身份授权、业务解码、冷读或20会话负载，不能与125列表p95计算改善比例。

### 3.3 分散数据反例与长历史限制

对每类对象各40个样本分别测试原样、独立提交摘要、再加不同名称和记录时刻，240个样本均完整往返。按相同22000背景规模外推，字面值及完整提交UUID的载荷分别为约108.96 MB、145.26 MB和221.46 MB；第三组尚未计结构、数据库页、索引、审计，已经超过200 MB。[敏感性证据](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/126-storage-research/entropy-sensitivity.json)是样本载荷外推，未运行三个完整物理库，也不是满足真实业务请求摘要规则的新压测。

该反例说明：此案例200 MB可行不能转化为一般客户200 MB承诺。相同历史数量的数据熵不同，底线成本不同。每条不可丢弃的新信息都需要空间。

当前原型的1000来源样本仍为整对象块，字面值203503字节，Python整史解码约0.8—0.91秒，当前投影约11—13ms。普通部门50来源整史解码约32—41ms；这些是解码探针，不是Go/API指标。未实现有界分块和直接来源定位，不能宣称确切来源500ms合同已通过。

## 4 实验修复与复验边界

准备阶段遇到两项脚本问题：PG初始化临时服务尚未退出便开始建表，连接报告shutting down；解码器把bytearray传入UUID导致类型断言。修正服务就绪判据和bytes转换后执行上述候选；不是产品缺陷或已通过的初次运行。

最初附加样本选择误用了深度/扇出命名，实际只重复检查了长历史和原型。随后明确选取`R6_DEPTH_099`、`R6_FAN_1999`、`R6_FAN_ROOT`补验，并在最终16位轮次纳入全部13个图；[首次结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/126-storage-research/candidate-validation.json)和[路径补验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/126-storage-research/candidate-path-validation.json)均保留。

首轮文档检查的73项脚本测试通过，但资料检查失败：收尾证据文件尚未形成，且数据附件中的Markdown复核说明未登记。保留[首次文档输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/126-storage-research/docs-check-first.txt)，完成真实收尾记录、将操作附件改为txt后复验；不将首轮记为通过。

脚本还明确了原始Go种子的UUID映射、JSON中的BU替换与完整主体范围，避免把部门BU映射到不存在的任意UUID。空间结果来自修正后的图，不用早期未入库的错误构造宣称通过。

## 5 未验证内容与技术风险

1. 未执行完整55 GiB逐事实迁移；背景内部物理事务元数据不等于原库，未运行对所有背景行的独立读取比较。
2. 未接产品存储端口、Go强类型编解码、API、当前授权、日志筛选、Outbox投递、完整结果恢复和浏览器正常菜单。
3. 未实现现行逐成员复合FK/期间排斥约束的替代、有界块、封口追加、损坏恢复、并发提交唯一性、跨实例隔离及维护后空间增长。原型分别校验字面值与结构摘要，未完成实例/对象/来源/字典/结构根的整体封套绑定；单实例字典也不能直接作为多实例授权边界。
4. 当前原型将辅助表内容存成JSONB并保留原空间预算，没有重建全部原辅助索引及鉴权运行表；实际生产结构还需实测。
5. 共享历史编码结构的收益依赖当前种子的重复模式；混合不同操作、时刻、自由文本、随机摘要及更大审计量必须重测。
6. 尚未验证20会话、冷启动、HTTP p95、写入WAL、崩溃恢复、备份恢复和完整API。正常容器重启不等于崩溃恢复验收。

因此本轮只关闭126的空间研究，S126-02—06均未实施；不关闭125或116的容量P1。若必要保护和查询索引使空间超过200 MB，须如实报告并重新评估，不把原型视为不能更改的指标承诺。

## 6 资源收尾与工作区保护

临时库完成正常停机/重启、数量和一个完整对象图核验后，按任务标签核对并删除容器和数据卷；辅助凭据副本随临时卷删除。原容量容器、数据卷及已有两个数据库未清理，保持停机。[资源收尾](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/126-storage-research/resource-cleanup.json)记录实际结果。

源库导出、构建二进制和实验临时文件在长期非敏感证据保存后回收；完整源案例继续由原持久库保存。本轮没有新建约55 GiB副本、改变数据库正式测量参数或使用源库进行业务写入。

交付执行`make docs`、研究脚本语法检查和`git diff --check`；具体日志见[文档检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/126-storage-research/docs-check.txt)及[最终保护核验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/126-storage-research/final-protection.json)。产品代码未改，未重复运行全量产品测试。两个既有未跟踪布局证据目录保留，索引、HEAD和stash保持。

## 7 证据与复核入口

[研究工具与复现边界](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/126-storage-research/reproduce.txt)、[工具摘要清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/126-storage-research/evidence-manifest.json)、[原型逐表数量](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/126-storage-research/bu-sample-summary.json)、[部门原型数量](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/126-storage-research/dep-sample-summary.json)、[实际附加对象/审计导出量](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/126-storage-research/export-extras.json)、[整库最终分项](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/126-storage-research/candidate-size-final.json)、[落库逐值比较](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/126-storage-research/persisted-readback.json)。原始完整导出不进入Git；重跑需原profile相同数据，不能只凭当前HEAD保证再次得到同样样本。
