# CHNHCM-140 阶段评审修复与复验

2026-11-22，wt-dev-b，基线 `b647031b8f64abf7447d81888d1dd399354bb4d5`。工作区及 stash 开始时为空。[路线图](../plans/140-module-cohesion-roadmap.md)的 A02、A04 在原分项验收后收到以下反例。[原签收](140-module-cohesion-implementation.md)

## 1 SQL 归属

在 Organization 查询目录分别放入两条真实表查询的独立样本。两条都经临时 PostgreSQL 的 `EXPLAIN` 成功。

| 样本 | 基线检查器退出码 | 修复后退出码 |
| --- | --- | --- |
| 括号连接首表为 `access.policy_guard`，连接本域业务单元 | 0，漏报 | 1，指出跨域表 |
| 仅连接两次本域业务单元，`IS DISTINCT FROM access.head_source_id` 使用本域表别名 | 1，误报 | 0，放行 |

在原实现上先运行扩充的检查器用例，发现五项不符。修复后用例全部通过，覆盖嵌套括号、带注释的 `IS NOT DISTINCT FROM`、`EXTRACT` 及比较子查询中的实际跨域读取。[检查器与回归](../../scripts/check-boundaries/main_test.go) · [目录合同](../architecture/004-repository-layout.md)

## 2 成对删除

基线在正常菜单复现两次。管理员先建立法律实体或地点，更正其已采用的国家目录名称，再插入停用与恢复记录。恢复记录从 2022-01-01 起采用新名。删除确认的 API 与页面均未列出变化，确认后“美国新名”变回“美国”，“澳门新名”变回“澳门”。按删除前原来源仍能读到新名。

修复在读取删除影响时保留采用版本不同的期间，再按当前权限投影并比较前后资料。受限标签及同名新版本不会单独生成变化行。页面展示获权字段值或其显示名称的差异。[按日区间](../../internal/contexts/organization/domain/dated/lifecycle_pair.go) · [删除确认](../../apps/web/src/features/organization/LifecyclePairDialog.tsx) · [技术合同](../specs/m02/013-m02-technical-design.md)

T04 的触发范围为两对象的历史写入、修订组装和提交恢复。本次改动归属 A04 的删除影响读取与展示，T04 继续由下一次触及共同历史规则的 M02 任务接续。

## 3 实际结果

| 菜单样本 | 服务端和确认框 | 取消、失权、确认与历史 |
| --- | --- | --- |
| 法律实体，登记国家为美国 | 恢复端采用更正后的美国名称。从 2022-01-01 起，确认框逐值列出新名到原名。 | 取消后来源相同。撤去国家历史读权，API 返回空变化集，页面隐藏变化区。恢复权限后确认删除，详情显示原名，按删除前来源仍读到新名。 |
| 地点，国家/地区为澳门 | 同一澳门身份在恢复端采用新名，显示相同开放期间和名称变化方向。 | 分别核对取消、国家历史撤权、确认及原来源，结果与上述预期相同。 |
| 地点，西乡塘地址 | 区划祖先更名，末级身份保持。前后完整路径都有准确的祖先名称。 | 撤去区划历史读权后变化集为空。另在 390×844 窄屏核对路径文字完整、变化区无横向溢出，取消及确认均可达，删除前原来源保留。 |

三组均在成对插入后将目录现名更正回原值，再读删除影响，确保显示取自两边准确采用的版本。复验脚本见[公共断言](../../tests/e2e/dated-object-support.ts)、[法律实体场景](../../tests/e2e/legal-entity206.spec.ts)和[地点场景](../../tests/e2e/location204.spec.ts)。

- `make check`、`make build`、`go test -count=1 ./internal/contexts/organization/domain/dated ./scripts/check-boundaries`、仓内引用和空白检查通过。
- 真实数据库按 `GOFLAGS='-run=Test(LocationRegionPickerCandidates|LifecyclePair|RedateJourney|Enterprise304FutureFirst|LegalEntity206(Pairs|ExactCountry)|Location204)' make integration` 执行。两个包分别耗时 56.417 秒和 528.948 秒。所选 16 个顶层用例中 15 项通过，需显式开启的 10,001 地点容量用例跳过。
- 临时生产构建使用[原 140 浏览器筛选命令](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/140-module-cohesion-implementation/results.txt)，覆盖原有 16 项和新增 3 项，当轮 19 项全过（3.9 分钟）。区划候选连续请求的状态为 200、200、200、409、200、200、409、200、200、403。另一临时环境按 `--no-deps -g '140.*成对删除显示准确采用' legal-entity206.spec.ts location204.spec.ts` 复验上述 3 项，加入窄屏断言后全部通过（29.5 秒）。
- 最终候选相对基线在 `apps/`、`internal/`、`scripts/`、`tests/` 中的 11 个修改文件，按路径排序，以每行“文件 SHA-256、两个空格、相对路径、换行”计算清单 SHA-256，结果为 `78f05a6bdfa5aaf5d776fe64d8ae5fe9d8fc4995a377f51d30ed4c8e79028f45`。
- 两个临时环境按 Compose 项目标识 `chnhcm-bu-468fd5f52e24`、`chnhcm-bu-22ca7e2a8280` 核对，容器、网络、卷各为零。本地验证限于上述范围。全量集成、全量浏览器、远端 CI 和人工业务签收均未执行。
