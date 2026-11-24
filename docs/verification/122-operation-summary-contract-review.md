# CHNHCM-122 BU操作摘要与日志表格化合同评审

登记信息：[registry.json](../governance/registry.json)的governance.verification_records。适用范围：[122草案](../plans/122-bu-operation-summary-and-change-log.md)、011—013及023本次对应增量，按[验证报告模板](../templates/verification.md)编写。本记录仅评审文档，不重审全M02，不证明产品已实现。

## 1 验证对象与受测基线

2026-11-04，固定工作树CHNHCM-wt-dev-b、分支wt-dev-b。开工HEAD为`16a18d7f00fbd3e16190034b5a8e4b38eaa67770`，工作区及stash为空，[开工摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-contract/before-inputs.json)保存原规格审阅登记及随后采集的3729份已有文件摘要。编写期间其他工作修改产品文件并提交`dbd40154087c0e1faa9a41f271ee2f78dc029cb9`，见[范围外观察](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-contract/out-of-scope-observation.json)；本轮未编写、提交或回退该产品变更。状态检查与摘要采集并非原子快照：其中10份产品文件摘要已含并行修改，见[采集边界核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-contract/baseline-capture-audit.json)，不把它们称为16a18d7原字节。

最终文档检查基线为dbd4015加本轮未提交文档差异：122、011—013/023、010/116交接、登记/导航/资产、来源及本报告。该产品提交没有修改本轮文档；在新HEAD静态重核SourceExplanation、来源级actor、记录修订创建source_id和删除原件读取，122主要缺口仍存在。受测文档摘要、最终差异及原件保护见第7节，不用HEAD单独标识整个脏工作区。

## 2 环境与依据

仅Python标准库、Git、仓库文档检查器及官方网页只读核验。没有启动本项目Web/API、DB、浏览器或容器，也没有调用相邻仓作为构建/运行依赖。PeopleSoft截图来自本对话前轮学习环境实机观察，本轮只检查并保全原字节，未重跑业务或验证存活状态。

