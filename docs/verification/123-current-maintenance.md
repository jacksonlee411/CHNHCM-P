# CHNHCM-123 今日列表与记录详情调整验证

登记信息：[registry.json](../governance/registry.json)的governance.verification_records。适用范围：[123决定](../adr/123-current-maintenance-and-dated-query-boundary.md)、BU功能调整及部门合同同步。

## 1 基线与范围

2026-11-04，CHNHCM-wt-dev-b固定wt-dev-b，HEAD为999d26cadb16b0bc9bed1e8c3393a73203a810a8，开始时工作区和stash为空。用户授权完善ADR和相关文档，检查通过后完成功能调整。已有122和116评审修订均在基线内，旧报告及来源保持；本批不实施部门或122日志功能。

## 2 合同复核

已核对正常菜单入口、今日有记录/未来首条、默认/显式/原件定位、字段权限、当前授权、未来安排及跨日选择的合同。ADR把部门路径限定为列表今日/未来首日/停用前和详情记录期间分段，不保留隐藏任意日期输入；当前归属授权与Hierarchy历史引用规则保持。本次为主代理复核，不冒充独立评审或人工业务签收。

## 3 检查记录

### 3.1 文档先行与当前检查

首次make docs通过：73项Python测试和结构/链接检查通过。实施前语义复核发现012布局示意与字段表仍保留旧查看日期描述，清理后复跑通过，才开始产品调整。输出见[实施前文档检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/123-current-maintenance/docs-before-implementation.txt.gz)。后续同步023旧案例、017运行指南及018/026/110/113替代关系；019/100只追加当前行为通知，保持已关闭阶段，旧定稿报告和源资产原件不改写。

| 检查 | 当前结果与证据 |
| --- | --- |
| Go/真实DB/API定向 | [三项通过](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/123-current-maintenance/dated-final.txt.gz)：今日资料优先、未来首条、三种状态、CSV、显式/缺失记录、日期注入、字段裁剪及同请求跨午夜一致性；包含受控时钟浏览器 |
| 受控时钟真实浏览器 | [最终代码复验通过](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/123-current-maintenance/dated-final.txt.gz)：上海/洛杉矶各两次时刻、自然停用/恢复、旧选择跨日409且零下载、重新选择后的CSV及文件日期；宿主与浏览器时钟未修改 |
| 初次重点Chromium | [27通过、2失败](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/123-current-maintenance/browser-initial29.txt.gz)；失败及修复见3.2，不能把该轮记为通过 |
| 完整Chromium及补验 | [整轮163通过/11失败](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/123-current-maintenance/browser-full.txt.gz)，34.9m；11项均在后续范围检查中通过，[逐项对应](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/123-current-maintenance/browser-results.json)。合计174个去重用例分批通过，不是单次全绿 |
| 静态、单元与构建 | [make check通过](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/123-current-maintenance/check-final.txt.gz)，含边界、Go单元/vet、运行脚本及类型检查；[make build通过](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/123-current-maintenance/build-final.txt.gz)。构建保留依赖use client/sourcemap与体积提示，不记为零警告 |
| 完整集成 | 两个完整包分批通过：tests/integration 532.076s（[第三轮记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/123-current-maintenance/integration-third.txt.gz)），bootstrap 347.264s（[修订后完整复验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/123-current-maintenance/bootstrap-final.txt.gz)）。最后只调整bootstrap的测试屏障；tests/integration通过时的产品和本包输入未变。三次make integration失败记录保留，不记为单次整条命令通过 |
| 最终文档与差异 | [make docs通过](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/123-current-maintenance/docs-final.txt.gz)：73项脚本测试、文档结构和引用；git diff --check及新增文本空白检查通过；[固定分支及引用检查通过](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/123-current-maintenance/refs-final.txt.gz) |
| 日常开发环境 | [最终代码admin/reader通过](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/123-current-maintenance/daily-verification-final.json)：正常菜单、7个现有BU、用户指定JOINT_5ac455db、无日期入口、所选记录状态及只读空菜单清理；零BU业务写请求 |

### 3.2 发现、修复与复验边界

1. 初次make check在末尾发现CSS尾部空行，已删除；后续边界检查发现测试直接引用Organization内部，已把仅integration构建可用的时钟接线放到bootstrap组合根，不扩大生产API或修改依赖规则。
2. 明确记录读取503时原页面只给返回当前资料，不能按原记录重试。现保留准确URL重试；501明确不支持仍给返回当前资料，不循环重试永久拒绝。
3. 移除查看日期后，只读或目标缺失详情的空更多菜单一并删除；旧测试相应改查无按钮。未来首条页面断言改为所选记录尚未生效，API仍保留NO_FACT与未来记录的区别。
4. [初次完整集成](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/123-current-maintenance/integration-initial.txt.gz)的12个根测试仍向HTTP发送businessAt。现通过隔离夹具的Go时钟验证原日期边界，保留原事实预期，不在测试客户端偷偷还原被取消的接口；普通HTTP注入另验400。
5. [第二次完整集成](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/123-current-maintenance/integration-second.txt.gz)剩会话绝对过期等待失败：旧测试用宿主时间等待数据库截止时刻，改为由数据库自身时钟计算等待；不改变Access过期规则。
6. 全量浏览器暴露旧测试把返回高亮当作勾选、连续排序未等待状态转换，以及查询辅助误接收上一条件的迟到响应。分别改查独立返回高亮、等待排序变化、按确切查询词等待响应；不弱化结果断言。
7. 静态复核发现同一读取请求先后取两次当前日期的午夜竞争。应用层只确定一次D，向列表/详情/选择投影传入同一天，避免默认记录和未来提示分别取跨日前后日期。写入的锁后时钟规则保持。
8. [第三次完整集成](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/123-current-maintenance/integration-third.txt.gz)中tests/integration已通过，bootstrap唯一失败为旧并发读取夹具等待第二次取日。改成测试专用连接的QueryTracer，在真实头/列表SQL完成后暂停读取，另一个连接提交更正后才放行；原来源/成员/数量断言保持，[两个子项定向复验通过](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/123-current-maintenance/snapshot-retry.txt.gz)，不向生产加入测试屏障。
9. 早期浏览器读取辅助曾把selectedRecord填入viewFact，导致改日到未来时误报今日已有事实。现辅助只从真实时间轴找到记录ID，再发真实selectedRecordId请求，不合成viewFact；[未来改日复验通过](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/123-current-maintenance/browser-redate-retry.txt.gz)。日期边界的真实按日断言保留于Go时钟夹具，不把浏览器定位断言冒充按日解析验证。
10. 删除拒绝COMMIT失败后的refresh在整轮出现一次等待超时；未改变业务规则或放宽测试时限，[同组retry/refresh定向复验通过](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/123-current-maintenance/browser-community-delete-retry.txt.gz)。保留该次失败记录。

