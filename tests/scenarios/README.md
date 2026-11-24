# tests/scenarios

当前正常菜单回归见 tests/e2e；业务规则及稳定案例见[023](../../docs/specs/m02/023-m02-timeline-test-cases.md)。

109退出通用联合编排后，其重启恢复、拒绝恢复、旧联合视口与t07-recovery运行脚本退役，原内容保存在[109归档](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/109-lifecycle-pair-maintenance/retired-general-runners.tar.gz)，不作为现行运行工具。固定两条的恢复、并发、取消及原件保护分别由 lifecycle-pair.spec.ts、lifecycle_pair_test.go 和已迁移的 bootstrap 真库故障用例验证。旧报告及其中证据保持原样。

111 T03的真实列表业务见 tests/e2e/community-list.spec.ts，需先选择并核验本工作树的隔离profile。该用例通过当前API建立唯一前缀的73个合成BU，记录非敏感fixtures.json；随后使用同一profile及输出目录运行 community-list-viewport.mjs 和 community-list-native-links.mjs，分别检查七组条件与原生复制链接地址。两者自有Xvfb/浏览器自动关闭，不启动或停止既有BU环境。完整命令、结果和资源收尾见[111 T03报告](../../docs/verification/111-community-t03.md)。
