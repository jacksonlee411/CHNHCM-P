# CHNHCM-317 个人资料地址类型主线容量复验

## 1 候选与口径

2026-11-28 在公共 P 仓将个人资料试点提交 `92af73b` 与主线 `fddabd8` 合为 `f7c1bed`。其固定前端在两个桌面视口与原生 200% 通过地址类型维护及员工整页保存，原地理目录的错误归焦与跨栏保存亦通过。本报告记录同一合并候选容量首败及修正后的结果。默认分支的实际集成状态和远程检查可由 PR 核对。[原试点](317-person-address-types.md) · [合并记录](https://github.com/jacksonlee411/CHNHCM-P/pull/41)

沿用 317 的万名人员、二十份九栏资料、连接池上限 12、每条链路预热后二百次的口径。本机 PostgreSQL 18.6，正式容量测量与本任务的构建和浏览器测试分时运行。单会话详情限 300 ms，其余单会话限 500 ms，20 并发临时限 1500 ms。[试点实施方案](../archive/plans/317-person-address-types.md) · [容量用例](../../tests/integration/personnel_nine_sections_capacity_test.go)

## 2 首败与处理

合并候选在 20 并发下保留原地址类型的整页办理 P95 为 1663.37 ms，改选为 1556.26 ms，两条链路均超限。用相同候选单跑 20 会话并采集 Go CPU profile，整页办理再次超限，观察到 PostgreSQL 读写通信耗时，写锁等待峰值为 11。首个远程运行在本地发现超限后取消。[首个远程运行](https://github.com/jacksonlee411/CHNHCM-P/actions/runs/46103378947)

修正提交 `eb963c585b5026dac0103750035f07ada47a1581` 在同一事务内按原顺序批量读取人员用途政策、专项用途及数据库判定时刻。数据库返回锁等待后的时刻，Go 再按该时刻筛选授予。个人资料先读确切修订，再将该修订的证件、适用的当前资料明细和联系方式一起发给数据库，按原顺序读取各查询的结果。办理当前九栏资料时，两处各减少两次数据库往返，历史来源、原采用及事务内锁顺序继续按原合同处理。[人员授权](../../internal/contexts/access/infra/postgres/personnel.go) · [人员存储](../../internal/contexts/personnel/infra/postgres/store.go)

## 3 同机复验

| HTTP 链路 | 首败候选 20 并发 P95 | 修正后专项 20 并发 P95 | 修正后完整复验 1 会话 P95 | 修正后完整复验 20 并发 P95 |
| --- | ---: | ---: | ---: | ---: |
| 九栏详情 | 715.76 ms | 353.28 ms | 27.28 ms | 310.87 ms |
| 保留原类型，整页更正、查回、重读 | 1663.37 ms | 1498.36 ms | 117.04 ms | 1383.20 ms |
| 当前住址行可选类型 | 190.92 ms | 223.11 ms | 24.80 ms | 153.25 ms |
| 改选类型，整页更正、查回、重读 | 1556.26 ms | 1043.46 ms | 111.75 ms | 1321.42 ms |

修正后的专项与完整复验分别通过。完整复验共 1600 次采样、3200 次 HTTP 请求，业务失败和超时均为零。两条写链路的锁等待峰值仍为 11。结论覆盖本机合成数据和约定并发；专项中保留原类型的 P95 距临时上限约 1.64 ms。首败、带采样的诊断及两次复验逐组留存在同一份 CSV 中。[各轮原始数值](data/317-person-address-types/capacity.csv)

```sh
CHNHCM_PERSONNEL_CAPACITY=1 go test -tags=integration -count=1 -timeout=20m -run '^TestPersonnelNineSectionsCapacity$' -v ./tests/integration
```