| 依据 | 实际支持范围 | 不可外推的结论 |
| --- | --- | --- |
| [Microsoft Audit History](https://learn.microsoft.com/en-us/power-platform/admin/manage-dataverse-auditing#use-the-audit-history-in-a-model-driven-app) | 操作历史和字段前后值、单独View Audit History权限 | 不照搬独立筛选下拉框、删除日志或环境级平台；不证明本仓修订模型 |
| [Account ModifiedBy/ModifiedOn](https://learn.microsoft.com/en-us/power-apps/developer/data-platform/reference/entities/account#modifiedby) | 修改者/时间是可读且可用于表单的只读属性 | 不证明默认页面底部位置或姓名账号并列 |
| [Oracle HRMS 9.1职务代码](https://docs.oracle.com/cd/E20215_01/hr91pbr1/eng/psbooks/hhaf/htm/hhaf14.htm)及[实机底部截图](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/reference/122-operation-summary/job-code-update-footer.png) | 职务代码概要字段之后、保存按钮之前有更新时间/更新用户，样本账号SAMPLE | 不是人员职务信息页；未测多记录归因、姓名并列及历史权限矩阵 |
| 本仓源码与012/013 | 已有单列条件交互、不可变修订创建来源及确切前后来源 | 有旧组件/字段不等于新摘要、完整差异或分页筛选已实现 |

来源日期、页面、样本、摘要、采纳/不采纳见[来源索引](../reference/122-operation-summary-sources.json)。姓名账号并列、修订归因、固定分页上界和删除链接是本项目合同，不能写成来源系统已经共同验证的行为。

## 3 验证过程与结果

| 检查 | 实际结果 | 证据及边界 |
| --- | --- | --- |
| 合同语义自审 | 已逐项审阅并修订10项范围/一致性问题；不是独立评审或人工业务签收 | [评审矩阵](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-contract/semantic-review.json)及第4节 |
| 首轮及修订后make docs | 通过；每轮73项Python检查，仓库结构/关联检查通过 | [首轮](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-contract/docs-first.log)、[修订后](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-contract/docs-final.log)；夹具中的profile文本不是启动真实服务 |
| 首次定稿引用检查 | 失败；报告已链接但final-inputs摘要尚未生成 | [失败输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-contract/refs-metadata-first-failed.log)，补齐摘要后复验 |
| 固定分支及仓内引用 | 通过 | [首轮引用](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-contract/refs-first.log)，定稿元数据再次核对见[最终引用](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-contract/refs-final.log) |
| 稳定ID、案例数量、草案/资产状态 | 通过；全M02 SCN27/BR35/AC44无冲突；023为386定义/342现行/44退役，原366行完全一致，新20例未运行 | [范围与编号](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-contract/scope-and-identifiers.json)；122草案未执行，116登记及原资产状态不变 |
| 旧报告/来源/产品范围保护 | 通过；3729份采集文件中3728份原字节不变，另1份仅验证导航README；旧报告/来源包未改，产品相对最终HEAD未改 | [保护核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-contract/evidence-protection.json)及[非原子采集边界](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-contract/baseline-capture-audit.json)，不混淆外部提交 |
| 首轮git diff --check | 失败；011文末多空行 | [首轮原始输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-contract/diff-first.json)，不覆盖失败 |
| 修复后git diff --check及新增文本检查 | 通过 | [复验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-contract/diff-final.json)；新增文件no-index退出1且无诊断代表文件不同，不是空白失败 |
| TL-367—386、Go/API/DB、Chromium、容量 | 未运行 | 新20例均未运行；make check/integration/smoke不在本轮范围 |

## 4 发现、修订与复核

| 问题 | 修订结果及验收去向 |
| --- | --- |
| 对象最新来源可能冒充所选修订的修改人 | BR034/013绑定record_revision.source_id；S1—S7及TL-368—370独立证明未变/派生/旧源 |
| 删除后链接可能继续定位墓碑原件 | 删除前用前源/原记录；删除后用后源并省略被删selectedRecordId；TL-373含同日新记录反例 |
| 旧012基本资料折叠来源说明与新目标冲突 | 019旧实现标明时点，3.2/3.4统一底部摘要与日志行内来源明细 |
| 初稿日志URL漏v1、动作白名单不明确 | 统一/api/v1及现行八动作；固定两条和旧联合分清，不提前启用117 R1分类 |
| 部门已占用稳定编号 | 使用SCN027、BR034/035、AC041—044及TL-367—386；旧定义/退役保留 |
| 摘要权限与历史身份缺失容易被混为一谈 | 基本bu.read可读最小摘要；历史另授权；新身份封存、旧源仅ID如实展示，不用当前账号回填 |
| 多字段/多记录筛选可能拆操作或泄露隐藏字段 | 服务端裁剪后筛选/计数、按操作分页及固定来源上界，H45给出45/23/22/15的手写预期 |
| 新案例初稿并列多个主要验证层 | 按024改为每例唯一主要层、明确代表性补充参数；TL-386归P层，旧366行不改 |
| 摘要位置可能误导移动已有顶部操作 | 012明确字段底部常显，保留既有操作入口；示意不新增底部按钮 |
| 编写中出现范围外提交 | 同时保存开工与最终基线、观察列表、非原子摘要采集边界及保护核对，本轮产品工作不冒领、不回退 |

验证时另发现011文末多余空行，已删除并复验通过；首次辅助脚本把新增文件no-index的差异退出码1误判为失败，已按退出码及诊断共同判定，原始输出保留。定稿引用检查又发现报告先链接尚未生成的final-inputs.json，已生成受测摘要并重新核对，失败输出另存，不覆盖。未修改仓库检查器。

以上“修订”指合同与评审记录修订。第二轮按角色入口、取值、授权、差异、筛选/分页、链接、状态和证据边界反查，新增条目均有下游案例。没有据此认定Go、SQL、X03或页面缺口已经修复。跨域维护角色已在计划中列明，本轮没有具名维护者或人工业务人员签收。

## 5 未覆盖范围与限制

122保持draft/not_started，新增20例全部未运行。未实施姓名/账号来源、身份快照、记录级摘要、完整日志差异、查询/上界、表格及新链接；未测试真实权限竞争、数据库原子性、浏览器响应式或容量。当前代码同时存在其他工作交付，未纳入本报告的产品审查。未启动116 R1—R6或部门真实接入，不替换116 R0结论；旧366案例、退役状态、旧报告及原来源证据各自保留。

主代理自审结论仅覆盖本次文档增量与受影响旧段落，不能当作独立评审、人工采纳或全M02合同完整性结论；已有页面样式、PeopleSoft样本或官方说明不等于本项目验收。未修改检查器或降低门禁，也未执行提交、推送、PR、合并或部署。

## 6 结论与后续事项

本次122计划草案、011—013/023目标增量及范围限定的合同自审记录已完成，文档结构/引用、稳定ID、范围和原证据保护检查通过；首轮空白检查失败及修复保留。规格review仅登记本次增量的主代理接受范围，原审阅元数据保存在开工摘要，旧报告不改写；122不因合同准备完成变为执行中或完成。

后续按122 P1/P2/P3实施并另建报告，分别给出真实角色/菜单/参数/结果、失败复验、资源清理及未测范围。116 R3后续提炼实际交付，不用BU结果推定部门通过。本轮无新增运行资源，资源收尾不适用；既有持久环境未操作，前轮用户请求的PeopleSoft环境未接管，其下次使用/停止时复查，见[资源记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-contract/resources.json)。

## 7 证据保存

长期证据置于[data/122-contract](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-contract)：before-inputs.json为开工摘要和原登记；out-of-scope-observation.json记录编写期间外部产品修改；semantic-review.json为范围、发现及20例not_run映射；resources.json为本轮资源边界。[检查汇总](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-contract/validation.json)、[范围/编号](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-contract/scope-and-identifiers.json)、[原件保护](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-contract/evidence-protection.json)和[受测文档摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/122-contract/final-inputs.json)保存最终结果；报告与输出自身不进入递归文件摘要。截图及其SHA256见来源索引，未复制会话凭据或学习系统数据库。
