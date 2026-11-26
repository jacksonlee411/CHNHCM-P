# CHNHCM-500 人员登记与主要证件修订评审

## 1 范围与基线

2026-11-22，依据用户采纳的登记简化方案，复核 [500 路线图](../archive/plans/500-personnel-roadmap.md#registration-revision-steps)、[501 业务验收](../specs/m01/501-person-registration-requirements.md#m01-fields)、[502 正常菜单](../specs/m01/502-person-directory-interactions.md#m01-page-register)及 [503 事务与权限](../specs/m01/503-person-registration-design.md#m01-recovery)。基线为固定工作树 `wt-dev-f` 的 `f7d84f192ff518555a60dfe939a78455d8a48597` 加工作区 T00/T01 变更。修订前的三份规格逐字节对应[原评审包](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/500-personnel-t00/reviewed-specs.zip)的第四稿。[原 T01 产品实测](500-t01-personnel-registration.md)对应材料必填的旧构建。

Oracle 将疑似重复人员提示与证件号码唯一性校验分别配置，Moka 的再入职记录说明按重复信息直接认人会造成误合并。本次采用用户确定的实例内号码排他范围；即使证件类型或签发地不同，同号仍阻止第二人占用。[Oracle 证件校验](https://docs.oracle.com/en/cloud/saas/human-resources/fawhr/how-national-identifiers-are-validated.html) · [Moka 再入职修订](https://mokapeople.moyincloud.com/d/1660565124237328385.html)

## 2 发现与修订

| 走查位置及问题 | 修订与复核 |
| --- | --- |
| [登记入口](../../internal/contexts/personnel/infra/postgres/store.go)把材料字段授权和可查候选范围计入可登记条件，[编辑器](../../apps/web/src/features/personnel/Editor.tsx)和[写命令](../../internal/contexts/personnel/infra/postgres/write.go)还校验确认令牌。仅撤掉界面必填仍办不成姓名登记。 | 分别沿入口、候选查询和直接保存走查。501、502、503 均允许只获姓名登记权者提交。姓名查询失败或出现新候选只影响提示，持有证号时仍由数据库约束排他。 |
| 从“王伟已换证”倒推旧单据重送，如果先检查当前证号，或者按新字段默认值重算原摘要，会把已办单据当成新冲突。 | 503 明定先识别原提交，再识别真实来源，最后才是新写入的非空证号。升级须保留原始摘要及绑定，旧材料不回填证件。升级用含实际 00026 结构和旧档的备份副本验证。 |
| 用仅有旧材料权的账号推演查看新证件，或让完整证号随默认详情先返回浏览器再遮罩，两种方式都会扩大读取范围。 | X03 单列证件字段授权。默认读、显式取原值、编辑取值及准确来源均有对应规则；冲突时未获权者只见不可办理提示，回执按当时权限重验。 |
| 按原 AC-009 只能验证材料保全，无法判断旧证处置是否误释放新证，或一般资料更正是否导致证件续期。 | 501 补明证件期限与现用占用的关系；503 按当前归属更新占用，单独处理历史。离职、普通读取截止均不等于换证或完成处置。 |

## 3 场景复核

下表为主代理依据四份文档逐项推演的验收预期。姓名与证件代号均为合成样本。实际产品测试在实施后执行。

| 输入或操作 | 应观察到的结果 | 对应验收 |
| --- | --- | --- |
| 客户甲先取消，再连续两次只填同一姓名，使用不同提交号，无来源引用 | 人数依次为 0、1、2。均无证号占用。同名提示无确认步骤，受限读者仅见获权集合 | [AC-001](../specs/m01/501-person-registration-requirements.md#m01-ac-001)、[页面流程](../specs/m01/502-person-directory-interactions.md#m01-page-register) |
| 两个同名者填不同证号、相同电话和邮箱；第三人用另一姓名和类型争用已有号。另一客户单独使用同号 | 客户甲可存前两人，第三人被拒；客户乙可存。两经办人在空库并发争同号时新增总数为一，败方无编号、来源或事件 | [AC-002](../specs/m01/501-person-registration-requirements.md#m01-ac-002)、[事务次序](../specs/m01/503-person-registration-design.md#m01-recovery) |
| 一人补证后以原号改姓名，换证时先填他人号，再填空闲号；以后处置其旧证来源 | 首次换证整笔回滚，第二次在原档成功。人员编号不变，旧来源按当下权限重现原值；处置旧来源不影响新证占用 | [AC-004](../specs/m01/501-person-registration-requirements.md#m01-ac-004)、[AC-009](../specs/m01/501-person-registration-requirements.md#m01-ac-009) |
| 经办人有录证权但无占用者读权，或先有读权、取回拒绝回执前撤权 | 写入均受实例内约束；无读权时的反馈不带出占用者身份、链接或尾数。只有旧材料权者即使持准确来源，也不能取得新证字段 | [AC-003](../specs/m01/501-person-registration-requirements.md#m01-ac-003)、[授权与访问集合](../specs/m01/503-person-registration-design.md#m01-registration-access) |
| 原提交丢回包后重试，同一真实单据换提交号重送，再换为全新单据以同证新增 | 前两次分别按原提交和单据查回同一办理，第三次因现用证号冲突被拒。无共同单据且都无证号时，系统不能仅凭姓名识别为同人 | [BR-006](../specs/m01/501-person-registration-requirements.md#m01-br-006)、[AC-007](../specs/m01/501-person-registration-requirements.md#m01-ac-007) |

## 4 结论与交接

**主代理方案复核通过，可进入实施。** 第 2 节发现的问题已在业务、页面、写入合同和验收条件中修正。四份审定稿的 SHA-256 如下。

| 文档 | 审定稿 SHA-256 |
| --- | --- |
| 500 路线图 | `c5906f35f2a4b32b99ac736ebb9e605e0fcead815cf68d97bbd4fdd392d8cc2e` |
| 501 业务 | `f3d3c3fd51bd2b7193d61b9e2938c99f555e05c6ceeb6679d5366caeb9720c5f` |
| 502 页面 | `79cbc42945325d99253b2eda226f4e701f71a87ed213a2153253c3b768d741ae` |
| 503 技术 | `32911d01c0de594108845186b55608b2b65d1e3921333530020639fa7214f4b2` |

首次 `make docs` 运行的 137 项脚本用例通过，仓库检查指出新评审记录缺少结果项。补登实际结果后复验通过。单独核对四份方案、评审记录和导航新增的 173 处本地链接（含 78 个锚点），原路线图的 8 个场景及 501 的 6 个规则、10 个验收标识均保留。用临时 Git 索引把未跟踪文档纳入本次 9 份文件的差异检查，`git diff --check` 通过。

本轮交付文档。已有产品实测仍对应原 T01 构建；新版的 Go、数据库竞争与恢复、菜单操作、容量和人事签收由[500 第 6.1 节](../archive/plans/500-personnel-roadmap.md#registration-revision-steps)接续。
