# CHNHCM-020 T05评审修复与复测

登记信息：[registry.json](../governance/registry.json)。适用范围：T05评审发现的共用终态反馈缺陷及当前实施说明。依据[010](../plans/010-m02-scope-and-plan.md)、[013事务恢复边界](../specs/m02/013-m02-technical-design.md)、[023案例](../specs/m02/023-m02-timeline-test-cases.md)。原[实施报告](020-delete-implementation.md)保留原结果。

## 1 验证对象与受测基线

2026-11-01，固定工作树CHNHCM-wt-dev-a、分支wt-dev-a，基线952b61fd97b388f932aef2508e4ea8d28468b886；开工工作区及stash为空。基线外仅修复共用命令及resolve的事务结果出口，补回归和本次文档。受测实现与测试见[输入摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-review-fixes/tested-inputs.json)和同目录source.patch.gz；报告最终提交不是所有测试的受测起点。

## 2 环境与前置条件

Go1.26.0、Node24.19.0、PostgreSQL18.6，DB/API使用独立Testcontainers；正常菜单验证使用新profile m02-t05-fixes（Web4476/API8476/DB54556）。旧4466及其他环境保留原程序与数据库。凭据只在仓外受保护目录，日志不记录Cookie、CSRF或密码。管理员从“设置 → 组织配置 → 业务单元 → 列表 → BU详情 → 更多操作 → 删除此记录”办理。

## 3 验证过程与结果

| 场景或检查 | 操作与实际结果 | 结论及证据 |
| --- | --- | --- |
| 原评审反例 | 三类DELETE分别在来源冲突及写后409的拒绝行写入后取消；原6组返回400/409但无终态，修复后均503且不含伪终态，resolve才封闭not_applied | 原失败保留；复验通过，[原结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-review-fixes/original-review-failure.log)、[复验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-review-fixes/original-probe-final.log) |
| 已开放写动作 | CREATE/ADD/CORRECT/DEACTIVATE/REACTIVATE及三类DELETE，共8动作写后409；原事务确认已存在rejected后取消。CREATE/DELETE另测截止耗尽和真实延迟约束COMMIT失败，共12参数。各例全部业务表不变，交替验证原身份重试先成功或resolve先封闭、唯一终态及迟到请求拒绝 | 通过，[完整DB/API](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-review-fixes/integration.log)，初轮独立12参数输出见[专项](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-review-fixes/after-focused.log) |
| 四种结果出口 | 原事务先读取并核实accepted/no_change/rejected/not_applied，再在COMMIT前取消；公共应用服务返回零业务结果及原取消错误，业务表保持原快照 | 通过，[专项](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-review-fixes/result-boundary-final.log)及完整集成回归 |
| 正常菜单故障恢复 | 管理员菜单创建甲、6月新增乙并预览删除；另一合法请求新增10月丙使来源过期。仅对该BU的rejected设置延迟约束失败，真实API返回503而无outcome。页面保留原引用；同页原正文重试后查回rejected，刷新后查回not_applied，两种均唯一终态、头和旧来源保持 | 通过，[专项2/2](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-review-fixes/browser-focused.log)；没有HTTP mock |
| make generate / make check / make build | 当前生成一致；文档、边界、Go测试/vet、前端类型、差异检查及构建通过。既有前端主块大小提示保持 | 通过，[生成](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-review-fixes/generate.log)、[检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-review-fixes/check-final.log)、[构建](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-review-fixes/build.log) |
| make integration | tests/integration及internal/bootstrap两个真实DB测试包全部通过；包含当前安装、创建/查询、维护/启停/删除、权限/历史及恢复回归，容量专项按原范围跳过 | 通过，[日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-review-fixes/integration.log) |
| 独立环境make smoke | 桌面Chromium完整61/61通过，含新增2条真实COMMIT失败恢复和全部原59条回归 | 通过，[完整日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-review-fixes/smoke.log) |
| 历史与运行一致性 | 757份既有报告、证据及来源原件保持；原登记记录和资产库存未改，仅导航追加。新profile程序与本轮构建摘要匹配 | 通过，[历史核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-review-fixes/history-protection.json)、[程序匹配](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-review-fixes/runtime-match.json) |

[TL及参数映射](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-review-fixes/tl-mapping.json)承接TL-201/202/204/209/211/213本次实际参数。代表验证不关闭完整联合、网络故障或后续批次案例。

## 4 失败、修复与复验

P1：拒绝行INSERT成功不代表外层COMMIT成功。原实现回调填入rejected，取消时虽回滚全部业务表，API仍按该临时结果返回400/409并导致页面清除引用。修复后WithinWrite报错即返回空业务结果及原错误；只有外层成功才交付accepted/no_change/rejected，resolve同样保护not_applied。没有后台补写或第二笔拒绝事务。

P3：011当前实施说明仍标T05入口未完成。同步到实际已接线及本次修复证据，T06/T08、T01/T02跨后续动作及T07仍保留。

新增12参数在修改运行代码前全部失败，见[修复前](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-review-fixes/before.log)，修复后通过。四结果专项首次夹具遗漏请求UUID，未抵达not_applied边界；补齐请求标识，并在原事务先核实目标临时终态，后续四参数及完整集成均通过，原[夹具失败](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-review-fixes/result-boundary.log)保留。文档草稿首次登记缺results被检查器拒绝，补齐实际结果后make check通过，保留[首次输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-review-fixes/check-draft-failure.log)。

## 5 未覆盖范围与限制

本次真实故障为提交前取消/截止时间及PostgreSQL延迟约束在COMMIT阶段失败；不等同所有网络通信未知、生产恢复或容量验收。页面恢复使用真实API、真实DB故障，没有模拟HTTP成功/失败响应。T06/T08、T07完整容量和真实下游消费者未实施。

## 6 结论与后续事项

再次评审由Codex执行者自查完成，不冒充独立人工批准。逐一复查两个WithinWrite调用点、技术错误与已持久业务拒绝的分流、原提交锁和savepoint、当前授权、不可变历史及页面引用清理；[复审记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-review-fixes/re-review.json)未发现新的遗留问题。P1与P3均已修复，完整受影响检查通过，本轮再次评审通过；该结论限于已列参数，不等于完整M02验收。

下一批任务保持010原顺序，本次不启动改日或联合；不推送、创建PR、合并或部署。

## 7 证据保存

必要原始输出与复核材料保存在[data/020-delete-review-fixes](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-review-fixes/tested-inputs.json)；.local/verification/t05-fixes仅为本机原始副本。cancelled-rejection-probe.txt保存原评审隔离Go测试，通过-overlay映射到internal/bootstrap/review_t05_test.go后运行对应TestReviewT05CancelledRejectionMustRemainPending；不修改受审源文件。正式新增回归为internal/bootstrap/command_commit_integration_test.go及tests/e2e/delete.spec.ts。历史报告保留原失败与结论，不改写为本轮通过。报告定稿后的文档及差异检查另见[封存检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/020-delete-review-fixes/docs-final.log)。

入库日志副本仅移除工具输出的行尾空白，以通过git diff --check；.local原始输出保留。提交前逐文件核验暂存证据摘要，避免被全局日志忽略规则遗漏。
