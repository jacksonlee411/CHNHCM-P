# CHNHCM-108 M02维护改造多轮复核记录

登记信息：[registry.json](../governance/registry.json)。适用范围：[108计划](../archive/plans/108-maintenance-consistency-review.md)、101直接保存、103/105位置新增、104统一更正及107首条通用维护。原报告、稳定ID及历史事实保持。

## 1 验证对象与受测基线

2026-11-02，`/home/vivobook/projects/CHNHCM-wt-dev-b`，固定分支wt-dev-b，基线`32bd8e14597765a8e7e6430b9547bca57cacc6b3`；开始工作区干净、stash为空。本轮按用户要求多轮检查并修复，非独立代理评审。未提交、推送、PR或合并本轮修改。

受测对象为基线加本轮工作区：[输入摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/108-maintenance-consistency-review/tested-inputs.json)与[源码及测试归档](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/108-maintenance-consistency-review/tested-inputs.tar.gz)冻结最终218个源码、测试辅助、构建及运行输入。完整144项的[原输入摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/108-maintenance-consistency-review/tested-inputs-full.json)及[原输入归档](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/108-maintenance-consistency-review/tested-inputs-full.tar.gz)另存；之后仅4文件发生[限定差异](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/108-maintenance-consistency-review/full-to-final-delta.json)，由最终28项定向复验及静态/构建承接，其他214文件保持。报告和登记元数据随后补齐。

## 2 环境与前置条件

任务独立profile为m02-maintenance-review：Web4548、API8548、DB54648，使用本任务合成数据，真实PostgreSQL18.6；Go1.26.0、Node24.19.0、pnpm10.24.0、Playwright1.55.1，见[运行核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/108-maintenance-consistency-review/runtime.txt)。浏览器默认桌面Chromium1440×900，管理员从“设置→组织配置→业务单元→列表→详情”进入；查询者、字段/动作撤权、未知提交等按用例执行，授权夹具finally恢复，顺序运行。

原Web4382/API8382的m02-current-b及来源4527仅登录查询；没有迁移、重建、清库或写业务数据。六个原样本从正常菜单核对日志/来源/新增插入，原环境全部7对象与107留存摘要一致，来源4527样本头版本亦保持。单独UI profile4549因Docker地址池耗尽未启动成功，保留[失败输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/108-maintenance-consistency-review/ui-up-retry.txt)；未清理主机或其他任务网络，改用任务主profile顺序验证。

## 3 多轮检查与处置

| 轮次 | 核对方式 | 发现与处置 | 证据 |
| --- | --- | --- | --- |
| R1交叉清点 | 导航→规格→ADR/计划→测试→报告引用 | 七组残留：103未接线/107误归草案、旧操作表、ADR入口替代、CORRECT旧算法/状态、测试旧别名、105明确未补参数、104日志被忽略 | [发现清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/108-maintenance-consistency-review/round1-findings.json) |
| R2修复及补验 | 对照Go/API及011独立预期，补测撤读权与并发改日 | 同步当前错误码及TL097/098显式联合路径；补TL272英文/日期撤读权、TL277锚点/后继改日四参数；暴露日期控件及旧上下文问题，修复后4项通过 | [复核记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/108-maintenance-consistency-review/round2-review.json)、[复验输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/108-maintenance-consistency-review/browser-round3.txt) |
| R3反向追踪 | 从实际保存守卫/菜单/失败用例反查当前合同，核对历史原件 | 找到启停/删除仍依赖辅助许可；4参数修复前1通过/3失败，修复后4通过。继续清理012证据表、021、T07待验状态及导航未置顶等遗漏；1792个受保护已跟踪文件未改写 | [第三轮发现](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/108-maintenance-consistency-review/round3-findings.json)、[资产核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/108-maintenance-consistency-review/asset-audit.json) |
| R4最终复验 | 核对完整回归到最终修订的差异、全部测试收集、视口及证据 | 完整144项与最终28项均通过；当前146个实例均有最近通过绑定，7组视口通过；补清007旧状态和归档空白规则 | [执行绑定](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/108-maintenance-consistency-review/browser-latest-inventory.json)、[最终差异](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/108-maintenance-consistency-review/full-to-final-delta.json)、[视口](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/108-maintenance-consistency-review/viewport/results.json) |

