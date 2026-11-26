# CHNHCM-100 运行隔离评审修复验证

登记信息：[registry.json](../governance/registry.json) 的同路径记录。适用范围：[100 T01](../archive/plans/100-ui-experience-improvement.md#ui100-runtime-isolation) 的两项 P2 评审问题；运行方法由 [017](../guides/017-local-bu-development.md) 持有。[原运行隔离报告](100-b-runtime-isolation.md)及其证据保留，本报告记录后续修复与补验。

## 1 对象与基线

本轮位于 `CHNHCM-wt-dev-b` / `wt-dev-b`，HEAD 为 `8ae0a44353ee8105dfdef9a1b15fde9af345a144`，dirty。开始时的 67 个待提交文件包含 b 工作树登记、100 草案和 T01 实施及验证；本轮按用户“修复，然后提交更改”执行两项修复和本地提交准备。T02 和 012 正文不在本轮修改范围。

本轮增量为运行归属模块、对照准备模块及调用方、Python/Node 回归测试和 `make check` 接线，另更新指南、计划交接与本报告。开始状态见[待提交输入摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime-review/before.json)，[117 份运行/验证输入](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime-review/tested-inputs.json)在本轮完整检查前采集，结束后逐项保持。环境沿用本机 WSL2 Linux 6.6.114.1、Go 1.26.0、Node 24.19.0/pnpm 10.24.0、Python 3.12.3、PostgreSQL 18.6 锁定镜像、Playwright 1.55.1/Chromium 140.0.7339.186。

## 2 发现与修复

| 问题 | 触发与影响 | 本次修改 |
| --- | --- | --- |
| 对照目标与凭据分属不同工作树 | 从 a 目录通过绝对路径运行 b 的对照脚本，目标由 b 的脚本位置解析，但凭据、Git 基线和默认输出仍跟随调用目录 | 将准备过程收敛到 [comparison-context.mjs](../../scripts/comparison-context.mjs)，与运行入口共享脚本根目录；凭据和 Git 命令明确 cwd，默认/相对输出锚定本工作树，显式绝对输出保留；目标拒绝时不读取凭据 |
| 不同 IP 上的同端口服务误判为外来进程 | 正常 b API 在 127.0.0.1:8381 监听，任务自有 127.0.0.2:8381 监听也被计入 PID 集合，导致归属检查失败 | [local_runtime.py](../../scripts/local_runtime.py)按本产品固定 IPv4 端点筛选监听器，保留 IPv4 通配监听；IPv6-only 不混入，真实双栈冲突仍在 IPv4 bind 预检被拒绝 |

## 3 验证过程与结果

| 检查 | 步骤与预期 | 实际结果与证据 | 结论 |
| --- | --- | --- | --- |
| 地址归属修复前 | 在正常 b API 外增设任务自有 127.0.0.2:8381 监听；运行归属检查 | 原实现误拒绝；新增测试中 IPv4 独立地址和 IPv6-only 两例失败，见[真实反例](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime-review/address-before.json)、[17 项初次结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime-review/python-before.txt) | 失败，保留原结果 |
| 地址归属修复后 | 运行原 13 项及新 4 项；核对独立 IPv4、IPv6-only、IPv4 通配及双栈冲突 | 17/17 通过，无跳过；其他地址不算目标进程，真实冲突仍拒绝，见[单元结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime-review/python-after.txt) | 通过 |
| 跨目录修复前 | 从仓外合成目录运行与原脚本相同的准备逻辑，使用无敏感信息的标记文件 | b 目标配上调用目录的凭据标记，见[反例](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime-review/cwd-before.json)；没有发送真实凭据 | 失败，保留原结果 |
| 跨目录自动回归 | 两个仓外临时检出夹具具有不同 Git HEAD、目标和凭据标记；从 a 调 b 的实际准备模块 | 目标、凭据、HEAD、默认/相对输出均归 b；绝对输出保持；拒绝发生在凭据查询和目录创建前。4/4 通过，见[Node 结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime-review/node-tests.txt) | 通过，合成边界测试 |
| 真实 b 准备入口 | 从带有诱饵脚本的仓外目录调用准备模块，仅在内存比较实际凭据 | 与 b 凭据、HEAD、目标一致，相对输出归 b；未创建输出或访问参考服务，见[实际准备结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime-review/live-comparison-context.json) | 通过，仅准备过程 |
| 完整静态与构建 | `make check`、`make build`；新 Node 测试已纳入 `make check` | 63 项 Python（含 17 项运行测试）、4 项 Node、Go 单元/边界/vet、类型检查及构建通过，见[check](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime-review/check.txt)、[build](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime-review/build.txt) | 通过 |
| 正常菜单和异常浏览器回归 | 构建后启动 b；整个 `make smoke` 期间，在 127.0.0.2 的 4381/8381 上保留任务自有监听 | 从设置菜单创建、查询、查看来源与返回，以及原有异常/会话/权限保护和组件用例 27/27 通过；测试前后 b 进程记录不变，见[smoke](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime-review/smoke.txt)、[环境断言](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime-review/smoke-environment.json) | 通过，真实 b Go/DB；原故障注入边界保持 |
| 文档、固定分支和差异 | 定稿后检查登记、导航、固定引用及 `git diff --check` | 见[最终文档检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime-review/docs-final.txt) | 通过 |
| 原证据及后端输入保护 | 比较原 T01 的 48 份报告/数据和本批 117 份受测输入 | 47 份原件原路径保持，1 份失败日志的精确字节归档后规范化阅读版末尾空行；117 份受测输入及 46 份 Go/API/DB 相关输入保持，见[保护核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime-review/protection.json)和[原件映射](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime-review/original-failure-log.json) | 通过，范围限所列输入 |
| Testcontainers 本轮重跑 | Go/API/迁移及相关 DB 输入未改 | 本轮未重跑，保留[原真实 DB 结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime/integration.txt)；本轮浏览器的真实创建查询另列，不合并为新的 Testcontainers 结果 | 未运行，原结果仅按未变范围复用 |

## 4 失败与复验

保留修复前真实地址冲突失败及新增测试中的两个失败：不同 IPv4 地址和 IPv6-only 被误计入。跨目录问题用仓外合成标记复现，不向错误目标发送真实凭据；修复后同一准备入口的四项用例通过。

两项 P2 均已修复并复验闭合。检查和构建一次通过，完整浏览器本轮一次通过；构建仍有既有依赖 `use client` 和大 chunk 提示，没有改变阈值或包拆分。未修改产品页面断言来迁就实现。本次为实施者自查与回归，不计独立代理评审或人工批准整个页面方案。

首次暂存差异检查发现原 T01 的 `reject-healthy-web-before-login.txt` 末尾多一个空行；先将其 589 字节原件存入[归档](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime-review/original-failure-log.zip)并逐字节核对，再仅删除阅读版最后一个空行。原 `.local/t01/` 输出保持，SHA 映射单列；没有删掉失败内容或更改原报告结论，也没有放宽 Git 空白检查规则。最终暂存检查通过。

## 5 未覆盖范围

没有运行 a 或 a/b 同时在线检查；没有重采 CubeBox 图面。跨目录测试只证明准备过程的工作树归属，不冒充外部图面比较。Go/API/数据库合同和迁移未改；旧库升级、容量、全浏览器、生产恢复及远程 CI 不在本轮范围。整体 100 仍在推进，T02 尚未开始。

并行工作期间 a 的索引文件摘要发生变化，核对时其 HEAD 未变；本轮未向 a 写入、未同步或回滚该索引，不宣称 a 整个工作区保持同一快照。本次提交只在 b 固定分支处理本会话已评审的计划、隔离实现与修复。

## 6 证据与交付

本轮原始非敏感输出保存于 `.local/t01-review/`；长期输出、受测输入和保护核对归档于 [data/100-runtime-review](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime-review/tested-inputs.json)。[输出检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime-review/output-scan.json)没有发现当前 b 的随机密码值，不代表一般秘密扫描认证。凭据留在仓外，本轮不保存 Cookie、CSRF、storageState 或 trace。原 T01、018/019 报告与资产状态保持。

必要检查完成后按用户要求提交 b 的本批更改；提交号以实际 Git 记录为准，不能用它替换第 1 节受测基线。没有推送、PR、合并或部署授权，下一实施任务仍为 T02。
