# 目标对象、聚合归属与写端口目录

本目录与各模块对象表从同一份设计数据生成。根、根内实体、不可变事实、值类型、投影及技术记录分别标明；概念连线不推导同一聚合。

[DDD合同](10-ddd-and-state-contracts.md)规定事务和公开端口。业务DecisionRecord归目标根或本域决定根，X04不持有业务决定。

## M01 人员身份与个人资料

| 目标对象 | 聚合归属及唯一写端口 |
| --- | --- |
| [Person（人员）](17-terminology-glossary.md#term-m01-person) | Person根 / RegisterPerson |
| [PersonalProfile（个人资料）](17-terminology-glossary.md#term-m01-personalprofile) | Person内资料项 / CorrectProfile |
| [ContactRelationship（联系人关系）](17-terminology-glossary.md#term-m01-contactrelationship) | Person内关系实体 / MaintainContact |
| [BankAccountIdentity（个人银行账户）](17-terminology-glossary.md#term-m01-bankaccountidentity) | BankAccountIdentity根 / RegisterAccount |
| [PersonNumberAlias（人员编号历史别名）](17-terminology-glossary.md#term-m01-personnumberalias) | Person内编号历史；客户内编号唯一登记守卫 / RegisterPerson、RenumberPerson |

## M02 企业、组织与地点

| 目标对象 | 聚合归属及唯一写端口 |
| --- | --- |
| [OrganizationObject（组织对象）](17-terminology-glossary.md#term-m02-organizationobject) | OrganizationObject根，含Enterprise/BU/Department/Location及LegalEntity用途 / PublishObject |
| [LegalEntityUse（法律实体用途）](17-terminology-glossary.md#term-m02-legalentityuse) | LegalEntity所属根的用途实体 / DeclareUse |
| [HierarchyVersion（组织架构版本）](17-terminology-glossary.md#term-m02-hierarchyversion) | Hierarchy根含版本及成员边 / PublishHierarchy |
| [OrganizationPublication（组织发布记录）](17-terminology-glossary.md#term-m02-organizationpublication) / [AdoptionReceiptIndex（组织采用回执索引）](17-terminology-glossary.md#term-m02-organizationadoptionreceiptindex) | 发布归源根；回执索引为派生投影 / RebuildIndex |
| [ContextBinding（组织上下文绑定）](17-terminology-glossary.md#term-m02-contextbinding) | ContextBinding根 / BindLegalPurpose |

## M03 职务、职级、职位与编制

| 目标对象 | 聚合归属及唯一写端口 |
| --- | --- |
| [Job（职务）](17-terminology-glossary.md#term-m03-job) / [JobFamily（职务系列）](17-terminology-glossary.md#term-m03-jobfamily) | Job与JobFamily独立根 / MaintainJob |
| [Grade（职级）](17-terminology-glossary.md#term-m03-grade) / [Ladder（职级阶梯）](17-terminology-glossary.md#term-m03-gradeladder) | Grade与GradeLadder独立根 / MaintainGrade |
| [Position（岗位）](17-terminology-glossary.md#term-m03-position) | Position定义根 / MaintainPosition |
| [WorkforcePlan（人员编制计划）](17-terminology-glossary.md#term-m03-workforceplan) | WorkforcePlan根 / PublishWorkforcePlan |
| [StaffingQuota（编制额度）](17-terminology-glossary.md#term-m03-positioncapacity) / [Reservation（招聘编制预留）](17-terminology-glossary.md#term-m03-positionreservation) / [Occupancy（任职占编）](17-terminology-glossary.md#term-m03-positionoccupancy) | StaffingQuota 按部门或岗位维护核定数，同日调剂一次提交所有变动对象 / ChangeStaffingQuota。岗位后续办理补员状态、限期交接、预留及占编 / SetRecruitmentStatus、ReserveStaffingQuota、ReleaseReservation、ConvertReservation、ReleaseOccupancy |
| [PositionHierarchy（岗位层级关系）](17-terminology-glossary.md#term-m03-positionhierarchy) / [HierarchyVersion（岗位层级关系版本）](17-terminology-glossary.md#term-m03-positionhierarchyversion) | PositionHierarchy根含版本及边 / PublishPositionHierarchy |
| [StaffingPolicy（任职与编制管理规则）](17-terminology-glossary.md#term-m03-staffingpolicy) | StaffingPolicy根 / PublishStaffingPolicy |

## M04 用工、任职与劳动合同

| 目标对象 | 聚合归属及唯一写端口 |
| --- | --- |
| [Employment（工作关系记录）](17-terminology-glossary.md#term-m04-employment) | Employment根 / Hire、RegisterAssociation、EndEmployment |
| [Assignment（任职）](17-terminology-glossary.md#term-m04-assignment) / [AssignmentRecord（任职有效期记录）](17-terminology-glossary.md#term-m04-assignmentrecord) | Assignment根含Record及经理关系 / Transfer、EndAssignment |
| [PersonContract（人员合同）](17-terminology-glossary.md#term-m04-personcontract) / [Process（合同办理记录）](17-terminology-glossary.md#term-m04-contractprocess) | Contract根含过程及后继 / RenewContract |
| [PrimarySelection（主要关系与任职选择）](17-terminology-glossary.md#term-m04-primaryselection) / [ManagementHome（主要管理归属）](17-terminology-glossary.md#term-m04-managementhome) | PrimarySelection根 / SelectPrimary |
| [BusinessActionRecord（人事业务办理记录）](17-terminology-glossary.md#term-m04-businessactionrecord) | 目标根内不可变动作记录及更正链 / ExecuteEmploymentAction、CorrectActionRecord、CancelAction |
| [EmploymentActionCatalog（人事业务配置目录）](17-terminology-glossary.md#term-m04-employmentactioncatalog) / [ActionDefinition（人事业务类型）](17-terminology-glossary.md#term-m04-actiondefinition) / [ReasonDefinition（人事业务原因）](17-terminology-glossary.md#term-m04-reasondefinition) / [AssignmentStatusDefinition（任职状态定义）](17-terminology-glossary.md#term-m04-assignmentstatusdefinition) | EmploymentActionCatalog根含定义及关联 / PublishEmploymentActions |
| [ManagerRelationship（汇报关系）](17-terminology-glossary.md#term-m04-managerrelationship) | Assignment根内关系实体 / SetManager、EndManagerRelationship |
| [PrimarySelectionPolicy（主要关系与任职选择规则）](17-terminology-glossary.md#term-m04-primaryselectionpolicy) | PrimarySelectionPolicy配置根 / PublishPrimaryPolicy、ActivatePrimaryPolicy |

## M05 招聘、录用与入职协同

| 目标对象 | 聚合归属及唯一写端口 |
| --- | --- |
| [Requisition（招聘需求）](17-terminology-glossary.md#term-m05-requisition) | Requisition根 / OpenRequisition |
| [Candidate（候选人）](17-terminology-glossary.md#term-m05-candidate) / [Application（求职申请）](17-terminology-glossary.md#term-m05-application) | Candidate与Application独立根 / Apply |
| [OfferVersion（录用方案版本）](17-terminology-glossary.md#term-m05-offerversion) | Offer根含不可变版本 / ApproveOffer、AcceptOffer |
| [HireHandoff（入职信息交接）](17-terminology-glossary.md#term-m05-hirehandoff) | HireHandoff根 / 随已接受录用与资料齐备生成，关联用工结果 |
| [RecruitingProcessDefinition（招聘流程定义）](17-terminology-glossary.md#term-m05-recruitingprocessdefinition) | RecruitingProcessDefinition根 / PublishRecruitingProcess |
| [Interview（面试）](17-terminology-glossary.md#term-m05-interview) / [Assessment（招聘评价）](17-terminology-glossary.md#term-m05-recruitingassessment) | Application根内评价实体及不可变证据 / RecordInterviewAssessment |

## M06 排班与工时制度

| 目标对象 | 聚合归属及唯一写端口 |
| --- | --- |
| [WorkPattern（排班循环模式）](17-terminology-glossary.md#term-m06-workpattern) / [Shift（班次）](17-terminology-glossary.md#term-m06-shift) | WorkPattern与Shift独立配置根 / PublishWorkPattern、PublishShift |
| [SchedulePublication（排班发布版本）](17-terminology-glossary.md#term-m06-schedulepublication) / [Candidate（排班候选方案）](17-terminology-glossary.md#term-m06-schedulecandidate) | SchedulePublication根含不可变发布版本；Candidate为未发布方案 / PublishSchedule、SwapSchedule、CancelSchedule、CorrectSchedule |
| [WorkingTimePolicy（工时制度规则）](17-terminology-glossary.md#term-m06-workingtimepolicy) | WorkingTimePolicy根 / PublishWorkingTimePolicy |
| [ScheduleGenerationProfile（排班生成方案）](17-terminology-glossary.md#term-m06-schedulegenerationprofile) | 独立配置根 / DefineGenerationProfile |
| [WorkingCalendar（工作日历）](17-terminology-glossary.md#term-m06-workingcalendar) | WorkingCalendar根 / PublishWorkingCalendar |
| [ScheduleBook（任职排班登记簿）](17-terminology-glossary.md#term-m06-schedulebook) | ScheduleBook根含发布成员引用 / PublishSchedule、SwapSchedule、CancelSchedule、CorrectSchedule |

## M07 考勤与实际工时

| 目标对象 | 聚合归属及唯一写端口 |
| --- | --- |
| [TimeEvent（原始考勤与工时事件）](17-terminology-glossary.md#term-m07-timeevent) / [DailyCheck（逐日考勤核对）](17-terminology-glossary.md#term-m07-dailycheck) | TimeEvent不可变事实；DailyCheck属TimeRecord / ConfirmTime |
| [TimeRecord（实际工时记录）](17-terminology-glossary.md#term-m07-timerecord) | TimeRecord根 / CorrectTime |
| [MonthlyClosing（工时月结）](17-terminology-glossary.md#term-m07-monthlyclosing) | MonthlyClosing根 / PublishClosing |
| [TimePublication（工时发布记录）](17-terminology-glossary.md#term-m07-timepublication) / [AdoptionReceiptIndex（工时采用回执索引）](17-terminology-glossary.md#term-m07-timeadoptionreceiptindex) | 发布属MonthlyClosing；采用回执为派生索引 / RebuildIndex |
| [TimePolicy（工时处理规则）](17-terminology-glossary.md#term-m07-timepolicy) / [Category（工时类别）](17-terminology-glossary.md#term-m07-timecategory) / [ConsumerSet（工时数据使用方集合）](17-terminology-glossary.md#term-m07-timeconsumerset) / [Profile（工时处理方案）](17-terminology-glossary.md#term-m07-timeprofile) / [Rule（工时规则）](17-terminology-glossary.md#term-m07-timerule) / [Layout（工时录入布局）](17-terminology-glossary.md#term-m07-timelayout) | TimePolicy配置根含可复用定义与关联 / PublishTimePolicy、RetireTimePolicy |
| [TimeDelivery（工时分发记录）](17-terminology-glossary.md#term-m07-timedelivery) / [SourceDefinition（考勤与工时来源配置）](17-terminology-glossary.md#term-m07-timesourcedefinition) | Delivery归TimeRecord或MonthlyClosing源根；SourceDefinition属TimePolicy / PublishTime、RegisterTimeSource |

## M08 休假、缺勤与权益账户

| 目标对象 | 聚合归属及唯一写端口 |
| --- | --- |
| [AbsencePlan（休假与缺勤方案）](17-terminology-glossary.md#term-m08-absenceplan) / [Eligibility（休假与缺勤资格）](17-terminology-glossary.md#term-m08-absenceeligibility) | AbsencePlan配置根；Eligibility不可变评估 / PublishAbsencePlan、RetireAbsencePlan、EvaluateEligibility |
| [EntitlementAccount（休假额度账户）](17-terminology-glossary.md#term-m08-entitlementaccount) / [GrantLot（休假额度发放批次）](17-terminology-glossary.md#term-m08-grantlot) / [Entry（休假额度变动流水）](17-terminology-glossary.md#term-m08-entitlemententry) / [ConsumptionAllocation（休假额度扣减分配）](17-terminology-glossary.md#term-m08-consumptionallocation) | EntitlementAccount根含授予批次/流水/分配 / Grant、Reserve、Consume、ReverseConsumption、Expire、CarryForward、AdjustGrant |
| [LeaveRequest（休假与缺勤申请）](17-terminology-glossary.md#term-m08-leaverequest) / [AbsenceEvent（生效缺勤记录）](17-terminology-glossary.md#term-m08-absenceevent) | AbsenceRequest根含Event、决定及更正 / SubmitAbsence、DecideAbsence、ConfirmAbsence、CancelAbsence、CorrectAbsence |
| [AbsenceCalculation（缺勤计算结果）](17-terminology-glossary.md#term-m08-absencecalculation) | AbsenceRequest下不可变结果 / CalculateAbsence |
| [AbsenceCatalog（休假与缺勤目录）](17-terminology-glossary.md#term-m08-absencecatalog) / [Type（缺勤类型）](17-terminology-glossary.md#term-m08-absencetype) / [Reason（缺勤原因）](17-terminology-glossary.md#term-m08-absencereason) / [Category（缺勤分类）](17-terminology-glossary.md#term-m08-absencecategory) | AbsenceCatalog配置根含定义和关联 / PublishAbsenceCatalog、RetireAbsenceCatalog |
| [AbsenceEnrollment（休假与缺勤方案参与记录）](17-terminology-glossary.md#term-m08-absenceenrollment) | AbsenceEnrollment根 / EnrollAbsencePlan、EndAbsenceEnrollment |
| [AbsenceTimeline（任职缺勤时间线）](17-terminology-glossary.md#term-m08-absencetimeline) | AbsenceTimeline根含确认覆盖引用 / ConfirmAbsence、CancelAbsence、CorrectAbsence |

## M09 薪酬体系、定薪与激励

| 目标对象 | 聚合归属及唯一写端口 |
| --- | --- |
| [CompensationPlan（薪酬计划）](17-terminology-glossary.md#term-m09-compensationplan) / [GradeRate（职级薪酬标准）](17-terminology-glossary.md#term-m09-graderate) | CompensationPlan与GradeRate独立根 / PublishPlan、PublishRate |
| [CompensationAward（个人薪酬核定）](17-terminology-glossary.md#term-m09-compensationaward) | Award根 / ConfirmAward |
| [AwardDelivery（薪酬核定下发记录）](17-terminology-glossary.md#term-m09-awarddelivery) | Award内Delivery实体 / DeliverAward |

## M10 工资核算、累计与追溯

| 目标对象 | 聚合归属及唯一写端口 |
| --- | --- |
| [PayrollContext（计薪上下文）](17-terminology-glossary.md#term-m10-payrollcontext) | PayrollContext根 / EstablishPayrollContext |
| [PayrollMembership（计薪归属）](17-terminology-glossary.md#term-m10-payrollmembership) / [Allocation（计薪分配）](17-terminology-glossary.md#term-m10-payrollallocation) | PayrollMembership根含Allocation，仅经集合协调入口 / AssignPayroll、EndPayrollMembership、CorrectAllocation |
| [PayrollDefinition（工资核算配置）](17-terminology-glossary.md#term-m10-payrolldefinition) / [Calendar（工资日历）](17-terminology-glossary.md#term-m10-payrollcalendar) | PayrollDefinition与Calendar独立根 / PublishPayrollDefinition、PublishPayrollCalendar |
| [Element（工资项目）](17-terminology-glossary.md#term-m10-element) / [PayrollInput（计薪输入）](17-terminology-glossary.md#term-m10-payrollinput) / [CalculationRule（工资计算规则）](17-terminology-glossary.md#term-m10-calculationrule) | Element与CalculationRule独立配置根；PayrollInput根含Entry / PublishElement、PublishCalculationRule、AdoptInput、RetirePayrollConfiguration |
| [PayrollRun（工资核算批次）](17-terminology-glossary.md#term-m10-payrollrun) / [Result（工资核算结果）](17-terminology-glossary.md#term-m10-payrollresult) / [Balance（工资累计）](17-terminology-glossary.md#term-m10-payrollbalance) | PayrollRun只持结果引用；PayrollResult与PayrollBalance独立根 / SealResult |
| [RetroCase（工资追溯事项）](17-terminology-glossary.md#term-m10-retrocase) | RetroCase根 / OpenRetroCase |
| [TimeAdoption（工时采用记录）](17-terminology-glossary.md#term-m10-timeadoption) / [PayTermAdoption（薪酬条款采用记录）](17-terminology-glossary.md#term-m10-paytermadoption) | 消费者采用属输入或结果根 / AdoptSource |
| [PayslipPublication（工资单发布记录）](17-terminology-glossary.md#term-m10-payslippublication) | PayslipPublication根 / PublishPayslip |
| [PayableSourceHead（工资应付来源版本链头）](17-terminology-glossary.md#term-m10-payablesourcehead) | PayableSourceHead根 / AppendPayableMovement |
| [PayrollFlowDefinition（工资流程定义）](17-terminology-glossary.md#term-m10-payrollflowdefinition) / [Instance（工资流程实例）](17-terminology-glossary.md#term-m10-payrollflowinstance) | FlowDefinition与FlowInstance独立根 / PublishPayrollFlow、StartPayrollFlow、RetirePayrollFlow |
| [BalanceDefinition（累计指标定义）](17-terminology-glossary.md#term-m10-balancedefinition) / [Dimension（累计维度）](17-terminology-glossary.md#term-m10-balancedimension) / [Group（累计指标组）](17-terminology-glossary.md#term-m10-balancegroup) | BalanceDefinition与BalanceGroup配置根；Dimension为定义内值类型 / PublishBalanceDefinition、PublishBalanceGroup |
| [PayrollAllocationSet（计薪分配集合）](17-terminology-glossary.md#term-m10-payrollallocationset) | PayrollAllocationSet根含成员引用与规则依据 / AssignPayroll、EndPayrollMembership、CorrectAllocation |
| [PayrollImpactPolicy（工资变更影响规则）](17-terminology-glossary.md#term-m10-payrollimpactpolicy) | PayrollImpactPolicy配置根 / PublishPayrollImpactPolicy |
| [ResultSeriesHead（工资结果版本链头）](17-terminology-glossary.md#term-m10-resultserieshead) | ResultSeriesHead根含当前完整结果引用 / SealResult、CorrectResult、CancelResult |

## M11 个人所得税扣缴与申报

| 目标对象 | 聚合归属及唯一写端口 |
| --- | --- |
| [WithholdingRegistration（扣缴义务人登记）](17-terminology-glossary.md#term-m11-withholdingregistration) | WithholdingRegistration根 / RegisterWithholding |
| [TaxIdentity（税收身份）](17-terminology-glossary.md#term-m11-taxidentity) / [Deduction（个税扣除信息）](17-terminology-glossary.md#term-m11-taxdeduction) | TaxIdentity与Deduction独立有效根 / ConfirmTaxFacts |
| [TaxLedger（税务累计账）](17-terminology-glossary.md#term-m11-taxledger) / [TaxAssessment（个税计算结果）](17-terminology-glossary.md#term-m11-taxassessment) | TaxLedger根；TaxAssessment不可变独立根 / Assess、RecognizeIncome |
| [Filing（个税扣缴申报）](17-terminology-glossary.md#term-m11-taxfiling) / [TaxMovement（税款资金流水）](17-terminology-glossary.md#term-m11-taxmovement) | Filing与TaxMovement独立根 / File、ApplyTaxMovement |
| [TaxCommitment（所得确认控制记录）](17-terminology-glossary.md#term-m11-taxcommitment) / [IncomeItemPlan（所得明细处理计划）](17-terminology-glossary.md#term-m11-incomeitemplan) / [DispatchPermit（涉税付款执行许可）](17-terminology-glossary.md#term-m11-dispatchpermit) | TaxLedger内成员与执行许可 / Commit、AcquirePermit |
| [TaxPolicy（税务规则）](17-terminology-glossary.md#term-m11-taxpolicy) / [IncomeEvidence（所得来源证据）](17-terminology-glossary.md#term-m11-incomeevidence) | TaxPolicy配置根；IncomeEvidence属评估/确认快照 / PublishTaxPolicy、AdoptIncomeEvidence |
| [PaymentTreatmentDecision（付款涉税处理认定）](17-terminology-glossary.md#term-m11-paymenttreatmentdecision) | TaxLedger内用途决定实体 / ClassifyRecoveryReturn |
| [TaxRemittanceAccount（应缴税款台账）](17-terminology-glossary.md#term-m11-taxremittanceaccount) | TaxRemittanceAccount根含义务行及来源head / ConfirmTaxLiability |
| [TaxRemittanceInstruction（税款缴纳指令）](17-terminology-glossary.md#term-m11-taxremittanceinstruction) / [SettlementAuthorization（税款结算授权）](17-terminology-glossary.md#term-m11-settlementauthorization) / [SettlementInvestigation（税款结算核查）](17-terminology-glossary.md#term-m11-settlementinvestigation) | TaxRemittanceInstruction根含分配、核准及核查 / PrepareInstitutionSettlement、ClaimInstitutionSettlement、ApplyInstitutionReceipt、ResolveUnknown |

## M12 社会保险与住房公积金

| 目标对象 | 聚合归属及唯一写端口 |
| --- | --- |
| [AgencyAccount（社保公积金经办账户）](17-terminology-glossary.md#term-m12-agencyaccount) / [Enrollment（参保与缴存登记）](17-terminology-glossary.md#term-m12-contributionenrollment) | AgencyAccount与Enrollment独立根 / Register、Enroll |
| [ObligationAccount（参缴义务账户）](17-terminology-glossary.md#term-m12-obligationaccount) / [ObligationVersion（应缴义务版本）](17-terminology-glossary.md#term-m12-obligationversion) | ObligationAccount根含Version及Delta / AssessObligation |
| [ContributionFiling（参缴申报）](17-terminology-glossary.md#term-m12-contributionfiling) / [LineReceipt（参缴申报明细回执）](17-terminology-glossary.md#term-m12-linereceipt) | FilingBatch根含LineReceipt / FileContribution |
| [ContributionSettlement（参缴实缴与退款记录）](17-terminology-glossary.md#term-m12-contributionsettlement) | AgencyMovement根含结算引用 / ApplyAgencyMovement |
| [EmployeeReceivable（应收个人缴费款）](17-terminology-glossary.md#term-m12-employeereceivable) / [DeductionReservation（代扣预留）](17-terminology-glossary.md#term-m12-deductionreservation) | ObligationAccount内员工子账及占用 / ReserveDeduction |
| [Correction（参缴义务更正）](17-terminology-glossary.md#term-m12-contributioncorrection) / [CollectionAllocation（个人缴费扣收分配）](17-terminology-glossary.md#term-m12-collectionallocation) | ObligationAccount内更正及分配 / Correct、AllocateCollection |
| [AgencyPayableAccount（应付经办机构缴款台账）](17-terminology-glossary.md#term-m12-agencypayableaccount) | AgencyPayableAccount根含机构执行意图、授权及核查 / ReserveAgencyPayment、Settle、ResolveUnknown |
| [ContributionPolicy（参缴规则）](17-terminology-glossary.md#term-m12-contributionpolicy) / [BaseAssessment（缴费基数评估）](17-terminology-glossary.md#term-m12-baseassessment) | ContributionPolicy配置根；BaseAssessment不可变评估 / PublishContributionPolicy、AssessContributionBase、RetireContributionPolicy |

## M13 企业福利

| 目标对象 | 聚合归属及唯一写端口 |
| --- | --- |
| [BenefitPlan（福利计划）](17-terminology-glossary.md#term-m13-benefitplan) / [Option（福利选项）](17-terminology-glossary.md#term-m13-benefitoption) | BenefitPlan根含Option / PublishBenefitPlan |
| [BenefitEnrollment（福利参与记录）](17-terminology-glossary.md#term-m13-benefitenrollment) / [LifeEvent（生活事件）](17-terminology-glossary.md#term-m13-lifeevent) | BenefitEnrollment与LifeEvent独立根 / EnrollBenefit |
| [BenefitCost（福利费用）](17-terminology-glossary.md#term-m13-benefitcost) / [Premium（保费）](17-terminology-glossary.md#term-m13-premium) | BenefitCost根含Premium及账单引用 / ConfirmBenefitCost |
| [BenefitClaim（福利待遇申请）](17-terminology-glossary.md#term-m13-benefitclaim) | BenefitClaim根 / DecideClaim |
| [BenefitSupplierPayable（应付福利供应商款）](17-terminology-glossary.md#term-m13-benefitsupplierpayable) | BenefitSupplierPayable根 / ConfirmSupplierPayable |
| [SupplierSettlementInstruction（福利供应商结算指令）](17-terminology-glossary.md#term-m13-suppliersettlementinstruction) / [SupplierMovement（福利供应商资金流水）](17-terminology-glossary.md#term-m13-suppliermovement) | SupplierSettlementInstruction根含授权/核查；SupplierMovement不可变根 / PrepareInstitutionSettlement、ClaimInstitutionSettlement、ApplyInstitutionReceipt、ResolveUnknown |

## M14 工资支付与银行用途

| 目标对象 | 聚合归属及唯一写端口 |
| --- | --- |
| [PayerAccount（机构付款账户）](17-terminology-glossary.md#term-m14-payeraccount) | PayerAccount根 / RegisterPayerAccount、PlacePayerHold、ReleasePayerHold、AcquirePayerExecution |
| [PaymentUse（账户支付用途）](17-terminology-glossary.md#term-m14-paymentuse) / [PayerConfiguration（付款方配置）](17-terminology-glossary.md#term-m14-payerconfiguration) | PaymentUse与PayerConfiguration独立根 / ConfigurePaymentUse |
| [PayableAccount（工资应付结算台账）](17-terminology-glossary.md#term-m14-payableaccount) / [SettlementEntry（工资结算分录）](17-terminology-glossary.md#term-m14-settlemententry) / [Reservation（付款额度占用）](17-terminology-glossary.md#term-m14-paymentreservation) | PayableAccount根含SettlementEntry及付款Reservation / ApplySettlementEntry、ReservePayable |
| [PaymentInstruction（付款指令）](17-terminology-glossary.md#term-m14-paymentinstruction) / [Authorization（付款授权）](17-terminology-glossary.md#term-m14-paymentauthorization) | PaymentInstruction与Authorization独立根 / Freeze、Authorize |
| [PaymentAttempt（付款执行尝试）](17-terminology-glossary.md#term-m14-paymentattempt) / [Receipt（付款渠道回执）](17-terminology-glossary.md#term-m14-paymentreceipt) / [Movement（工资付款资金流水）](17-terminology-glossary.md#term-m14-cashmovement) | 尝试属指令；Receipt观察去重后CashMovement不可变 / ApplyReceipt |
| [RecoveryAccount（工资超付追收台账）](17-terminology-glossary.md#term-m14-recoveryaccount) / [Allocation（追收分配）](17-terminology-glossary.md#term-m14-recoveryallocation) | RecoveryAccount根含追收占用及Allocation；实际分录归PayableAccount / ReserveRecovery、AllocateRecovery |

## M15 人工成本与财务衔接

| 目标对象 | 聚合归属及唯一写端口 |
| --- | --- |
| [CostFact（人工成本事实）](17-terminology-glossary.md#term-m15-costfact) / [CostAllocation（成本分摊）](17-terminology-glossary.md#term-m15-costallocation) | CostFact不可变根；CostAllocation集合根 / AdoptCost、AllocateCost、FreezeAllocation |
| [AccountingMapping（会计映射）](17-terminology-glossary.md#term-m15-accountingmapping) / [Period（会计期间）](17-terminology-glossary.md#term-m15-accountingperiod) | AccountingMapping与Period独立根 / PublishMapping、BeginClosePeriod、ClosePeriod |
| [JournalRequest（会计分录请求）](17-terminology-glossary.md#term-m15-journalrequest) / [Receipt（会计回执）](17-terminology-glossary.md#term-m15-accountingreceipt) | AccountingBatch根含冻结请求及回执 / PrepareJournal、ClaimJournal、ApplyJournalReceipt |
| [CostCorrection（成本更正）](17-terminology-glossary.md#term-m15-costcorrection) | CostCorrection根引用原事实 / CorrectCost |

## M16 人才档案、技能与资格

| 目标对象 | 聚合归属及唯一写端口 |
| --- | --- |
| [ContentType（人才档案内容类型）](17-terminology-glossary.md#term-m16-contenttype) / [Item（人才档案内容项）](17-terminology-glossary.md#term-m16-contentitem) / [RatingModel（评分模型）](17-terminology-glossary.md#term-m16-ratingmodel) | ContentDefinition及RatingModel独立根 / PublishContent、PublishRatingModel、RetireDefinition |
| [Profile（人才档案）](17-terminology-glossary.md#term-m16-talentprofile) / [Section（档案栏目）](17-terminology-glossary.md#term-m16-profilesection) | TalentProfile根含Section及所采用类型/章节版本 / MaintainProfile、AdoptProfileTypeVersion |
| [Qualification（人员资格）](17-terminology-glossary.md#term-m16-qualification) / [Assessment（资格评价证据）](17-terminology-glossary.md#term-m16-qualificationassessment) | Qualification根；Assessment为不可变证据 / RecordEvidence |
| [Questionnaire（问卷）](17-terminology-glossary.md#term-m16-questionnaire) / [Response（问卷作答记录）](17-terminology-glossary.md#term-m16-questionnaireresponse) | Questionnaire配置根 / PublishQuestionnaire、RetireQuestionnaire；Response归所属评价聚合 / SubmitResponse |
| [QualificationDecision（资格认定决定）](17-terminology-glossary.md#term-m16-qualificationdecision) | Qualification内DecisionRecord / DecideQualification |
| [ProfileType（档案类型）](17-terminology-glossary.md#term-m16-profiletype) / [SectionDefinition（档案栏目定义）](17-terminology-glossary.md#term-m16-sectiondefinition) | ProfileType配置根含SectionDefinition / PublishProfileType |

## M17 目标与绩效

| 目标对象 | 聚合归属及唯一写端口 |
| --- | --- |
| [GoalDefinition（绩效目标定义）](17-terminology-glossary.md#term-m17-goaldefinition) / [GoalPlan（绩效目标计划）](17-terminology-glossary.md#term-m17-goalplan) | GoalDefinition及GoalPlan独立配置根 / PublishGoalDefinition、PublishGoalPlan、RetireGoalConfiguration |
| [ReviewPeriod（绩效考核周期）](17-terminology-glossary.md#term-m17-reviewperiod) / [PerformanceTemplate（绩效模板）](17-terminology-glossary.md#term-m17-performancetemplate) | ReviewPeriod及PerformanceTemplate独立配置根 / PublishReviewPeriod、PublishTemplate、RetireTemplate |
| [PerformanceDocument（绩效评价文档）](17-terminology-glossary.md#term-m17-performancedocument) / [StageInstance（绩效阶段实例）](17-terminology-glossary.md#term-m17-stageinstance) | PerformanceDocument根含StageInstance及反馈 / CreatePerformanceDocument、Evaluate、AdvanceStage、PublishPerformance、CorrectPerformance |
| [Feedback（绩效反馈）](17-terminology-glossary.md#term-m17-feedback) / [Appeal（绩效申诉）](17-terminology-glossary.md#term-m17-appeal) / [Calibration（绩效校准）](17-terminology-glossary.md#term-m17-calibration) | PerformanceDocument内决定及后继 / Appeal、Calibrate |
| [GoalAssignment（个人绩效目标分配）](17-terminology-glossary.md#term-m17-goalassignment) / [GoalInstance（个人绩效目标）](17-terminology-glossary.md#term-m17-goalinstance) / [Progress（目标进度）](17-terminology-glossary.md#term-m17-goalprogress) | GoalAssignment根含GoalInstance及Progress / AssignGoalPlan、AddGoal、UpdateGoalProgress、ConfirmGoals |
| [PerformanceProcessDefinition（绩效流程定义）](17-terminology-glossary.md#term-m17-performanceprocessdefinition) / [DocumentType（绩效文档类型）](17-terminology-glossary.md#term-m17-performancedocumenttype) | PerformanceProcessDefinition及DocumentType独立配置根 / PublishPerformanceProcess、PublishDocumentType、RetirePerformanceConfiguration |

## M18 学习与发展

| 目标对象 | 聚合归属及唯一写端口 |
| --- | --- |
| [LearningCatalog（学习目录）](17-terminology-glossary.md#term-m18-learningcatalog) / [Offering（课程开设安排）](17-terminology-glossary.md#term-m18-learningoffering) | Course与LearningOffering独立根 / PublishCourse、OpenOffering、ChangeCapacity、CloseOffering |
| [LearningEnrollment（学习报名）](17-terminology-glossary.md#term-m18-learningenrollment) / [SeatReservation（学习名额占用）](17-terminology-glossary.md#term-m18-seatreservation) | LearningEnrollment根；SeatReservation及WaitlistEntry归LearningOffering / Enroll、CancelEnrollment、PromoteWaitlist、TransferEnrollment |
| [LearningRecord（学习记录）](17-terminology-glossary.md#term-m18-learningrecord) | LearningRecord根 / ConfirmCompletion |
| [DevelopmentPlan（个人发展计划）](17-terminology-glossary.md#term-m18-developmentplan) / [DevelopmentGoal（个人发展目标）](17-terminology-glossary.md#term-m18-developmentgoal) / [Action（发展行动）](17-terminology-glossary.md#term-m18-developmentaction) | DevelopmentPlan根含DevelopmentGoal及Action / PlanDevelopment、AddDevelopmentGoal、UpdateDevelopmentAction、CompleteDevelopmentGoal |
| [LearningPath（学习路径）](17-terminology-glossary.md#term-m18-learningpath) / [Audience（学习适用人群）](17-terminology-glossary.md#term-m18-learningaudience) | LearningPath及Audience独立配置根 / PublishPath、DefineAudience |
| [LearningAssignment（学习任务）](17-terminology-glossary.md#term-m18-learningassignment) | LearningAssignment根 / AssignLearning、ReviseLearningRequirement、CancelLearningAssignment |
| [DevelopmentGoalDefinition（发展目标定义）](17-terminology-glossary.md#term-m18-developmentgoaldefinition) | DevelopmentGoalDefinition配置根 / PublishDevelopmentGoal、RetireDevelopmentGoal |

## M19 人才盘点、人才池与继任

| 目标对象 | 聚合归属及唯一写端口 |
| --- | --- |
| [TalentReviewSession（人才盘点会议）](17-terminology-glossary.md#term-m19-talentreviewsession) | TalentReview根 / ConfirmReview |
| [TalentPool（人才池）](17-terminology-glossary.md#term-m19-talentpool) / [Membership（人才池成员关系）](17-terminology-glossary.md#term-m19-talentpoolmembership) | TalentPool根含Membership / EnterPool、ExitPool |
| [SuccessionPlan（继任计划）](17-terminology-glossary.md#term-m19-successionplan) / [Candidate（继任候选人）](17-terminology-glossary.md#term-m19-successorcandidate) | SuccessionPlan根含Subject、Candidate及发展行动引用 / CreateSuccessionPlan、NominateSuccessor、WithdrawCandidate、CloseSuccessionPlan |
| [ReadinessAssessment（继任准备度评估）](17-terminology-glossary.md#term-m19-readinessassessment) | SuccessionPlan内ReadinessAssessment及决定 / ConfirmReadiness |
| [TalentReviewTemplate（人才盘点模板）](17-terminology-glossary.md#term-m19-talentreviewtemplate) / [Dimension（人才盘点评价维度）](17-terminology-glossary.md#term-m19-talentreviewdimension) | TalentReviewTemplate配置根含Dimension / PublishReviewTemplate |

## M20 员工经理服务与HR任务

| 目标对象 | 聚合归属及唯一写端口 |
| --- | --- |
| [ServiceRequest（人力资源服务请求）](17-terminology-glossary.md#term-m20-servicerequest) | ServiceRequest根 / Open、ResolveRequest |
| [TaskProjection（业务待办视图）](17-terminology-glossary.md#term-m20-taskprojection) | 可重建投影 / RefreshTaskProjection |
| [JourneyInstance（员工历程实例）](17-terminology-glossary.md#term-m20-journeyinstance) / [StepInstance（历程步骤实例）](17-terminology-glossary.md#term-m20-stepinstance) | JourneyInstance根含StepInstance / StartJourney、ConfirmStepByEvidence、ReassessStep、CancelJourney |
| [Communication（员工沟通内容）](17-terminology-glossary.md#term-m20-communication) / [DeliveryProjection（消息送达视图）](17-terminology-glossary.md#term-m20-deliveryprojection) | Communication根；Delivery为X04/X05投影 / PublishCommunication |
| [PayslipView（工资单授权视图）](17-terminology-glossary.md#term-m20-payslipview) | M10发布件授权投影 / ReadPayslip |
| [JourneyTemplate（员工历程模板）](17-terminology-glossary.md#term-m20-journeytemplate) / [StepDefinition（历程步骤定义）](17-terminology-glossary.md#term-m20-stepdefinition) | JourneyTemplate配置根含StepDefinition / PublishJourneyTemplate、RetireJourneyTemplate |

## X01 身份、有效期、修订与采用历史

| 目标对象 | 聚合归属及唯一写端口 |
| --- | --- |
| [Identity（稳定对象标识）](17-terminology-glossary.md#term-x01-identity) / [EffectiveRange Types（有效期间）](17-terminology-glossary.md#term-x01-effectiverange) | 共享值类型 无业务写者 |
| [Revision（修订版本）](17-terminology-glossary.md#term-x01-revision) / [Adoption（版本采用）](17-terminology-glossary.md#term-x01-adoption) / [Event Envelope（事件信封）](17-terminology-glossary.md#term-x01-event-envelope) | 共享协议 无业务写者 |
| [Impact Lookup Protocol（影响查询协议）](17-terminology-glossary.md#term-x01-impact-lookup-protocol) | 各域接口约定 无全局索引事实根 |
| [DecisionRef Envelope（决定引用信封）](17-terminology-glossary.md#term-x01-decisionref-envelope) | 共享协议 决定仍归目标域 |

## X02 共享配置、适用范围、默认与政策版本

| 目标对象 | 聚合归属及唯一写端口 |
| --- | --- |
| [SharedReference（共享对象引用）](17-terminology-glossary.md#term-x02-sharedreference) | 引用描述值类型，实际对象归所属域 |
| [ApplicabilityPolicy（适用范围规则）](17-terminology-glossary.md#term-x02-applicabilitypolicy) | ApplicabilityPolicy根 / PublishApplicability、RetireApplicability、ResolveApplicability |
| [DefaultRule（默认值规则）](17-terminology-glossary.md#term-x02-defaultrule) / [DefaultProposal（默认值建议）](17-terminology-glossary.md#term-x02-defaultproposal) | DefaultRule配置根 / PublishDefaultRule、RetireDefaultRule；DefaultProposal为派生值 / ProposeDefault |
| [PolicyVersion（政策版本）](17-terminology-glossary.md#term-x02-policyversion) / [Evidence（政策来源证据）](17-terminology-glossary.md#term-x02-policyevidence) | PolicyEvidence及发布版本根 / PublishEvidence |

## X03 身份权限与隐私

| 目标对象 | 聚合归属及唯一写端口 |
| --- | --- |
| [Principal（访问主体）](17-terminology-glossary.md#term-x03-principal) / [RoleGrant（角色授权）](17-terminology-glossary.md#term-x03-rolegrant) | Principal与RoleGrant独立根 / RegisterPrincipal、DisablePrincipal、GrantRole、AdoptGrantVersion、RevokeGrant |
| [DataScope（数据访问范围）](17-terminology-glossary.md#term-x03-datascope) / [FieldPolicy（字段访问策略）](17-terminology-glossary.md#term-x03-fieldpolicy) | AccessPolicy根 / PublishAccessPolicy、RetireAccessPolicy |
| [ProcessingPurpose（个人信息处理目的）](17-terminology-glossary.md#term-x03-processingpurpose) / [Retention（数据保留策略）](17-terminology-glossary.md#term-x03-retentionpolicy) | PurposePolicy与RetentionPolicy独立根 / DefinePurpose |
| [AccessDecision（访问判定结果）](17-terminology-glossary.md#term-x03-accessdecision) / [ExportAudit（导出审计）](17-terminology-glossary.md#term-x03-exportaudit) | AccessDecision请求内计算结果 / EvaluateAccess；ExportAudit最小操作事件 / RecordExportEvent |
| [RoleDefinition（角色定义）](17-terminology-glossary.md#term-x03-roledefinition) / [CapabilityRef（操作能力引用）](17-terminology-glossary.md#term-x03-capabilityref) | RoleDefinition配置根 / PublishRoleDefinition、RetireRoleDefinition |

## X04 任务、作业与人工核对

| 目标对象 | 聚合归属及唯一写端口 |
| --- | --- |
| [TaskProjection（技术任务视图）](17-terminology-glossary.md#term-x04-technicaltaskprojection) / [Job（技术作业）](17-terminology-glossary.md#term-x04-job) / [Attempt（作业执行尝试）](17-terminology-glossary.md#term-x04-attempt) | 技术作业及尝试记录 / DispatchJob |
| [Delivery（事件投递）](17-terminology-glossary.md#term-x04-delivery) / [ProcessingEvidence（处理证据）](17-terminology-glossary.md#term-x04-processingevidence) | 投递及处理证据记录 / DeliverEvent |
| [NotificationRequest（通知请求）](17-terminology-glossary.md#term-x04-notificationrequest) / [DeliveryReceipt（通知送达回执）](17-terminology-glossary.md#term-x04-deliveryreceipt) | 通知尝试及回执记录 / SendNotification |

## X05 导入导出与外部集成

| 目标对象 | 聚合归属及唯一写端口 |
| --- | --- |
| [Endpoint（集成端点）](17-terminology-glossary.md#term-x05-endpoint) / [MappingVersion（接口映射版本）](17-terminology-glossary.md#term-x05-mappingversion) | 适配配置版本 / ConfigureAdapter |
| [ImportBatch（导入批次）](17-terminology-glossary.md#term-x05-importbatch) / [Observation（外部观察记录）](17-terminology-glossary.md#term-x05-observation) | 不可变来源观察记录 / ObserveImport |
| [ExecutionIntentRef（外部执行意图引用）](17-terminology-glossary.md#term-x05-executionintentref) / [ChannelAttempt（渠道执行尝试）](17-terminology-glossary.md#term-x05-channelattempt) | 技术尝试记录 / DispatchCommittedIntent |
| [Receipt（接口回执）](17-terminology-glossary.md#term-x05-integrationreceipt) / [Reconciliation（对账与核对）](17-terminology-glossary.md#term-x05-reconciliation) | 原始观察及核对记录 / ObserveReceipt |

## X06 报表、指标与事实追溯

| 目标对象 | 聚合归属及唯一写端口 |
| --- | --- |
| [MetricDefinition（指标口径定义）](17-terminology-glossary.md#term-x06-metricdefinition) | MetricDefinition根 / PublishMetric |
| [AuthorizedProjection（授权分析视图）](17-terminology-glossary.md#term-x06-authorizedprojection) | 可重建读模型 / ProjectFacts |
| [ReportSnapshot（报表快照）](17-terminology-glossary.md#term-x06-reportsnapshot) / [Lineage（数据来源追溯）](17-terminology-glossary.md#term-x06-lineage) | ReportSnapshot不可变根含Lineage / PublishReport |
| [ExportRecord（报表导出记录）](17-terminology-glossary.md#term-x06-exportrecord) | ExportRecord不可变根 / ExportReport |
