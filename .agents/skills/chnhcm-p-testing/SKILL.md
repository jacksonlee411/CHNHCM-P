---
name: chnhcm-p-testing
description: 为 CHNHCM-P 的业务变更设计、修改或执行回归测试，以及核对其验证证据时使用。按 P 仓 Makefile、业务合同与实际改动选择检查。
---

# 验证 CHNHCM-P 变更

用当前工作树的[项目约定](../../../AGENTS.md)、对象合同和[验证规范](../../../docs/governance/137-actions-and-validation.md)确定独立预期。业务用例写清角色、菜单、前置和预期结果。按变更涉及的行为补充拒绝、取消、更正、历史查回或重试用例。

从本分支的 [Makefile](../../../Makefile)选择入口。文档登记与链接用 `make docs`；模块、SQL 边界用 `make boundaries`；通用代码检查用 `make check`；真实 DB/API 路径用 `make integration` 或其对应分片；运行冒烟用 `make smoke`。候选的 CI 影响范围可用 `make ci-preview` 查询。

浏览器按[共享交互规范](../../../docs/specs/m02/012-m02-functional-and-interaction-design.md)和对象合同实测角色、视口及关键流程。先核对 profile 和端口，需要写入或模拟失败时使用任务自有环境；按[运行指南](../../../docs/guides/017-local-bu-development.md)回收临时资源并保护持久验收环境。

记录基线、实际命令、首败、修复后的复验和未执行项。前置检查与业务结果分别记载；旧版本上的成功只证明当时的受测范围。