当前规则、代码和测试的逐项对应见[复核矩阵](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/108-maintenance-consistency-review/scope-matrix.json)。未删Go只读preview协议、日志中的改日/分类修改面、联合内分类编辑、否定断言或历史段落；这些不是已退役单条菜单。

## 4 失败、修复及测试更新

### 4.1 两处实际页面问题

新增/插入表单撤生效日期读权后，原页面仍显示空日期控件；能力函数对undefined参数默认回退旧additionContext，可能提示“核对后保存”但实际禁存。已隐藏不可读日期控件，并显式使用重读后的上下文，保留仍可读草稿及取消路径，不自动换锚点或清空合法字段。最终将此日期控件修复限定在普通ADD：其位置上下文依赖日期读取，缺上下文不可办理；启停沿用独立Go动作授权，可明确输入自己的新日期而不获得历史日期读取。两项额外UI权限用例保护该边界。

TimelineEditor的按钮及save处理仍为DEACTIVATE/REACTIVATE/DELETE保留assisting/assistanceError门槛。管理员已填合法原因/日期，辅助503仍无法确认。已移除两处门槛，保留当前授权、在途、重读、冲突、待确认、分类转停用确认及越权编辑保护；Go最终校验和历史事务不变。新增四参数分别验证一次真实提交、实际持久事实及旧来源原件；原修复前[1通过/3失败](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/108-maintenance-consistency-review/assistance-before-main.txt)和修复后[4通过](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/108-maintenance-consistency-review/assistance-after.txt)均保留。

### 4.2 文档与用例一致性

AGENTS、README/规格/验证导航、007/010、011—013/023、020/021/022/103及017按当前行为同步；100/102/104/105保留原批次并明确当前合同去向。更正记录包括名称/日期/按需分类，普通保序、跨节点显式联合；最新新增/非最新插入及Go默认，首条直接更正后按需另次新增，均有当前说明。不再把原T07、早期四组迁移或首条保留选择当成现行待办。

023保持306个原编号、301现行和5退役（132/241/242/246/247），不回收或新增重复ID；修正TL164辅助失败预期及TL038/067/068错误码，澄清TL097/098为显式联合。新增8个浏览器参数对应TL164/272/277，另补AC009启停新日期写入/历史日期读取分权两项UI断言；旧分类菜单测试别名改为明确进入“更正记录→业务分类”，权限夹具共用grants.ts。修改对象可从[矩阵](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/108-maintenance-consistency-review/scope-matrix.json)定位。

### 4.3 首次失败及中断边界

首轮定向浏览器26项为22通过/4失败：[原输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/108-maintenance-consistency-review/browser-round2.txt)。其中1项暴露日期撤读权问题，另3项为本轮新增测试把Request误当Response调用request()，已修正为method()，四项复验通过，不将测试错误算作产品缺陷。

发现辅助门槛时主动中断当时140项命令：[37通过/1中断/102未运行](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/108-maintenance-consistency-review/browser-full.txt)，不计完整通过。另一次4549浏览器尝试因运行环境未就绪在setup失败，见[原输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/108-maintenance-consistency-review/assistance-before.txt)。这些输出不被最终重跑覆盖。

## 5 检查结果与覆盖边界

