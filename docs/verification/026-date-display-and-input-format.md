# CHNHCM-026 日期格式统一验证

登记信息：[registry.json](../governance/registry.json)。适用范围：共享日期显示、输入及当前M02；上游为[026](../adr/026-date-display-and-input-format.md)、[012](../specs/m02/012-m02-functional-and-interaction-design.md)和[013](../specs/m02/013-m02-technical-design.md)。

## 1 验证对象与受测基线

2026-11-03，wt-dev-a开工HEAD为f751175b1824cb24792928e51c9edb5abd160f21，工作区与stash干净。受测内容为该基线加本轮日期显示/输入、测试及文档差异；最终[输入摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/026-date-display-and-input-format/input-sha256.json)及[源文件归档](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/026-date-display-and-input-format/runtime-inputs.tar.gz)标识本轮差异；完整首轮另保留当时输入。

## 2 环境与前置条件

make smoke使用独占可丢弃profile、当前模型空库、真实API及桌面Chromium。组件夹具和正常菜单证据分开记录；凭据只在仓外，不修改已有持久环境。

## 3 验证过程与结果

| 检查 | 实际结果与边界 | 证据 |
| --- | --- | --- |
| make check | 已通过；Python文档/治理、Go单元/静态、边界、6项Node运行检查、前端类型与差异检查；不等同完整DB集成 | [输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/026-date-display-and-input-format/check-first.txt) |
| make build | 已通过Go构建、前端类型及Vite生产构建；保留原有大chunk提示，不扩大为部署验收 | [输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/026-date-display-and-input-format/build.txt) |
| 显示函数 | 纯日期保持0001/9999及闰日原日；北京时间跨UTC午夜、秒和无效时刻检查通过 | [Node输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/026-date-display-and-input-format/runtime-tests.txt) |
| 首轮浏览器 | 5项为3通过/2失败，失败为新用例误用当前Playwright不存在的toBeInvalid断言；正常菜单1项、日期轴/跨页签2项通过 | [原输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/026-date-display-and-input-format/browser-first.txt) |
| 组件与正常菜单复测 | 更换为读取浏览器validity.valid后3项通过；zh-CN/en-US输入及日历回填、闰日、0001/9999、无效/空值、只读/禁用、min/max；真实菜单创建/更正、查看日取消/应用及确切来源 | [输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/026-date-display-and-input-format/browser-component-retest.txt) |
| 完整Chromium首轮 | 137项为136通过/1失败；唯一失败仍要求日期弹窗常驻旧中文回显。整轮按失败保存，不计单次全绿 | [完整原输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/026-date-display-and-input-format/browser-full-first.txt.gz)、[遮罩截图](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/026-date-display-and-input-format/full-first-failure-masked.png) |
| 最终受影响复验 | 更新旧回显断言为ISO文本、占位、无重复说明及不完整日期反馈；另补两locale的320/390宽度检查。两测试文件共8项全部通过，包含完整首轮的失败用例 | [输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/026-date-display-and-input-format/browser-final-focused.txt)、[中文320图面](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/026-date-display-and-input-format/component-zh-CN-320.png) |
| 最近结果及复用边界 | 完整轮后运行代码未改，仅两份测试变化，8项均复测；其余129项按不变输入复用。137项最近结果通过，非单次137项全绿 | [汇总](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/026-date-display-and-input-format/checks.json) |
| 临时资源收尾 | 四轮独占profile的容器、网络、卷、仓外凭据/配置及临时构建已回收；44个原容器状态及挂载保持 | [核验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/026-date-display-and-input-format/cleanup-final.json) |

## 4 失败、修复与复验

首轮类型检查发现MUI通用htmlInput未直接声明min/max，收窄为HTMLInputAttributes后复验通过，见[首轮](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/026-date-display-and-input-format/typecheck-first.txt)及[复验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/026-date-display-and-input-format/typecheck.txt.gz)。首轮浏览器新用例误用toBeInvalid/toBeValid，改为轮询真实浏览器约束有效性，原业务步骤与断言目的保持；复测3项通过。

完整首轮唯一失败是ui-improvements沿用旧的MuiFormHelperText常驻断言。改为验证可见ISO文本框、YYYY-MM-DD占位、正常时无重复回显、不完整日期有错误反馈并禁应用，恢复合法值后错误移除；不回加旧说明来迁就旧测试。最终8项复验同时补充390/320宽度的文本完整显示及无横向溢出，源代码未改。

提交前暂存差异检查发现两份原始日志的尾随空格/EOF空行，已原字节gzip归档并核对解压一致，未修改运行代码或测试源码。归档引用的重复扩展名已修正，保留[引用检查输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/026-date-display-and-input-format/docs-archive-link-first.txt)，最终重新核验。

末轮文档首检在证据归档完成前执行，发现最终输入清单/归档及检查记录尚未生成；已生成文件后重新检查，保留[首检输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/026-date-display-and-input-format/docs-evidence-order-first.txt)，不以文件存在代替检查通过。

完整回归前已冻结[输入摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/026-date-display-and-input-format/full-first-input-sha256.json)及[源文件归档](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/026-date-display-and-input-format/full-first-inputs.tar.gz)。早期类型/断言失败保留原输出，未逐字冻结全部早期测试源码；不宣称首轮全绿，不用新格式改写旧定稿报告。

## 5 未覆盖范围与限制

不改Go业务内核/API/DB格式，不重跑完整DB集成、容量、受控午夜专项、Firefox、真实消费者或生产恢复，不触发远程CI。浏览器业务流程使用真实API/DB，但不等同全M02验收。组件选日检查覆盖按钮调用及原生输入回填，未承诺跨操作系统原生弹出面板的视觉一致性；未运行真实200%/400%缩放矩阵。历史报告和截图保留原结果。

## 6 结论与后续事项

026格式决定、012交互、013纯日期/时刻边界、100局部替代说明及AGENTS/导航已同步。当前页面统一完整ISO日期，记录时刻保留时分秒和北京时间；没有修改业务历史、协议、日期算法或存储。

make check/build通过；浏览器最近137项为129项不变输入复用加8项最终复验，首轮136/1结果原样保留。本次为主代理实施后自查，非独立评审或人工批准；没有把组件夹具计为业务完成，也不声明全M02完成。最终文档及差异检查结果见[记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/026-date-display-and-input-format/docs-final.txt)。

四轮临时环境均已回收，无本任务保留运行环境；原44个容器状态和挂载核对一致，未对持久环境执行启停、迁移或清库。原任务环境继续由各自维护者负责，在下次所属任务交接/新建环境时复查。没有提交、推送或部署。

## 7 证据保存

非敏感本机输出位于.local/date-format-20261103，必要原件和输入快照已保存到本报告关联目录，入口为[最终汇总](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/026-date-display-and-input-format/checks.json)。四份组件图面仅说明组件在zh-CN/en-US及320/390宽度下的输入布局，不代替正常菜单业务证据。不复制仓外状态、凭据、Cookie或数据库文件；旧定稿报告和来源资产不改写。