### 3.3 浏览器补验与最终输入

| 补验范围 | 结果 | 覆盖的整轮失败 |
| --- | --- | --- |
| [BU创建/查询、详情操作完整两文件](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/123-current-maintenance/browser-bu-detail-retry.txt.gz) | 33项通过，4.5m | BU的5项 |
| [Community查询与删除恢复](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/123-current-maintenance/browser-community-delete-retry.txt.gz) | 3项通过，58.9s | 查询1项、删除1项 |
| [固定启停与单条启停](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/123-current-maintenance/browser-lifecycle-retry.txt.gz) | 3项通过，40.0s | 启停3项 |
| [未来首条改日](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/123-current-maintenance/browser-redate-retry.txt.gz) | 1项通过，11.9s | 改日1项 |

这些补验含重复覆盖，不能与174相加宣称新用例数。完整轮的API在同请求单次取日修订前启动；最终Go已另跑跨午夜三项和两个完整集成包，最终UI对应修改有上述定向补验。未重新运行最终工作区单次174项，按影响复用其余已通过范围。[最终产品/测试文件摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/123-current-maintenance/final-inputs.json)用于复核最终修改，不能冒充整轮启动时的冻结输入。

## 4 实施与未覆盖范围

### 4.1 已调整功能

普通BU列表/详情和共享组件不再提供通用查看日期，普通GET及选择/导出DTO拒绝businessAt输入。列表仍每BU一行：今日有资料取今日记录，未来首条仍可搜索、筛选、选择和导出，状态为未生效。当前停用而未来恢复的对象仍显示停用。

详情默认今日记录或未来首条，时间轴、保存结果、准确来源与删除原件按记录身份定位；名称、状态和适用期间统一表达所选记录。未来提示由Go返回并受日期字段权限裁剪，头部当日状态退出。新增/更正/启停、源版本和内部按日解析保留，数据库及业务历史无需重建。

选择和CSV服务端每请求重判D与授权集合；跨午夜旧选择冲突后清空并重查，绝不自动纳入新集合。前后对象导航也核对集合版本。新合同、OpenAPI、生成客户端及实际调用同步，没有兼容日期字段或双格式。

### 4.2 案例与限制

023新增TL-387—393覆盖本批BU，[逐例参数与证据](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/123-current-maintenance/case-results.json)及[未来首条详情截图](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/123-current-maintenance/future-first-detail.png)可复核；实际日期、角色、层级和失败以本报告及对应日志为准，不把编号映射解释为所有参数全通过。TL-394部门分段路径仅合同：119/120/121定义首日安排、记录期间分段、停用前路径和确切来源证据；部门/Hierarchy仍留116 R4/R5，R1—R6未因此完成。

122操作摘要/日志仍draft/not_started，TL-367—386仍未运行。专项组织快照、员工快照、外域消费者、容量/生产SLA及生产恢复不在本批。未运行新的CubeBox运行对照或Windows Edge；旧原型和旧报告不作新产品通过证据。本轮为主代理实施/复核，非独立评审或人工业务签收。

## 5 环境与资源

make smoke使用本任务的temporary profile，成功或失败均由脚本回收容器、网络、数据卷与私有配置；真实DB测试由各自Testcontainers夹具回收。[最终资源核验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/123-current-maintenance/resources-final.json)确认本任务临时profile、配套网络/卷均无残留，本轮日志记录的219个独立集成容器ID均已不存在。本工作树只保留m02-current-b运行，其余既有持久环境和数据保持。

日常m02-current-b复用原持久环境，保留数据；[最终API同步](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/123-current-maintenance/daily-restart-final.txt.gz)及[入口归属核验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/123-current-maintenance/daily-owner-final.txt.gz)后实际入口为http://127.0.0.1:4382（API8382、DB54373），两角色正常菜单核验见3.1。保留运行供用户复核，责任为wt-dev-b本轮实现维护者，下次开发/验证交接复查。未操作a工作树环境。实施验证结束时尚未提交；随后用户授权将123的ADR、相关文档、产品调整与验证证据形成独立本地提交。提交时产品与测试的61个文件摘要与前述最终输入一致，原分批验证结论保持；推送、PR、合并或部署不在本次范围。