| 检查 | 实际结果 | 证据 |
| --- | --- | --- |
| make check（两处页面修复后） | 通过；文档结构、边界、Go测试/vet、生成一致性、前端类型及diff检查 | [原输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/108-maintenance-consistency-review/check-complete.txt) |
| make build（两处修复后） | 通过；保留既有包体积提示，未以此声明性能验收 | [原输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/108-maintenance-consistency-review/build-complete.txt) |
| 定向真实DB/API | 11个测试函数及参数通过，Go自报60.372s；使用Testcontainers隔离真库，命令见下 | [原输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/108-maintenance-consistency-review/integration-round2.txt) |
| 完整默认Chromium（完整回归输入） | 144/144通过，21.3分钟；随后仅4文件差异，由下项补验承接 | [原输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/108-maintenance-consistency-review/browser-full-passed.txt) |
| 最终受影响Chromium | 28/28通过，4.4分钟；含新增/插入、启停、直接保存及两项新日期权限边界；静态/构建同时在最终输入通过 | [原输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/108-maintenance-consistency-review/browser-scope-final.txt) |
| 最终测试收集及最近结果绑定 | 19文件146个实例，与完整144及最终28的集合匹配；26项以最终复验替代旧结果、新增2项；不是单次146项命令 | [收集输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/108-maintenance-consistency-review/browser-list-final.txt)、[逐项绑定](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/108-maintenance-consistency-review/browser-latest-inventory.json) |
| 七组视口/真实缩放 | 桌面、短屏、窄屏、320宽、320×256、真实200%及400%全部通过；49张截图，核对位置新增的输入/日期/禁用、键盘/滚动和保存 | [逐组结果与截图路径](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/108-maintenance-consistency-review/viewport/results.json)、[原输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/108-maintenance-consistency-review/viewport-final.txt) |
| 原环境只读样本 | 六样本UI与新增/插入位置通过；4382全部7对象摘要与107一致，4527样本来源头不变 | [UI结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/108-maintenance-consistency-review/current-samples/ui-result.json)、[最终数据摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/108-maintenance-consistency-review/current-data-after-final.json)、[输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/108-maintenance-consistency-review/current-samples.txt) |
| 最终文档/格式复核 | make docs与git diff --check通过；相关登记、链接与格式闭合，语义结论仍限本报告范围 | [文档输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/108-maintenance-consistency-review/docs-final.txt)、[空白核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/108-maintenance-consistency-review/whitespace-audit.json) |
| 案例ID、历史及冻结输入 | 通过；306编号/5退役保持，1792原文件未改写，218受测输入摘要保持，报告链接存在且不被忽略 | [资产核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/108-maintenance-consistency-review/asset-audit.json) |

```sh
go test -tags=integration ./tests/integration \
  -run '^(TestFirstRecordIndependentCorrectionAndAddition|TestJointBackfillClassificationAndSafePreview|TestPositionedAddition.*|TestUnifiedCorrection.*)$' -count=1
CHNHCM_LOCAL_PROFILE=m02-maintenance-review corepack pnpm test:smoke \
  --output .local/maintenance-review/browser-final
```

Go生产、API和存储未变，本轮未再运行完整DB/容量/时钟故障专项；定向真库回归与最终页面回归各自界定。未运行Firefox/WebKit，不证明全306编号/任意参数、真实消费者、具名业务人员或生产验收。未优化通用联合入口，未新增历史停用再启用专用交互。

## 6 结论与后续事项

已完成四轮检查与修订，本轮发现均已处置并在相应层复验；当前菜单、规则、文档及测试按101/103/104/105/107一致承接，历史原件和原样本保持。受测范围内未留待修复发现，逐项关闭见[最终复核](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/108-maintenance-consistency-review/round4-final-review.json)。本次仍为主代理多轮自查；旧定稿报告保留原结论，原通过不会自动提升为本轮通过。

## 7 证据保存

本轮长期证据位于data/108-maintenance-consistency-review，非敏感原始输出保留.local/maintenance-review；原104的24份被忽略日志按[原字节摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/108-maintenance-consistency-review/restored-evidence.json)恢复为可跟踪变更，尚未提交，未将补归档写成重新执行。报告引用、受测输入、历史保护及Git跟踪资格由[audit-assets.py](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/108-maintenance-consistency-review/audit-assets.py)核验；[敏感值定向检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/108-maintenance-consistency-review/evidence-safety-check.json)不输出任何凭据内容。认证会话值和真实数据库口令不保存在仓库。

本轮新日志与源文件分别处理：104两份原始日志的行尾空白保留原字节，仅在该目录的日志上豁免空白检查；新grants.ts多余文件尾空行已删除，源文件规则不放宽，见[空白核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/108-maintenance-consistency-review/whitespace-audit.json)。最终任务环境4548/4549已停止并保留卷，原4382就绪核验见[输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/108-maintenance-consistency-review/current-runtime-final.txt)；来源4527仍仅只读核对。
