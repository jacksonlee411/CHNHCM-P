# CHNHCM-100 b运行隔离验证

登记信息：[registry.json](../governance/registry.json) 的同路径验证记录。适用范围：[100 的 T01](../archive/plans/100-ui-experience-improvement.md#ui100-runtime-isolation)，操作方法由 [017](../guides/017-local-bu-development.md) 持有。本次完成 b 独立运行及其必要反例验证；a/b 同时在线未运行，留 T05 补验。

## 1 验证对象与受测基线

2026-11-01，工作树 `CHNHCM-wt-dev-b`，分支 `wt-dev-b`，HEAD `8ae0a44353ee8105dfdef9a1b15fde9af345a144`，dirty。进入本任务前已有 AGENTS、005、registry、100 草案与导航修改；继续保留。本批新增运行配置、Vite/浏览器入口接线及必要隔离测试，更新 017 和计划进度；Go 领域、API、迁移、012 正文及产品页面行为未修改。

实现位置为 [local-bu.py](../../scripts/local-bu.py)、[归属与配置模块](../../scripts/local_runtime.py)、[Node 入口](../../scripts/local-runtime.mjs)、Vite 配置和页面启动命令、Playwright 配置/准备/夹具、BU 请求 Origin 及可选图面对照目标。配置单元检查在 [test_local_runtime.py](../../scripts/test_local_runtime.py)。[115 份输入摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime/tested-inputs.json)在生命周期验证前采集，结束后逐项一致；完整浏览器回归后没有继续修改运行及测试代码。

## 2 环境与前置条件

本机 WSL2 Linux 6.6.114.1、Go 1.26.0、Node 24.19.0、pnpm 10.24.0、Python 3.12.3、Docker 29.1.3、Compose 2.40.3、Playwright 1.55.1 / Chromium 140.0.7339.186。b 执行 `make install` 独立安装锁定依赖，PostgreSQL 18.6-alpine 使用仓内锁定镜像。查验空闲后保存 b Web/API/DB 端口 **4381/8381/54371**，全部只绑定 127.0.0.1。

运行配置目录按工作树真实路径摘要 `39bc5023cef2` 区分；Compose 项目为 `chnhcm-bu-39bc5023cef2`。配置和随机凭据只在仓外受保护文件，日志和报告不保存其内容。a 的默认 4380/8380/54370 未发现服务，本轮不启动 a；现有 CubeBox、Odoo 等监听不属于本任务。

## 3 验证过程与结果

| 场景或检查 | 前置与操作 | 预期与实际 | 结论 | 证据 |
| --- | --- | --- | --- | --- |
| 配置和进程边界 | 执行 13 项新增单元检查；使用任务自有 socket、子进程和仓外临时目录 | 默认/非默认配置、非法端口、冲突环境、私有文件、生命周期锁均按约束处理；外来进程、变更的启动时间和目录不被停止 | 通过 | [完整 check](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime/check-second.txt) |
| 静态、单元、类型及构建 | `make check`、`make build` | 59 项 Python 检查及 Go 单元/边界/vet、前端类型检查通过；独立产物构建成功 | 通过 | [check](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime/check-second.txt)、[build](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime/build.txt) |
| 完整 Chromium 回归 | b API/DB 已运行，单 worker 从正常菜单执行既有 27 项用例 | 创建、未来首条、字段拒绝、取消/未知提交、当前授权、来源历史、查询返回及公共组件检查 27/27；既有受控故障注入仍按原边界解释 | 通过 | [smoke](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime/smoke-first.txt) |
| b 实例真实创建与来源 | 独立 context 管理员从设置菜单新建虚构 `T01_d94af57b`，返回查询，打开详情/日志/当次来源 | API 会话实例与 b 仓外实例一致；列表恰好 1 个对应 BU，日志 1 条，来源版本明确；不是健康接口或 mock 证明 | 通过 | [浏览器生命周期](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime/browser-lifecycle.json) |
| 重启与当前库维护 | 连续两次 `up`，每次重新从菜单查同一 BU；之后停止、重新启动、执行 `migrate` 再查回 | BU 身份和来源版本保持；初始化不复制事实；同一容器、卷、端口和凭据保留；迁移后仍可读确切来源 | 通过 | [37 项生命周期检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime/lifecycle.json)、[浏览器结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime/browser-lifecycle.json) |
| 外来服务与误目标 | 任务自建 Web/API 冲突服务；错误 origin、端口占用、伪装 200/标识、已有库改端口 | `up` 在停止/写库前拒绝；错误 origin 零 HTTP；伪装 Web 被 Playwright 全局准备拒绝，只有就绪 GET、零 POST 和零凭据头；已有 API 在冲突拒绝后 PID 保持 | 通过 | [生命周期](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime/lifecycle.json)、[拒绝日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime/reject-healthy-web-before-login.txt) |
| 停止范围 | 先正常停止，随后在冲突服务仍在线时执行 `down` | b 自身 API/Web 退出、DB 停止且卷保留；任务自建外来服务仍监听；不按端口猜测 PID | 通过 | [生命周期](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime/lifecycle.json) |
| 独立 context | b 管理员和查询者分别登录；管理员退出后重载查询者页面 | 管理员会话 401，查询者仍 200 且能查列表；查询者新建入口保持禁用 | 通过；仅 b 内两个 context | [浏览器生命周期](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime/browser-lifecycle.json) |
| 生产预览及 CI 启动形式 | `up --no-web` 后分别启动 preview 和 Playwright 自有 Vite；额外指定错误 Vite 端口 | preview 归属与真实 BU 来源查回通过；错误 CLI 端口拒绝；自动 webServer 路径的 1 个完整 BU 流程加 2 个组件用例 3/3，套件结束自行关闭 Web | 通过；本机非远程 CI | [preview](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime/browser-preview.txt)、[CI 形式](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime/ci-webserver-path.txt)、[CLI 拒绝](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime/reject-vite-cli-port.txt) |
| 独立真实 DB 回归 | `make integration` | Testcontainers 套件成功，不复用 b 开发库；容量开关未启用 | 通过 | [integration](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime/integration.txt) |
| 代码、来源及其他工作保护 | 核对输入、Git、历史资料与已有监听 | 115 份受测输入一致；332 份已跟踪后端/来源/历史资料保持 HEAD 内容；a/b HEAD、a index 保持，b 未暂存；已有监听仍在 | 通过；a 文档并行变化另见第 5 节 | [保护核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime/protection.json) |
| 文档与登记 | 补齐报告导航，完成登记后重新执行文档/引用与差异检查 | 最终报告、元数据、路径、固定分支登记及差异格式有效 | 通过 | [最终文档检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime/docs-final.txt)、[固定引用](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime/refs.txt) |
| a/b 同时在线 | 本轮 a 未运行，未启动 a 或写入其数据 | 无实测结果；任务自有冲突服务不替代该项 | 未运行 | [保护核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime/protection.json) |

`lifecycle.json` 的 37 项为命令与状态断言，不是 37 项业务验收；其中预期返回非零的拒绝用例以“拒绝发生且状态保留”为通过条件。最终 b 已重新启动并通过 `verify`，当前入口为 `http://127.0.0.1:4381`，见[最终状态](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime/final-status.txt)。

## 4 失败、修复与复验

首次 `make check` 的 59 项 Python 测试通过，但仓库检查发现新报告不可从入口到达；补入验证导航后完整重跑通过。[首次日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime/check-first.txt)仍保留。

补充生命周期浏览器探针前三次失败：[第一次](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime/browser-seed.txt)及[细分阶段后的第二次](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime/browser-seed-second.txt)把详情标题误写为仅名称，实际合同是“编码 · 名称”；[第三次](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime/browser-seed-final.txt)误以为只读新建入口应消失，实际应禁用。核对现有代码及原浏览器合同后只修正探针预期，[第四次](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime/browser-seed-fourth.txt)创建/来源/context 验证通过，后续两次重启、preview 和 migrate 的同对象查回均通过。前三次实际创建的虚构记录保留，没有为使测试通过清库或改产品行为。原完整 27 项浏览器回归一次通过。

构建存在已有依赖的 `use client` 指令和大 chunk 提示，本批不调整包拆分或降低阈值。实施者完成差异自查与本机验证，没有执行独立代理评审，也不记为人工批准整个页面方案。

## 5 未覆盖范围与限制

a/b 同时在线验证未运行，留 T05 补齐；同一浏览器 profile 的多端口 Cookie 不隔离，本批采用不同 profile/context。仅支持本机 Linux/WSL2 与本机 Docker，未验证远程 Docker、原生 Windows或其他浏览器。既有数据库带数据改端口明确拒绝；没有重演 00003 旧库升级，保留原升级顺序及 018 历史结果，不将当前库 `migrate` 当旧库升级验收。

a 的 HEAD/index 未变，但期间 registry、010、011、013 内容及工作区状态有并行变更；因此不声称 a 整个工作区前后摘要相同。本轮对 a 只读核对，没有覆盖、回滚或同步这些文件。已有服务的可见 PID/端口保持只证明本轮监听未被停止，不扩大为这些产品的业务验证。

T02 问题基线、012 正式修订、R01—R10 页面优化及真实 200%/400% 缩放均未执行；可选 CubeBox 对照入口仅完成配置接线，没有重采 019 图面。容量、生成、远程 CI/PR、发布、生产恢复和完整 M02 不在本轮结果中。

## 6 结论与后续事项

T01 的独立配置、真实创建查询、错误目标拒绝和停止范围满足本批完成条件，可以进入 T02。100 整体保持草案，执行进度为 in_progress；后续页面提案不自动成为现行合同。下一批应先从正常菜单复现 R01—R09，形成问题基线，再审阅和修订 012；a/b 同时在线检查由 T05 补齐。未提交、推送、建 PR 或合并。

## 7 证据保存

长期证据位于 [data/100-runtime](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime/lifecycle.json)，原始非敏感输出保留在 `.local/t01/`。留存[浏览器探针](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime/browser-probe.mjs)和[生命周期探针](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime/lifecycle-probe.py)用于复核；执行时它们位于 `.local/t01/`，相对路径按该位置解析，直接在文档目录运行不属于本次步骤。

[输出检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-runtime/output-scan.json)核对了本轮输出、运行日志和 Playwright 目录中没有当前 b 的四类随机密码值；这不是一般性秘密扫描认证。Cookie、CSRF、storageState 和 trace 不进入证据；报告只保存虚构 BU 身份与来源引用，不保存会话主体或登录值。现有 019、018 报告和来源资产状态不提升、不覆盖。
