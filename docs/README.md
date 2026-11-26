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
