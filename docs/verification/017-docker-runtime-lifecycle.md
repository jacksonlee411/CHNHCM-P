# CHNHCM-017 Docker地址池与临时验收环境验证

登记信息：[registry.json](../governance/registry.json)。适用范围：Docker地址池、独占临时验收环境及资源收尾；上游为[003](../adr/003-implementation-stages.md#development-environment-cleanup)和[017](../guides/017-local-bu-development.md)。

## 1 验证对象与受测基线

2026-11-03，工作树wt-dev-a，开工HEAD为`a27caf82e6c30b406dc574c6d809659ac3931742`，工作区与stash干净。本次修改开发运行脚本、Compose标签、Makefile、CI接线及操作文档，不修改Go业务模型和数据库迁移。受测实现、测试和浏览器输入见[摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/017-docker-runtime-lifecycle/input-sha256.json)及[输入归档](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/017-docker-runtime-lifecycle/runtime-inputs.tar.gz)。正常菜单检查后，仅补充列表端口显示、孤立资源保护测试及回收时构建文件/元数据的删除顺序；最终Python/Node检查覆盖这些变更，不据此宣称重跑完整浏览器套件。

## 2 环境与前置条件

Ubuntu、Docker Engine 29.1.3、Compose 2.40.3。临时验收使用独占profile、自动选择空闲端口、当前模型新库及合成BU；凭据保持仓外，不进入证据。既有持久环境及旧验证报告保持原样。

修改前共33个bridge网络，其中30个CHNHCM网络。默认池已占用15个/16和15个/20，剩余192.168.0.0/20与局域网重叠；已有3个10.*网段为此前任务绕过地址池耗尽而创建。Windows路由192.168.70.0/24、192.168.146.0/24还分别与旧Docker网络192.168.64.0/20、192.168.144.0/20重叠。本次只选择无当前路由重叠的新池，不重编旧网络或宣称这些旧网段的连通性已经验证。

宿主机配置为`default-address-pools:[{base:10.246.0.0/16,size:24}]`；先reload启用live-restore，再显式保留原bip=172.17.0.1/16并重启应用地址池。4个基线运行数据库的PID及启动时间均未变化，原容器状态和挂载一致，见[应用核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/017-docker-runtime-lifecycle/daemon-applied.json)。这是本次实例的实测，非所有Docker重启场景保证。

## 3 验证过程与结果

| 检查 | 操作与实际结果 | 结论与证据 |
| --- | --- | --- |
| 地址池及原环境 | 对照Windows/WSL及Docker当前路由；新建不指定subnet的任务网络，实际自动分配10.246.0.0/24，随后回收；原容器/网络/卷保持 | 通过；[路由](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/017-docker-runtime-lifecycle/pool-route-check.json)、[分配](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/017-docker-runtime-lifecycle/network-allocation.json)、[收尾](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/017-docker-runtime-lifecycle/cleanup-final.json) |
| Python及文档检查 | 最终make docs通过73项Python测试及登记、链接、来源检查；含临时属性、持久库拒绝转换/删除、元数据归属、部分分配、回收重试、孤立网络和失败保留原退出码 | 通过；[输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/017-docker-runtime-lifecycle/docs.txt) |
| Node运行入口 | make runtime-tests，4项通过，验证当前本地连接及目标归属检查继续有效 | 通过；[输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/017-docker-runtime-lifecycle/node-runtime.txt) |
| 停机与再次启动 | 任务独占可丢弃真库夹具执行up --no-web；写入探针行后down，原容器和卷保留；再次up后容器/挂载身份及探针行一致；最后destroy回收 | 通过；[结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/017-docker-runtime-lifecycle/lifecycle-real.json)、[原输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/017-docker-runtime-lifecycle/lifecycle-real.txt)；实测的是同一启停路径，持久分类的删除拒绝另由单元测试覆盖 |
| 部分创建回收 | 任务独占临时profile先声明属性，再Compose up --no-start，真实分配Created容器、网络及卷；尚无runtime.json，destroy仍完整回收 | 通过；[结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/017-docker-runtime-lifecycle/lifecycle-real.json) |
| 正常菜单浏览器 | make smoke SMOKE_ARGS='tests/e2e/bu.spec.ts --grep 设置菜单创建未来BU'；管理员由设置菜单创建未来BU，检查字段错误、当日事实、确切来源/历史与返回上下文；1项通过，finally回收 | 通过；[输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/017-docker-runtime-lifecycle/browser.txt)，桌面Chromium，仅此定向用例 |
| 测试失败回收 | 使用明确不匹配任何用例的过滤条件，Playwright实际返回1，make返回2；原错误保留，finally回收资源及仓外凭据 | 回收与退出码验证通过，业务用例未运行；[原失败输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/017-docker-runtime-lifecycle/browser-expected-failure.txt) |
| SIGTERM取消回收 | 独占smoke在浏览器进入Running 1 test后收到SIGTERM；退出130，浏览器及配套环境回收，数据库/网络/卷/凭据均不再存在 | 回收验证通过，业务用例中断不计通过；[结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/017-docker-runtime-lifecycle/browser-cancel.json)、[输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/017-docker-runtime-lifecycle/browser-cancel.txt) |
| profile列表 | 显示30个既有环境的工作树、名称或摘要、用途、状态、端口、网段和temporary；元数据缺失的历史环境按持久保留，不打开runtime.json | 通过；[最终列表](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/017-docker-runtime-lifecycle/profiles-final.json) |
| 差异和文档语义 | git diff --check通过；主代理自查003/017、AGENTS、CI和脚本的持久/临时语义及结果边界 | 通过；非独立评审或人工业务验收 |

## 4 失败、修复与复验

本轮未出现非预期的业务或环境验证失败。上述“无匹配测试”和SIGTERM为故意触发的失败/中断，原输出保留；只证明对应回收路径，不计业务通过。

实施自查补充两点：无临时标记但仍有网络的profile不能误报“已清理”；删除临时构建先于删除元数据，失败时保留重试依据。旧环境无ports.json时，列表从实际数据库端口绑定补足，其余端口保持未知。相应新增/调整测试后最终make docs通过73项。

## 5 未覆盖范围与限制

不重跑全M02、全量DB集成、容量、Firefox、真实消费者或生产恢复；CI仅修改接线，不代表GitHub远程检查已运行。没有修改Testcontainers夹具，不把本次Compose验证扩大为其所有中断场景已通过。SIGKILL和主机故障不在自动退出钩子的保证范围，按003/017在下次任务开始或交接时盘点残留。临时profile的空状态目录/锁及非敏感日志可留存，不持有Docker网络、运行进程或数据库凭据。

## 6 结论与后续事项

地址池已实际应用；make smoke默认新建独占临时环境，成功/失败/可处理取消自动回收；list、显式临时声明和幂等destroy已接线。持久环境继续stop保留，不能自动转为可丢弃。相关操作已写入003/017、AGENTS及CI说明。

本任务新建的两个真库生命周期夹具、三个浏览器profile及一个分配验证网络均已回收；所有原容器、网络、卷仍存在，原容器状态无变化，4382和4527入口HTTP200，见[最终核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/017-docker-runtime-lifecycle/cleanup-final.json)。未重新启动或迁移旧库，也未做历史存量清理、发布、推送或合并。

基线运行数据库包括a的m02-single-lifecycle-a、m02-t07-business，b的m02-current-b及CubeBox的odoo20；继续由各自原任务维护者负责当前验证/参考用途，本任务不接管其生命周期，下次所属任务交接或新建环境时复查。入口HTTP200只证明可达，不能代替对应环境的业务验收或完整归属verify。

## 7 证据保存

本机非敏感原始输出位于`.local/docker-runtime-20261103/`；本报告所需原件已保存至关联目录，入口为[输入清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/017-docker-runtime-lifecycle/input-sha256.json)。输入归档仅包含实现、测试、配置样例及依赖锁，不包含仓外状态、真实凭据、Cookie或数据库文件。旧定稿报告及其证据不改写。
