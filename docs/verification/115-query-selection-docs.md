# CHNHCM-115 BU全查询结果选择与导出文档评审

登记信息：[registry.json](../governance/registry.json)的governance.verification_records。适用范围：[114决定](../adr/114-bu-query-selection-and-export.md)、[115计划](../plans/115-bu-query-selection-and-export.md)及受影响长期规格的本轮文档编写与评审。从[验证模板](../templates/verification.md)起草；不证明产品已实现。

## 1 对象与资料基线

2026-11-03，固定wt-dev-b，基线`0e3a0920875741263e7ae8f085e50deca82b5ceb`，开始时工作区和stash为空。受测工作区为该基线加本轮文档差异：新增114/115、SCN018、BR022/023、AC032/033、012/013目标合同及TL-349—366；同步110/113部分替代、010/111原范围说明、导航、来源与资产。产品源码、OpenAPI和原验证记录未改。正式输入摘要及检查输出在第7节列明。

## 2 环境与来源

仅使用本工作树的Python标准库、Git及文档检查器，主代理分轮语义复核，不是独立评审或具名人工批准。当前源码事实为Community单返回高亮、多选禁用、服务端分页/授权查询，不能视为目标选择能力已经存在。CubeBox来源定位/摘要见[来源记录](../reference/114-bu-query-selection-source.json)，本轮仅静态核对，不把前轮浏览器观察登记成本轮新验收。

本轮不运行浏览器、Go/DB业务、make check/integration/smoke或容量测试，不创建/启动/停止Web/API、容器、profile或数据卷；无任务自有运行资源需清理，已有用户环境未操作。

## 3 检查与结果

| 检查 | 实际结果 | 证据及边界 |
| --- | --- | --- |
| 首轮make docs | 通过；73项Python单元检查及仓库结构/关联检查 | [首轮输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/115-query-selection-docs/docs-first.txt)；包含模拟运行资源的单元夹具输出，不是启动真实profile |
| 首轮check-repository --check-refs | 通过；固定分支引用及仓内关联合法 | [首轮引用检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/115-query-selection-docs/refs-first.txt) |
| 首轮git diff --check | 通过 | [首轮差异检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/115-query-selection-docs/diff-first.txt) |
| 主代理两轮语义复核 | 通过；6项发现已修订，12类要求均有规格/案例映射，无剩余文档阻塞项 | [评审矩阵](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/115-query-selection-docs/semantic-review.json)；非独立评审或人工签收 |
| 修订后的make docs | 通过；73项Python检查及仓库结构/引用检查 | [复验输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/115-query-selection-docs/docs-final.txt)；检查器/测试输入未修改，最后语义勘误与定稿元数据另由最终引用检查覆盖 |
| 最终check-repository --check-refs | 通过 | [最终引用检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/115-query-selection-docs/refs-final.txt)；覆盖修订后的合同 |
| Go/API/DB、Chromium、容量及18例新增案例 | 未运行 | 产品未实施，TL-349—366全部保留未运行 |

## 4 分轮语义评审

第一轮按用户目标、规范、角色/入口/结果、权限/历史、A/D与API/计数逐项对照：发现纯排序案例与已有应用D语义冲突、旧范围对导出的未限定排除、版本格式/预算/序列化失败处理不够明确。已分别修订TL-353/354、110/113/012和013，未变更既定全量要求。

第二轮沿所有选择入口及边界反查：发现原生网格Ctrl/Cmd+A仍可能绕过新表头而只选当前页，以及超过100页的规模例缺精确输入。已把网格快捷键接同一全结果动作、保留输入框文本全选，TL-365补验；TL-366明确2021BU/20条每页反例。另区分校验/生成失败的零CSV响应与网络传输中断：已发字节无法追回，但前端未完整接收不创建下载，修订013及TL-363，避免不可实现的承诺。重新阅读114/115、011—013/023及范围/导航后，各要求都有对应合同和待执行验收，剩余文档阻塞项为0。

集合版本检测同数成员替换、来源及有效读取授权；导出按完整所选集合核验独立权限且失败零文件，均已写入合同。本报告只确认文档语义具备实施依据，未用来源行为或已有组件能力证明新产品能力。

## 5 限制

文档结构、引用或代理语义复核不等于业务/性能验收；尚无新增OpenAPI、Go、SQL、导出授权或页面运行结果。114/115可以采纳为目标合同，115的execution_status仍为not_started，不能从T00完成推出T01—T05完成。CubeBox源行为、111旧通过和本轮新全结果语义分别记录。

## 6 结论及交接

本轮文档编写、主代理两轮语义评审及结构/引用检查通过，无剩余文档阻塞项。114/115及受影响规格的本次修订已登记accepted；115产品实施仍为not_started，新增案例和资产采用均不标记为产品通过。定稿元数据及完整差异另经最后只读复核，输出保存在第7节证据目录。

用户本轮授权为制定文档并评审检查；没有实施、提交、push、PR、合并或部署动作。后续按115实施后另建实际报告，保留本次资料基线及未实施边界。本轮未创建真实运行资源，无自有profile/浏览器需清理，既有用户环境未操作。

## 7 证据保存

正式检查输出、语义复核矩阵、最终范围/编号核对和受测合同文件摘要保存在[data/115-query-selection-docs](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/115-query-selection-docs)。input-files.sha256标识本轮合同、登记、来源与资产输入；报告及证据文件自身不放入递归摘要。scope-and-identifiers.json记录366个唯一案例和新增18例未运行、原报告/产品输入未改，以及计划状态。定稿元数据复核见refs-metadata.txt，完整新增文件空白检查见diff-final.txt；不以仅存在.local的材料作为唯一长期依据。原报告和来源原件不覆盖。
