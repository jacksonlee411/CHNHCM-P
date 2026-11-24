# CHNHCM-309 G1职级实施与验收

登记信息：[registry.json](../governance/registry.json)。范围为[309 T01／T02](../plans/309-m03-grade-ladder.md)及[310](../specs/m03/310-grade-ladder-contract.md)的Grade适用部分。G1实施、分层验收及五轮主代理复评已通过，T01／T02关闭。阶梯、Job可用职级及引用关系属于G2，尚未实施。

## 1 基线和运行保护

工作树`/home/jason/projects/CHNHCM-wt-dev-d`、分支`wt-dev-d`、起始HEAD `f87b7b9f7ed53336af9e85065ec0134362462567`，工作区干净、stash为0；main `a9e5daca`已包含于该HEAD。源码摘要、迁移清单及环境见[基线](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g1/baseline.json)。本轮新增00018，不改正式交付00017。

独立profile `m03-grade-d`，Web4381／API8381／DB54371。用途为本批开发及用户验收，责任wt-dev-d，用户验收结束或下次更新复查。4380及其他持久环境保持原程序和数据。随机数据库凭据由运行工具保存于仓外；本地admin／reader采用公开开发密码`admin1234567`。

## 2 实施落点

GradeData／GradePatch在Staffing领域约束文字字段，映射到本域已有标量时间线；新建默认日、位置维护、日期及分类更正、启停与固定期间仍由Go计算。独立Grade表组封存来源、修订、终态和Outbox，按日投影包含半开daterange。企业下界端口在同一物理事务与共同守卫下执行。

公开资源`/api/v1/grades`、grade授权空间及生成的Grade DTO接入标准Staffing页面，正常菜单为“设置 → 职务配置 → 职级”。职级只有编码升序、名称／英文名称升降序；状态和日期仅筛选。日志直接维护与筛选后退仍复用308修复，新增Grade实测。

## 3 评审与验证流水

首轮接线复核发现原代码中的二选一分支不能直接承接Grade：已明确Grade类型、原因、字段、独立锁／表组及CSV，Family反向引用检查仅适用Family。Grade排序单独收窄，未知系列筛选参数拒绝。

[首轮真实DB/API](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g1/integration-first.txt)六个根测试通过：企业同物理事务、企业双向锁序、身份／字段／来源／恢复、固定期间与未来记录、字段权限／CSV／排序、完整按日维护。真实PostgreSQL重启后的原来源和回执保持；这不代替浏览器或容量验证。

五轮主代理评审覆盖接线与边界、API与权限、事务及升级、实际页面、验收反例及交付证据。G1无遗留阻断项，分层结果见第6节；这不是独立代理或人工业务验收。


## 4 分轮发现与处置

| 轮次／问题 | 发现与实际修改 | 状态 |
| --- | --- | --- |
| R1 类型接入 | Family／Job二选一分支须显式加入Grade；GradeData／Patch、原因、锁、存储、读投影、权限、路由和页面逐项接线 | 首轮真实DB/API通过 |
| R2 导出排序 | 原公共Export单独使用`in.Sort`，仅检查捕获查询的排序；现与列表执行同一排序／字段权校验，Grade编码降序与状态排序导出均拒绝 | 新反例及第二轮DB/API通过 |
| R2 Grade DTO | Grade不继承Job字段或Family独有采用；GradeSelectionInput保留initialIds及固定选择修订协议，前端提交显式走Grade生成类型 | 类型检查通过 |
| R3 升级与故障 | 从真实00001—00017布局升级00018，对全部原Job／Family头、来源、修订、采用、回执、Outbox及投影逐表比较；另验两锁共同预算、取消释放、迟到提交查回及写后回滚 | 有效领域资料、原采用及准确操作字段夹具复验通过 |
| 首轮浏览器 | 开发服务器中筛选后退重查等待超时、一次初始页面未加载；宽屏日志维护、41项选择CSV及V01完整旅程通过 | 原失败保留；同候选固定构建两项复验通过，未据此断言根因 |
| 原生缩放前置 | 本机缺Xvfb／xdpyinfo／xwininfo，两项未进入验收；仓外解包工具，以临时挂载提供xkbcomp，不改系统路径 | 独立X11检查及双角色原生缩放通过；仓外临时工具收尾回收 |
| R4 窄屏日期定位 | 连续维护旅程发现退出动画期间旧菜单项与日期控件均带aria-current，通配定位发生strict mode冲突；检查限定到实际日期导航，仍要求选择目标日期且完成读取 | 14单元复验全部通过，含双角色原生200%／400% |
| R4 完整集成时限 | make integration两包各超过20分钟总限额，超时位于原Department场景；无完整通过结论。按Staffing、企业及改动的迁移／授权实际消费者分别执行，超时遗留自有容器已回收 | 分组结果另记，不等同全仓通过 |
| R4 证据落盘 | list reporter不持久化body附件；矩阵复验启用JSON reporter保存实际视口、Grant、参考页测量及滚轮路径，截图仍为实际原生画面 | JSON报告及14单元实测已落盘 |
| R4 生成类型复核 | 恢复Job／Family原结果枚举后，联合路由的unwrap泛型推断过窄；明确以公共StaffingResult解包，保留运行时类型／提交号校验 | 首次最终check类型失败保留，定向类型复验通过 |
| R5 日期边界夹具 | 追加北京时间零点场景时，首版夹具误用了1月边界且比较更正前来源；改为独立预期2035-02-01零点前一秒／当天，并比较已更正来源 | 首失败保留，修正夹具后复验通过，产品代码未变 |
| R5 登记状态 | 收尾将验证报告状态误填为accepted，结构检查拒绝；按014报告枚举改为final，方案execution仍为in_progress | 原失败保留，make docs复验通过 |
| R5 原始日志空白 | 暂存检查发现工具日志自带行尾空白；原字节另存压缩件，阅读副本规范行尾，产品与断言不变 | [规范记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g1/log-normalization.json)；最终暂存检查通过 |

