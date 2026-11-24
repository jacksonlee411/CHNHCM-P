# CHNHCM-311 第二批岗位层级受托签收记录

## 1 验收范围

**验收人**为用户在本会话委托的 Codex。基线为 `89ebdd068cd7f469e2bd5fd8ee223e5eab5faa02`。使用独立检出、空库和合成业务资料。验收状态与正式 CI、发布分别登记。 第一批之上的第二批增量补丁 SHA-256 为 `bb3228160ac7414557b6c6952ae758c2d85268d2f928e06b926ed04def39c290`。本批从「设置 → 职务配置 → 岗位层级」走查，按[312 对象合同](../specs/m03/312-position-contract.md#position-acceptance)的 AC053—055 作出签收决定。历史[开发验证](311-position-hierarchy.md)保留其原范围和日期。

## 2 签收结论

**2026-11-18 19:47（北京时间），Codex 受托签收通过。** 第一批签收后执行本批用例。签收环境 `accept311-p2-16e553` 使用 Web 48259、API 45659、DB 48459，最高迁移为 00022。独立检出的应用、API、迁移、内部代码及其测试共 1710 个文件与工作区逐文件一致。

| 合同项目 | 实际观察 |
| --- | --- |
| AC053 并发互设 | 两个不同管理员账号，使用独立登录会话，从同一版本同时通过 HTTP 发布互为上级的两笔关系。实收 200 和 409；败方按新版本重试，仍收到 409，提示 2027-02-01 成环。查询确认只有胜方的边，版本数和 Outbox 事件数各增一。锁内暂停的对照用例还分别验证了两种先后顺序。 |
| AC054 未来换上级及旧版 | 从管理汇报菜单发布第 1 版，A 自 2027-01-01 向 B 汇报。先取消一次换上级，原版号保持；再发布第 2 版，04-01 起换为 C。最新资料查 03-31 为 B、04-01 为 C；同查 04-01 选择第 1 版仍为 B，返回当前资料又为 C。切换职能协作用途可独立查阅。 |
| AC055 整段校验 | 在上述样本追加从 01-01 起让 C 向 A 汇报，页面点出首次成环的 **04-01**，第 2 版保持。真实数据库另验 05-01 停用端点的正向与反向冲突、先撤边再停用；Go 日期段用例覆盖重叠双父、三节点闭环及首尾相接的合法反向关系。 |

本轮 Chromium 走查包括独立发布场景，以及管理员、只读账号各自的 1440×900、390×844 菜单流程，共 5 项，退出码为 0。只读账号能查旧版，页面无发布动作。原开发记录保留完整视口及原生缩放实测。本次范围到岗位层级为止，实际人员经理关系仍归 M04。[结果摘录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/311-position-hierarchy/signoff-results.txt) · [04-01 查看第 1 版的页面](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/311-position-hierarchy/signoff-historical-parent.png)

复核时可把[签收脚本](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/311-position-hierarchy/signoff.spec.ts.txt)放入对应独立检出的 `tests/e2e/accept311.spec.ts`，加入 Playwright 的 testMatch，与原岗位层级菜单 V01、V03 用例同跑。数据库复核运行 `TestPositionHierarchyDates` 和 `TestStaffing312Hierarchy`。在后者的测试文件上用 Go overlay 装载[HTTP 并发补验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/311-position-hierarchy/signoff-admins.diff.txt)，两个客户端使用各自身份和提交号，在共同起点同时发送请求。
