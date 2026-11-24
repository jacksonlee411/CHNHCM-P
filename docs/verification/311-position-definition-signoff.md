# CHNHCM-311 第一批岗位定义受托签收记录

## 1 验收范围

**验收人**为用户在本会话委托的 Codex。基线为 `89ebdd068cd7f469e2bd5fd8ee223e5eab5faa02`。使用独立检出、空库和合成业务资料。验收状态与正式 CI、发布分别登记。 第一批补丁 SHA-256 为 `1253a114d4f24d627f9e543fbf49d9570fd0f97ef5861f70eb691958c1d30aae`。本批从「设置 → 职务配置 → 岗位」走查，按[312 对象合同](../specs/m03/312-position-contract.md#position-acceptance)的 AC049—052 作出签收决定。历史[开发验证](311-position-definition.md)保留其原范围和日期。

## 2 签收结论

**2026-11-18 19:44（北京时间），Codex 受托签收通过。** 候选的 55 个文件摘要全部匹配。签收环境 `accept311-p1-f3ff5e` 使用 Web 47045、API 48575、DB 48437，最高迁移为 00021。

| 合同项目 | 实际观察 |
| --- | --- |
| AC049 正向 | 在职务菜单安排 2027-04-01 移除 G5，再建立 2027-01-01 起的长期岗位。岗位表单显示期间内无可选职级；带 G5 直接写入返回 409 并指出 04-01。留空职级后从菜单保存成功（201）。有限段恰在 04-01 结束时仍可用 G5，多延一天返回冲突日 04-01。 |
| AC050 反向 | 在另一职务下先建使用 G5 的岗位，再从职务菜单移除 G5。页面指出“自 2027-04-01 起仍有岗位采用此职务或职级”，职务版本保持原值。补测两个不同管理员主体，两种真实抢锁顺序均只有先到者成功。主动在 04-01 清空岗位职级后可移除，03-31 仍保留原职级。 |
| AC051 引用 | 已采用的部门、地点、职务和职级缩短有效期时被拒；缺失必填引用返回 400。地点更名后现行资料和原采用分别显示新名、原名；撤回地点历史权限后原来源按权限隐藏。 |
| AC052 菜单 | 管理员新建无职级岗位，02-01 新增职级、放弃一次修改、同日更正，05-01 停用；从变更日志回到当次资料。只读账号从同一菜单查阅；写接口返回 403。 |

业务菜单使用桌面 1440×900、窄屏 390×844 和两个角色，并单独从职务与岗位菜单走 G5 正反顺序，共 5 项。首次外层等待命令退出 1，浏览器记录为 5 项通过；用新样本直接运行 Chromium 后复验 5 项通过，Playwright 退出码为 0。完整视口与原生缩放覆盖见原开发记录。本次签收以新走查及真实 PostgreSQL 断言为依据。[结果摘录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/311-position-definition/signoff-results.txt) · [页面冲突提示](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/311-position-definition/signoff-reverse.png)

复核时可把[签收脚本](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/311-position-definition/signoff.spec.ts.txt)放入第一批独立检出的 `tests/e2e/accept311.spec.ts`，加入 Playwright 的 testMatch，与原岗位菜单 V01、V03 用例同跑。数据库复核运行 `TestStaffing312Position` 和领域的 `TestPosition*`，再用[测试差异](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/311-position-definition/signoff-admins.diff.txt)给同一并发用例配置第二个独立主体。本次使用 Go overlay 装载这份测试差异。
