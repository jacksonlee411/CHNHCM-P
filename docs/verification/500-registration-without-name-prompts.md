# CHNHCM-500 人员登记取消同名提示的方案评审

## 1 本次修订

2026-11-22，按用户确认的规则复核人员登记方案。同一客户的人员档案只按现用主要证件号码判断人员是否重复。姓名仍是登记必填项。输入姓名、离开姓名栏和保存时均不查同名、不提示，也不用经办人确认。经办人仍可在员工名录中主动按姓名找人。[501 业务规则](../specs/m01/501-person-registration-requirements.md#m01-br-002) · [502 登记页面](../specs/m01/502-person-directory-interactions.md#m01-page-register)

工作基线为 `wt-dev-f` 的 `f7d84f192ff518555a60dfe939a78455d8a48597` 及已有 T00/T01 工作区。上次方案还要求在姓名旁列出同名人员，本次已取消。上次评审引用的四份工作区原稿按该报告的 SHA-256 核对并保存。[上次方案评审](500-primary-document-registration-review.md) · [上次审定原稿](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/500-primary-document-registration/reviewed-documents.zip)

## 2 业务与页面复核

| 核对事项 | 方案结论 |
| --- | --- |
| 两个王伟，身份证号码不同 | 分别建档，得到两个人员编号。录入第二人时没有同名提示。独立登记暂缺主要证件号码，同样可以保存。 |
| 两位经办人各自登记同一身份证号码 | 两笔普通登记只能有一笔建档。即使一人写错姓名，也不能重复建档。经办人无权查看原档时，页面只提示无法按此证件办理。 |
| 同一张有单号的登记单原样又办一次 | 单据类型、单号和原内容都相符，有权查看的人可以查回上次办理结果。换一张单再次登记同一证号，仍按证号阻止重复建档。手工登记可以不填单号。 |

逐项对照业务、页面和保存规则。员工名录的姓名搜索仅由经办人主动发起。登记表、保存操作和直接调用登记接口均不按姓名查人。目标设计和技术合同已删除原来要求返回同名人员的约定，实施范围包括删除旧查询接口和保存时的确认字段。人员编号的分配及已经办理的登记单继续按原记录查回。[业务验收样例](../specs/m01/501-person-registration-requirements.md#m01-registration-examples) · [保存与重复办理](../specs/m01/503-person-registration-design.md#m01-recovery) · [M01 模块](../architecture/target/modules/m01.md)

## 3 评审与实施

文档检查 `make docs` 通过，含 137 项检查器用例。模块生成器复核 57 份产物一致。另逐条核对 500–503 的原有业务标识和 48 处相关锚点引用，并将未跟踪的规格及评审文件纳入差异检查，`git diff --check` 通过。

本次方案复核通过。现有登记页面和服务端仍执行原 T01 规则。新版的系统改动、同时登记同一证号的数据库验证，以及人事从菜单试办，均待实施。[原构建的实测记录](500-t01-personnel-registration.md) · [人事实施步骤](../plans/500-personnel-roadmap.md#registration-revision-steps)
