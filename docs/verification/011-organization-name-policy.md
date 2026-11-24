# CHNHCM-011 组织名称规则实施复测

登记信息：[registry.json](../governance/registry.json)。适用范围：[BR002](../specs/m02/011-m02-business-requirements.md#m02-br-002)、[013字段合同](../specs/m02/013-m02-technical-design.md)及[023名称案例](../specs/m02/023-m02-timeline-test-cases.md#m02-tl-011)，创建/新增/普通更正名称。按用户采纳的研究建议实施并自查，不冒充独立人工评审或业务批准。

## 1 验证对象与受测基线

2026-11-01，wt-dev-a，开始HEAD为`7c89dc2c9664e76470f92de6371e25c824b3c6d4`，工作区及stash为空。本次差异包含名称Go内核、NFC共享纯函数、精确依赖许可、同源API提示、页面、测试及合同。受测文件摘要与日志摘要见[执行清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/011-organization-name-policy/execution.json)，报告自己的最终提交号不替代脏工作区输入。

原第一批验证及其日志保留当时结论；其emoji可保存属于旧名称合同，不能推导当前准入通过。T01/T02后续及T04—T08范围不扩大。

## 2 环境与前置条件

Go1.26.0、Unicode15.0.0（Go分类、x/text v0.43.0 NFC及固定属性数据）、Node24、PostgreSQL18.6锁定镜像及桌面Chromium。任务新建独立空库profile `m02-name-current`，页面4406/API8406/DB54486；Compose项目chnhcm-bu-094e815539e1。原持久库、旧卷及相邻工作树保留，未改迁移或对旧库运行新迁移。

组织管理员从“设置 → 组织配置 → 业务单元”创建并维护；测试凭据只在仓外0700/0600位置或独立测试进程内。原4396环境仅在检查后重启自身API，沿用已有身份和数据库。

## 3 验证过程与结果

### 3.1 名称规则与业务流程

| 场景及参数 | 实际结果与证据 |
| --- | --- |
| 合法多语言、名称标点、数字及4字节汉字 | Go手写表覆盖华东事业部（一期）、AI研发、R&D/C++/#2、O’Connor、西安·、𠮷祥、重音及希伯来文字；合法名称保留，未用受测算法生成预期 |
| NFC与200/201码点 | 分解重音按手写预期归为预组合文字；200个分解重音原输入400码点仍可保存为200，201拒绝；全角和罗马数字不作NFKC改写。浏览器以198个中加𠮷文验证200可完整保存，201定位名称并保留输入 |
| emoji及非法名称 | 普通表情、旗帜、肤色、ZWJ职业组合、带/不带VS16键帽、心形、图形字母ℹ拒绝；普通数字不因Emoji属性误拒。控制字符、首尾/内部换行制表、Zl/Zp、私用/未分配/替代字符、孤立标记、纯标点及非法不可见拒绝 |
| 合法文字上下文 | 手写波斯文ZWNJ、透明标记间连接、印度文字virama后的ZWJ/ZWNJ、汉字/蒙古文字体变体通过；脱离文字或跨脚本连接、独立/重复选择符等反例拒绝 |
| 三入口、预览及原子拒绝 | Testcontainers真实DB/API在name/nameEn对11组非法值分别调用CREATE/ADD/CORRECT及预览；66笔保存拒绝可留技术终态，但身份、记录、修订、来源、成功日志和Outbox七表完整快照不变 |
| 规范等价与历史保护 | 正常NFC等价更正no_change；Go另验证已存分解形式的等价更正不重写原文。显式字段继续核验权限，未声明英文及后继独立快照不清洗，旧来源不重新规范化。真实API核对新增默认、原身份resolve、后续更正及原来源 |
| 正常菜单名称旅程 | Chromium真实页面先拒绝创建emoji并保留输入/聚焦；修正为𠮷祥Café研发（一期）及Société R&D后保存；新增拒绝键帽、修正后保存；英文更正拒绝职业emoji、改为O’Connor R&D后保存。刷新及原来源查询准确；页面提示由Go提供 |
| 既有M02回归 | 完整集成和33项Chromium同时覆盖创建/查询、资料新增/普通更正、当前撤权/字段权、原结果恢复、并发、离开保护、日期与返回上下文；启停夹具不计相应业务入口完成 |

### 3.2 原环境用户指定样本

在原4396环境以正常API对`TL_04a597a4`执行一次普通更正，未直接更新业务表：当前名称为198个“中”＋“𠮷文”（200码点），英文仍为空，记录身份和日期保持。当前来源从2追加为3；旧来源2的198个“中”＋“😀文”仍按原文只读。原来源/修订行摘要与操作前完全相同；非法emoji预览400且定位changes.name，精确重试和resolve返回同一个新来源。见[实际请求结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/011-organization-name-policy/live-sample.json)和[历史摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/011-organization-name-policy/history-protection.json)。

该样本仍供边界验证，人工正常名称旅程使用独立环境的NAME_前缀业务单元；没有批量清洗其他既有测试数据。

### 3.3 检查与案例映射

| 检查 | 结果 | 证据 |
| --- | --- | --- |
| make generate | 通过；离线Unicode属性、sqlc及OpenAPI客户端一致 | [生成](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/011-organization-name-policy/generate.log) |
| make check | 通过；46项文档工具测试、文档/边界、Go测试/vet、前端类型及差异检查；边界许可含限定正反例 | [检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/011-organization-name-policy/check.log) |
| make build | 通过；API、upgrader及页面 | [构建](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/011-organization-name-policy/build.log) |
| GOFLAGS=-v make integration | 通过；23项真实DB/API/事务及1项HTTP，容量测试1项显式跳过 | [集成](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/011-organization-name-policy/integration.log) |
| 独立profile make smoke | 通过；桌面Chromium 33/33，约2.8分钟 | [浏览器](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/011-organization-name-policy/smoke.log) |
| Go名称独立预期 | 通过；Unicode版本守卫、合法/非法参数、继承/旧文本及规范等价无变化 | [独立表](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/011-organization-name-policy/domain.log) |
| git diff --check、原报告/来源保护及最终文档检查 | 通过；原定稿报告字节保持，当前源码与归档摘要一致 | 执行清单、最终文档日志 |

TL-011：主G覆盖当前字符目录、NFC后1/200/201、文字/图形/上下文与旧文本参数；补A覆盖三入口、预览、NFC长度及无业务事实拒绝；补B覆盖菜单创建/维护、200/201、原来源与刷新。TL-012：主A补充两字段的控制/不可见/emoji拒绝及英文显式空值，补D由现有完整集成验证持久空值与原来源；无效UTF-8仍以既有API/Go断言为准，不把浏览器当编码错误主层。其余TL仅复用本批原已实现的回归范围，不新增关闭253案例的计数。

## 4 失败、修复与复验

- 初次专项集成夹具误用不存在的setup，编译失败；改为既有fresh。随后快照断言包含应保留的command_result，因66笔安全拒绝而失败；改为七张业务事实表全列不变，并单独断言66个rejected，未删除终态或弱化业务回滚要求。完整集成最终通过。
- 初次边界检查拒绝领域直接导入x/text；移到共享层NFC纯函数后，共享层原检查仍将其视为外部适配器。004明确仅unicode.go对norm的精确纯函数许可，新增其他文件、领域直接导入、网络/存储的拒绝测试，名称规则仍归本域。未开放通用第三方依赖。两次失败日志保留。
- 初次名称浏览器专项1/2：新增字段错误说明与顶部Alert同时出现，旧文本定位器匹配两处；改为精确断言名称字段helper的错误文本及焦点/输入保留。专项2/2及最终完整33/33通过，保留[初次失败](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/011-organization-name-policy/smoke-first-failure.log)。

暂存差异检查还发现Unicode原件自带行末空白及Testcontainers日志行末空白；原件改用固定ZIP保留完整字节，生成器仍核验原件SHA-256，属性产物未变；归档日志仅去行末空白。生成/检查复验通过后复用未变化的构建、DB及浏览器输入证据。

## 5 未覆盖范围与限制

不实施T04启停、T05删除、T06改日、T08联合/首条补全/分类纠错，T01/T02后续逐项保持。未运行T07全容量、Firefox、外部系统交换、GB18030符合性认证、所有语言正字法或全部IVD注册/字体显示验收。本批只校验已明确的文字上下文，不宣称完整Unicode标识符规范。

页面构建仍有约666KB主JS块大于500KB建议值的提示，构建成功，未据此宣称性能验收。无推送、PR、合并或对外部署。

## 6 结论与后续事项

用户采纳的名称规则及案例已落实到合同、Go、API/客户端及正常菜单，并完成受影响复测。旧emoji正例保留为历史验证，当前新输入拒绝。既有来源和原终态仍可按当前权限读取，不因本次收紧失效或被改写。后续业务批次沿用当前规则，不自动启动。

## 7 证据保存

本目录data下保存非敏感命令日志（归档仅移除行末空白，本地原输出保留）、TL参数映射、实际样本结果及受测输入/日志摘要。凭据、cookie、CSRF、trace及客户数据未归档；Unicode数据版权和固定来源摘要另见NOTICE、许可文件及生成脚本。原第一批报告及全部旧定稿验证报告未覆盖。
