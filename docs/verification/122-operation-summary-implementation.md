# CHNHCM-122 操作摘要与变更日志实施验证

登记信息：[registry.json](../governance/registry.json)。承接用户“批准122，请完成122的实施”，关联[122计划](../archive/plans/122-bu-operation-summary-and-change-log.md)、011—013及023第25章。原合同评审、修订和123后复核报告保持原时点与原字节。

## 1 基线与授权

2026-11-04，固定wt-dev-b，基线`9ec5d53124f458d1ae44315633875cd5e154485d`，开始时工作区与stash为空。本轮完成122的P1—P3：身份封存、所选资料摘要、表格日志及筛选/分页/原件链接、真实验证和交接。未扩大到116 R1—R6、部门、生产或完整M02；没有提交、推送、PR、合并或部署。

当前输入是该基线加工作区差异，最终文件摘要见[input-files.sha256](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-implementation/input-files.sha256)。[基线清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-implementation/baseline.json)及[保护复核](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-implementation/preservation.json)确认2541份既有来源/验证资产未改变；旧通过结果不计作本轮新结果。

## 2 实施与环境

### 2.1 基本资料及身份

入口为“设置 → 组织配置 → 业务单元 → 详情”。正文大块来源说明已移除，字段后显示次级文字“操作者：姓名（账号） 操作时间”。摘要归属于所选业务修订的形成操作，不跟随BU最新来源；期间派生修订归因实际形成操作。未来首条没有今日事实仍显示摘要；显式缺失记录不自动换条。

X03公开端口提供经过认证的身份，00004迁移新增最小姓名/账号/类别快照，与新来源在原事务中封存。身份失败整笔回滚；客户端伪造身份或时间被拒绝。旧封口来源只显示原主体标识，未从当前账号补造历史姓名。改名、改账号或停用不改既有快照；服务主体明确标注，缺可选姓名仍显示账号。

### 2.2 日志及权限

语义表格按操作分组，默认显示获权字段原值/新值，标明直接修改与随期间变化。来源明细在组内展开，当次资料、删除前原件、删除后版本均为真实链接。删除证据入口保持墓碑修订，摘要准确指向删除前业务修订；固定两条分别归因。

复用原筛选对话框的应用/取消/清除，增加日志字段目录；Go裁剪后筛选、计数和分页，返回前后记录上下文，客户端不计算业务差异。20/50/100按操作分页，历史上下文与固定上界绑定URL。返回恢复条件、页码、上界、滚动和焦点，新标签不改原页。

摘要只要求当前bu.read，历史日志和确切来源仍要求bu.history.read；字段目录取两种读权交集。撤权清除正文和旧日志条件，重新读取可保留合法摘要；非法条件可重置。取消和会话检查阻止迟到结果覆盖新页面。

### 2.3 存储与实现

查询采用一次来源/修订流加固定授权与窗口查询，不逐行请求详情。revision_source索引和LATERAL定位避免长历史关联计划膨胀。Organization持有自己的公开身份投影，由基础设施接X03端口，不增加跨域公开包依赖或API对身份表的更新权。只有一套当前DTO、生成客户端和页面。

持久环境m02-current-b执行增量迁移，没有重建或清空。九类组织业务表行数与旧字段摘要前后相同，含38个原来源，见[迁移前](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-implementation/persistent-before.json)/[迁移后](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-implementation/persistent-after.json)。

## 3 验证结果

### 3.1 检查结果

完整输出和各轮状态见[验证清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-implementation/validation.json)，失败与复验分别保存。

| 范围 | 实际结果与边界 |
| --- | --- |
| make generate、make check、make build | 生成、73项Python及运行工具检查、Go单元/vet、148文件导入边界、TypeScript及构建通过。既有大于500kB分块提示仍在；构建不代替业务验收 |
| 完整真实DB集成 | tests/integration最后整包498.523秒通过；bootstrap整包359.033秒通过。来自分列记录的运行，不能将此前失败的make integration改写为整轮通过 |
| 122独立DB/API专项 | 4个测试50.408秒通过，含S/P/F/H45/H1000、身份故障、停用操作者、跨实例来源、差异与上下文分离及全部日期操作符。完整整包不冒称包含后加断言 |
| Chromium受影响回归 | 113项首轮108通过、5失败；定向7项及最后6项均通过，覆盖全部5个失败场景。最后6项另含条件重置、撤权恢复和原生200%缩放；不宣称113项同轮全绿 |
| 正常菜单新功能 | operation-summary与operation-permissions各3项通过：准确归因、两类删除原件、未来首条、H45分页筛选、新标签/返回、真实权限及在途响应失效 |
| 日常环境 | 管理员与查询者经正常菜单读取原资料及日志，旧来源显示主体标识，业务写请求均为0；见[persistent-ui.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-implementation/persistent-ui.json) |
| 原资料保护 | 2541份原证据不变；持久组织事实不变；只追加122采用证据，不提升来源资产整项状态 |

### 3.2 案例与独立预期

