# CHNHCM-005 GitHub CI 发布准备验证

登记信息：[registry.json](../governance/registry.json) 的 `governance.verification_records`。
适用范围：CHNHCM 远程发布准备及 CI 定义，不记录尚未发生的远程结果。
上游依据：[005](../governance/005-branches-and-documents.md)、[017](../guides/017-local-bu-development.md)、[018 多轮评审](018-multiround-review.md)。

## 1 验证对象与受测基线

2026-10-31，工作树 `/home/vivobook/projects/CHNHCM-wt-dev-a`，固定分支 `wt-dev-a`，基线 `1798a2a8a14194701bafaaee0591a2b8dbaf18a7`。该提交已保存前轮评审修复。本次差异仅为 CI、005 协作规则、AGENTS、导航及登记/本报告；受测文件摘要见[证据](005-github-ci-preparation-evidence.json)。不修改业务实现、原规格和历史验证结果。

## 2 环境与前置条件

本机 Linux/WSL，Go 1.26.0、Node 24.19.0、pnpm 10.24.0、Corepack 0.35.0；工作流目标为 ubuntu-24.04，版本取现有锁定文件。用户授权创建远程仓库、推送、PR、CI 通过后合并及同步全部分支。main 与 a 各自固定工作树，b–f 仅有分支和预留目录。认证凭据始终在仓外维护，不写入报告或输出。

## 3 验证过程与结果

| 检查 | 操作 | 预期 | 实际 | 结论 | 证据 |
| --- | --- | --- | --- | --- | --- |
| 工作流语法 | actionlint v1.7.12 | 无错误 | 退出 0 | 通过 | 同名 evidence.json |
| 静态与单元检查 | make check | 现有检查通过 | 退出 0；46 项 Python 检查、Go、类型及文档检查通过 | 通过 | 同名 evidence.json |
| 构建 | make build | Go 和前端可构建 | 退出 0 | 通过 | 同名 evidence.json |
| 生成一致性 | make generate 后核对差异 | 生成文件无漂移 | 退出 0，生成文件未变 | 通过 | 同名 evidence.json |
| 修订后的文档与登记 | make docs、git diff --check | 通过 | 退出 0，登记与链接检查通过 | 通过 | 同名 evidence.json |
| 历史保护 | 对照基线文件摘要 | 原评审证据不变 | 018 两轮报告及证据原字节保留 | 通过 | 同名 evidence.json |
| GitHub 执行 | PR 的 quality、integration、browser；合并与 refs 同步 | 全部成功后合并 | 本报告截点尚未运行 | 未运行 | 以后以实际 PR、Actions、refs 为准 |

## 4 失败、修复与复验

发布准备时本机旧 gh 凭据返回 HTTP 401；用户报告登录后本环境仍未恢复。旧版 gh 设备授权遇到 slow_down 中断，使用任务专用新版客户端重试认证，未修改系统安装。认证失败不记为 CI 失败，也不以连接器身份替代尚不可用的仓库创建权限。失败日志不包含令牌。

## 5 未覆盖范围与限制

本次本地检查不重复 018 的数据库和 Chromium 业务复测，业务受测输入没有改变。GitHub CI 会在独立运行环境重新执行这两部分；本地通过不能代替远程通过。后续变更、更正、删除、停用等未实施动作不在本次范围；不部署应用，不连接客户系统。

## 6 结论与后续事项

工作流通过本地语法、现有构建及资料检查。获得有效认证后创建私有 CHNHCM 远程，以 main 初始化提交为基线打开开发分支 PR；只在最新提交三项 CI 均成功时合并，再快进同步七个分支。远程地址和配置写入登记表，最终 PR、CI 与同步结果在交付中提供可核验链接及 SHA；本准备记录不回填为远程成功记录。

## 7 证据保存

[同名证据](005-github-ci-preparation-evidence.json)保留受测文件和非敏感原始日志的 SHA-256、执行结果及历史保护摘要。原始本地日志位于 `.local/publish-ci/`；摘要及本报告为仓内长期证据，Git 保留实际文件内容。日志不包含授权凭据；GitHub 浏览器任务仅上传经过测试遮罩的失败 PNG，不上传 Cookie、trace 或仓外状态。