## 5 验收对应关系

以下映射按310独立预期归类，最终执行结果见本记录的分层汇总；G2集合／关系部分不在G1出口内。

| 310验收 | G1实施／验证入口 | G2保留范围 |
| --- | --- | --- |
| AC031／032 | Grade领域正反例、企业下界／跨实例、全部维护、独立后继及固定两条；正常菜单、取消与未来首条 | Ladder及集合语义 |
| AC038／039 | 实际Grant、日志固定上界／原时刻、删除前原件、撤权／读失败；Grade无关系来源候选 | 成员采用、关系源读权及候选迟到 |
| AC040／041 | Grade与企业同事务及双向锁序、累计预算和取消、写前后回滚、未知提交、原号／跨类型、响应丢失／换账号／真实重启 | Grade资格与Ladder／Job关系竞争 |
| AC042—044／046 | Grade14单元的BU标准页比较、日志直接维护／取消返回、筛选后退、全表单及原生缩放；原Family／Job回归 | Ladder／Job新增集合字段 |
| AC045 | Grade六列CSV逐值、41项跨页选择、固定上界排除／取消及撤权 | Ladder七列及JSON成员顺序 |
| AC047／048 | Go边界／生成、真实1—17升级、原事实保护、投影重建、容量与运行收尾 | G2关系格式及容量 |

AC033—037均为G2，保持未实施／未运行。14单元不以两名账号代替精确Grant专项，容量不以浏览器成功代替。

## 6 分层结果与运行交接

| 层次 | 实际结果 | 证据 |
| --- | --- | --- |
| 静态／单元／边界／类型 | 最终make check通过；128项工具测试、文档／边界、Go单元／vet、TS及diff检查。此前一次最终类型失败及修复保留；收尾make docs再次通过 | [最终check](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g1/check-closure.txt)／[首次类型失败](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g1/check-type-first-failure.txt) |
| 生成与构建 | make generate输出一致；最新前端5个构建文件与已验收固定构建逐字节一致；原完整make build通过 | [生成](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g1/generated-check.json)／[构建一致](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g1/web-equivalence.json) |
| DB/API主批 | 39个根测试通过，覆盖307／308原消费者、309 Grade及改动的迁移准入；3个显式条件测试跳过，不能计通过 | [主批](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g1/integration-scoped-final.txt) |
| 企业／提交复核 | 11个企业根测试及1个Grade未知COMMIT确认丢失测试通过 | [企业及提交](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g1/integration-enterprise-final.txt) |
| 补充反例 | Grade固定选择跨日／导出撤权及北京时间零点边界定向复验通过；同名根测试为复验，不重复累计独立根数 | [选择](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g1/integration-selection-final.txt)／[日期边界](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g1/integration-clock-retest.txt) |
| 浏览器功能 | 主批18项通过，2项缺X11前置、3项窄屏定位冲突失败；后续矩阵12个测试涵盖14单元，全部通过 | [首次合批](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g1/browser-combined-first.txt)／[矩阵复验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g1/browser-matrix-final.txt) |
| 原消费者及异常 | Family／Job筛选后退、双尺寸日志直接维护及41项选择共8项，加Grade撤权／503／换账号恢复／迟到详情4项，全12项通过 | [回归及异常](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g1/browser-regression-final.txt) |
| 容量／重启 | 1200正式HTTP、252次预热分开记录；12组均达标、零语义错误，1643原回执及准确来源重启保持 | [容量及重启](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g1/capacity-first.txt) |

