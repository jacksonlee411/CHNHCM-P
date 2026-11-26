# CHNHCM-116 R1资料与引用比较评审修复

登记信息：[registry.json](../governance/registry.json)的verification_records。适用范围：R1提交后评审发现的P2测试夹具问题。上游为[116](../archive/plans/116-m02-shared-dated-maintenance-roadmap.md)、[120比较约定](../specs/m02/120-dated-maintenance-technical-contract.md)及[121](../specs/m02/121-dated-object-admission-matrix.md)；[原实施报告](116-r1-kernel-and-directory.md)保持原记录。

## 1 验证对象与受测基线

2026-11-05，wt-dev-b，基线e7b72606298732182c317ec3773ddde424285731，开始时工作区及stash干净。本次仅修改dated包的candidate_test.go和impact_test.go，以及本轮文档及证据。受测测试差异见[source.patch](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1-review-fixes/source.patch)，摘要见[source.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1-review-fixes/source.json)。

## 2 环境与前置条件

Go 1.26.0，Linux/amd64。本轮是纯Go测试夹具修复，由主代理执行和复核；不涉及登录角色、页面入口或业务数据写入。未创建或启停容器、浏览器profile、持久服务及数据卷。

## 3 验证过程与结果

原夹具的EqualRevision比较完整fixtureData，名称变化也被当作引用结构变化。修复后仅比较准确关系引用Ref。三类案例保持同一关系目标root-a，只分别改变名称、引用版本或两者，独立断言候选资料、NoChange、数据影响期间、REFERENCE结构面、受影响记录及无期间变化。

| 场景或检查 | 操作及预期 | 实际结果 | 结论 | 证据 |
| --- | --- | --- | --- | --- |
| 原夹具新增反例 | 仅名称变化应有数据影响、没有结构影响 | 纯资料案例错误返回REFERENCE；另两类通过，测试退出1 | 失败，确认缺陷 | [首次结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1-review-fixes/regression-before.txt) |
| 纯资料 | 名称same→new，引用保持edge-v1 | 数据影响为2026-01-01起，结构变化为空，目标记录保留为实际变更 | 通过 | [复验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1-review-fixes/regression-after.txt) |
| 纯引用 | 名称保持same，引用edge-v1→edge-v2 | 数据影响为空，准确产生REFERENCE，仍须形成修订 | 通过 | 同上 |
| 资料及引用同时变化 | 名称及引用版本同时修改 | 数据影响与REFERENCE均保留 | 通过 | 同上 |
| Organization全包 | go test ./internal/contexts/organization/... -count=1 | 应用、授权适配、BU领域及dated测试通过；包含已有复制、继承和无变化断言 | 通过 | [全包结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1-review-fixes/organization.txt) |
| 仓库综合检查 | make check | 文档、边界、单元、vet、类型及差异检查通过 | 通过 | [检查输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1-review-fixes/check.txt)、[退出记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1-review-fixes/check.json) |

## 4 失败、修复与复验

本问题记为R1-09，承接提交e7b7260后的P2评审发现。先只加入三类独立预期，用原夹具重现错误，再修改EqualRevision。原两者同时变化的单一测试扩充为完整三类断言，保留原有引用单独新增、更正、停用继承及恢复独立资料测试。运行代码和120合同均不需要修改。

首次失败输出与修复后通过分别保存；原R1报告和final-review.json仍只表示原批次检查。本次补验关闭的范围是测试夹具对资料与结构影响的区分，不把合成类型称作部门真实接入。

## 5 未覆盖范围与限制

本轮未运行DB、浏览器或产品构建，运行源码未变化；本轮通过不替代原报告的分层证据和浏览器多轮复验边界。R2—R6、部门及Hierarchy保持原未实施状态。无独立评审、人工签收、远程CI或生产验收。

## 6 结论与后续事项

夹具修正、三类反例、Organization全包及仓库检查通过；主代理复核确认R1-09已关闭，无本次未解决发现，见[复核记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1-review-fixes/review.json)。本次结论补齐R1的资料/结构比较验证，不扩大原实施范围。本轮不产生临时运行资源，环境收尾不适用；已有持久环境和证据保留。本轮更改尚未提交。运行源码、既有验证/引用/资产未改动，2634份原保护清单文件保持原字节，见[范围核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1-review-fixes/scope-check.json)。

## 7 证据保存

新证据位于[data/116-r1-review-fixes](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1-review-fixes)，含首次失败、复验、全包日志和测试源码差异；不覆盖[data/116-r1](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r1)原结果。
