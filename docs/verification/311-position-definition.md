# CHNHCM-311 岗位定义第一批实施与验收记录

登记：[registry.json](../governance/registry.json)。独立预期：[312 AC049—052](../specs/m03/312-position-contract.md#position-acceptance)。这是开发侧实测，不代表业务人员已经签收，不包含岗位层级。

## 1 基线与环境

2026-11-18，`wt-dev-d`，受测基线`89ebdd068cd7f469e2bd5fd8ee223e5eab5faa02`加本批工作区差异。开工时工作区与stash均为空，未切换分支。[候选清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/311-position-definition/candidate.sha256)记录55个变更的代码、迁移、配置及测试文件；可在仓根执行`sha256sum -c docs/verification/data/311-position-definition/candidate.sha256`复核。WSL2、Go 1.26.8、pnpm 10.24.0；DB/API用Testcontainers隔离的PostgreSQL 18.6，浏览器用可丢弃独立profile、正式前端构建和Chromium。岗位用例只用合成管理BU、部门、澳门地点、职务及职级，不创建人员或占用容量。

## 2 实现与实际业务结果

| 验收项 | 操作与证据 |
| --- | --- |
| AC049：未来引用 | J从2027-04-01移除G5；拟建2027-01-01起长期使用J/G5的岗位，候选排除，直接写返回409和2027-04-01。身份、来源、修订、Outbox零业务写入，原提交可查回拒绝。同一职务不指定职级可建；恰在04-01结束的岗位段仍可选G5，延长一天拒绝。见[Go边界](../../internal/contexts/staffing/domain/position_test.go)与[真实API](../../internal/bootstrap/staffing312_position_integration_test.go)。 |
| AC050：反向与竞争 | 先有岗位时，反向移除得到相同冲突日；先安排04-01清空岗位职级则移除成功。两独立管理员会话在提交前人为停在真实锁内，观察到另一事务等待；两种先后次序都只有先到者成功，败方不改变岗位头或职务Outbox。 |
| AC051：其他引用及准确历史 | 部门、职务和地点必填。实际停用部门、地点、Grade和Job分别因相交的岗位启用段被拒；非资格更名不受阻，当前展示新名而原来源仍显示旧名。撤销地点历史权后，精确岗位来源不再返回地点ID、原名或采用证据。岗位停用后同日可停用其地点。 |
| AC052：正常菜单 | 用管理员从设置 → 职务配置 → 岗位创建不指定职级的岗位，2027-02-01新增职级、取消草稿、同日更正、05-01停用，日志回看当次资料再返回。reader从同一菜单只读，直接写403。桌面、窄屏、短屏及私有X显示器的原生缩放分别验证；参照页是业务单元而不是截图。见[菜单旅程](../../tests/e2e/staffing312-support.ts)和[浏览器记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/311-position-definition/browser.txt)。 |

## 3 数据、恢复与集成

按对象保存独立编码、权威头、不可变来源、修订、回执和Outbox；资格在Go按完整岗位段决定。候选可用索引缩小检索范围，但保存和源端反查不依赖候选或读取投影。M02只通过公开端口与M03共享同一写事务。00021单独追加，不修改已交付迁移。用旧1—20库演进，原职务/系列各表和G2成员清单逐表摘要保持；旧账本、缺岗位身份表或禁用不可变修订触发器拒绝程序运行。没有用回滚迁移构造旧库。

在隔离库删除Grade读投影后仍能合法建岗；再移除岗位读投影，反向更正职务依然拒绝。由原来源重建投影，核对岗位来源数未增加。对照基线解析OpenAPI：新增22条岗位路径、15种岗位结构，原路径未移除或改变；只有四个共享Staffing结构增加了岗位字段。

## 4 首次失败与复验

未来G5领域反例先于判定实现，初次因缺少判定而不通过。首次路由检测发现候选路径与对象预览冲突，改为独立端点的`field`参数。地点前置最初选了不在本地支持范围的国家，改为澳门。停用测试初次锚到职务第一条而非最新未来条，得到400，而不是预期的期间409；改用最新锚点后验证了业务拒绝。候选分页初版把Grade的生成范围列照搬到Job，真实API返回503；按两种对象共同保存的起止日读取半开区间。首失败与各次运行结果分开保留，见[检查记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/311-position-definition/checks.txt)。

边界检查首次发现Organization公开合同依赖了Access公开包，改由Organization描述部门范围事实，Staffing再组装授权目标。故意移除投影的用例初次在破坏后才读普通职务详情，返回503而未执行预期写操作；改为先记住合法写入锚点，再破坏投影，分别验证保存、反向阻断及重建。

撤去管理员的地点历史字段权后，第一次尝试自2027-05-01停用岗位竟返回200；改为按受影响日期逐段核对新旧引用，复验返回403，恢复同一权限后可用原锚点停用。表单实测另发现补选地点后还显示上一笔必填错误；现只在该字段确有修改时清除旧错误，不将此视为新值已通过服务端校验。

扩展M02回归的第一次8分钟组合命令在全量区划目录初始化时超时；该组没有整体通过。仅回收其已核实归属的Testcontainers容器及匿名卷，单独用20分钟上限重跑`TestLocation204Acceptance`，全部10个子场景通过（443.62秒）；部门叶子互锁用例及最终岗位用例也分别实测通过。最终浏览器profile为`smoke-ecd9fbfb3718`（Web 47003、API 48081、DB 47655），结束已回收，不能作为手工验收地址。私有X11工具已回收；核对原4380／4381／4382及其API、DB仍监听。

## 5 范围与交接

生成、`make check`及`make build`均已执行；另以两片运行既有Staffing非容量功能，307/308共23根、309/310共26根通过，[回归摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/311-position-definition/regression.txt)列出命令及逐根结果。岗位菜单浏览器最终12项通过：两角色各五个100%视口，每角色一项走原生200%和400%，合计14个角色×视口/缩放单元。可对照[桌面新建](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/311-position-definition/desktop-create.png)、[窄屏新建](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/311-position-definition/narrow-create.png)和[只读400%查来源](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/311-position-definition/reader-native-400.png)；通过依据是菜单操作、日期/内容断言、键盘与滚动，而非截图数量。撤权后的期间写入由另行运行的真实API检验。

岗位层级发布、并发防环、旧版结构属于第二批待实施的AC053—055，本报告不判通过；容量／FTE、招聘预留及人员任职不在本批。未把定向测试当作全仓回归、生产容量或正式CI。没有提交、推送、PR、客户发布或人工签收。用户保留环境4380、4381、4382及其他既有环境不更新或回收；由wt-dev-d按原约定继续负责。

## 6 业务人员如何验收

验收机需要候选版本和单独的数据库。可按[017独立环境指南](../guides/017-local-bu-development.md#31-启动独立环境)给新profile分配未占用端口并用`up --fixed-web`启动，以工具实际报告的地址访问；不要把既有固定环境升级成岗位试验库。

1. 在本候选的独立实例中，以管理员打开「设置 → 职务配置 → 岗位」。用新岗位编码，填名称、所属部门、职务、工作地点，保留“未指定职级”即可保存。不需要人员或招聘申请。
2. 使用两组独立的职务和岗位。第一组先在职务安排2027-04-01移除G5，再建2027-01-01长期岗位：职级候选不含G5，不选职级可建。第二组先建使用G5的岗位，再安排同日从职务移除G5：应提示2027-04-01。先安排同日清空岗位职级，再改职务即可；回看截至03-31的岗位记录应仍显示G5。
3. 打开岗位，新增未来记录，再更正该条名称；取消另一笔修改，确认原值未变。停用后从记录历史查看原来源。切换到只读账号，同样能查已授权资料，但不应看到保存动作。
