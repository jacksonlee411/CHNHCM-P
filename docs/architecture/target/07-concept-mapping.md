# HCM核心概念逐项映射

本表将[参考文档](../../reference/not-imported.md#source-a9f68bab28ef)第1—7节映射为131条对标条目（部分为组合概念）。条目数不是功能数、完成率或128计划编号。

目标处置是China HCM独立架构决定[D]，按HCM概念问题、中国业务边界与DDD原则逐项说明。原型观察只作历史参考；采纳不表示已实现。本仓采用的来源提交及初始摘要见[基线记录](../../reference/baselines.json)，后续变更见[本仓修订链](../../reference/local-revisions.json)；[来源证据记录](../../reference/not-imported.md#source-5801fdf4bf92)仅作历史路由，不作为当前全部设计的验收结论。

各行的主责是逻辑设计，组合主责表示引用/生产消费关系；事实的唯一维护域以[归属表](02-target-architecture.md#2-事实归属与依赖)为准。

## 1. 人员、工作关系与任职

| 条目 | 参考概念 | 目标处置 | 责任模块及聚合入口 | 目标对象或边界 | 图与节点定位 |
| --- | --- | --- | --- | --- | --- |
| C001 | Person<br>[Person（人员）](17-terminology-glossary.md#term-m01-person) | 采纳并调整创建路径 | [M01](modules/m01.md) | 稳定自然人身份；允许身份核对登记先于工作关系，登记目的、权限与保留政策独立管理 | [c01-core-hcm-concepts](diagrams/c01-core-hcm-concepts.svg)：P |
| C002 | Work Relationship<br>[Work Relationship（工作关系）](17-terminology-glossary.md#term-m04-work-relationship) | 采纳并区分关系性质 | [M04](modules/m04.md) | Employment记录人与明确组织的系统关系；劳动关系核对用人单位，非劳动/非工作关联按实际角色；entityId不自动等于雇主 | [c01-core-hcm-concepts](diagrams/c01-core-hcm-concepts.svg)：E |
| C003 | Assignment<br>[Assignment（任职）](17-terminology-glossary.md#term-m04-assignment) | 采纳 | [M04](modules/m04.md) | 关系下任职独立身份和时间轴 | [c01-core-hcm-concepts](diagrams/c01-core-hcm-concepts.svg)：A |
| C004 | Primary relationship<br>[Primary Work Relationship（主要工作关系）](17-terminology-glossary.md#term-m04-primary-work-relationship) | 语义替代 | [M04](modules/m04.md) | 按日期及用途的主关系选择，不以人员主岗位字段替代 | [c01-core-hcm-concepts](diagrams/c01-core-hcm-concepts.svg)：PR |
| C005 | Primary assignment<br>[Primary Assignment（主要任职）](17-terminology-glossary.md#term-m04-primary-assignment) | 采纳 | [M04](modules/m04.md) | 工作关系内主任职；不推导全部税费与权限 | [c01-core-hcm-concepts](diagrams/c01-core-hcm-concepts.svg)：PR |
| C006 | Action<br>[ActionDefinition（人事业务类型）](17-terminology-glossary.md#term-m04-actiondefinition) | 采纳 | [M04](modules/m04.md) | M04可复用ActionDefinition与真实BusinessActionRecord分别有身份；运行保留采用的目录版本 | [m04](diagrams/m04.svg)：DEF, ACT, L |
| C007 | Action Reason<br>[ReasonDefinition（人事业务原因）](17-terminology-glossary.md#term-m04-reasondefinition) | 采纳 | [M04](modules/m04.md) | ReasonDefinition按动作关联、期间启停；旧动作保留原定义版本 | [m04](diagrams/m04.svg)：DEF, ACT |
| C008 | Contact Relationship<br>[ContactRelationship（联系人关系）](17-terminology-glossary.md#term-m01-contactrelationship) | 纳入目标设计 | [M01](modules/m01.md) | 联系人关系和必要资料；可显式关联已核Person | [m01](diagrams/m01.svg)：D, P |
| C009 | Person Types<br>[Person Type（人员类型）](17-terminology-glossary.md#term-m04-person-type) | 语义替代 | [M04](modules/m04.md) | 人员类型在工作关系上下文，非Person全局标签 | [c01-core-hcm-concepts](diagrams/c01-core-hcm-concepts.svg)：E |
| C010 | Name Styles / Name Formats<br>[Name Style（姓名结构）](17-terminology-glossary.md#term-m01-name-style) / [Name Format（姓名显示格式）](17-terminology-glossary.md#term-m01-name-format) | 语义替代 | [M01](modules/m01.md) | 姓名展示及来源规范；不改变稳定身份 | [m01](diagrams/m01.svg)：D, P |
| C011 | Assignment Status<br>[AssignmentStatusDefinition（任职状态定义）](17-terminology-glossary.md#term-m04-assignmentstatusdefinition) | 采纳 | [M04](modules/m04.md) | 版本化AssignmentStatusDefinition与任职状态实例分离；关系、合同、计薪状态分别维护 | [m04](diagrams/m04.svg)：DEF, A |

## 2. 组织、职位与共享数据

| 条目 | 参考概念 | 目标处置 | 责任模块及聚合入口 | 目标对象或边界 | 图与节点定位 |
| --- | --- | --- | --- | --- | --- |
| C012 | Enterprise<br>[Enterprise（企业）](17-terminology-glossary.md#term-m02-enterprise) | 语义替代 | [M02](modules/m02.md) | 客户实例及企业总体上下文，不等同一个部门 | [m02](diagrams/m02.svg)：E |
| C013 | Legal Entity<br>[Legal Entity（法律实体）](17-terminology-glossary.md#term-m02-legal-entity) | 采纳 | [M02](modules/m02.md) | 法律实体稳定身份 | [m02](diagrams/m02.svg)：L, B |
| C014 | Legal Employer<br>[Legal Employer（用人单位）](17-terminology-glossary.md#term-m02-legal-employer) | 大陆语境映射 | [M02](modules/m02.md)、[M04](modules/m04.md) | 劳动关系语境首选用人单位，保留Legal Employer对照与检索别名；法律身份、允许用途与具体人员关系分别确定 | [m02](diagrams/m02.svg)：L, B |
| C015 | Business Unit<br>[Business Unit（业务单元）](17-terminology-glossary.md#term-m02-business-unit) | 语义替代 | [M02](modules/m02.md) | 业务处理及允许值上下文；不默认等同法人 | [m02](diagrams/m02.svg)：BU |
| C016 | Department<br>[Department（部门）](17-terminology-glossary.md#term-m02-department) | 采纳 | [M02](modules/m02.md) | 按日有效的管理部门 | [m02](diagrams/m02.svg)：D |
| C017 | Job<br>[Job（职务）](17-terminology-glossary.md#term-m03-job) | 采纳 | [M03](modules/m03.md) | 工作类型，不是人员占用席位 | [m03](diagrams/m03.svg)：JF, J |
| C018 | Job Family<br>[JobFamily（职务系列）](17-terminology-glossary.md#term-m03-jobfamily) | 采纳 | [M03](modules/m03.md) | 职务分类及有效关系 | [m03](diagrams/m03.svg)：JF, J |
| C019 | Position<br>[Position（岗位）](17-terminology-glossary.md#term-m03-position) | 采纳 | [M03](modules/m03.md) | 独立岗位，按日核编，空缺仍保留身份 | [m03](diagrams/m03.svg)：P, Q, R |
| C020 | Grade<br>[Grade（职级）](17-terminology-glossary.md#term-m03-grade) | 采纳 | [M03](modules/m03.md) | 职级身份与结构 | [m03](diagrams/m03.svg)：G, L |
| C021 | Grade Ladder<br>[GradeLadder（职级阶梯）](17-terminology-glossary.md#term-m03-gradeladder) | 采纳 | [M03](modules/m03.md) | 级别进阶结构；晋级动作与结构分开 | [m03](diagrams/m03.svg)：G, L |
| C022 | Grade Rate<br>[GradeRate（职级薪酬标准）](17-terminology-glossary.md#term-m09-graderate) | 采纳 | [M09](modules/m09.md) | 职级金额/范围、币种频率及有效期由薪酬维护 | [m09](diagrams/m09.svg)：G, R |
| C023 | Location<br>[Location（地点）](17-terminology-glossary.md#term-m02-location) | 采纳 | [M02](modules/m02.md) | 实际地点、地址及适用上下文 | [m02](diagrams/m02.svg)：LOC |
| C024 | Reference Data Set / Common Set<br>[Reference Data Set（参考数据集）](17-terminology-glossary.md#term-x02-reference-data-set) / [Common Set（公共参考数据集）](17-terminology-glossary.md#term-x02-common-set) | 语义替代 | [X02](modules/x02.md) | 稳定共享对象＋按类型/BU等上下文的适用策略、默认及明确采用；支持企业共享与专用范围，保持同名对象身份独立 | [c05-reference-access-concepts](diagrams/c05-reference-access-concepts.svg)：REF, APP, DEFAULT |
| C025 | SetID / TableSet<br>[SetID（集合标识）](17-terminology-glossary.md#term-x02-setid) / [TableSet（表集合共享机制）](17-terminology-glossary.md#term-x02-tableset) | 明确不采纳 | [X02](modules/x02.md) | 不将SetID/TableSet作为China HCM的目标机制；对象级适用策略承担引用范围职责，属于本设计取舍 | [c05-reference-access-concepts](diagrams/c05-reference-access-concepts.svg)：REF, APP, DEFAULT |
| C026 | Tree<br>[Tree（树）](17-terminology-glossary.md#term-m02-tree) | 采纳组织树；岗位树作明确取舍 | [M02](modules/m02.md)、[M03](modules/m03.md) | M02组织用途树、M03岗位层级分别有根、节点边和有效版本；不另采纳多套可独立编辑Position Tree，历史采用保留 | [m02](diagrams/m02.svg)：T, D；[m03](diagrams/m03.svg)：PH, P |
| C027 | Tree Version<br>[HierarchyVersion（组织架构版本）](17-terminology-glossary.md#term-m02-hierarchyversion) / [PositionHierarchyVersion（岗位层级关系版本）](17-terminology-glossary.md#term-m03-positionhierarchyversion) | 采纳组织树；岗位树作明确取舍 | [M02](modules/m02.md)、[M03](modules/m03.md) | M02组织用途树、M03岗位层级分别有根、节点边和有效版本；不另采纳多套可独立编辑Position Tree，历史采用保留 | [m02](diagrams/m02.svg)：T, D；[m03](diagrams/m03.svg)：PH, P |
| C028 | Position hierarchy / Manager relationship<br>[PositionHierarchy（岗位层级关系）](17-terminology-glossary.md#term-m03-positionhierarchy) / [ManagerRelationship（汇报关系）](17-terminology-glossary.md#term-m04-managerrelationship) | 采纳 | [M03](modules/m03.md)、[M04](modules/m04.md) | M03岗位层级与M04双方任职的实际经理关系独立；不自动同步 | [c01-core-hcm-concepts](diagrams/c01-core-hcm-concepts.svg)：PH, POS, MG |

## 3. 法定边界与薪资

| 条目 | 参考概念 | 目标处置 | 责任模块及聚合入口 | 目标对象或边界 | 图与节点定位 |
| --- | --- | --- | --- | --- | --- |
| C029 | LDG<br>[Legislative Data Group（法规数据组）](17-terminology-glossary.md#term-x02-legislative-data-group) | 语义替代 | [X02](modules/x02.md) | 法规上下文、地区币种与政策适用，非物理复制Oracle对象 | [c02-payroll-concepts](diagrams/c02-payroll-concepts.svg)：LAW |
| C030 | PSU<br>[Payroll Statutory Unit（薪资法定单位）](17-terminology-glossary.md#term-m02-payroll-statutory-unit) | 语义替代 | [M02](modules/m02.md)、[M10](modules/m10.md) | M02声明薪资责任用途；M10 PayrollContext绑定人员和该责任用途，不并入单一组织树 | [c02-payroll-concepts](diagrams/c02-payroll-concepts.svg)：LEG, CTX |
| C031 | TRU<br>[Tax Reporting Unit（税务报告单位）](17-terminology-glossary.md#term-m11-tax-reporting-unit) | 语义替代 | [M11](modules/m11.md) | 扣缴/报告登记与用途，按中国申报责任表达 | [m11](diagrams/m11.svg)：REG |
| C032 | Payroll Relationship<br>[Payroll Relationship（薪资关系）](17-terminology-glossary.md#term-m10-payroll-relationship) | 语义替代 | [M10](modules/m10.md)、[M11](modules/m11.md) | PayrollContext连接Person与薪资责任主体；PayrollMembership连接任职、计薪上下文及工资定义；税务/参缴累计独立 | [c02-payroll-concepts](diagrams/c02-payroll-concepts.svg)：PERSON, CTX, MEM |
| C033 | Country Extension<br>[Country Extension（国家扩展）](17-terminology-glossary.md#term-x02-country-extension) | 语义替代 | [X02](modules/x02.md)、[M11](modules/m11.md)、[M12](modules/m12.md) | 中国政策包、地区适用和领域规则版本 | [c02-payroll-concepts](diagrams/c02-payroll-concepts.svg)：LAW |
| C034 | Organization Calculation Card<br>[Organization Calculation Card（组织计算卡）](17-terminology-glossary.md#term-m11-organization-calculation-card) | 语义替代 | [M11](modules/m11.md)、[M12](modules/m12.md) | 主体登记、资格/扣缴/参缴事实及规则引用，不照搬卡片形状 | [m11](diagrams/m11.svg)：REG, ID, POL；[m12](diagrams/m12.svg)：P, BASE |
| C035 | Payroll Definition<br>[PayrollDefinition（工资核算配置）](17-terminology-glossary.md#term-m10-payrolldefinition) | 采纳 | [M10](modules/m10.md) | 工资定义及周期、所属期和运行规则 | [c02-payroll-concepts](diagrams/c02-payroll-concepts.svg)：DEF, RUN |
| C036 | Time Definition<br>[Time Definition（计薪日期与期间规则）](17-terminology-glossary.md#term-m10-time-definition) | 采纳 | [M10](modules/m10.md)、[X01](modules/x01.md) | 计算日期和期间语义；与实际工时不同 | [c02-payroll-concepts](diagrams/c02-payroll-concepts.svg)：DEF, RUN |
| C037 | Consolidation Group<br>[Consolidation Group（工资归并组）](17-terminology-glossary.md#term-m10-consolidation-group) | 语义替代 | [M10](modules/m10.md) | 运行/报告归并范围，不改变事实身份 | [c02-payroll-concepts](diagrams/c02-payroll-concepts.svg)：DEF, RUN |
| C038 | Payroll Period / Calendar<br>[PayrollPeriod（工资期间）](17-terminology-glossary.md#term-m10-payrollperiod) / [PayrollCalendar（工资日历）](17-terminology-glossary.md#term-m10-payrollcalendar) | 采纳 | [M10](modules/m10.md) | 期间、计薪日、支付日和日历 | [c02-payroll-concepts](diagrams/c02-payroll-concepts.svg)：DEF, RUN |
| C039 | Run Type<br>[RunType（工资核算类型）](17-terminology-glossary.md#term-m10-runtype) | 采纳 | [M10](modules/m10.md) | 普通、非周期等运行类型及允许处理 | [c02-payroll-concepts](diagrams/c02-payroll-concepts.svg)：DEF, RUN |
| C040 | Element<br>[Element（工资项目）](17-terminology-glossary.md#term-m10-element) | 采纳 | [M10](modules/m10.md) | 有身份的工资项目及输入输出语义 | [c02-payroll-concepts](diagrams/c02-payroll-concepts.svg)：EL |
| C041 | Element Classification<br>[Element Classification（工资项目分类）](17-terminology-glossary.md#term-m10-element-classification) | 采纳 | [M10](modules/m10.md)、[M11](modules/m11.md) | 工资项目分类与税务所得性质分别映射 | [c02-payroll-concepts](diagrams/c02-payroll-concepts.svg)：EL |
| C042 | Rate<br>[Rate（计薪标准）](17-terminology-glossary.md#term-m09-rate) | 采纳 | [M09](modules/m09.md)、[M10](modules/m10.md) | 费率单位、频率、精度及版本 | [c02-payroll-concepts](diagrams/c02-payroll-concepts.svg)：RULE |
| C043 | Calculation Value<br>[Calculation Value（计薪参数值）](17-terminology-glossary.md#term-m10-calculation-value) | 采纳 | [M10](modules/m10.md) | 计算参数、金额/数量和适用条件 | [c02-payroll-concepts](diagrams/c02-payroll-concepts.svg)：RULE |
| C044 | Fast Formula<br>[Fast Formula（快速公式）](17-terminology-glossary.md#term-m10-fast-formula) | 语义替代 | [M10](modules/m10.md) | 受控计算规则/表达式及版本，不要求使用Oracle语言 | [c02-payroll-concepts](diagrams/c02-payroll-concepts.svg)：RULE |
| C045 | UDT<br>[User-Defined Table（用户定义表）](17-terminology-glossary.md#term-x02-user-defined-table) | 语义替代 | [X02](modules/x02.md)、[M10](modules/m10.md) | 有主责、版本和适用条件的规则查表数据 | [c02-payroll-concepts](diagrams/c02-payroll-concepts.svg)：RULE |
| C046 | Element Entry<br>[ElementEntry（工资项目输入记录）](17-terminology-glossary.md#term-m10-elemententry) | 采纳 | [M10](modules/m10.md) | 来源唯一的项目输入及期间/实例 | [c02-payroll-concepts](diagrams/c02-payroll-concepts.svg)：INPUT |
| C047 | Assignment payroll input<br>[Assignment Payroll Input（任职计薪输入）](17-terminology-glossary.md#term-m10-assignment-payroll-input) | 采纳 | [M09](modules/m09.md)、[M10](modules/m10.md) | 任职授予和工资采用分开，含来源版本 | [c02-payroll-concepts](diagrams/c02-payroll-concepts.svg)：SOURCE, INPUT |
| C048 | Run Result<br>[PayrollResult（工资核算结果）](17-terminology-glossary.md#term-m10-payrollresult) | 采纳 | [M10](modules/m10.md) | 冻结输入下的候选及确认结果版本 | [c02-payroll-concepts](diagrams/c02-payroll-concepts.svg)：RUN |
| C049 | Balance Definition<br>[BalanceDefinition（累计指标定义）](17-terminology-glossary.md#term-m10-balancedefinition) | 采纳 | [M10](modules/m10.md)、[M11](modules/m11.md)、[M12](modules/m12.md) | 工资、税务、参缴各自指标定义和记账时点 | [c02-payroll-concepts](diagrams/c02-payroll-concepts.svg)：BD, BAL, TAX；[m10](diagrams/m10.svg)：BD, B |
| C050 | Balance Dimension / Group<br>[BalanceDimension（累计维度）](17-terminology-glossary.md#term-m10-balancedimension) / [BalanceGroup（累计指标组）](17-terminology-glossary.md#term-m10-balancegroup) | 采纳 | [M10](modules/m10.md)、[M11](modules/m11.md) | 人员、主体、年度、所得/指标等维度及分组 | [c02-payroll-concepts](diagrams/c02-payroll-concepts.svg)：BD, BAL, TAX；[m10](diagrams/m10.svg)：BD, B |
| C051 | Retro Event / Event Group<br>[Retro Event（追溯触发事件）](17-terminology-glossary.md#term-m10-retro-event) / [Event Group（追溯事件组）](17-terminology-glossary.md#term-m10-event-group) | 采纳 | [M10](modules/m10.md)、[X01](modules/x01.md) | M10 PayrollImpactPolicy版本化选择/归组来源及规则变更，运行形成影响事项和RetroCase；X01仅提供信封协议 | [m10](diagrams/m10.svg)：IMPACT, D |
| C052 | Retro Action<br>[Retro Action（追溯处理）](17-terminology-glossary.md#term-m10-retro-action) | 采纳 | [M10](modules/m10.md)、[M11](modules/m11.md)、[M12](modules/m12.md) | 按ResultSeriesHead当前完整版本求差，串行推进后继；税费各域独立更正，不按原始结果重复补差 | [m10](diagrams/m10.svg)：HEAD, D, R |
| C053 | Payment Method<br>[Payment Method（支付方式）](17-terminology-glossary.md#term-m14-payment-method) | 采纳 | [M14](modules/m14.md) | 银行、现金等方式及不同必需条件 | [m14](diagrams/m14.svg)：U, I |
| C054 | Payslip<br>[Payslip（工资单）](17-terminology-glossary.md#term-m10-payslip) | 采纳 | [M10](modules/m10.md)、[M20](modules/m20.md) | M10 PayslipPublication独立发布/定时/撤回/替代，M20受控查询；封存、通知、支付不代替发布，读取结合X03当前权限 | [c02-payroll-concepts](diagrams/c02-payroll-concepts.svg)：SLIP |
| C055 | Costing<br>[Costing（人工成本核算）](17-terminology-glossary.md#term-m15-costing) | 采纳 | [M15](modules/m15.md) | 确认成本来源、归属、分摊和差额 | [m15](diagrams/m15.svg)：C, A, M, J |
| C056 | Accounting Structure<br>[Accounting Structure（会计核算结构）](17-terminology-glossary.md#term-m15-accounting-structure) | 语义替代 | [M15](modules/m15.md) | 科目/成本中心/项目映射及会计期间 | [m15](diagrams/m15.svg)：C, A, M, J |
| C057 | Payroll Flow definition / execution<br>[PayrollFlowDefinition（工资流程定义）](17-terminology-glossary.md#term-m10-payrollflowdefinition) / [PayrollFlowInstance（工资流程实例）](17-terminology-glossary.md#term-m10-payrollflowinstance) | 采纳 | [X04](modules/x04.md)、[M10](modules/m10.md) | 任务定义与一次运行实例分开 | [m10](diagrams/m10.svg)：FLOWDEF, FLOW, R |
| C058 | Compensation<br>[Compensation（薪酬）](17-terminology-glossary.md#term-m09-compensation) | 采纳 | [M09](modules/m09.md) | 薪酬体系和授予，不等于已发工资 | [m09](diagrams/m09.svg)：P, A, D |
| C059 | Benefits<br>[Benefits（企业福利）](17-terminology-glossary.md#term-m13-benefits) | 纳入目标设计 | [M13](modules/m13.md) | 计划、资格、登记和费用；独立福利领域职责 | [m13](diagrams/m13.svg)：L, E, N, CLAIM, C |
| C060 | Payroll<br>[Payroll（工资核算）](17-terminology-glossary.md#term-m10-payroll) | 采纳 | [M10](modules/m10.md) | 准备、计算、累计、结果、追溯及下游采用 | [c02-payroll-concepts](diagrams/c02-payroll-concepts.svg)：CTX, INPUT, RUN, BAL |
| C061 | Social insurance / Housing fund<br>[Social Insurance（社会保险）](17-terminology-glossary.md#term-m12-social-insurance) / [Housing Fund（住房公积金）](17-terminology-glossary.md#term-m12-housing-fund) | 语义替代 | [M12](modules/m12.md) | 中国独立参缴义务、申报、扣收和实缴 | [m12](diagrams/m12.svg)：P, O, F, E, S |

## 4. 排班、实际工时与缺勤

| 条目 | 参考概念 | 目标处置 | 责任模块及聚合入口 | 目标对象或边界 | 图与节点定位 |
| --- | --- | --- | --- | --- | --- |
| C062 | Work Pattern<br>[WorkPattern（排班循环模式）](17-terminology-glossary.md#term-m06-workpattern) | 采纳 | [M06](modules/m06.md) | 班次模式与周期 | [c03-time-absence-concepts](diagrams/c03-time-absence-concepts.svg)：WP, SCH |
| C063 | Shift<br>[Shift（班次）](17-terminology-glossary.md#term-m06-shift) | 采纳 | [M06](modules/m06.md) | 计划班段与跨午夜归属 | [c03-time-absence-concepts](diagrams/c03-time-absence-concepts.svg)：WP, SCH |
| C064 | Schedule<br>[Schedule（排班）](17-terminology-glossary.md#term-m06-schedule) | 采纳 | [M06](modules/m06.md) | ScheduleBook按任职统一守卫跨日发布/替换，发布版本固定实际时段和来源 | [m06](diagrams/m06.svg)：BOOK, S |
| C065 | Generation Profile<br>[ScheduleGenerationProfile（排班生成方案）](17-terminology-glossary.md#term-m06-schedulegenerationprofile) | 纳入目标设计 | [M06](modules/m06.md) | 排班生成条件与约束；生成不等于发布 | [m06](diagrams/m06.svg)：GEN, S |
| C066 | Time Category<br>[TimeCategory（工时类别）](17-terminology-glossary.md#term-m07-timecategory) | 采纳 | [M07](modules/m07.md) | M07 TimePolicy内可复用分类条件/单位与运行分类值分开，记录采用版本 | [m07](diagrams/m07.svg)：CAT, T |
| C067 | Consumer Set<br>[TimeConsumerSet（工时数据使用方集合）](17-terminology-glossary.md#term-m07-timeconsumerset) | 语义替代 | [M07](modules/m07.md)、[X05](modules/x05.md) | M07消费者选择及核准/释放规则定义与发布实例分开；消费者独占采用 | [m07](diagrams/m07.svg)：CONS, R, Y |
| C068 | Time Profile<br>[TimeProfile（工时处理方案）](17-terminology-glossary.md#term-m07-timeprofile) | 语义替代 | [M07](modules/m07.md)、[X02](modules/x02.md) | 工时录入/校验规则的日期有效适用 | [m07](diagrams/m07.svg)：PROF, D, T |
| C069 | Time Layout<br>[TimeLayout（工时录入布局）](17-terminology-glossary.md#term-m07-timelayout) | 语义替代 | [M07](modules/m07.md)、[M20](modules/m20.md) | 角色任务的录入展示，不承载另一套事实 | [m07](diagrams/m07.svg)：PROF, D, T |
| C070 | Time Rule<br>[TimeRule（工时规则）](17-terminology-glossary.md#term-m07-timerule) | 采纳 | [M07](modules/m07.md) | 验证、分类和转换规则及版本 | [m07](diagrams/m07.svg)：PROF, D, T |
| C071 | Device<br>[Device（考勤采集设备）](17-terminology-glossary.md#term-m07-device) | 采纳 | [M07](modules/m07.md)、[X05](modules/x05.md) | 时间事件来源、设备身份、序列和重复校验 | [c03-time-absence-concepts](diagrams/c03-time-absence-concepts.svg)：EVT, TR |
| C072 | Actual Time / Time Record<br>[TimeRecord（实际工时记录）](17-terminology-glossary.md#term-m07-timerecord) | 采纳 | [M07](modules/m07.md) | 原始时间与核对结果分别保留 | [c03-time-absence-concepts](diagrams/c03-time-absence-concepts.svg)：EVT, TR |
| C073 | Time validation / confirmation<br>[Time Validation（工时校验）](17-terminology-glossary.md#term-m07-time-validation) / [Time Confirmation（工时确认）](17-terminology-glossary.md#term-m07-time-confirmation) | 语义替代 | [M07](modules/m07.md)、[X04](modules/x04.md) | 工时确认责任、证据及DecisionRecord；所需核验/审批由WorkTime政策确定 | [m07](diagrams/m07.svg)：D；[c05-reference-access-concepts](diagrams/c05-reference-access-concepts.svg)：REQ, CMD |
| C074 | Time distribution<br>[Time Distribution（工时数据分发）](17-terminology-glossary.md#term-m07-time-distribution) | 采纳 | [M07](modules/m07.md)、[M10](modules/m10.md) | 来源版本和唯一输入路径；月结不重复加金额 | [c03-time-absence-concepts](diagrams/c03-time-absence-concepts.svg)：TR, CLOSE, ADOPT |
| C075 | Absence Type<br>[AbsenceType（缺勤类型）](17-terminology-glossary.md#term-m08-absencetype) | 采纳 | [M08](modules/m08.md) | AbsenceCatalog中假种定义及版本，独立引用计划与允许原因 | [m08](diagrams/m08.svg)：CAT, P, R |
| C076 | Absence Reason<br>[AbsenceReason（缺勤原因）](17-terminology-glossary.md#term-m08-absencereason) | 采纳 | [M08](modules/m08.md) | 可复用原因定义与请求原因采用分开，多个假种显式关联 | [m08](diagrams/m08.svg)：CAT, R |
| C077 | Absence Plan<br>[AbsencePlan（休假与缺勤方案）](17-terminology-glossary.md#term-m08-absenceplan) | 采纳 | [M08](modules/m08.md) | 计划定义余额型、资格待遇或无权益模式；个人参与与资格采用独立 | [m08](diagrams/m08.svg)：P, ENR, B |
| C078 | Absence Category<br>[AbsenceCategory（缺勤分类）](17-terminology-glossary.md#term-m08-absencecategory) | 采纳 | [M08](modules/m08.md) | 目录中的假种分类，不替代计划规则或个人评估 | [m08](diagrams/m08.svg)：CAT, P |
| C079 | Eligibility<br>[Eligibility（参与资格）](17-terminology-glossary.md#term-x02-eligibility) | 采纳 | [M08](modules/m08.md)、[M12](modules/m12.md)、[M13](modules/m13.md) | 领域各自拥有资格结果，X02提供适用政策合同 | [c03-time-absence-concepts](diagrams/c03-time-absence-concepts.svg)：AP, REQ |
| C080 | Entitlement / Balance<br>[Entitlement（休假权益）](17-terminology-glossary.md#term-m08-entitlement) / [AbsenceBalance（可用休假余额）](17-terminology-glossary.md#term-m08-absencebalance) | 采纳 | [M08](modules/m08.md) | 授予批次/有效期、消费分配、原路退回和到期/结转可追溯 | [m08](diagrams/m08.svg)：B, C |
| C081 | Absence Request / Event<br>[LeaveRequest（休假与缺勤申请）](17-terminology-glossary.md#term-m08-leaverequest) / [AbsenceEvent（生效缺勤记录）](17-terminology-glossary.md#term-m08-absenceevent) | 采纳 | [M08](modules/m08.md) | 请求与确认缺勤分别成立，所有覆盖任职/计划由共同时间线守卫 | [m08](diagrams/m08.svg)：R, TL |
| C082 | Cancellation / Correction<br>[AbsenceCancellation（缺勤取消）](17-terminology-glossary.md#term-m08-absencecancellation) / [AbsenceCorrection（缺勤更正）](17-terminology-glossary.md#term-m08-absencecorrection) | 采纳 | [M08](modules/m08.md)、[M10](modules/m10.md) | 原消费分配反向及已到期保护；已采用工资经后继差额，不改历史 | [m08](diagrams/m08.svg)：B, C, TL, R |

## 5. 招聘、人才与学习

| 条目 | 参考概念 | 目标处置 | 责任模块及聚合入口 | 目标对象或边界 | 图与节点定位 |
| --- | --- | --- | --- | --- | --- |
| C083 | Candidate<br>[Candidate（候选人）](17-terminology-glossary.md#term-m05-candidate) | 纳入目标设计 | [M05](modules/m05.md) | 候选身份与Person显式关联 | [m05](diagrams/m05.svg)：C, PER |
| C084 | Application<br>[Application（求职申请）](17-terminology-glossary.md#term-m05-application) | 纳入目标设计 | [M05](modules/m05.md) | 同一候选可有多个需求申请 | [m05](diagrams/m05.svg)：C, A, R |
| C085 | Recruiting Process Configuration<br>[RecruitingProcessDefinition（招聘流程定义）](17-terminology-glossary.md#term-m05-recruitingprocessdefinition) | 纳入目标设计 | [M05](modules/m05.md) | 招聘阶段/评价过程模板与实际申请状态分开 | [m05](diagrams/m05.svg)：PROC, A |
| C086 | Offer<br>[Offer（录用方案）](17-terminology-glossary.md#term-m05-offer) | 纳入目标设计 | [M05](modules/m05.md) | 条款版本、接受/撤回及入职交接 | [m05](diagrams/m05.svg)：A, O |
| C087 | Hire conversion<br>[Hire Conversion（录用转入职）](17-terminology-glossary.md#term-m05-hire-conversion) | 采纳并扩展 | [M05](modules/m05.md)、[M01](modules/m01.md)、[M04](modules/m04.md) | 身份核对后建立用工；预留转化不双占 | [m05](diagrams/m05.svg)：H, PER, STAFF, CAP |
| C088 | Goal Library<br>[Goal Library（目标库）](17-terminology-glossary.md#term-m17-goal-library) | 纳入目标设计 | [M17](modules/m17.md) | 绩效目标定义与个人目标/进度分开；发展目标库归M18 | [m17](diagrams/m17.svg)：G, GA |
| C089 | Goal Plan<br>[GoalPlan（绩效目标计划）](17-terminology-glossary.md#term-m17-goalplan) | 纳入目标设计 | [M17](modules/m17.md) | 版本化GoalPlan配置和个人GoalAssignment实例；个人目标供绩效显式采用 | [m17](diagrams/m17.svg)：G, GA, D |
| C090 | Review Period<br>[ReviewPeriod（绩效考核周期）](17-terminology-glossary.md#term-m17-reviewperiod) | 采纳并扩展 | [M17](modules/m17.md) | 考核周期及任职分段 | [m17](diagrams/m17.svg)：T, H, D |
| C091 | Performance Template<br>[PerformanceTemplate（绩效模板）](17-terminology-glossary.md#term-m17-performancetemplate) | 纳入目标设计 | [M17](modules/m17.md) | 版本化模板/章节、资格及参与规则，引用流程定义和文档类型 | [m17](diagrams/m17.svg)：T, PROC, D |
| C092 | Performance Document<br>[PerformanceDocument（绩效评价文档）](17-terminology-glossary.md#term-m17-performancedocument) | 采纳并扩展 | [M17](modules/m17.md) | 个人周期文档，按多任职期间分配评价责任 | [m17](diagrams/m17.svg)：D, R |
| C093 | Process Flow<br>[PerformanceProcessDefinition（绩效流程定义）](17-terminology-glossary.md#term-m17-performanceprocessdefinition) | 语义替代 | [M17](modules/m17.md)、[X04](modules/x04.md) | M17流程定义与文档阶段实例；X04只提供分派运行，不拥有决定和发布 | [m17](diagrams/m17.svg)：PROC, T, D, R |
| C094 | Feedback / Appeal<br>[Feedback（绩效反馈）](17-terminology-glossary.md#term-m17-feedback) / [Appeal（绩效申诉）](17-terminology-glossary.md#term-m17-appeal) | 采纳 | [M17](modules/m17.md) | 反馈、申诉、依据及已发布结果更正 | [m17](diagrams/m17.svg)：F, R |
| C095 | Content Type<br>[ContentType（人才档案内容类型）](17-terminology-glossary.md#term-m16-contenttype) | 采纳 | [M16](modules/m16.md) | 人才内容类型定义 | [m16](diagrams/m16.svg)：C |
| C096 | Content Item<br>[ContentItem（人才档案内容项）](17-terminology-glossary.md#term-m16-contentitem) | 采纳 | [M16](modules/m16.md) | 技能/资格项目身份及语义版本 | [m16](diagrams/m16.svg)：C |
| C097 | Profile Type<br>[ProfileType（档案类型）](17-terminology-glossary.md#term-m16-profiletype) | 采纳 | [M16](modules/m16.md) | 版本化ProfileType及SectionDefinition由M16发布；人员或岗位档案固定采用类型及章节版本 | [m16](diagrams/m16.svg)：PT, P, J, C；[c04-talent-concepts](diagrams/c04-talent-concepts.svg)：TYPE, PP, JP |
| C098 | Profile Section<br>[ProfileSection（档案栏目）](17-terminology-glossary.md#term-m16-profilesection) | 采纳 | [M16](modules/m16.md) | 章节结构、字段与读写权限 | [m16](diagrams/m16.svg)：P, J, C |
| C099 | Rating Model<br>[RatingModel（评分模型）](17-terminology-glossary.md#term-m16-ratingmodel) | 采纳 | [M16](modules/m16.md) | 量表等级定义，旧结果保留原尺度 | [m16](diagrams/m16.svg)：C |
| C100 | Questionnaire<br>[Questionnaire（问卷）](17-terminology-glossary.md#term-m16-questionnaire) | 纳入目标设计 | [M16](modules/m16.md) | 模板版本与个人回答分开 | [m16](diagrams/m16.svg)：Q, E |
| C101 | Catalog Item<br>[Catalog Item（学习目录项）](17-terminology-glossary.md#term-m18-catalog-item) | 纳入目标设计 | [M18](modules/m18.md) | 学习内容目录身份 | [m18](diagrams/m18.svg)：C |
| C102 | Course<br>[Course（课程）](17-terminology-glossary.md#term-m18-course) | 纳入目标设计 | [M18](modules/m18.md) | 课程定义与版本 | [m18](diagrams/m18.svg)：C |
| C103 | Learning Event / Offering<br>[LearningEvent（学习活动场次）](17-terminology-glossary.md#term-m18-learningevent) / [LearningOffering（课程开设安排）](17-terminology-glossary.md#term-m18-learningoffering) | 纳入目标设计 | [M18](modules/m18.md) | 开班容量/座位/候补由Offering共同守卫；指派要求与实际报名分开 | [m18](diagrams/m18.svg)：O, LA, E |
| C104 | Learning Path<br>[LearningPath（学习路径）](17-terminology-glossary.md#term-m18-learningpath) | 纳入目标设计 | [M18](modules/m18.md) | 学习路径前置条件、个人发展目标与行动；仅学习行动形成学习要求 | [m18](diagrams/m18.svg)：PATH, DG, P, LA, E |
| C105 | Audience<br>[LearningAudience（学习适用人群）](17-terminology-glossary.md#term-m18-learningaudience) | 纳入目标设计 | [M18](modules/m18.md)、[X03](modules/x03.md) | 受众适用与可见权限分开 | [m18](diagrams/m18.svg)：AUD, E |
| C106 | Learning Record<br>[LearningRecord（学习记录）](17-terminology-glossary.md#term-m18-learningrecord) | 纳入目标设计 | [M18](modules/m18.md) | 实际学习/完成证据，由学习证据形成确认记录 | [m18](diagrams/m18.svg)：R, Q |
| C107 | Talent Review dimension / template<br>[TalentReviewDimension（人才盘点评价维度）](17-terminology-glossary.md#term-m19-talentreviewdimension) / [TalentReviewTemplate（人才盘点模板）](17-terminology-glossary.md#term-m19-talentreviewtemplate) | 纳入目标设计 | [M19](modules/m19.md) | 盘点评价维度、模板与本次结论分开 | [m19](diagrams/m19.svg)：CFG, R |
| C108 | Talent Pool<br>[TalentPool（人才池）](17-terminology-glossary.md#term-m19-talentpool) | 纳入目标设计 | [M19](modules/m19.md) | 成员资格、入退池及历史 | [m19](diagrams/m19.svg)：T |
| C109 | Succession Plan<br>[SuccessionPlan（继任计划）](17-terminology-glossary.md#term-m19-successionplan) | 纳入目标设计 | [M19](modules/m19.md) | Job/Position/Incumbent三种计划主题；直接或人才池/招聘候选，不占编制 | [m19](diagrams/m19.svg)：P, T, S, H |
| C110 | Readiness<br>[Readiness（继任准备度）](17-terminology-glossary.md#term-m19-readiness) | 纳入目标设计 | [M19](modules/m19.md) | 准备度及依据，不等于自动任命 | [m19](diagrams/m19.svg)：S, A |

## 6. 有效日期与历史

| 条目 | 参考概念 | 目标处置 | 责任模块及聚合入口 | 目标对象或边界 | 图与节点定位 |
| --- | --- | --- | --- | --- | --- |
| C111 | Stable logical object / dated slices<br>[Identity（稳定对象标识）](17-terminology-glossary.md#term-x01-identity) / [Dated Slice（有效期切片）](17-terminology-glossary.md#term-x01-dated-slice) | 采纳 | [X01](modules/x01.md) | 逻辑身份与日期切片分开 | [e04-effective-adoption](diagrams/e04-effective-adoption.svg)：O, D1, D2 |
| C112 | Effective Date query<br>[Effective Date Query（按生效日期查询）](17-terminology-glossary.md#term-x01-effective-date-query) | 采纳 | [X01](modules/x01.md) | 业务日期选择对应有效版本 | [e04-effective-adoption](diagrams/e04-effective-adoption.svg)：O, D1, D2 |
| C113 | Update<br>[Update（有效期更新）](17-terminology-glossary.md#term-x01-update) | 采纳 | [X01](modules/x01.md) | 新增有效业务切片，不覆盖历史 | [e04-effective-adoption](diagrams/e04-effective-adoption.svg)：O, D1, D2 |
| C114 | Correction<br>[Correction（数据更正）](17-terminology-glossary.md#term-x01-correction) | 采纳 | [X01](modules/x01.md) | 同日更正保留审计修订和旧采用 | [e04-effective-adoption](diagrams/e04-effective-adoption.svg)：R1, R2, AU |
| C115 | End Date<br>[End Date（结束日期）](17-terminology-glossary.md#term-x01-end-date) | 采纳 | [X01](modules/x01.md)、[M04](modules/m04.md) | 终结有效范围，非删除身份 | [x01](diagrams/x01.svg)：K, V |
| C116 | Future slice deletion<br>[Record Deletion（删除记录）](17-terminology-glossary.md#term-x01-future-slice-deletion) | 采纳 | [X01](modules/x01.md) | 明确允许范围、引用检查及历史保护 | [x01](diagrams/x01.svg)：K, V |
| C117 | Date interval overlap<br>[Date Interval Overlap（有效期间重叠）](17-terminology-glossary.md#term-x01-date-interval-overlap) | 采纳 | [X01](modules/x01.md) | 同时间轴/作用域区间不可冲突 | [x01](diagrams/x01.svg)：K, V |
| C118 | Same-day effective sequence<br>[Effective Sequence（同日生效序号）](17-terminology-glossary.md#term-x01-effective-sequence) | 明确不采纳 | [M04](modules/m04.md)、[X01](modules/x01.md) | 选择日级任职最终状态；真实动作另记发生依据，不支持日内任职状态查询/计量，审计修订非等价替代 | [e04-effective-adoption](diagrams/e04-effective-adoption.svg)：D1, ACT, AU, N |
| C119 | Parent-child date alignment<br>[Parent-child Date Alignment（父子有效期约束）](17-terminology-glossary.md#term-x01-parent-child-date-alignment) | 采纳 | [X01](modules/x01.md)、[M04](modules/m04.md) | 主子对象适用日期和动作分别校验 | [x01](diagrams/x01.svg)：K, V |
| C120 | Audit timestamp / revision<br>[Audit Timestamp（审计记录时间）](17-terminology-glossary.md#term-x01-audit-timestamp) / [Revision（修订版本）](17-terminology-glossary.md#term-x01-revision) | 采纳 | [X01](modules/x01.md) | 系统记录顺序与真实动作发生依据分开；不能用反序补录重建业务生效先后 | [e04-effective-adoption](diagrams/e04-effective-adoption.svg)：R1, R2, AU |
| C121 | Adopted history / result version<br>[Adopted History（采用历史）](17-terminology-glossary.md#term-x01-adopted-history) / [Result Version（结果版本）](17-terminology-glossary.md#term-x01-result-version) | 采纳 | [X01](modules/x01.md)、[M10](modules/m10.md) | 已采用快照与新事实、计算结果分别版本化 | [e04-effective-adoption](diagrams/e04-effective-adoption.svg)：A, OLD, NEW |
| C122 | Concurrent editing / stale authorization<br>[Concurrent Editing（并发编辑）](17-terminology-glossary.md#term-x01-concurrent-editing) / [Stale Authorization（失效授权）](17-terminology-glossary.md#term-x01-stale-authorization) | 语义替代 | [X01](modules/x01.md)、[X03](modules/x03.md)、[X04](modules/x04.md) | expectedRevision与确切版本核准；失效决定重新评估，最终执行重核当前权限 | [c05-reference-access-concepts](diagrams/c05-reference-access-concepts.svg)：TARGET, REQ, CMD |

## 7. 适用、权限与执行许可

| 条目 | 参考概念 | 目标处置 | 责任模块及聚合入口 | 目标对象或边界 | 图与节点定位 |
| --- | --- | --- | --- | --- | --- |
| C123 | Job Role / functional permission<br>[Job Role（职务角色）](17-terminology-glossary.md#term-x03-job-role) / [Functional Permission（功能权限）](17-terminology-glossary.md#term-x03-functional-permission) | 语义替代 | [X03](modules/x03.md) | RoleDefinition职责配置引用领域动作；授予按完整角色/范围/字段/用途单元匹配 | [c05-reference-access-concepts](diagrams/c05-reference-access-concepts.svg)：ROLE, GRANT, ALLOW |
| C124 | HCM Data Role<br>[HCM Data Role（HCM数据角色）](17-terminology-glossary.md#term-x03-hcm-data-role) | 语义替代 | [X03](modules/x03.md) | RoleGrant完整绑定角色、数据范围、字段、用途及期限；禁止跨授予拼接放大 | [c05-reference-access-concepts](diagrams/c05-reference-access-concepts.svg)：ROLE, SCOPE, GRANT, ALLOW |
| C125 | Security Profile<br>[Security Profile（安全配置文件）](17-terminology-glossary.md#term-x03-security-profile) | 语义替代 | [X03](modules/x03.md) | 分对象层次的范围/字段策略与授予关联，当前权限不由历史业务日期恢复 | [c05-reference-access-concepts](diagrams/c05-reference-access-concepts.svg)：SCOPE, GRANT, TARGET |
| C126 | Person vs relationship access<br>[Person-level Access（人员级访问）](17-terminology-glossary.md#term-x03-person-level-access) / [Relationship-level Access（工作关系级访问）](17-terminology-glossary.md#term-x03-relationship-level-access) | 采纳 | [X03](modules/x03.md)、[M04](modules/m04.md) | 能见Person不等于能见全部关系、任职及工资 | [c05-reference-access-concepts](diagrams/c05-reference-access-concepts.svg)：SCOPE, TARGET |
| C127 | Business applicability<br>[Business Applicability（业务适用性）](17-terminology-glossary.md#term-x02-business-applicability) | 采纳 | [X02](modules/x02.md) | 对象、地区、日期和用途允许范围 | [c05-reference-access-concepts](diagrams/c05-reference-access-concepts.svg)：CTX, APP |
| C128 | Default selection<br>[Default Selection（默认值选取）](17-terminology-glossary.md#term-x02-default-selection) | 采纳 | [X02](modules/x02.md) | DefaultRule配置生成带版本依据的建议，执行时重新核验后由消费者显式采用 | [c05-reference-access-concepts](diagrams/c05-reference-access-concepts.svg)：DR, DEFAULT, CMD, ADOPT |
| C129 | Execution permission / approval<br>[Execution Permission（执行许可）](17-terminology-glossary.md#term-x03-execution-permission) / [Approval（审批）](17-terminology-glossary.md#term-x03-approval) | 语义替代 | [X03](modules/x03.md)、[X04](modules/x04.md) | 业务域拥有DecisionRequirement/DecisionRecord；核准、业务生效、访问授权及外部执行许可分别定义 | [c05-reference-access-concepts](diagrams/c05-reference-access-concepts.svg)：REQ, CMD, TASK |
| C130 | OAuth / authentication<br>[OAuth（OAuth授权协议）](17-terminology-glossary.md#term-x05-oauth) / [Authentication（身份认证）](17-terminology-glossary.md#term-x03-authentication) | 语义替代 | [X03](modules/x03.md)、[X05](modules/x05.md) | 认证技术身份不替代业务权限，凭据不得进业务事实 | [c05-reference-access-concepts](diagrams/c05-reference-access-concepts.svg)：PERSON, USER |
| C131 | AvailableFlag / CSV / execution_mode<br>[AvailableFlag（目录可用标志）](17-terminology-glossary.md#term-x05-availableflag) / [CSV（逗号分隔值文件）](17-terminology-glossary.md#term-x05-csv) / [execution_mode（执行模式）](17-terminology-glossary.md#term-x05-execution-mode) | 边界区分 | [X05](modules/x05.md) | FSCR执行目录元数据不直接成为China HCM业务能力；适配能力和权限仍分开 | [x05](diagrams/x05.svg)：M, I, A |

## 关键差异的判断

- Person独立登记服务身份核对与分阶段办理；这是China HCM的选择，不声称与Oracle创建路径相同。
- LDG/PSU/TRU与PayrollRelationship按法规上下文及责任边界吸收；中国法定口径仍靠官方政策和独立案例验证。
- 共享引用问题由对象适用策略及明确采用承接，不复刻SetID/TableSet。日级人事不采用业务EFFSEQ；真实动作与数据更正分别留证，审计修订不能等价替代日内任职状态，当前不支持日内状态查询/计量。
- 招聘、人才、学习、福利和员工服务均纳入目标设计，设计完整性不受原型阶段限制。
- 图节点定位检查证明引用存在；关键关系、语义替代和主责正确性仍需独立人工评审。
- 技能目录、执行模式和页面占位不等于业务模块实现或正式接口可用。
