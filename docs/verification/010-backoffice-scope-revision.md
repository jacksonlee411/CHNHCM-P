# CHNHCM-010 后台操作范围修订验证

登记信息：[registry.json](../governance/registry.json) 的 `governance.verification_records`。适用范围：[010](../plans/010-m02-scope-and-plan.md) 的范围、入口与操作路径修订，以及 [008](../plans/008-next-steps.md) 和文档导航同步；依据 [014](../governance/014-document-management.md) 保留历史证据。本报告不采纳范围、不验收业务实现。

## 1 验证对象与受测基线

日期：2026-10-31；仓库 CHNHCM，固定工作树 `/home/vivobook/projects/CHNHCM-wt-dev-a`，分支 `wt-dev-a`，HEAD 为 `f21c99ac42d1662bd7fc077399e27fd144d7a8d0`。本轮开始时工作区已包含上一轮 010 起草及配套资料的未提交变更，不能仅用 HEAD 标识修订前内容。

修改前将原起草的 9 个输入文件、原报告及原输入摘要共 11 个文件保存到[修订前快照](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/010-scope-before-backoffice.zip)，并核对其与[原输入摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/010-scope-draft-inputs.json)一致。该 ZIP 仅用于复核历史版本，不是当前方案入口；当前正文始终由 010 持有。[本轮输入摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/010-backoffice-scope-inputs.json)分别记录 Git 基线、修订前工作区及修订后文件摘要，报告与本轮摘要自身排除以避免自引用。

## 2 环境与前置条件

本机 Linux、Python 3.12.3 标准库与 Git；仅使用本仓文档检查，无新增工具。源包为 [CubeBox 固定引用包](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/reference/sources/cubebox-m02-2deaf927.zip)，对应提交 `2deaf9276086f68e07aaa0c2cf7f35025046a9ee`；不连接相邻源码仓或运行源应用。

用户已明确本批聚焦后台操作，不引入申请、撤回申请、审批、独立人工发布流程或工作台。本轮按此修订 010，保留已有 Go 权威、日期、权限与历史保护要求；未改动目标设计合同、资产状态或产品代码。

## 3 验证过程与结果

| 检查 | 操作与预期 | 实际结果 | 结论 |
| --- | --- | --- | --- |
| 源 BU 入口依据 | 读取 ASSET-0002 源脚本及 ASSET-0133/0165 登记，核对正常入口与资产归属 | 源脚本第 9、23 行为“设置 → 组织配置 → 业务单位”，第 15—18 行为列表新建、保存后到详情；010 使用本项目“业务单元”术语，优先采用 BU 直接资产 | 通过 |
| 范围与路径自查 | 逐节核对 010 的角色、范围、阶段、风险与验收；排除工作台和审批前置 | 已改为管理员直接保存；设置/组织配置仅为导航分组；区分对象操作、记录操作、取消编辑、三个日期含义及返回上下文；必要源版本与事件不形成独立人工发布步骤 | 通过 |
| 文档与治理检查 | `make docs`：现有回归、登记、编号、入口、来源与生成一致性通过 | 44 项 Python 回归通过；99 个 Markdown、2310 个本地链接、53 张图稿及 197 项来源资产检查通过 | 通过 |
| 固定分支与资料检查 | `python3 -B scripts/check-repository.py --check-refs`：六个固定分支 ref 及资料通过 | 六个固定分支 ref 存在，资料检查通过 | 通过 |
| 差异格式 | `git diff --check`：无格式错误 | 无格式错误 | 通过 |
| 输入摘要及保护范围 | 核对历史 ZIP、原报告/摘要字节、当前输入摘要、未改动范围及草案状态 | 11 个快照成员与原 9 个受检输入一致；当前 11 个输入摘要一致；原报告/摘要、前轮其他修改、目标设计、来源、197 项资产状态及运行代码均保留；010 及 011—013 状态符合记录 | 通过 |

上述内容自查由主代理完成，不是独立语义评审或人工批准；源文件静态核对也不等于新系统菜单已运行。

## 4 失败、修复与复验

本轮资料检查未出现失败或中断。原[010 起草报告](010-scope-draft.md)及其摘要保持原字节，新结果只记录在本报告。补齐本报告结果、登记及当前输入摘要后，再检查资料与引用、摘要和差异格式；不重复运行未受影响的产品检查。

## 5 未覆盖范围与限制

没有进行业务实现、导航接线、011—013 编写或范围正式采纳，没有运行 Go 业务、数据库、API、构建或浏览器验收。没有重新核对法律、Oracle 最新文档或全部 R25 后增量。本轮没有新增审批、工作台、其他 M02 对象、父子组织结构或复杂适用配置；保存未来记录、记录撤销和停用仍需在 011—013 定义具体允许条件并实际实现验证。

## 6 结论与后续事项

010 已按用户要求修订为后台直接维护方案，范围、入口和配套资料检查通过。010 保持 `draft`、`review: null`、执行 `not_started`；008 编写为 `in_progress`，011—013 仍为 `planned` 且无正文。报告定稿不改变这些状态，197 项资产的实现/验证状态仍未提升。下一步继续在既定范围内细化 011—013。

## 7 证据保存

- [原起草报告](010-scope-draft.md)、[原输入摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/010-scope-draft-inputs.json)及[修订前快照](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/010-scope-before-backoffice.zip)共同保存原受检版本，避免把旧结果用于本次新范围。
- [本轮输入摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/010-backoffice-scope-inputs.json)保存当前输入与快照 SHA256；正文、登记和本报告保存当前修订及验证范围。
- 本轮原始命令输出保存在 `.local/010-backoffice/`，仅作非敏感辅助记录；长期复核以仓内正文、报告、快照和摘要为准，不依赖本地临时文件。
