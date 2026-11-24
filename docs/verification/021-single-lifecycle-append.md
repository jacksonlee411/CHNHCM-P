# CHNHCM-021 单条启停末尾追加与表单一致性验证

登记信息：[registry.json](../governance/registry.json)的同路径verification_records。适用范围：单条停用/重新启用位置、能力、默认、日期、直接反馈及保护。依据[BR008](../specs/m02/011-m02-business-requirements.md#m02-br-008)、[BR018及AC015](../specs/m02/011-m02-business-requirements.md#m02-ac-015)、[012](../specs/m02/012-m02-functional-and-interaction-design.md)、[013](../specs/m02/013-m02-technical-design.md)、[现行TL343—348](../specs/m02/023-m02-timeline-test-cases.md#m02-tl-343)。旧T04、104—109报告保留原结果。

编号勘误：原来源分支的TL323—328现映射为TL343—348，业务内容与执行结论不变；[原报告逐字节保存](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/023-parallel-case-identifiers/original-report.md.txt)，映射与摘要见[023编号记录](023-parallel-case-identifiers.md)。

## 1 基线与业务范围

2026-11-03，wt-dev-a，HEAD 94dd06466ffc44ee233e600d99f3bb82218a143b及本轮未提交差异。管理员从设置→组织配置→业务单元→当前来源详情选中末条，在更多操作办理单条停用或恢复。Go按末条生效后状态开放对应动作，日期晚于末条，默认max（服务端今天，末条次日），恢复名称进入即带出，原因留空。中间空档仍用固定成对操作；单条恢复可确认或调整名称，不恢复手动预览。

单条DTO及自动辅助携带anchorRecordId；事务内重验准确来源、末条位置、状态及完整候选。自动辅助业务错误就地显示并禁止确认，修改后重验；网络辅助失败仍允许交给保存事务校验。初始默认不算修改，真实编辑、来源冲突、撤权和未知结果继续保护。012的“多目标另需联合入口权”残留已删除；021适用说明、010、017和文档导航同步。

## 2 环境与资源归属

独立profile m02-single-lifecycle-a，Web4550/API8550/DB54650，项目chnhcm-bu-ccd2cbd8bef0；真实PostgreSQL18.6、Go1.26.0、Node24.19.0、Playwright1.55.1桌面Chromium。使用新合成样本及Testcontainers隔离库。公开本地账号遵循017，数据库凭据仍在仓外。

未在4382或4527写入业务，也未更新其程序。4382页面及8382 readiness在本轮前后均返回200；未对原库作全量数据核验，不能据此声明其全部业务内容已复验。原环境继续由原工作树维护。

## 3 验证过程与结果

| 检查 | 实际结果与证据 |
| --- | --- |
| make check | 通过，含文档/脚本、Go单元与vet、边界、前端类型及diff；[完整输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-single-lifecycle-append/check-complete.txt.gz) |
| 报告/登记收尾检查 | make docs通过；[最终文档输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-single-lifecycle-append/docs-final.txt.gz)，结构通过不替代业务验收 |
| make build | 通过；[完整输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-single-lifecycle-append/build-complete.txt.gz)，保留原MUI/打包体积提示，未改阈值 |
| 完整真实DB/API默认套件 | integration完整复跑通过481.204s；[输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-single-lifecycle-append/integration-final.txt.gz)。bootstrap在完整首轮通过256.439s，其后Go/API/SQL及该套件输入保持；[首轮原输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-single-lifecycle-append/integration-all.txt.gz)保留另一包的失败，不把整条首轮命令记通过 |
| Chromium受影响45例 | 扩大轮44通过/1旧前置失败（8.1m）；修正唯一样式前置后，该例定点通过（13.3s）。[扩大轮](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-single-lifecycle-append/browser-final.txt.gz)、[定点通过](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-single-lifecycle-append/browser-layout-retest3.txt.gz)、[逐例最近结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-single-lifecycle-append/browser-latest-results.json)。不是单次45项全绿 |
| 受测输入 | 238份当前程序/合同/测试输入哈希；唯一末轮差异为样式用例前置及原断言位置。基线加24份差异文件重建哈希全部一致；[重建核验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-single-lifecycle-append/input-verification.json) |
| 主代理自查 | 当前规则、请求绑定、权限、配对入口、查回和文档互相核对，无剩余已知发现；[记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-single-lifecycle-append/review.json)，非独立评审 |
| 环境收尾 | 本任务Web/API及DB停止，三个端口无监听，容器exited 0、卷及网络保留；[核验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-single-lifecycle-append/cleanup-verification.json)、[原输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-single-lifecycle-append/cleanup.txt.gz) |

| 场景 | 真实正常菜单/API/领域证据范围 |
| --- | --- |
| 所选末条可办性 | 非末条没有单条启停，当前末条只开放符合生效后状态的一项；API拒绝旧锚点在末尾追加，拒绝可精确重放；读者/旧来源没有入口，配对权限独立 |
| 初始值、日期边界与未来状态 | 默认今天或末条次日；未来首条/停用及恢复、闰日/月末/年末/9999上界；恢复名称进入即带出，原因留空；同日/早于末条及缺锚点拒绝 |
| 就地反馈及取消 | 越界业务错误自动显示、聚焦并禁确认，保留输入，合法修改后直接保存；初始取消无提示，真实编辑可继续/放弃；单条恢复仍能调整中文及清空英文 |
| 冲突、权限、历史及结果未知 | 新来源后重读仍绑定原锚点，原锚点变非末条时禁存；明确重选末条后可办；字段撤权清受限缓存并忽略迟到响应，保留可读输入；未知按原身份查回，原来源及恢复独立快照保持 |
| 相邻回归 | 位置新增/插入、固定成对插入/删除、辅助503仍直接保存、日期不可读但可写、详情布局和日期弹窗；详情用例含1440桌面、390窄屏及既有短屏步骤，未新增全视口/缩放专项 |

## 4 失败、修复与复验

首次环境启动的输出目录未先建立，重试前未发生业务写入；随后Docker默认网段池耗尽。仅为本任务创建带准确Compose标签的10.245.50.0/24网络，再正常启动，未清理其他项目网络或数据。初次文档检查发现章节应为21及登记须有实际结果项，已修正；旧失败输出保留。

首轮定向真库发现两处联动：成对入口复用单条能力导致误关；新anchorRecordId拒绝位置未进入结果解码白名单，精确重放返回503。分别改为独立核验配对双方权限、允许单条新位置诊断查回。次轮只剩空恢复changes的报错优先级，保持格式拒绝400，位置重验仍在事务内。两轮原输出分别见[首轮](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-single-lifecycle-append/integration-targeted.txt.gz)、[次轮](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-single-lifecycle-append/integration-targeted-round2.txt.gz)。之后完整默认DB/API验证通过。

完整真库首轮integration只因T07编码保护夹具沿用缺锚点请求失败，补为当前DTO后完整复跑通过；未改编码保护断言或产品代码。首轮bootstrap通过，整条首轮命令仍按失败保存。

浏览器首轮24项为22通过/2失败（4.7m）：旧成对恢复夹具未选末条就要求停用；新案例辅助查询漏传日期。[原输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-single-lifecycle-append/browser-round1.txt.gz)与[输出附件](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-single-lifecycle-append/browser-round1-output.tar.gz)保留。扩大45项为44通过/1失败，旧样式例未先停用便打开恢复；定点过滤首次未匹配测试，记未执行。补停用后又发现原例末尾重复要求停用，实际仍为停用状态；将原停用样式及取消焦点断言移到启用阶段，保留全部原断言，并真实保存停用后再检查恢复。最终该例通过，原[第二次失败](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-single-lifecycle-append/browser-layout-retest2.txt.gz)及[当时源码](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-single-lifecycle-append/layout-retest2-source.ts.gz)保留。

## 5 未覆盖范围与限制

不声明全M02、284现行案例全部层次/参数、全浏览器、容量、真实消费者或生产恢复验收。未运行Firefox、独立受控时钟浏览器和新增全视口缩放专项；默认套件中环境开关关闭的专项不计通过。首次修复过程中的早期失败未逐次形成完整源码快照，保留原始输出与修复说明；本轮冻结输入及扩大浏览器轮有精确清单和差异档。

浏览器45项是扩大轮44例加唯一修正前置的定点1例，最新逐例结果不等于单次45项全绿。Go/API/SQL、客户端和页面在扩大轮后未再变化；最后仅改第一详情样式例的准备与原断言位置，其余44例的执行逻辑保持。

## 6 结论与资源收尾

本轮实际范围内的单条启停位置、准备、反馈及保护完成；旧单条业务含义及配对操作差异保持，未改业务历史格式或迁移。主代理实施后复核，非独立评审或人工批准。分支HEAD仍为94dd064，本轮未提交、推送、PR、合并或部署。

自有profile已停止，PostgreSQL容器exited 0，4550/8550/54650均无监听；持久卷chnhcm-bu-ccd2cbd8bef0_database和本任务网络保留供复现，不保留运行服务。恢复按[017](../guides/017-local-bu-development.md#local-runtime-cleanup)设置同一profile，再执行local-bu.py up。原4382仍可用，由原工作树维护，在该环境下一次任务前复查；本次修复未发布到4382。

## 7 证据保存

[当前输入清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-single-lifecycle-append/tested-inputs.json)与[最终差异输入](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-single-lifecycle-append/changed-inputs-final.tar.gz)基于第1节提交重建238份受测文件，已逐字节哈希核验；归档只放24份差异文件，未重复打包基线已有的大型来源资产。[扩大轮输入](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-single-lifecycle-append/browser45-inputs.json)与[扩大轮差异档](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-single-lifecycle-append/changed-inputs-browser45.tar.gz)保留唯一布局前置修订之前的精确输入。原始命令输出以gzip保留原字节，截图附件只含既有遮罩输出；仓内不保存数据库凭据或会话；浏览器不持久保存草稿正文。
