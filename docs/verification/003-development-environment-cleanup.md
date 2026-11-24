# CHNHCM-003 开发测试环境收尾规范验证

登记信息：[registry.json](../governance/registry.json)。适用范围：按用户要求，将开发/测试容器与配套进程收尾纳入实施规范；上游为[003](../adr/003-implementation-stages.md#development-environment-cleanup)、[014](../governance/014-document-management.md)及[025](../adr/025-greenfield-implementation-policy.md)。本记录区分前轮实际清理、本轮只读核验和文档检查。

## 1 对象与受测基线

工作树`wt-dev-a`，开工HEAD为`c600be6568848c92501f5858fd92d3ee84aabd80`，工作区和stash干净。受测内容为该基线加本次文档差异，摘要见[输入清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/003-development-environment-cleanup/input-sha256.json)。不修改程序、迁移、测试代码、运行配置或旧定稿报告。

## 2 环境与前置

前一轮由用户授权清理已完成的旧profile。实际停止12套profile的23个记录进程组及数据库容器；运行数据库15降至3，原45个卷全部保留，目标容器、仓外凭据及恢复资料保持。该事实来自前轮非敏感[结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/003-development-environment-cleanup/cleanup-result.json)和[停止清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/003-development-environment-cleanup/cleanup-stopped.json)，本轮没有重新执行启停或删除。

本轮只读核对12套目标仍停止、对应卷仍在，保留的当前4527入口归属与就绪检查成功。wt-dev-b及CubeBox数据库仍运行，未纳入前轮已确认旧环境范围。记录见[只读核验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/003-development-environment-cleanup/readonly-check.json)。非敏感证据复制入本目录，凭据和程序归档不复制入仓。

## 3 规范修改与检查

| 对象 | 修改或核对 | 实际结果与边界 |
| --- | --- | --- |
| AGENTS、003 | 通用收尾责任、批次/容量/任务完成或取消的时机；归属核验；持久profile停止保留与明确可丢弃的临时资源回收；例外/失败及交接核验 | 已写入本次文档差异；003持有统一规则，AGENTS引用 |
| 013、017 | M02引用通用要求；本地盘点、保存恢复依据、受控停止、旧记录失配处理与清理后核验 | 已与现有脚本核对；017明确历史环境保留不等于永久常驻，当前用户验证环境有责任与复查节点 |
| 计划/验证模板 | 关闭条件和报告必须记录资源归属、停止/回收、数据保留、仍运行对象及理由和未解决项 | 已补充；文档任务不适用时如实记录，不为填表新启容器 |
| 脚本及测试静态核对 | local-bu的down使用Compose stop；local_runtime核对进程身份及数据库归属；集成夹具有t.Cleanup/Terminate；CI有always收尾 | 仅核对已有路径，不证明全部创建失败、超时或强制终止场景已运行通过 |
| make docs | 登记、标题、链接、来源摘要及治理检查器回归 | 通过；65个治理检查器用例及登记/链接/来源检查通过 |
| git diff --check | 本次文档及证据差异 | 通过；内容由本次文档自查核对 |

## 4 自查与处置

自查区分停止持久profile和删除测试临时资源，避免将数据保留写成常驻要求，也避免将常规停止扩大为删除持久卷或迁移权限。其他工作树或仓库只有明确纳入清理范围且归属核对通过时才处理；不新增逐次确认已获授权的自有临时资源清理流程。

新规则要求分配资源后立即登记清理，并处理创建失败时已产生的部分资源。本文不宣称现有全部测试已满足新规则的每条异常路径；测试后续修改或发现遗留资源时按003核对和补齐。本次为Codex文档自查，不记独立评审或人工业务批准。

## 5 未覆盖范围

不变更资源管理代码或建设自动清理平台；没有为本次文档修订新启数据库，也未重跑业务Go、DB/API、容量或浏览器验收。SIGKILL、主机故障等无法执行退出钩子的现场不计自动回收通过，按归属在下次任务开始或交接时核查。旧T07及其他定稿报告的范围和原始字节保持。

## 6 结论与后续

通用规则、M02引用、本地步骤和模板已连通，文档与差异检查均通过；结论仅覆盖本次规范和静态核对。后续实施者按003执行收尾，交付时列明保留运行的环境及理由；当前4527环境在下次新建验证环境或任务交接时复查运行需要。

## 7 证据保存

长期证据为本目录的清理结果、停止清单、只读核验、[输入摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/003-development-environment-cleanup/input-sha256.json)和[检查记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/003-development-environment-cleanup/checks.json)。前轮本机原始记录位于`.local/runtime-cleanup-20261102/`，仓内已保存本记录使用的非敏感原件及摘要，不依赖该忽略目录作为唯一证据。
