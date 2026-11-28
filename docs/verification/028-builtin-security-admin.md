# CHNHCM-028 内置 admin 安全管理权限验证

## 1 范围与首败

2026-11-28 在 `wt-dev-a` 的 `44145442c415b107dd21a3aad33d802ae44196e8` 上验证本轮工作树变更。更新前，从 4627 以 `admin` 登录成功，Go 返回的安全管理权限为空，账号、角色和授权接口均为 403。同一地址的 `x03-security` 持有全部六项权限。该库已有业务授予、逐份撤销和客户指定管理员。[028 业务规则](../specs/x03/028-local-account-and-authorization.md#x03-br-001) · [原固定库样例](027-t05-opening-acceptance.md) · [逐项实测](data/028-builtin-security-admin/results.json)

Go 复用部署初始化保存的主体声明，在本实例识别内置 `admin`。账号、凭据、角色、策略、业务授予、管理职责委派均使用原管理命令和审计。原 00059 数据结构继续承载客户指定资格及回执。旧库中由日常开户建立的同名账号沿用自己的授权。[身份识别](../../internal/contexts/access/infra/postgres/management_admin.go) · [受控交接](../../internal/contexts/access/infra/postgres/management_recovery.go)

## 2 固定端口实页

| 登录账号及入口 | 实际结果 |
| --- | --- |
| `admin` 从「设置 → 安全管理」打开账号、角色、授权 | 三个接口分别为 200。打开自己的授权详情，可见「管理职责 → 内置管理员 → 全部安全管理权限 · 长期」。固定库原本有较多业务授予，初版把这行放在明细末尾，需要滚动。将管理职责移到业务授予之前，1440×540、1440×900 首屏均可见，直接地址登录后也回到同一详情。 |
| `admin` 在账号页开户，在授权页选择「T04 甲部门名录 · V1」并按份撤销 | 新建的 `x03-admin-check-1128` 起初查询部门为 403。授予后从部门菜单只看到 X03DA，撤销后同一查询回到 403。验毕由 `admin` 从账号页停用该账号。四笔提交均可用原提交号查回，数据库记录的办理人均为 `admin`。 |
| 原客户指定管理员 `x03-security`、原只读账号 `reader` | 前者仍具六项管理权，从授权页查到 T05 现场账号的三份已撤销授予。后者三个管理接口均为 403，从「部门编制分配」看到 X03P1，可见小计为 3。`admin` 仍可按原业务授予查到 X03DA、X03DB。 |

[固定端口登录](http://127.0.0.1:4627/) · [admin 的管理职责](http://127.0.0.1:4627/settings/security/grants/1d003ed4-e19d-44e6-838c-b22cb0a30a05) · [短屏原图](data/028-builtin-security-admin/admin-540.png) · [HTTP、回执及授权明细](data/028-builtin-security-admin/results.json)

## 3 旧库、交接与重启

隔离实库从 00058 升至 00059，重复升级、重复初始化后，管理回执校验密钥摘要一致，初始业务授予数仍为零。另一实例按旧管理页面的开户声明建立同名主体，升级未给它建立内置管理资格。新实例在首次指定客户管理员前，内置账号已经可以开户、重置、发布角色与两项策略、授予业务权限及委派管理职责。客户指定和后续交接保留内置资格。也实测将客户指定资格交给 `admin` 再交出，原会话为 401。新任管理员为 `admin` 单独授予并撤掉一份业务权后，重新登录的 `admin` 仍有六项管理权。单独恢复内置账号后，客户指定资格仍归原现任。[真实 PostgreSQL 用例](../../internal/bootstrap/access_builtin_admin_integration_test.go)

固定环境使用工作树默认的 `x03-manual-a`，Web 4627、API 8627、数据库 54627。先在仓外保存原构建及可列出目录的数据库备份，随后实际执行以下命令。启动输出确认运行了迁移检查、运行角色检查、两个原账号的初始化及目录载入。

```sh
export CHNHCM_LOCAL_PROFILE=x03-manual-a
python3 scripts/local-bu.py down
python3 scripts/local-bu.py up --fixed-web
python3 scripts/local-bu.py verify --origin http://127.0.0.1:4627
```

PostgreSQL 于北京时间 14:49:13 从原容器、原卷重新启动。首次停启前后核对 18 张表、1,308 行。其中 17 张表逐表摘要相同，覆盖主体、凭据、指定管理员及回执校验配置、原授予及修订、管理回执、部门和核编历史。剩余一张为会话及授权检查使用的 `access.policy_guard`，两个目录任务分别以 `admin` 登录并退出，其序号由 34 增至 36，其他四个原主体序号相同。完成本次四笔现场办理后，再用 `up --fixed-web` 更新首屏排序；更新前后对账 18 张表、1,322 行，原四笔回执均可在最终页面构建下查回。[构建摘要、逐表对账与停启步骤](data/028-builtin-security-admin/results.json)

## 4 仓内检查与交付范围

10 个不同的 PostgreSQL / Go API 用例均通过，覆盖内置身份、重复初始化、首次指定及管理交接、日常管理边界、代理到期、原权限撤销、业务会话、部门并发撤权及岗位历史。最终固定前端的 Chromium 12 项均通过，其中安全管理 7 项、企业前置 5 项，包含短屏、普通桌面、原生 200% 及委派失效。最初一轮 Playwright 输出 8 项通过，外层命令返回 143。随后两轮管理分片各 12 项通过，均以 0 退出，并回收各自临时实例。

`make check build` 通过。OpenAPI 独立重新生成的 TypeScript 定义与仓内文件逐字节相同。固定入口实际返回的 JavaScript 与构建摘要一致。构建完成后扩充了交接回归用例，补测通过。[逐项结果](data/028-builtin-security-admin/results.json) · [浏览器用例](../../tests/e2e/access-management.spec.ts)

固定 4627 保留运行供本机验收，后续更新由 `wt-dev-a` 复查。隔离浏览器实例按临时 profile 收尾。范围限本地 PostgreSQL、Go API 与桌面 Chromium，全仓集成、远程 CI 和客户生产环境未在本次验证。
