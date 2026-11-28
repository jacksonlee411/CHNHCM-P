# 公共仓数据库迁移登记

六个开发分支共用这一本账。维护 goose 迁移时先领号，再以返回值创建 `db/migrations/<五位编号>_<名称>.sql`。已有 Git 写权限的同事可直接领取。单独克隆登记分支，配置自己的 Git 提交身份。

```sh
git clone --single-branch --branch migration-registry https://github.com/jacksonlee411/CHNHCM-P.git CHNHCM-P-migration-registry
cd CHNHCM-P-migration-registry
python3 registry.py check
```

以下给出 CHNHCM-027 T02 的实际领号命令。相同参数重复执行会返回同一条登记。办理其他任务时替换负责分支、SQL 名称、用途和依赖。依赖使用五位编号，允许引用已登记的在途迁移，本次指定已合入的公共升级基线。文件后缀在全仓唯一。

```sh
python3 registry.py reserve --branch wt-dev-a --name access_grant_versions \
  --purpose 'CHNHCM-027 T02 逐份授予、固定策略版本与存量授权迁移' --depends-on 00057
```

命令每次读取远端登记和 main，核对已用 SQL，再在临时工作树中提交一条记录，使用普通 Git push。以远端接受为领号成功。两人从同一父提交领取时，后收到的提交会被拒绝。工具刷新账本，遇到同一任务返回已有号，遇到其他任务重新计算编号后提交。如果提示工具已更新，在本克隆执行 `git pull --ff-only` 再重试。断线后也可原样重试。需要查看最新账本文件时使用 `git pull --ff-only`。

SQL 合入 main 后执行以下命令，将示例参数替换为实际负责分支和编号。工具会核对远端 main 中同号同名的 SQL，并记录主线提交与 SQL 的 Git 对象 ID。

```sh
python3 registry.py mark --branch wt-dev-a --number 00058 --status merged
```

取消尚未合入的迁移时使用 `--status cancelled`。如有下游有效登记，先协调其依赖。已取消号永久保留。合并业务变更前复查前置迁移、最新公共库升级和应用启动准入。取消预留留下的空号须与本仓准确迁移集合核对。

初始盘点记录在 `baseline.json`，对应公共 main 的已合入 SQL、文件名及 Git 对象 ID。历史负责人与依赖沿用原仓记录。启用后的领取、合入、取消均向 `events.jsonl` 追加一行，每次一个提交。若 main 出现账外迁移、错名、已合入 SQL 被改动，工具停止领号，由维护者对照实际提交核对账本。

远端规则仅匹配 `refs/heads/migration-registry`，要求线性历史、禁止强推与删除。具有写权限的协作者按上述命令登记。Git 的快进检查保护提交历史，登记字段、状态及依赖由工具校验。其他主机尚未推送的迁移须先在这里领号，再进入公共交付。

## 2026-11-28 编号校准

主线已交付 00058 的逐份授权、00059 的职级薪标、00060 的薪酬方案，以及 00061、00062 的人员地址类型和区间。登记工具首次停在 00059 名称冲突。此次按主线 `ed11f04156e7909bc3f02e865b625d365e7a132d` 核对全部 62 份 SQL 的 Git 对象，已交付的内容和编号沿用主线。[主线迁移目录](https://github.com/jacksonlee411/CHNHCM-P/tree/ed11f04156e7909bc3f02e865b625d365e7a132d/db/migrations)

原基线和两条领取事件按原字节封存在本分支。00058 的原负责分支、用途和依赖已随合入记录保留。00059 曾由 wt-dev-a 领取用于安全管理，4627 实际使用的旧 SQL 对象为 `d037b33d034e77e6516069b1113fa93666096e88`。安全管理需重新领号，产品合并时须验证此前按原登记安装的库可保留授权、回执与旧账本升级。[原基线](../history/2026-11-28-00059/baseline.json) · [原始领取记录](../history/2026-11-28-00059/events.jsonl) · [原登记提交](https://github.com/jacksonlee411/CHNHCM-P/commit/bbe39c7ea7995f844b7af6f213af3d4c7eaa7e0a)

00063 为 C 已提交的按任期定薪迁移，状态为在途；核对 PR 头 `fb0088e3524862bea0c40af6384e51749d2b75de` 的 SQL 对象 `09e2106136282794c29f9de76c896dfcbdb0a53e`。合入后由责任方按实际主线登记。之后新增迁移继续运行本工具普通领号。[C 的 PR #43](https://github.com/jacksonlee411/CHNHCM-P/pull/43)