DB/API主批与企业批合计51个不同根测试通过，另1个独立Grade容量根测试通过；额外反例及升级夹具复验不重复计数。主批跳过的是旧307容量、旧308固定时钟浏览器及单独运行的309容量；Grade新矩阵和时钟边界有本批独立实测，不以旧跳过项填数。完整make integration在20分钟限额超时，保留失败及自有容器回收记录；上述结果是分组验证，不是全仓完整通过。尚未运行远程CI或人工业务验收。

[最终候选及源码摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g1/candidate.json)、[固定构建](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g1/fixed-build.json)、[14单元实际视口](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g1/viewport-summary.json)均归档。主批首失败及历次复验保留，全部文件摘要见[证据清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g1/evidence-index.json)。

独立验收入口为[4381](http://127.0.0.1:4381)，正常菜单“设置→职务配置→职级”，admin／reader密码均为公开开发密码`admin1234567`。搜索`G1-`可找到初级职级、带2035年未来更名的专业职级、带2024停用及2025恢复历史的演示职级，均经公开API创建，具体对象链接见[验收样本](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g1/handoff-samples.json)。自动验收数据保留，未清空历史。

[4380](http://127.0.0.1:4380)只做无登录只读复核：API／前端摘要、容器身份与启动时刻及1—17账本仍匹配308保留记录；本批没有迁移、重启或业务写入该环境。[只读保护结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g1/protected4380-check.json)已归档。4381责任为wt-dev-d，用户验收结束或下次更新复查。临时DB属于测试自身，退出即回收；完整集成超时遗留的一个自有容器已精确回收。

## 7 容量结果与评审出口

每组100个正式完整HTTP样本，预热每会话2次单列；读组为admin／reader，写组为获权admin且20会话各写独立对象。下表单位ms，全部12组语义错误为0。原始每次计时及预热见[样本](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g1/capacity-first.json)。

| 路径／角色 | 会话 | P50 | P95 | P99 | P95目标 |
| --- | --- | --- | --- | --- | --- |
| grade/list/admin | 1 | 43.60 | 55.33 | 57.69 | 300 |
| grade/detail/admin | 1 | 43.41 | 55.27 | 57.37 | 300 |
| grade/list/reader | 1 | 42.17 | 56.04 | 60.02 | 300 |
| grade/detail/reader | 1 | 41.58 | 58.83 | 69.14 | 300 |
| grade/ordinary-save | 1 | 54.86 | 80.40 | 84.50 | 500 |
| grade/original-resolve | 1 | 26.17 | 35.87 | 39.60 | 500 |
| grade/list/admin | 20 | 189.60 | 364.04 | 374.77 | 500 |
| grade/detail/admin | 20 | 127.36 | 152.39 | 161.56 | 500 |
| grade/list/reader | 20 | 128.70 | 194.01 | 205.98 | 500 |
| grade/detail/reader | 20 | 141.17 | 171.82 | 174.89 | 500 |
| grade/ordinary-save | 20 | 342.39 | 455.02 | 482.07 | 500 |
| grade/original-resolve | 20 | 62.61 | 78.78 | 80.03 | 500 |

Linux/amd64、Go1.26.8、18逻辑CPU，真实PostgreSQL使用fsync／同步提交，普通API连接池保持12。测试API与驱动位于同一Go测试进程，经完整HTTP和真实DB执行；测试进程及子进程峰值RSS约374MiB，Grade表组约16.65MiB，总耗时5分48秒含制数、正式测量及重启核验，不能当作单次产品请求时延。数据库配置与逐表体积见[资源](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g1/capacity-first.json.resources.json)。其他构建和测试在正式测量时均已结束，持久环境保持运行。

重启后逐笔恢复1643个原回执并读取1643个准确来源；全部原身份、来源、修订、回执及Outbox指纹保持，新增来源／回执均为0。全部自有测试库及X11进程／临时工具已回收，保留4381供用户验收，收尾核对见[资源清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g1/resource-closure.json)。

G1的Grade独立出口通过；G2阶梯／Job可用职级及全期间双向引用保护仍未实施，因此309整体保持in_progress。下一步按T03推进G2，同批交付关系写入与Grade反向守卫。工资金额仍归M09；旧125等容量接续不由本次结果关闭。推送、PR、合并及固定4380更新均未执行，发布前仍须最终候选三项正式CI通过；[实际差异选择](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-g1/future-ci-impact.json)要求完整8个integration及11个browser分片，本地分组结果不替代该门禁。