[逐例映射](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-implementation/case-matrix.json)覆盖TL-367—386及实际层次、样本边界。S用两操作者与三记录验证旧修订不随头变化、派生归因、删除原件及同日新身份；P两原件各属不同操作；F区分无今日事实与所选资料。预期未调用产品差异算法生成。

H45从无英文名创建，45次操作中英文差异恰15组，操作者23/22，20分页为20/20/5，50/100均一页；S5仅改名称，不因上下文有日期/原因命中对应差异筛选。非法/重复条件、越权字段、未知/跨BU/跨实例来源、历史上界越界分别拒绝。合法空值、未登记与无权省略分别核对。

主层规则与补充层代表旅程分别执行，不声称每个参数都在所有层重复。分类取封存标签，投影测试验证不读取当前目录；当前没有可修改目录的产品入口。既有联合来源沿同一只读格式读取，由固定启停对和既有联合回读测试覆盖多记录结构，不重开旧通用写入口。

### 3.3 展示与视口

实际覆盖1920×1080、1366×768、390×844、320宽和原生Chromium200%。390使用CSS视口仿真；200%使用独立Xvfb中的原生缩放，1366×768变为683×384 CSS px/DPR2，二者不混记。长姓名及字段值换行、键盘展开/收起、筛选返回焦点通过。见[视口记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-implementation/viewport-matrix.json)、[桌面](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-implementation/log-desktop.png)、[无历史权摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-implementation/summary-without-history.png)、[200%](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-implementation/native-200.png)。

### 3.4 长历史预算

H1000由真实API写入1000来源，英文差异333次。测量20/100分页、尾页、身份/差异筛选及两轮20并发，见[性能输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-implementation/logs/budget4.log.gz)及最后专项。保留轮单请求约180—260ms；20并发冷连接池p50/p95/max为2.310/3.418/3.544秒，暖连接为0.928/1.755/1.836秒，未超当前5秒读取预算。未清OS/PG页缓存，不称冷数据库缓存，不把500ms确切来源读取参考目标外推为聚合日志已经达标。

同轮Go进程堆采样4213568→5785160字节，累计分配增量305848312字节，含API和HTTP测试客户端，不含PG，也不是峰值保证。真实阻塞SQL下取消至PG事务及连接释放约15.7ms；无后台继续查询。环境及PG配置见[environment.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-implementation/environment.json)。本轮独立于旧列表容量，属于本机工程证据，不是生产SLA。

## 4 失败、修复与复验

早期身份查询FOR SHARE需要额外更新权限，改为原授权守卫覆盖下读取并原子封存，未扩大权限。长日志宽关联超预算503，增加索引和LATERAL定位后H1000通过。公开包跨域引用、处理器复制含锁对象由边界/vet发现并修正。

H45初次浏览器夹具填写Alpha导致16组英文差异，改为空英文创建后复验15组。旧UI测试由按钮/串接文本/链接序号改为真实链接、前后值单元格及确切来源记录定位，未降低业务预期。StrictMode取消请求可能先消耗503注入，改为持续故障至实际观察再释放。筛选清除失焦及选项可访问名称已修复。

113项期间两次白页/登录等待与产品编辑时段重叠，但证据不足以认定根因；停止产品编辑后两项分别复验通过。一次集成鉴权存储故障恢复仍503，孤立连续3次和最后完整整包通过，未将偶发原因写成已查明。准入测试原用迁移4表示未知版本，004存在后改999并完整复验。原生200%普通截图发生可视区裁切，改用CDP无clip截图并验证1366×768物理像素、新值和查看链接在视口内，单项15.1秒复验通过；产品源码未改。原失败输出均保留，不追溯提升较早证据。

## 5 交付、限制与资源

122限定P1—P3完成，011—013、023、导航、登记和三类采用记录同步。本文为主代理实施验证，非独立评审或人工业务签收。116后续共用提炼与部门接入继承实际合同，并分别重新验证。

本次只验证本地Chromium与锁定PostgreSQL，未验证Windows原生Edge、Firefox、客户升级、生产恢复或真实下游消费者。并发结果不能替代生产评估；旧历史缺姓名显示主体标识属于约定行为。

临时smoke profiles及其容器、网络、卷和私有配置自动回收，Testcontainers及原生缩放的浏览器/Xvfb由cleanup/finally释放，见[resources.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-implementation/resources.json)。其他工作树及既有停止环境未修改。保留m02-current-b供手工体验：Web `http://127.0.0.1:4382`、API8382、DB54373；本工作树实现/验证维护者负责，在用户结束本轮使用或下次版本交接时复查是否停机，持久卷默认保留。

## 6 证据保存

[data/122-implementation](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-implementation)保留逐例映射、环境、保护摘要、实际截图和压缩日志，不保存Cookie、CSRF、数据库密码或客户资料。报告不递归纳入自身摘要；基线加最终源码/合同/测试摘要明确输入。最后文档、引用和diff检查属于结构验证，不代替第3节产品证据。
