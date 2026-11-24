# 命令、事务与验收索引

本页由模块对象归属与[data/command-contract-index.json](data/command-contract-index.json)生成，仅提供导航。业务规则以[DDD合同](10-ddd-and-state-contracts.md)及模块说明为准；不复制另一套金额、状态或事务算法。

接口名称表达目标命令/查询族，不表示已有实现；QueryHistory、ConfirmTime、AssessImpact、ResolveImpact、ApplyRecovery是跨具体端口的导航名称。所有AC场景均为后续产品验收要求；来源设计阶段的文档/图纸检查不等于本仓验收。本仓已执行范围见[验证记录入口](../../verification/README.md)，来源增量边界见[来源说明](../../reference/README.md)。

## K01 晋升与调薪分步完成

命令/查询：StartJourney、ConfirmStepByEvidence、ReassessStep、CancelJourney。

| 对象及所属域 | 同源聚合/写端口 |
| --- | --- |
| [M20](modules/m20.md) [JourneyInstance（员工历程实例）](17-terminology-glossary.md#term-m20-journeyinstance) / [StepInstance（历程步骤实例）](17-terminology-glossary.md#term-m20-stepinstance) | JourneyInstance根含StepInstance / StartJourney、ConfirmStepByEvidence、ReassessStep、CancelJourney |
| [M04](modules/m04.md) [Assignment（任职）](17-terminology-glossary.md#term-m04-assignment) / [AssignmentRecord（任职有效期记录）](17-terminology-glossary.md#term-m04-assignmentrecord) | Assignment根含Record及经理关系 / Transfer、EndAssignment |
| [M09](modules/m09.md) [CompensationAward（个人薪酬核定）](17-terminology-glossary.md#term-m09-compensationaward) | Award根 / ConfirmAward |
| [M10](modules/m10.md) [RetroCase（工资追溯事项）](17-terminology-glossary.md#term-m10-retrocase) | RetroCase根 / OpenRetroCase |

提交点、守卫或查询能力：[规范合同](10-ddd-and-state-contracts.md#213-跨域业务完成晋升并调薪)；恢复：[更正/恢复合同](10-ddd-and-state-contracts.md#62-消费者影响案与闭环)；验收：[AC01](10-ddd-and-state-contracts.md#ac01)。

## K02 主选择资格、唯一及覆盖

命令/查询：Hire、SelectPrimary、EndAssignment、ActivatePrimaryPolicy。

| 对象及所属域 | 同源聚合/写端口 |
| --- | --- |
| [M04](modules/m04.md) [PrimarySelection（主要关系与任职选择）](17-terminology-glossary.md#term-m04-primaryselection) / [ManagementHome（主要管理归属）](17-terminology-glossary.md#term-m04-managementhome) | PrimarySelection根 / SelectPrimary |
| [M04](modules/m04.md) [PrimarySelectionPolicy（主要关系与任职选择规则）](17-terminology-glossary.md#term-m04-primaryselectionpolicy) | PrimarySelectionPolicy配置根 / PublishPrimaryPolicy、ActivatePrimaryPolicy |
| [M04](modules/m04.md) [Employment（工作关系记录）](17-terminology-glossary.md#term-m04-employment) | Employment根 / Hire、RegisterAssociation、EndEmployment |

提交点、守卫或查询能力：[规范合同](10-ddd-and-state-contracts.md#210-人员编号与主选择覆盖)；恢复：[更正/恢复合同](10-ddd-and-state-contracts.md#22-核心人事的配置层级与完整生命周期)；验收：[AC02](10-ddd-and-state-contracts.md#ac02)。

## K03 同日动作及跨午夜

命令/查询：CancelAction、CorrectEndDate、ConfirmTime、PublishSchedule。

| 对象及所属域 | 同源聚合/写端口 |
| --- | --- |
| [M04](modules/m04.md) [BusinessActionRecord（人事业务办理记录）](17-terminology-glossary.md#term-m04-businessactionrecord) | 目标根内不可变动作记录及更正链 / ExecuteEmploymentAction、CorrectActionRecord、CancelAction |
| [M06](modules/m06.md) [WorkingTimePolicy（工时制度规则）](17-terminology-glossary.md#term-m06-workingtimepolicy) | WorkingTimePolicy根 / PublishWorkingTimePolicy |
| [M07](modules/m07.md) [TimeRecord（实际工时记录）](17-terminology-glossary.md#term-m07-timerecord) | TimeRecord根 / CorrectTime |

提交点、守卫或查询能力：[规范合同](10-ddd-and-state-contracts.md#211-日级人事与跨午夜时间)；恢复：[更正/恢复合同](10-ddd-and-state-contracts.md#27-worktime的配置时间覆盖与权益)；验收：[AC03](10-ddd-and-state-contracts.md#ac03)。

## K04 机构义务和执行

命令/查询：ConfirmTaxLiability、ConfirmSupplierPayable、PrepareInstitutionSettlement、ClaimInstitutionSettlement、ApplyInstitutionReceipt。

| 对象及所属域 | 同源聚合/写端口 |
| --- | --- |
| [M11](modules/m11.md) [TaxRemittanceAccount（应缴税款台账）](17-terminology-glossary.md#term-m11-taxremittanceaccount) | TaxRemittanceAccount根含义务行及来源head / ConfirmTaxLiability |
| [M11](modules/m11.md) [TaxRemittanceInstruction（税款缴纳指令）](17-terminology-glossary.md#term-m11-taxremittanceinstruction) / [SettlementAuthorization（税款结算授权）](17-terminology-glossary.md#term-m11-settlementauthorization) / [SettlementInvestigation（税款结算核查）](17-terminology-glossary.md#term-m11-settlementinvestigation) | TaxRemittanceInstruction根含分配、核准及核查 / PrepareInstitutionSettlement、ClaimInstitutionSettlement、ApplyInstitutionReceipt、ResolveUnknown |
| [M12](modules/m12.md) [AgencyPayableAccount（应付经办机构缴款台账）](17-terminology-glossary.md#term-m12-agencypayableaccount) | AgencyPayableAccount根含机构执行意图、授权及核查 / ReserveAgencyPayment、Settle、ResolveUnknown |
| [M13](modules/m13.md) [BenefitSupplierPayable（应付福利供应商款）](17-terminology-glossary.md#term-m13-benefitsupplierpayable) | BenefitSupplierPayable根 / ConfirmSupplierPayable |
| [M13](modules/m13.md) [SupplierSettlementInstruction（福利供应商结算指令）](17-terminology-glossary.md#term-m13-suppliersettlementinstruction) / [SupplierMovement（福利供应商资金流水）](17-terminology-glossary.md#term-m13-suppliermovement) | SupplierSettlementInstruction根含授权/核查；SupplierMovement不可变根 / PrepareInstitutionSettlement、ClaimInstitutionSettlement、ApplyInstitutionReceipt、ResolveUnknown |
| [M14](modules/m14.md) [PayerAccount（机构付款账户）](17-terminology-glossary.md#term-m14-payeraccount) | PayerAccount根 / RegisterPayerAccount、PlacePayerHold、ReleasePayerHold、AcquirePayerExecution |

提交点、守卫或查询能力：[规范合同](10-ddd-and-state-contracts.md#32-机构付款责任与外部财务委托)；恢复：[更正/恢复合同](10-ddd-and-state-contracts.md#33-unknown的运营处置)；验收：[AC04](10-ddd-and-state-contracts.md#ac04)。

## K05 薪酬口径与工资期运算

命令/查询：ConfirmAward、DeliverAward、AdoptInput、SealResult。

| 对象及所属域 | 同源聚合/写端口 |
| --- | --- |
| [M09](modules/m09.md) [AwardDelivery（薪酬核定下发记录）](17-terminology-glossary.md#term-m09-awarddelivery) | Award内Delivery实体 / DeliverAward |
| [M10](modules/m10.md) [Element（工资项目）](17-terminology-glossary.md#term-m10-element) / [PayrollInput（计薪输入）](17-terminology-glossary.md#term-m10-payrollinput) / [CalculationRule（工资计算规则）](17-terminology-glossary.md#term-m10-calculationrule) | Element与CalculationRule独立配置根；PayrollInput根含Entry / PublishElement、PublishCalculationRule、AdoptInput、RetirePayrollConfiguration |
| [M10](modules/m10.md) [PayrollRun（工资核算批次）](17-terminology-glossary.md#term-m10-payrollrun) / [Result（工资核算结果）](17-terminology-glossary.md#term-m10-payrollresult) / [Balance（工资累计）](17-terminology-glossary.md#term-m10-payrollbalance) | PayrollRun只持结果引用；PayrollResult与PayrollBalance独立根 / SealResult |

提交点、守卫或查询能力：[规范合同](10-ddd-and-state-contracts.md#212-薪酬口径转换与期间分摊)；恢复：[更正/恢复合同](10-ddd-and-state-contracts.md#24-工资配置影响规则与连续更正)；验收：[AC05](10-ddd-and-state-contracts.md#ac05)。

## K06 历史查询及采用解释

命令/查询：QueryHistory、ExplainResult、ReproduceResult。

| 对象及所属域 | 同源聚合/写端口 |
| --- | --- |
| [X01](modules/x01.md) [Revision（修订版本）](17-terminology-glossary.md#term-x01-revision) / [Adoption（版本采用）](17-terminology-glossary.md#term-x01-adoption) / [Event Envelope（事件信封）](17-terminology-glossary.md#term-x01-event-envelope) | 共享协议 无业务写者 |
| [M04](modules/m04.md) [Assignment（任职）](17-terminology-glossary.md#term-m04-assignment) / [AssignmentRecord（任职有效期记录）](17-terminology-glossary.md#term-m04-assignmentrecord) | Assignment根含Record及经理关系 / Transfer、EndAssignment |
| [M10](modules/m10.md) [PayrollRun（工资核算批次）](17-terminology-glossary.md#term-m10-payrollrun) / [Result（工资核算结果）](17-terminology-glossary.md#term-m10-payrollresult) / [Balance（工资累计）](17-terminology-glossary.md#term-m10-payrollbalance) | PayrollRun只持结果引用；PayrollResult与PayrollBalance独立根 / SealResult |

提交点、守卫或查询能力：[规范合同](10-ddd-and-state-contracts.md#61-历史查询能力与计算依赖)；恢复：[更正/恢复合同](10-ddd-and-state-contracts.md#61-历史查询能力与计算依赖)；验收：[AC06](10-ddd-and-state-contracts.md#ac06)。

## K07 消费者影响闭环

命令/查询：AssessImpact、ResolveImpact、CorrectResult。

| 对象及所属域 | 同源聚合/写端口 |
| --- | --- |
| [M10](modules/m10.md) [RetroCase（工资追溯事项）](17-terminology-glossary.md#term-m10-retrocase) | RetroCase根 / OpenRetroCase |
| [M10](modules/m10.md) [PayrollImpactPolicy（工资变更影响规则）](17-terminology-glossary.md#term-m10-payrollimpactpolicy) | PayrollImpactPolicy配置根 / PublishPayrollImpactPolicy |
| [X01](modules/x01.md) [Impact Lookup Protocol（影响查询协议）](17-terminology-glossary.md#term-x01-impact-lookup-protocol) | 各域接口约定 无全局索引事实根 |

提交点、守卫或查询能力：[规范合同](10-ddd-and-state-contracts.md#62-消费者影响案与闭环)；恢复：[更正/恢复合同](10-ddd-and-state-contracts.md#24-工资配置影响规则与连续更正)；验收：[AC07](10-ddd-and-state-contracts.md#ac07)。

## K08 人员编号与误建核对

命令/查询：RegisterPerson、RenumberPerson。

| 对象及所属域 | 同源聚合/写端口 |
| --- | --- |
| [M01](modules/m01.md) [PersonNumberAlias（人员编号历史别名）](17-terminology-glossary.md#term-m01-personnumberalias) | Person内编号历史；客户内编号唯一登记守卫 / RegisterPerson、RenumberPerson |
| [M01](modules/m01.md) [Person（人员）](17-terminology-glossary.md#term-m01-person) | Person根 / RegisterPerson |

提交点、守卫或查询能力：[规范合同](10-ddd-and-state-contracts.md#210-人员编号与主选择覆盖)；恢复：[更正/恢复合同](10-ddd-and-state-contracts.md#210-人员编号与主选择覆盖)；验收：[AC08](10-ddd-and-state-contracts.md#ac08)。

## K09 未知执行升级及查明

命令/查询：ResolveUnknown、ApplyInstitutionReceipt、PlacePayerHold、ReleasePayerHold。

| 对象及所属域 | 同源聚合/写端口 |
| --- | --- |
| [M14](modules/m14.md) [PaymentInstruction（付款指令）](17-terminology-glossary.md#term-m14-paymentinstruction) / [Authorization（付款授权）](17-terminology-glossary.md#term-m14-paymentauthorization) | PaymentInstruction与Authorization独立根 / Freeze、Authorize |
| [M11](modules/m11.md) [TaxRemittanceInstruction（税款缴纳指令）](17-terminology-glossary.md#term-m11-taxremittanceinstruction) / [SettlementAuthorization（税款结算授权）](17-terminology-glossary.md#term-m11-settlementauthorization) / [SettlementInvestigation（税款结算核查）](17-terminology-glossary.md#term-m11-settlementinvestigation) | TaxRemittanceInstruction根含分配、核准及核查 / PrepareInstitutionSettlement、ClaimInstitutionSettlement、ApplyInstitutionReceipt、ResolveUnknown |
| [M13](modules/m13.md) [SupplierSettlementInstruction（福利供应商结算指令）](17-terminology-glossary.md#term-m13-suppliersettlementinstruction) / [SupplierMovement（福利供应商资金流水）](17-terminology-glossary.md#term-m13-suppliermovement) | SupplierSettlementInstruction根含授权/核查；SupplierMovement不可变根 / PrepareInstitutionSettlement、ClaimInstitutionSettlement、ApplyInstitutionReceipt、ResolveUnknown |
| [X04](modules/x04.md) [TaskProjection（技术任务视图）](17-terminology-glossary.md#term-x04-technicaltaskprojection) / [Job（技术作业）](17-terminology-glossary.md#term-x04-job) / [Attempt（作业执行尝试）](17-terminology-glossary.md#term-x04-attempt) | 技术作业及尝试记录 / DispatchJob |
| [M12](modules/m12.md) [AgencyPayableAccount（应付经办机构缴款台账）](17-terminology-glossary.md#term-m12-agencypayableaccount) | AgencyPayableAccount根含机构执行意图、授权及核查 / ReserveAgencyPayment、Settle、ResolveUnknown |
| [M15](modules/m15.md) [JournalRequest（会计分录请求）](17-terminology-glossary.md#term-m15-journalrequest) / [Receipt（会计回执）](17-terminology-glossary.md#term-m15-accountingreceipt) | AccountingBatch根含冻结请求及回执 / PrepareJournal、ClaimJournal、ApplyJournalReceipt |
| [M14](modules/m14.md) [PayerAccount（机构付款账户）](17-terminology-glossary.md#term-m14-payeraccount) | PayerAccount根 / RegisterPayerAccount、PlacePayerHold、ReleasePayerHold、AcquirePayerExecution |

提交点、守卫或查询能力：[规范合同](10-ddd-and-state-contracts.md#33-unknown的运营处置)；恢复：[更正/恢复合同](10-ddd-and-state-contracts.md#32-机构付款责任与外部财务委托)；验收：[AC09](10-ddd-and-state-contracts.md#ac09)。

## K10 保留连续更正与净追收不变量

命令/查询：CorrectResult、CancelResult、ApplyRecovery。

| 对象及所属域 | 同源聚合/写端口 |
| --- | --- |
| [M10](modules/m10.md) [ResultSeriesHead（工资结果版本链头）](17-terminology-glossary.md#term-m10-resultserieshead) | ResultSeriesHead根含当前完整结果引用 / SealResult、CorrectResult、CancelResult |
| [M14](modules/m14.md) [RecoveryAccount（工资超付追收台账）](17-terminology-glossary.md#term-m14-recoveryaccount) / [Allocation（追收分配）](17-terminology-glossary.md#term-m14-recoveryallocation) | RecoveryAccount根含追收占用及Allocation；实际分录归PayableAccount / ReserveRecovery、AllocateRecovery |
| [M11](modules/m11.md) [TaxLedger（税务累计账）](17-terminology-glossary.md#term-m11-taxledger) / [TaxAssessment（个税计算结果）](17-terminology-glossary.md#term-m11-taxassessment) | TaxLedger根；TaxAssessment不可变独立根 / Assess、RecognizeIncome |

提交点、守卫或查询能力：[规范合同](10-ddd-and-state-contracts.md#24-工资配置影响规则与连续更正)；恢复：[更正/恢复合同](10-ddd-and-state-contracts.md#31-付款账户与超付追收)；验收：[AC10](10-ddd-and-state-contracts.md#ac10)。
