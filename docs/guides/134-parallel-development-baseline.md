# CHNHCM-134 并行开发共同基线与独立环境

登记信息：[registry.json](../governance/registry.json)。
适用范围：本仓固定分支、本机及其他Linux/WSL主机的独立开发环境、私有组织来源的保管边界。
上游依据：[005协作规则](../governance/005-branches-and-documents.md)、[017运行](017-local-bu-development.md)、[025全新实施](../adr/025-greenfield-implementation-policy.md)、[本次检查](../verification/134-parallel-baseline.md)。

## 1 共同基线与开工边界

客户实库案例的组织明细由私有原仓保存。

客户实库案例的组织明细由私有原仓保存。 [133结果](../verification/133-property-rename.md) · [最新决定](../adr/132-query-and-validation-boundary.md#15-正式采用20记录场景并保留模型5)

Git同步不会启动或升级数据库。当前4390由wt-dev-b维护，供用户观察，固定程序与原54392库的具体状态见017第16节；原4387/4388及历史环境保持。新环境从当前迁移空库初始化，不复制4390的账号、会话、凭据或数据库卷。

## 2 新主机准备代码工作树

需要当前私有仓库的访问权限、Git及SSH认证。SSH密钥由本机管理，不写入仓库。以下只用于尚无该检出的主机；已有目录先检查，不覆盖。

```bash
CHNHCM_WORKSPACE="$HOME/projects"
CHNHCM_BRANCH=wt-dev-c
mkdir -p "$CHNHCM_WORKSPACE"
git clone --branch main ssh://git@ssh.github.com:443/jacksonlee411/CHNHCM.git "$CHNHCM_WORKSPACE/CHNHCM"
cd "$CHNHCM_WORKSPACE/CHNHCM"
git fetch origin
git status --short
git worktree list
git rev-parse origin/main
for branch in wt-dev-a wt-dev-b wt-dev-c wt-dev-d wt-dev-e wt-dev-f; do
  if ! git show-ref --verify --quiet "refs/heads/$branch"; then
    git branch --track "$branch" "origin/$branch" || exit 1
  fi
done
git rev-list --left-right --count "origin/main...$CHNHCM_BRANCH"
```

首次共同基线同步完成后，最后一行应为`0 0`。若分支已有新提交，先确认负责者和起点，不移动已有ref来伪造相同基线。确认所选分支未在该仓其他工作树检出、目标目录不存在后：

```bash
git worktree add "$CHNHCM_WORKSPACE/CHNHCM-$CHNHCM_BRANCH" "$CHNHCM_BRANCH"
cd "$CHNHCM_WORKSPACE/CHNHCM-$CHNHCM_BRANCH"
pwd
git branch --show-current
git rev-parse HEAD
git status --short
git stash list
python3 -B scripts/check-repository.py --check-refs
```

主目录始终固定main，开发工作树始终固定同名分支。只建立实际使用的工作树，c—f无需全部创建；本机登记表绝对路径不要求其他主机使用同一用户名。各主机自行记录实际目录、负责者及所用分支，换主机写同一远程分支前交接现场；共享分支不能两台主机同时推进。

已有工作树更新时先`git fetch origin`，核对分支、工作区、stash及`git rev-list --left-right --count HEAD...origin/main`。仅在工作区干净且HEAD是目标祖先时使用`git merge --ff-only origin/main`；有独有提交或未提交工作时保留现场，按任务协调集成。禁止reset、强推、自动stash或切换他人固定工作树。

## 3 安装依赖与建立独立运行环境

### 3.1 工具和检查

工具版本以[go.mod](../../go.mod)、[package.json](../../package.json)、锁文件及[CI](../../.github/workflows/ci.yml)为准；当前为Go1.26.0、Node24.19.0、pnpm10.24.0，另需Python3、Docker及Compose。镜像由[锁文件](../../deploy/local/postgres-image.txt)指定。浏览器安装及原生缩放需要Chromium、xvfb、x11-utils、libxtst6；Ubuntu的软件包安装由主机维护者执行。仓库内依赖分别安装，不复用其他工作树node_modules或软链接。

```bash
go version
node --version
corepack pnpm --version
python3 --version
docker version
docker compose version
make install
corepack pnpm exec playwright install --with-deps chromium
python3 scripts/check-x11.py
make check
```

若Node安装未带Corepack，按CI采用的Corepack0.35.0在本机配置，再使用packageManager锁定的pnpm；不临时升级项目依赖来绕过安装问题。

### 3.2 运行自己的数据库与页面

以下是wt-dev-c的示例值，使用前检查该主机端口和profile是否已有责任者。同主机各工作树使用不同端口；不同主机拥有各自Docker与进程时可使用相同端口。

```bash
export CHNHCM_LOCAL_PROFILE=m02-dev-c-host2
python3 scripts/local-bu.py list
python3 scripts/local-bu.py configure --web-port 4403 --api-port 8403 --db-port 54403 --purpose 'wt-dev-c独立功能开发'
python3 scripts/local-bu.py connection
python3 scripts/local-bu.py up
python3 scripts/local-bu.py verify --origin http://127.0.0.1:4403
```

`up`在本机创建独立数据库、随机数据库密码、迁移和最小开发账号；profile、工作树路径共同决定项目与卷。运行配置只在仓外0700目录、文件0600中，不复制他人的runtime.json。需要固定版本观察时使用`up --fixed-web`并记录Git SHA和构建摘要。正常开发中同模型迁移由upgrader统一处理；现有库模型不匹配时保留原库和匹配程序，另用当前模型空库，不自动重建持久数据。

浏览器使用verify确认的实际页面入口，管理员admin、查询者reader的本地开发预设密码均为admin1234567。从设置→组织配置分别检查业务单元、部门；用本机合成对象验证创建、更正、历史、原提交恢复和查询者读权。新库没有万科数据是正常初态。远程主机页面可通过SSH端口转发到本机浏览器，保持同一页面Origin；公开开发预设不能直接用于客户或公网环境。

### 3.3 检查和收尾

`make check`检查静态、单元、生成物及类型；受影响DB/API执行`make integration`，页面执行`make smoke`。smoke自动创建独占临时环境并回收，不使用日常手工库。普通停止执行`python3 scripts/local-bu.py down`，保留持久容器、卷和历史；重新up后核对同一对象及历史。只有创建时明确为temporary的任务环境才可destroy，不运行全局prune。

其他主机开工记录至少包含Git SHA、负责分支、路径、profile、实际三端口、模型及迁移、双角色菜单和自己的业务验证结果。本文是操作指引，不是这些主机已经部署成功的证明。

## 4 公共变更的协作次序

规则由005第6节持有。每项公共变更用任务或PR写明以下内容后，与直接受影响分支协调；无依赖的开发继续：

| 内容 | 必须明确的事项 |
| --- | --- |
| 负责与基线 | 唯一负责分支、实际起点SHA、受影响分支及消费者 |
| 共同合同 | Go规则、API/类型、授权、提交恢复、共享页面流程是否变化 |
| 存储 | 模型/迁移号、字段及完整性保护；核对其他分支在途迁移编号 |
| 集成顺序 | 公共修改先合入，消费者随后同步；冲突保留双方实际业务语义 |
| 验证与恢复 | 正常操作、错误/撤权、历史、幂等恢复及资源收尾；不得借冲突修复降低保护 |

同一交付同步实现、合同、测试、登记及证据；Go保持唯一业务权威，不为并行分支建设兼容双轨。存储候选先证明成员完整性、来源绑定、普通SQL损坏拒绝和原子发布，再以同输入比较成本；当前存储保留决定不会因新分支启动而改变。

## 5 CI与同步

PR 面向 main。CI 维护者按 137 确定正式门禁、分片及运行时长，按第 12.1 节接续托管分钟试点。工作流在 quality 和矩阵之间执行前置回归，提交稳定候选前可运行 `make ci-preflight`。限定范围独立选择 DB/API 及浏览器，full、selected、docs 正式 PR 的等价 main 结果按 137 复用。各片使用独立临时环境，workers=1、retries=0，分片按实际用例和耗时分配。[分片规则](../governance/137-actions-and-validation.md#7-分片缓存与运行资源) · [托管分钟观察](../governance/137-actions-and-validation.md#hosted-minutes-observation) · [当前工作流](../../.github/workflows/ci.yml) · [首批发布](../verification/136-actions-publication.md) · [限定范围实施](../verification/136-impact-selection.md)

交付前先完成本地受影响检查及生成物核对，再推送稳定候选；在依赖需要或准备交付时同步main，不为保持所有分支同SHA反复更新开放PR。正式PR运行与手动诊断分开，手动成功不能代替合并门禁。最终检查核对head、base、受测引用、实际范围和全部计划结果；失败job补验仅按[137第6节](../governance/137-actions-and-validation.md#6-失败与重跑)执行，明确分批结果及原失败，不表述为单轮全绿。

三项门禁成功后保留提交历史合并。获得固定分支同步授权时，fetch后逐个核对本地/远程祖先关系；main及已建工作树在各自目录快进，未检出ref核验旧值后快进，再非强制推送开发分支。任何独有提交、脏工作区或并发ref变化均保留现场。其他主机需要由其负责者再次fetch并按第2节同步；本机同步不等于远端主机工作区已更新。

正式容量计时须独占主机资源，避免同时构建、smoke、数据导入或其他负载；普通功能CI在独立runner并行。容量失败继续保留首失败与复验，不与功能准入混算。

## 6 组织来源与公开范围

客户组织工作簿、原行、编码映射、回执、数据库、真实环境的截图和导出仅在授权的私有环境保管。公共仓库使用合成输入启动开发环境、运行集成及浏览器测试。有权使用离线原件时，来源目录和重放账本都应位于检出目录之外。

## 7 常见问题与依据

| 现象 | 处理 |
| --- | --- |
| clone或push无权限 | 检查本机SSH身份和私有仓库授权，不把令牌写入配置示例 |
| 分支不能快进 | 保留独有提交、工作区及stash，协调实际合并，不reset或强推 |
| 端口或profile已存在 | 核对责任者；新任务选独立值，不覆盖已有配置 |
| 模型不匹配 | 保留库与匹配程序，用当前模型独立空库；旧环境升级需单独任务 |
| 来源SHA不匹配 | 停止导入，检查Git版本及文件，不重新生成清单来掩盖漂移 |
| prepare提示已有账本 | 使用原任务目录继续恢复，不覆盖已提交请求及回执 |

本机检查与限制见[134记录](../verification/134-parallel-baseline.md)。产品正确性依据原133报告，完整来源导入的历史结果依据127报告；其他主机运行成功、后续全量重放和容量达标均需各自实际证据。
