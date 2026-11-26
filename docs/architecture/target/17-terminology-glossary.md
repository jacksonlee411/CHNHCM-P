# China HCM 术语、中文名称与定义

[返回总览](README.md) · [概念映射](07-concept-mapping.md) · [聚合目录](14-aggregate-catalog.md)

本表是 China HCM 当前目标设计的统一术语入口，按20个业务模块和6项横向能力分组。中文名称为本项目首选用语；定义说明业务含义及相邻概念边界，详细规则、生命周期和事务约束以模块说明及[DDD契约](10-ddd-and-state-contracts.md)为准。

当前共 **449 个独立词条**，逐项关联[概念映射](07-concept-mapping.md)的 **131 条参考条目**及[聚合目录](14-aggregate-catalog.md)的 **154 组对象**。组合对象分别链接各词条，同一来源概念被多模块使用时不按使用方重复定义。词条数不是聚合根数或功能完成率。

范围包含当前参考概念、模块对象及理解设计所需的关键术语；不把每个命令名、字段名和状态码都当作业务概念，命令导航见[16](16-command-acceptance-index.md)。标记“参考概念”的条目用于解释对标来源，采纳、替代或不采纳以07的逐项处置为准，不能据词条存在推定目标实现。

同名词按语境解释：M03的Job是职务，X04的Job是技术作业；M04的Assignment是任职，M18的LearningAssignment是学习任务；人才档案与工时处理方案分别定义。别名含原文简称、历史称呼及检索名称，不表示所有别名都适合在新正文继续使用。

## 分领域索引

| 领域 | 词条数 |
| --- | --- |
| [M01 人员身份与个人资料](#domain-m01) | 8 |
| [M02 企业、组织与地点](#domain-m02) | 20 |
| [M03 职务、职级、岗位与编制](#domain-m03) | 14 |
| [M04 用工、任职与劳动合同](#domain-m04) | 21 |
| [M05 招聘、录用与入职协同](#domain-m05) | 12 |
| [M06 排班与工时制度](#domain-m06) | 10 |
| [M07 考勤与实际工时](#domain-m07) | 18 |
| [M08 休假、缺勤与权益账户](#domain-m08) | 22 |
| [M09 薪酬体系、定薪与激励](#domain-m09) | 11 |
| [M10 工资核算、累计与追溯](#domain-m10) | 41 |
| [M11 个人所得税扣缴与申报](#domain-m11) | 19 |
| [M12 社会保险与住房公积金](#domain-m12) | 18 |
| [M13 企业福利](#domain-m13) | 11 |
| [M14 工资支付与收付款账户](#domain-m14) | 19 |
| [M15 人工成本与财务衔接](#domain-m15) | 12 |
| [M16 人才档案、技能与资格](#domain-m16) | 18 |
| [M17 目标与绩效](#domain-m17) | 17 |
| [M18 学习与发展](#domain-m18) | 19 |
| [M19 人才盘点、人才池与继任](#domain-m19) | 10 |
| [M20 员工与经理服务及HR任务](#domain-m20) | 10 |
| [X01 对象标识、有效期、修订与采用历史](#domain-x01) | 46 |
| [X02 共享配置、适用范围、默认值与政策版本](#domain-x02) | 18 |
| [X03 身份、权限与隐私](#domain-x03) | 21 |
| [X04 任务、作业与人工核对](#domain-x04) | 11 |
| [X05 导入导出与外部集成](#domain-x05) | 13 |
| [X06 报表、指标与事实追溯](#domain-x06) | 10 |

<a id="domain-m01"></a>

## M01 人员身份与个人资料

业务依据与详细契约：[M01模块说明](modules/m01.md)。

| 术语 | 中文名称 | 定义 |
| --- | --- | --- |
| <a id="term-m01-person"></a>Person | 人员 | 在客户实例中具有稳定身份的自然人；可先登记再建立工作关系，跨单位、多段关系和离职再聘仍关联同一人。 |
| <a id="term-m01-personalprofile"></a>PersonalProfile | 个人资料 | 人员身份、联系方式等按目的维护的资料集合；不等同于记录技能和资格的人才档案。 |
| <a id="term-m01-contactrelationship"></a>ContactRelationship | 联系人关系 | 人员与紧急联系人、家属等联系对象之间有类型和期限的关系，可关联已核对的人员身份。 |
| <a id="term-m01-bankaccountidentity"></a>BankAccountIdentity<br>别名／原称：个人银行账户身份 | 个人银行账户 | 归属于个人、具有稳定标识的银行账户及资料；工资收款用途、分配和付款快照由支付域管理。 |
| <a id="term-m01-personnumberalias"></a>PersonNumberAlias | 人员编号历史别名 | 人员改号时登记并永久保留的旧编号与变更记录，旧号持续解析到同一personId且不能另分配；编号历史通过确切登记来源版本追溯，不建立businessAt业务生效切片或系统时间截点查询。 依据：[ddd-contracts](#source-ddd-contracts)。 |
| <a id="term-m01-name-style"></a>Name Style<br>别名／原称：姓名样式 | 姓名结构 | 姓名的组成字段及结构约定，例如姓、名和本地文字字段；不同于姓名显示格式。 |
| <a id="term-m01-name-format"></a>Name Format | 姓名显示格式 | 按用途排列和显示姓名字段的规则；显示名称不作为人员唯一身份。 |
| <a id="term-m01-employee-number"></a>Employee Number | 员工编号 | 企业为员工分配的业务编号，供人事办理和检索使用；与内部稳定人员标识分别维护，改号保留旧号关系，不以某家企业的位数或流水规则作为通用规则。 依据：[cr2018](#source-cr2018)。 原文定位：华润PDF第243页（DS_CRH0_HR_010001）。 |

<a id="domain-m02"></a>

## M02 企业、组织与地点

业务依据与详细契约：[M02模块说明](modules/m02.md)。

| 术语 | 中文名称 | 定义 |
| --- | --- | --- |
| <a id="term-m02-enterprise"></a>Enterprise | 企业 | 本设计中客户实例及其企业总体上下文，容纳多个组织和法律实体。 |
| <a id="term-m02-legal-entity"></a>Legal Entity | 法律实体 | 具有可识别法律身份并承担相应权利义务的组织实体；本项目不将其一概限定为独立法人公司，分支机构按登记、资格与适用依据建模。用人、用工、薪资、付款等允许用途分别声明。 |
| <a id="term-m02-legal-employer"></a>Legal Employer<br>别名／原称：法定雇主、合法雇主 | 用人单位 | 在中国劳动关系语境中，与劳动者建立劳动关系并承担相应用人责任的组织主体。劳务派遣中的派遣单位与接受派遣的用工单位分别识别；符合条件的分支机构也可作为用人单位。<br>来源说明：首选中文是本项目大陆劳动关系语境的选词；保留 Legal Employer 英文映射和原词条 ID。Oracle 在工作关系层记录此字段，但其人员类型还包括非员工及非工作关联，不能仅凭该字段认定中国劳动关系中的用人单位；非劳动关联使用对应角色。 依据：[cn-labor-contract-law-009](#source-cn-labor-contract-law-009)、[cn-labor-contract-regulations-009](#source-cn-labor-contract-regulations-009)、[oracle-legal-employer-009](#source-oracle-legal-employer-009)、[oracle-work-relationships-009](#source-oracle-work-relationships-009)、[oracle-assignments-zh-009](#source-oracle-assignments-zh-009)。 |
| <a id="term-m02-business-unit"></a>Business Unit<br>别名／原称：业务单位、业务部门 | 业务单元 | 业务处理和参考数据适用范围的组织上下文，可与法律实体、部门采用不同划分。 |
| <a id="term-m02-department"></a>Department | 部门 | 用于组织管理和人员归属的日期有效组织单位，与业务单元和法律实体分别建模。 |
| <a id="term-m02-location"></a>Location | 地点 | 带地址和适用范围的实际工作或组织地点；地点本身不自动决定税务或参缴责任。 |
| <a id="term-m02-organizationobject"></a>OrganizationObject | 组织对象 | 组织域统一管理企业、业务单元、部门、地点及法律实体身份的目标对象，具体职责由对象类型确定。 |
| <a id="term-m02-legalentityuse"></a>LegalEntityUse | 法律实体用途 | 声明组织法律身份在给定期间可承担的用人、派遣接收用工、薪资、扣缴、参缴、付款等用途及资格；不同用途分别核定。允许用途不等于与具体人员已建立劳动关系，运营登记和账户归消费域。 |
| <a id="term-m02-organizationhierarchy"></a>OrganizationHierarchy<br>别名／原称：组织用途树 | 组织架构 | 按管理或报告等用途组织节点的层级结构，拥有独立身份和有效版本；可有多套用途，不限于行政组织架构。 |
| <a id="term-m02-hierarchyversion"></a>HierarchyVersion<br>别名／原称：Tree Version、组织树版本 | 组织架构版本 | 某一用途的组织架构在指定期间采用的节点和父子关系版本；历史引用保留当时版本。 |
| <a id="term-m02-tree"></a>Tree | 树 | 以节点及父子关系表达层级的结构；本设计分别维护组织用途树和岗位层级。 |
| <a id="term-m02-organizationpublication"></a>OrganizationPublication | 组织发布记录 | 组织域对外发布某一组织对象或树版本的记录；发布本身不修改下游已采用事实。 |
| <a id="term-m02-organizationadoptionreceiptindex"></a>OrganizationAdoptionReceiptIndex<br>别名／原称：AdoptionReceiptIndex | 组织采用回执索引 | 按消费者回执重建的组织版本采用索引，用于查找影响；实际采用记录归消费者。 |
| <a id="term-m02-contextbinding"></a>ContextBinding | 组织上下文绑定 | 将BU、地点等上下文与具体组织及允许用途按有效期关联；用人、用工、付款等角色分别解析，不从一个组织或地点引用推导全部责任。 |
| <a id="term-m02-payroll-statutory-unit"></a>Payroll Statutory Unit<br>别名／原称：PSU | 薪资法定单位 | **参考概念。** Oracle 参考概念，界定薪资法定处理和报告责任；本设计通过法律实体用途及计薪上下文承接，不直接复制此对象。 依据：[reference-model](#source-reference-model)。 |
| <a id="term-m02-unified-social-credit-code"></a>Unified Social Credit Code | 统一社会信用代码 | 组织登记所使用的一类外部识别代码；与内部组织身份分开，适用主体及校验规则按相应登记标准确定，不要求每种组织对象都具有该代码。 依据：[cr2018](#source-cr2018)。 原文定位：华润PDF第5页（DS_CRH0_PB_030002）。 |
| <a id="term-m02-labor-host-unit"></a>Labor Host Unit<br>别名／原称：派遣接收单位 | 用工单位 | 在劳务派遣语境中接受被派遣劳动者工作的单位，与作为用人单位的派遣单位分别识别；工作地点、系统员工类别或工资付款方不能单独决定这一角色。<br>来源说明：本词条英文为项目说明名，不宣称是 Oracle 固定字段或官方译名；其他服务接收关系按其实际性质命名。 依据：[cn-labor-contract-law-009](#source-cn-labor-contract-law-009)。 |
| <a id="term-m02-associated-organization"></a>Associated Organization<br>别名／原称：关系关联组织 | 关联单位 | 某段工作关系或非工作关联所引用的组织；具体可为委托方、服务接收方等。该中性关联本身不表示用人、派遣接收、付款或扣缴责任。<br>来源说明：本项目用语；Employment.entityId 引用其 M02 组织身份，角色和依据另行明确。 |
| <a id="term-m02-business-unit-reactivation"></a>Business Unit Reactivation | 重新启用 | BU在已记录的停用期间之后发生恢复启用的业务变化；保留BU身份、编码及此前停用期间，新增有独立完整资料快照的重新启用记录。<br>来源说明：业务类型REACTIVATE、记录处理ADD，过去/当天/未来按完整候选合法状态办理。022允许在已有后继时与停用联合补入完整期间，独立资料快照不自动跟随前序更正；用户须明确原因与资料。沿用BU身份，不自动恢复人员、岗位、权限或采用；英文为项目说明名，非运行完成声明。 依据：[chnhcm-m02-terms-011](#source-chnhcm-m02-terms-011)、[chnhcm-bu-lifecycle-021](#source-chnhcm-bu-lifecycle-021)、[chnhcm-joint-timeline-022](#source-chnhcm-joint-timeline-022)。 |
| <a id="term-m02-timeline-revision"></a>Joint Timeline Revision | 联合修订 | 在同一BU、同一当前来源上显式列出有限新增、更新和删除项，按最终候选时间线与全部守卫统一校验并一次保存；不发布中间状态，不隐式级联修改。<br>来源说明：覆盖关联启停、固定首条历史补全及分类纠错；已有独立快照未选字段保持，一份新来源和逐项前后证据。限制于单BU固定动作，不是跨BU批处理、脚本或通用导入平台。英文为项目说明名，合同采纳不等于运行能力。 依据：[chnhcm-m02-terms-011](#source-chnhcm-m02-terms-011)、[chnhcm-joint-timeline-022](#source-chnhcm-joint-timeline-022)。 |

<a id="domain-m03"></a>

## M03 职务、职级、岗位与编制

业务依据与详细契约：[M03模块说明](modules/m03.md)。

| 术语 | 中文名称 | 定义 |
| --- | --- | --- |
| <a id="term-m03-job"></a>Job | 职务 | 对工作职责和任职要求的分类定义，可被多个岗位及任职引用，本身不表示人员或编制占用。 |
| <a id="term-m03-jobfamily"></a>JobFamily<br>别名／原称：Job Family、职务族、职族、职系 | 职务系列 | 按相关职责、资格或专业领域归类的一组职务，用于职业与工作结构管理。 依据：[cn-analytics](#source-cn-analytics)。 |
| <a id="term-m03-grade"></a>Grade | 职级 | 表示工作或人员等级的结构定义，可关联薪酬标准；职级身份与具体薪酬金额分别维护。 |
| <a id="term-m03-gradeladder"></a>GradeLadder<br>别名／原称：Grade Ladder、Ladder | 职级阶梯 | 按进阶关系组织职级及级内档次的结构；维护结构不等于执行人员晋级。 |
| <a id="term-m03-position"></a>Position<br>别名／原称：职位、独立席位 | 岗位 | 具有独立身份的组织岗位，关联职务与部门，可单独核编，空缺时仍然存在。 |
| <a id="term-m03-workforceplan"></a>WorkforcePlan | 人员编制计划 | 某期间的人员目标和预算。管理员可按业务日明确采用相应版本，调整核定编制。 |
| <a id="term-m03-positioncapacity"></a>StaffingQuota | 编制额度 | M03 按生效日期维护的部门核定总额或岗位定额，单位为非负整数名额。已核部门的待分配编制为总额减当日直属启用岗位的定额合计。 |
| <a id="term-m03-positionreservation"></a>PositionReservation<br>别名／原称：Reservation | 招聘编制预留 | M05 为招聘保留的单人名额，有效期间和状态由招聘记录。M03 据此计算可用数。经录用入职时按原接受版本转换。 |
| <a id="term-m03-positionoccupancy"></a>PositionOccupancy<br>别名／原称：Occupancy | 任职占编 | M04 的有效任职按已采用规则使用一个岗位名额，自终结生效日起释放。M03 按其期间计算岗位用量。 |
| <a id="term-m03-positionhierarchy"></a>PositionHierarchy<br>别名／原称：Position hierarchy、岗位层级 | 岗位层级关系 | 岗位间按生效日期维护的管理汇报关系；与员工之间实际汇报关系分别维护。 |
| <a id="term-m03-positionhierarchyversion"></a>PositionHierarchyVersion<br>别名／原称：HierarchyVersion、岗位层级版本 | 岗位层级关系版本 | 某岗位层级关系在指定期间的节点和父子边快照，消费者保存所采用版本。 |
| <a id="term-m03-staffingpolicy"></a>StaffingPolicy<br>别名／原称：任职与编制管理政策 | 任职与编制管理规则 | 按企业、人员类别、用途及有效期间决定是否须有岗位、是否占编的规则。 |
| <a id="term-m03-positionmanaged"></a>PositionManaged | 按岗位管理 | 要求任职关联有效岗位的管理模式；是否占编仍由人员类别和用途规则决定。 |
| <a id="term-m03-jobonly"></a>JobOnly | 按职务管理 | 任职引用职务、组织等必要信息而不引用岗位的管理模式，不产生虚构岗位占用。 |

<a id="domain-m04"></a>

## M04 用工、任职与劳动合同

业务依据与详细契约：[M04模块说明](modules/m04.md)。

| 术语 | 中文名称 | 定义 |
| --- | --- | --- |
| <a id="term-m04-work-relationship"></a>Work Relationship | 工作关系 | 人员与某组织之间按类型记录的一段系统关系，可承载劳动关系、非劳动工作关系及非工作关联；本设计用 Employment 表达。劳动关系识别用人单位，其他关系按实际性质说明关联单位与责任，不一概等同于劳动合同。 依据：[oracle-work-relationships-009](#source-oracle-work-relationships-009)、[cn-labor-contract-law-009](#source-cn-labor-contract-law-009)。 |
| <a id="term-m04-employment"></a>Employment<br>别名／原称：用工关系 | 工作关系记录 | 本设计保存一段工作关系的对象，具有起止日期、人员类型及明确关系性质。entityId 引用关联组织，不自动表示用人、用工、付款或扣缴角色；具体角色按事实分别记录或引用，关系变化及再聘保留旧关系和采用证据。<br>来源说明：任职要求按关系类型定义：实际入职或派驻Hire须建首任职；明确的非工作关联通过M04受控动作登记，可没有任职，按真实关联登记。 |
| <a id="term-m04-assignment"></a>Assignment | 任职 | 人员在一段工作关系下承担工作的记录，引用职务、岗位、部门、经理及工时等信息，具有独立身份和有效期间。 依据：[assignments](#source-assignments)。 |
| <a id="term-m04-assignmentrecord"></a>AssignmentRecord | 任职有效期记录 | 任职在一个有效日期段内的业务状态及引用；系统修订记录与业务有效期间分开。 |
| <a id="term-m04-personcontract"></a>PersonContract<br>别名／原称：Contract | 人员合同 | 与用工关系关联的合同身份、条款及期限，续订和更正保留前后继；合同状态与任职状态分开。 |
| <a id="term-m04-contractprocess"></a>ContractProcess<br>别名／原称：Process、合同办理过程 | 合同办理记录 | 围绕签订、续订或结束合同的一次办理及结果记录，不替代合同本身。 |
| <a id="term-m04-primary-work-relationship"></a>Primary Work Relationship<br>别名／原称：Primary relationship、主关系 | 主要工作关系 | 按业务日期和用途从人员的工作关系中选出的主要关系，不直接决定所有税费或数据权限。 |
| <a id="term-m04-primary-assignment"></a>Primary Assignment<br>别名／原称：Primary assignment、主任职 | 主要任职 | 一段工作关系中按日期和用途选定的主要任职；人员总体主要任职需结合主要工作关系确定。 依据：[assignments](#source-assignments)。 |
| <a id="term-m04-overall-primary-assignment"></a>Overall Primary Assignment | 人员总体主要任职 | 由人员的主要工作关系及该关系内主要任职组合得到的单值任职视图，不另建独立维护事实。 依据：[assignments](#source-assignments)。 |
| <a id="term-m04-primaryselection"></a>PrimarySelection | 主要关系与任职选择 | 分别在人员层和工作关系层记录主要项选择的目标对象，保存适用期间、用途和所选身份。 |
| <a id="term-m04-managementhome"></a>ManagementHome | 主要管理归属 | 按主要关系及任职选择形成的管理归属视图，不替代各业务域独立判断责任和适用范围。 |
| <a id="term-m04-primaryselectionpolicy"></a>PrimarySelectionPolicy<br>别名／原称：主要项选择政策 | 主要关系与任职选择规则 | 规定主要关系或任职的候选资格、唯一性和期间覆盖要求的版本化规则。 |
| <a id="term-m04-businessactionrecord"></a>BusinessActionRecord<br>别名／原称：人事业务操作记录 | 人事业务办理记录 | 一次实际人事业务办理的对象、前后版本、原因和执行依据；数据更正不冒充新的人事业务动作，也不是点击或页面操作日志。 |
| <a id="term-m04-employmentactioncatalog"></a>EmploymentActionCatalog<br>别名／原称：人事操作目录 | 人事业务配置目录 | 版本化维护人事业务类型、原因和任职状态定义及其关联的配置目录。 |
| <a id="term-m04-actiondefinition"></a>ActionDefinition<br>别名／原称：Action、人事操作定义 | 人事业务类型 | 可复用的人事业务类型及适用规则，例如入职、调动；不同于具体办理记录。 |
| <a id="term-m04-reasondefinition"></a>ReasonDefinition<br>别名／原称：Action Reason、操作原因定义 | 人事业务原因 | 与人事业务类型关联的可选原因及有效期间，用于说明该项人事业务为何发生。 |
| <a id="term-m04-assignmentstatusdefinition"></a>AssignmentStatusDefinition<br>别名／原称：Assignment Status | 任职状态定义 | 任职状态代码、语义及允许转换的配置；本设计的关系、合同和计薪状态分别维护。 |
| <a id="term-m04-managerrelationship"></a>ManagerRelationship<br>别名／原称：Manager relationship、经理关系 | 汇报关系 | 以双方任职身份建立的上级汇报关系，可按直线或项目等用途区分；不要求上级职务名称是经理。 |
| <a id="term-m04-person-type"></a>Person Type<br>别名／原称：Person Types | 人员类型 | 系统用于工作关系的人员分类，例如员工或非员工；同一人的不同关系可有不同类型。人员类别与组织责任是不同维度，类别本身不能认定劳动关系或把派遣接收方变为用人单位。 |
| <a id="term-m04-hire-date"></a>Hire Date | 入职日期 | 某段工作关系开始的业务日期；加入集团、加入现单位及工龄起算日期按各自用途保存，不因名称相近合并为同一个日期。 依据：[cr2018](#source-cr2018)。 原文定位：华润PDF第243页（DS_CRH0_HR_010010）；华润PDF第270页（DS_CHA0_HR_010006）。 |
| <a id="term-m04-service-start-date"></a>Service Start Date | 工龄起算日期 | 某一工龄或年资口径采用的计量起点，须明确用途和依据；不自动等同入职、加入集团或加入现单位的日期。 依据：[cr2018](#source-cr2018)。 原文定位：华润PDF第243页（DS_CRH0_HR_010011）。 |

<a id="domain-m05"></a>

## M05 招聘、录用与入职协同

业务依据与详细契约：[M05模块说明](modules/m05.md)。

| 术语 | 中文名称 | 定义 |
| --- | --- | --- |
| <a id="term-m05-requisition"></a>Requisition | 招聘需求 | 用人方提出的招聘对象、人数、要求及组织范围，按岗位管理模式决定是否预留编制。 |
| <a id="term-m05-candidate"></a>Candidate | 候选人 | 招聘过程中识别和联系的应聘对象；经身份核对后可与人员记录关联，不因进入候选库自动建立用工。 |
| <a id="term-m05-application"></a>Application<br>别名／原称：应聘申请、需求申请 | 求职申请 | 候选人针对某一招聘需求提交并参与甄选的申请；与企业提出招聘需求分别建模。 依据：[cn-recruiting](#source-cn-recruiting)。 |
| <a id="term-m05-offer"></a>Offer | 录用方案 | 向候选人提出的录用条件及其接受、失效等状态；正式对外通知文件可称录用通知书。 依据：[cn-recruiting](#source-cn-recruiting)。 |
| <a id="term-m05-offerversion"></a>OfferVersion<br>别名／原称：录用条款 | 录用方案版本 | 某一录用方案不可变的条款版本及接受依据；接受方案不直接生成任职或已生效工资。 依据：[cn-recruiting](#source-cn-recruiting)。 |
| <a id="term-m05-hirehandoff"></a>HireHandoff<br>别名／原称：入职交接 | 入职信息交接 | 已接受的录用在资料齐备时自动形成的交接记录，关联身份、录用、适用的预留及实际用工结果。 |
| <a id="term-m05-hire-conversion"></a>Hire Conversion | 录用转入职 | 将招聘结果经身份匹配和用工办理转为人员工作关系及任职的业务过程，各域保存自己的事实。 |
| <a id="term-m05-recruitingprocessdefinition"></a>RecruitingProcessDefinition<br>别名／原称：Recruiting Process Configuration | 招聘流程定义 | 招聘阶段、评价要求和所需审批的可复用配置；申请实例固定采用版本。 |
| <a id="term-m05-interview"></a>Interview | 面试 | 针对求职申请开展的沟通评价活动，保存参与者、过程版本和评价依据。 |
| <a id="term-m05-recruitingassessment"></a>RecruitingAssessment<br>别名／原称：Assessment | 招聘评价 | 针对某份求职申请形成的评价结果和证据，保留评价者及量表版本。 |
| <a id="term-m05-recruitment-plan"></a>Recruitment Plan | 招聘计划 | 围绕特定期间的招聘目标，对岗位、数量、时间、渠道和预算等作出的活动安排；与具体招聘需求分别表达。 依据：[cr2018](#source-cr2018)。 原文定位：华润PDF第245页（DS_CRH0_HR_040002）。 |
| <a id="term-m05-recruitment-channel"></a>Recruitment Channel | 招聘渠道 | 组织与招聘对象之间开展招聘信息发布、沟通和人才获取的途径；与某一候选人、求职申请及招聘流程分别记录。 依据：[cr2018](#source-cr2018)。 原文定位：华润PDF第246页（DS_CRH0_HR_040007）。 |

<a id="domain-m06"></a>

## M06 排班与工时制度

业务依据与详细契约：[M06模块说明](modules/m06.md)。

| 术语 | 中文名称 | 定义 |
| --- | --- | --- |
| <a id="term-m06-workpattern"></a>WorkPattern<br>别名／原称：Work Pattern、工作模式 | 排班循环模式 | 由工作日、休息日及班次循环构成的计划工作模式，用于形成具体排班。 |
| <a id="term-m06-shift"></a>Shift | 班次 | 带开始结束时间、休息安排及跨午夜归属规则的计划工作时段。 |
| <a id="term-m06-schedule"></a>Schedule | 排班 | 将工作模式和班次安排到具体任职及日期的计划时间；不直接等于实际出勤。 |
| <a id="term-m06-schedulepublication"></a>SchedulePublication | 排班发布版本 | 已确认发布的任职工作时段及来源版本；替换或撤销需要保留历史。 |
| <a id="term-m06-schedulecandidate"></a>ScheduleCandidate<br>别名／原称：Candidate | 排班候选方案 | 尚未发布的排班计算或编制结果，不形成当前有效班表。 |
| <a id="term-m06-workingtimepolicy"></a>WorkingTimePolicy<br>别名／原称：工时制度政策 | 工时制度规则 | 确定工时制度适用、计划时间和相关校验的版本化规则，引用政策证据。 |
| <a id="term-m06-schedulegenerationprofile"></a>ScheduleGenerationProfile<br>别名／原称：Generation Profile | 排班生成方案 | 控制排班生成所用条件、限制和参数的配置；生成结果仍需明确发布。 |
| <a id="term-m06-workingcalendar"></a>WorkingCalendar | 工作日历 | 规定工作日、休息日及例外日期的版本化日历，供计划时间与相关计算引用。 |
| <a id="term-m06-schedulebook"></a>ScheduleBook | 任职排班登记簿 | 按任职管理有效排班发布及替换的权威登记，协调跨日排班和换班冲突。 |
| <a id="term-m06-working-time-system"></a>Working Time System<br>别名／原称：工时制度 | 工时制 | 工作时间制度的类别及其业务含义；与班次、排班循环和具体排班分开，实际适用、校验及计量由版本化工时制度规则确定。 依据：[cr2018](#source-cr2018)。 原文定位：华润PDF第244页（DS_CRH0_HR_010022）。 |

<a id="domain-m07"></a>

## M07 考勤与实际工时

业务依据与详细契约：[M07模块说明](modules/m07.md)。

| 术语 | 中文名称 | 定义 |
| --- | --- | --- |
| <a id="term-m07-timeevent"></a>TimeEvent<br>别名／原称：原始时间事件 | 原始考勤与工时事件 | 打卡或其他采集产生的带时间戳和来源身份的不可变事件；尚不表示已核实工时。 |
| <a id="term-m07-dailycheck"></a>DailyCheck | 逐日考勤核对 | 对某任职某日的原始时间事件、计划及异常进行核对的结果。 |
| <a id="term-m07-timerecord"></a>TimeRecord<br>别名／原称：Time Record、Actual Time | 实际工时记录 | 经记录或核验的实际工作时段、数量和分类；本身不等于工资金额。 |
| <a id="term-m07-monthlyclosing"></a>MonthlyClosing | 工时月结 | 固定核对范围、人员清单和源事实版本的月结记录；通过引用汇总，不重复产生工时或工资事实。 |
| <a id="term-m07-timepublication"></a>TimePublication | 工时发布记录 | 对外发布工时或月结结果版本的源域记录；使用方是否采用由使用方确认。 |
| <a id="term-m07-timeadoptionreceiptindex"></a>TimeAdoptionReceiptIndex<br>别名／原称：AdoptionReceiptIndex | 工时采用回执索引 | 依据下游回执重建的工时版本使用索引；工资域拥有真实采用记录。 |
| <a id="term-m07-timepolicy"></a>TimePolicy<br>别名／原称：工时处理政策 | 工时处理规则 | 组织分类、使用方、录入和校验等配置的版本化规则，运行结果固定采用版本。 |
| <a id="term-m07-timecategory"></a>TimeCategory<br>别名／原称：Time Category、Category | 工时类别 | 按条件和单位对工时进行归类的配置，供核算、校验和分发使用。 |
| <a id="term-m07-timeconsumerset"></a>TimeConsumerSet<br>别名／原称：Consumer Set、ConsumerSet、消费者集合 | 工时数据使用方集合 | 指定哪些下游系统接收工时，以及各自审批、释放和分发条件的配置集合。 |
| <a id="term-m07-timeprofile"></a>TimeProfile<br>别名／原称：Time Profile、Profile | 工时处理方案 | 将录入、校验、计算等工时规则按有效日期应用到指定人员或任职范围的配置。 |
| <a id="term-m07-timerule"></a>TimeRule<br>别名／原称：Time Rule、Rule | 工时规则 | 对时间数据进行验证、分类或转换的版本化规则，不等同于某次处理结果。 |
| <a id="term-m07-timelayout"></a>TimeLayout<br>别名／原称：Time Layout、Layout | 工时录入布局 | 按角色和业务场景组织工时字段的展示与录入方式，不单独拥有工时事实。 |
| <a id="term-m07-timedelivery"></a>TimeDelivery | 工时分发记录 | 向指定使用方传递某源事实版本及用途的记录，用于追踪与去重。 |
| <a id="term-m07-timesourcedefinition"></a>TimeSourceDefinition<br>别名／原称：SourceDefinition、时间数据来源定义 | 考勤与工时来源配置 | 对来源系统、设备和数据格式的配置，支持识别原始事件及重复记录。 |
| <a id="term-m07-device"></a>Device<br>别名／原称：时间采集设备 | 考勤采集设备 | 采集打卡等考勤与工时事件的终端或设备来源，需具有可识别身份和序列信息。 |
| <a id="term-m07-time-validation"></a>Time Validation | 工时校验 | 按规则检查时间数据的完整性、格式和业务约束，不表示人工已确认。 |
| <a id="term-m07-time-confirmation"></a>Time Confirmation | 工时确认 | 责任方依据核对证据确认可供后续业务使用的工时结果，所需审批由业务政策规定。 |
| <a id="term-m07-time-distribution"></a>Time Distribution | 工时数据分发 | 按使用方和用途传递已发布工时及来源身份的过程，接收成功不代表已计入工资。 |

<a id="domain-m08"></a>

## M08 休假、缺勤与权益账户

业务依据与详细契约：[M08模块说明](modules/m08.md)。

| 术语 | 中文名称 | 定义 |
| --- | --- | --- |
| <a id="term-m08-absenceplan"></a>AbsencePlan<br>别名／原称：Absence Plan、休假与缺勤计划 | 休假与缺勤方案 | 定义缺勤待遇、额度或资格规则及适用期间的配置，可采用余额管理、资格待遇或仅登记缺勤模式；不是个人休假日程。 |
| <a id="term-m08-absenceeligibility"></a>AbsenceEligibility<br>别名／原称：Eligibility | 休假与缺勤资格 | 按计划规则及人员事实判断某人是否可以参与计划或享受待遇的结果。 |
| <a id="term-m08-entitlement"></a>Entitlement | 休假权益 | 人员按适用规则享有的可休时长或缺勤待遇；存在权益不必然表示存在余额账户。 |
| <a id="term-m08-absencebalance"></a>AbsenceBalance<br>别名／原称：Balance | 可用休假余额 | 在指定期间内，按授予批次、已用及预留等计算的可使用额度，不是工资金额。 |
| <a id="term-m08-entitlementaccount"></a>EntitlementAccount<br>别名／原称：休假权益账户 | 休假额度账户 | 按人员、方案和权益范围记录额度批次及变动的账户，仅余额型方案要求建立；不代表全部休假权益。 |
| <a id="term-m08-grantlot"></a>GrantLot<br>别名／原称：权益授予批次 | 休假额度发放批次 | 一次具有来源和有效期限的休假额度发放；后续使用、取消和结转必须能追溯到批次。 |
| <a id="term-m08-entitlemententry"></a>EntitlementEntry<br>别名／原称：Entry、权益变动流水 | 休假额度变动流水 | 账户中授予、扣减、恢复、到期或结转等不可变变动记录。 |
| <a id="term-m08-consumptionallocation"></a>ConsumptionAllocation<br>别名／原称：权益使用分配 | 休假额度扣减分配 | 记录某次休假从哪些授予批次扣减多少额度，供取消时按原批次恢复。 |
| <a id="term-m08-leaverequest"></a>LeaveRequest<br>别名／原称：Absence Request、AbsenceRequest | 休假与缺勤申请 | 请求对明确任职和时段登记缺勤的业务对象，保存依据及决定；申请不等同于已生效缺勤。 |
| <a id="term-m08-absenceevent"></a>AbsenceEvent<br>别名／原称：Absence Event | 生效缺勤记录 | 已确认覆盖指定任职和时段的缺勤事实，支持取消及更正并保留历史。 |
| <a id="term-m08-absencecalculation"></a>AbsenceCalculation | 缺勤计算结果 | 依据规则和来源版本得到的缺勤数量、待遇比例和权益分配；工资金额由工资域计算。 |
| <a id="term-m08-absencecatalog"></a>AbsenceCatalog | 休假与缺勤目录 | 维护缺勤类型、原因、分类及与计划关联的版本化配置目录。 |
| <a id="term-m08-absencetype"></a>AbsenceType<br>别名／原称：Absence Type、Type、假种 | 缺勤类型 | 用于登记和处理某类休假或缺勤的定义，可关联原因及一个或多个计划；假期子集可称假种。 |
| <a id="term-m08-absencereason"></a>AbsenceReason<br>别名／原称：Absence Reason、Reason | 缺勤原因 | 解释某次休假或缺勤发生缘由的可复用定义，可供多个缺勤类型使用。 |
| <a id="term-m08-absencecategory"></a>AbsenceCategory<br>别名／原称：Absence Category、Category | 缺勤分类 | 对多个缺勤类型进行业务归组的分类，不等同于具体假期类型或个人计划。 |
| <a id="term-m08-absenceenrollment"></a>AbsenceEnrollment<br>别名／原称：缺勤计划参与记录 | 休假与缺勤方案参与记录 | 个人在某期间参加某休假与缺勤方案及所用资格结果的记录，不由入职自动推导所有参与关系。 |
| <a id="term-m08-absencetimeline"></a>AbsenceTimeline | 任职缺勤时间线 | 汇集某任职所有已确认缺勤的覆盖引用，用于跨计划校验时间重叠和兼容性。 |
| <a id="term-m08-absencecancellation"></a>AbsenceCancellation<br>别名／原称：Cancellation | 缺勤取消 | 撤销某份缺勤请求或已确认缺勤的业务操作；恢复额度须遵循原批次及有效期。 |
| <a id="term-m08-absencecorrection"></a>AbsenceCorrection<br>别名／原称：Correction | 缺勤更正 | 纠正缺勤时段、类型或结果并保留原版本及后继的操作；已封存工资另走追溯。 |
| <a id="term-m08-balancetracked"></a>BalanceTracked | 余额管理模式 | 通过权益账户记录授予和扣减，并检查可用额度的缺勤计划模式。 |
| <a id="term-m08-qualification"></a>Qualification | 资格待遇模式 | 按资格及适用期间确定缺勤待遇比例或分类的计划模式，不要求建立虚构额度余额。 |
| <a id="term-m08-noentitlement"></a>NoEntitlement<br>别名／原称：无权益模式 | 仅登记缺勤模式 | 登记和核验缺勤而不授予休假额度或待遇的方案模式；不表示人员没有休假权，是否带薪另由规则判断。 |

<a id="domain-m09"></a>

## M09 薪酬体系、定薪与激励

业务依据与详细契约：[M09模块说明](modules/m09.md)。

| 术语 | 中文名称 | 定义 |
| --- | --- | --- |
| <a id="term-m09-compensation"></a>Compensation | 薪酬 | 对基本工资、津贴、奖金及非现金待遇等报酬安排的统称，涵盖设计和核定而不限于支付。 |
| <a id="term-m09-compensationplan"></a>CompensationPlan | 薪酬计划 | 规定薪酬组成、适用对象、预算和分配规则的可复用配置。 |
| <a id="term-m09-graderate"></a>GradeRate<br>别名／原称：Grade Rate、职级费率 | 职级薪酬标准 | 与职级关联的金额或区间，包含币种、频率和有效期；用于定薪校验及预算参考。 |
| <a id="term-m09-rate"></a>Rate<br>别名／原称：费率、工资率 | 计薪标准 | 按币种、单位和频率表示的金额、单价或比例，可用于时薪、津贴等计算；具体类型应明确。 依据：[rates](#source-rates)。 |
| <a id="term-m09-compensationaward"></a>CompensationAward<br>别名／原称：个人授予、薪酬授予 | 个人薪酬核定 | 确认某人或任职的定薪、调薪或激励安排及依据的目标对象；核定不等同于工资支付。 |
| <a id="term-m09-awarddelivery"></a>AwardDelivery | 薪酬核定下发记录 | 向工资域交付适用于该任职的约定金额或小时单价、币种、频率、单位及准确授予版本。工资域依自身计薪规则保存采用、换算及期间分摊。 依据：[ddd-contracts](#source-ddd-contracts)。 |
| <a id="term-m09-budgetestimate"></a>BudgetEstimate<br>别名／原称：预算报价、职位报价 | 薪酬预算测算值 | 依据薪酬标准等估算的编制预算参考金额；不是个人已生效薪酬或外部商业报价。 |
| <a id="term-m09-basic-salary"></a>Basic Salary | 基本工资 | 按岗位或任职约定的基本报酬项目，金额、频率、包含范围及对应工资项目由有效薪酬配置确定。 依据：[cr2018](#source-cr2018)。 原文定位：华润PDF第255页（CD_CRH0_2301）。 |
| <a id="term-m09-allowances-and-subsidies"></a>Allowances and Subsidies<br>别名／原称：津贴/补贴 | 津贴补贴 | 按岗位、工作条件或企业安排等设置的津贴和补贴项目；种类、标准、适用及计薪税务处理分别采用相应规则。 依据：[cr2018](#source-cr2018)。 原文定位：华润PDF第255页（CD_CRH0_2301）。 |
| <a id="term-m09-performance-pay"></a>Performance Pay | 绩效工资 | 根据绩效规则确定的工资组成项；与绩效奖金分别定义，金额、频率和实际计薪采用明确的薪酬与工资规则。 依据：[cr2018](#source-cr2018)。 原文定位：华润PDF第255页（CD_CRH0_2301）。 |
| <a id="term-m09-performance-bonus"></a>Performance Bonus | 绩效奖金 | 根据个人、团队或组织绩效结果及约定条件核定的奖金项目；与绩效工资分别定义，适用条件、核定金额和交付工资域的版本分别保存。 依据：[cr2018](#source-cr2018)。 原文定位：华润PDF第256页（CD_CRH0_2301）。 |

<a id="domain-m10"></a>

## M10 工资核算、累计与追溯

业务依据与详细契约：[M10模块说明](modules/m10.md)。

| 术语 | 中文名称 | 定义 |
| --- | --- | --- |
| <a id="term-m10-payroll"></a>Payroll<br>别名／原称：薪资处理 | 工资核算 | 准备工资输入、计算、核对、确认累计及输出结果的业务过程，与定薪和资金支付分别负责。 |
| <a id="term-m10-payroll-relationship"></a>Payroll Relationship | 薪资关系 | **参考概念。** Oracle 参考概念，将人员与薪资法定单位及相关任职关联；本设计用计薪上下文和计薪归属承接，不据此合并税务累计。 依据：[reference-model](#source-reference-model)。 |
| <a id="term-m10-payrollcontext"></a>PayrollContext | 计薪上下文 | 连接人员与薪资责任主体的目标对象，可容纳多份计薪归属；不直接作为税务或社保累计键。 |
| <a id="term-m10-payrollmembership"></a>PayrollMembership<br>别名／原称：计薪成员 | 计薪归属 | 按日期连接任职、计薪上下文和工资定义的记录，保存所需分配事实。 |
| <a id="term-m10-payrollallocation"></a>PayrollAllocation<br>别名／原称：Allocation | 计薪分配 | 一份计薪归属中对任职计薪用途、比例或金额等的分配事实，总量由共同集合校验。 |
| <a id="term-m10-payrolldefinition"></a>PayrollDefinition<br>别名／原称：Payroll Definition、工资定义 | 工资核算配置 | 规定工资处理周期、期间、核算类型和相关日期规则的配置，不等同于整个薪酬体系。 |
| <a id="term-m10-payrollcalendar"></a>PayrollCalendar<br>别名／原称：Calendar、Payroll Calendar | 工资日历 | 规定工资期间、计薪日及支付日的日期安排，与实际工时记录分别管理。 |
| <a id="term-m10-payrollperiod"></a>PayrollPeriod<br>别名／原称：Payroll Period | 工资期间 | 某工资定义下用于组织计算及报告的一段期间，不能与支付日或税务所属期混为一谈。 |
| <a id="term-m10-time-definition"></a>Time Definition<br>别名／原称：计算时间定义 | 计薪日期与期间规则 | 规定工资计算引用何种日期和期间的定义，区别于员工实际工作时长。 |
| <a id="term-m10-consolidation-group"></a>Consolidation Group | 工资归并组 | Oracle 参考的运行或报告归并范围；本设计将归并范围放入工资定义，不改变结果身份。 |
| <a id="term-m10-runtype"></a>RunType<br>别名／原称：Run Type、工资运行类型 | 工资核算类型 | 普通或非周期等工资核算类别及其允许处理范围，是版本化配置而非某次执行状态。 |
| <a id="term-m10-element"></a>Element | 工资项目 | 有稳定身份的工资计算组成项，定义输入输出语义及分类，例如基本工资或某项扣款。 |
| <a id="term-m10-element-classification"></a>Element Classification | 工资项目分类 | 按处理性质对工资项目进行归组的定义；与税务所得性质分别映射。人工成本科目用于成本归类，二者可以映射而不合并身份。 依据：[cr2018](#source-cr2018)。 原文定位：华润PDF第245页（DS_CRH0_HR_030001）。 |
| <a id="term-m10-payrollinput"></a>PayrollInput<br>别名／原称：工资输入 | 计薪输入 | 工资域按来源、用途和期间明确采用的计算输入，保存去重身份与来源版本。 |
| <a id="term-m10-elemententry"></a>ElementEntry<br>别名／原称：Element Entry、工资项目录入项 | 工资项目输入记录 | 某工资项目在人员或任职及期间上的具体输入实例，可来自人工录入或上游采用。 |
| <a id="term-m10-assignment-payroll-input"></a>Assignment Payroll Input | 任职计薪输入 | 关联明确任职的薪酬、工时或其他工资输入，保留业务来源和采用版本。 |
| <a id="term-m10-calculationrule"></a>CalculationRule | 工资计算规则 | 工资域执行的版本化计算表达式和规则，引用参数、查表数据及计薪标准；其中ConversionProrationPolicy固定频率换算、期间分摊及舍入规则。 依据：[ddd-contracts](#source-ddd-contracts)。 |
| <a id="term-m10-conversionprorationpolicy"></a>ConversionProrationPolicy<br>别名／原称：计薪换算与分摊政策 | 计薪换算与分摊规则 | 工资计算规则中的换算与分摊规则。每个版本确定金额频率、单位换算、完整期分母、入离职及调薪分段、金额精度与舍入顺序。 依据：[ddd-contracts](#source-ddd-contracts)。 |
| <a id="term-m10-proration"></a>Proration | 期间分摊 | 将已换算到工资期的约定固定金额按有效覆盖期间分摊。按小时分摊时，覆盖小时与完整参考期分母均取同一任职日历。时薪依确认的计薪小时计算。 依据：[ddd-contracts](#source-ddd-contracts)。 |
| <a id="term-m10-calculation-value"></a>Calculation Value<br>别名／原称：计算值 | 计薪参数值 | 工资计算使用的金额、数量、系数等参数及其适用条件，与计算规则分别记录。 |
| <a id="term-m10-fast-formula"></a>Fast Formula | 快速公式 | **参考概念。** Oracle 的规则表达语言及公式机制；本设计采用受控计算规则，不要求使用该语言。 依据：[reference-model](#source-reference-model)。 |
| <a id="term-m10-payrollrun"></a>PayrollRun<br>别名／原称：工资核算运行 | 工资核算批次 | 一次以稳定runId组织固定输入范围与规则版本的工资准备、计算、核对和封存过程；技术重试不另立业务批次，结果及累计分别管理。 |
| <a id="term-m10-payrollresult"></a>PayrollResult<br>别名／原称：Result、Run Result | 工资核算结果 | 一次计算形成的工资金额及相关结果版本；确认后不可覆盖，更正通过后继版本。 |
| <a id="term-m10-payrollbalance"></a>PayrollBalance<br>别名／原称：Balance | 工资累计 | 按累计定义、维度和记账时点汇总的工资指标及历史，不等同于待支付工资余额。 |
| <a id="term-m10-balancedefinition"></a>BalanceDefinition<br>别名／原称：Balance Definition | 累计指标定义 | 规定某工资累计指标的含义、计入项目及记账时点的版本化定义。 |
| <a id="term-m10-balancedimension"></a>BalanceDimension<br>别名／原称：Balance Dimension、Dimension | 累计维度 | 规定累计按哪些人员、主体、期间或其他范围汇总的维度定义。 |
| <a id="term-m10-balancegroup"></a>BalanceGroup<br>别名／原称：Balance Group、Group | 累计指标组 | 为计算或报告组织多个累计指标的分组，不能改变各指标身份和口径。 |
| <a id="term-m10-retrocase"></a>RetroCase | 工资追溯事项 | 因历史来源或规则变化而重新评估工资、形成差额及后继的事项，保留原确认结果。 |
| <a id="term-m10-retro-event"></a>Retro Event | 追溯触发事件 | 可能影响历史工资的来源或规则变化，由影响政策判断是否进入追溯。 |
| <a id="term-m10-event-group"></a>Event Group | 追溯事件组 | 对影响工资的变化按条件归组的配置，用于确定追溯评估范围。 |
| <a id="term-m10-retro-action"></a>Retro Action | 追溯处理 | 按当前确认完整结果与更正后结果求差并形成后继的业务操作，税费域独立更正。 |
| <a id="term-m10-timeadoption"></a>TimeAdoption | 工时采用记录 | 工资域确认采用某工时来源版本及用途的记录，负责计薪去重；不归考勤域维护。 |
| <a id="term-m10-paytermadoption"></a>PayTermAdoption | 薪酬条款采用记录 | 工资域采用个人薪酬核定版本作为计薪条款的记录，区别于薪酬域下发记录。 |
| <a id="term-m10-payslip"></a>Payslip | 工资单 | 向人员展示某期间工资项目、扣款和结果的业务文件；可见、发布和实际付款状态分别确定。 |
| <a id="term-m10-payslippublication"></a>PayslipPublication<br>别名／原称：工资单发布件 | 工资单发布记录 | 工资单的草稿、定时发布、撤回及替代记录，读取时同时检查发布状态和当前权限。 |
| <a id="term-m10-payablesourcehead"></a>PayableSourceHead<br>别名／原称：工资应付来源头 | 工资应付来源版本链头 | 记录工资结果、更正和取消的连续应付变动及最新版本，用于支付域追平来源后执行。 |
| <a id="term-m10-payrollflowdefinition"></a>PayrollFlowDefinition | 工资流程定义 | 组织工资业务步骤、顺序及完成条件的可复用配置，不拥有技术重试状态。 |
| <a id="term-m10-payrollflowinstance"></a>PayrollFlowInstance<br>别名／原称：Instance | 工资流程实例 | 按某流程定义启动的一次业务运行，保存业务进度；技术执行尝试归任务设施。 |
| <a id="term-m10-payrollallocationset"></a>PayrollAllocationSet | 计薪分配集合 | 按任职及用途汇集跨上下文、跨工资定义的计薪归属，统一校验有效分配总量和重叠。 |
| <a id="term-m10-payrollimpactpolicy"></a>PayrollImpactPolicy<br>别名／原称：工资影响政策 | 工资变更影响规则 | 选择和归组来源或规则变化以识别计薪分段、追溯及复核范围的版本化规则。 |
| <a id="term-m10-resultserieshead"></a>ResultSeriesHead<br>别名／原称：工资结果系列头 | 工资结果版本链头 | 某计算责任、期间及币种下当前确认完整结果和后继指针，多次更正据此连续求差。 |

<a id="domain-m11"></a>

## M11 个人所得税扣缴与申报

业务依据与详细契约：[M11模块说明](modules/m11.md)。

| 术语 | 中文名称 | 定义 |
| --- | --- | --- |
| <a id="term-m11-tax-reporting-unit"></a>Tax Reporting Unit<br>别名／原称：TRU | 税务报告单位 | **参考概念。** Oracle 参考概念，承担特定税务报告责任的组织用途；本设计按中国扣缴和申报登记承接。 依据：[reference-model](#source-reference-model)。 |
| <a id="term-m11-organization-calculation-card"></a>Organization Calculation Card | 组织计算卡 | **参考概念。** Oracle 参考的组织级法定扣缴或参缴计算配置；本设计拆为主体登记、资格事实及规则引用。 依据：[reference-model](#source-reference-model)。 |
| <a id="term-m11-withholdingregistration"></a>WithholdingRegistration<br>别名／原称：扣缴主体登记 | 扣缴义务人登记 | 本系统记录承担个税扣缴申报责任的扣缴义务人及其登记范围；区别于员工税收身份，建立该记录不代表已办结外部税务登记。 依据：[cn-tax-terms](#source-cn-tax-terms)。 |
| <a id="term-m11-taxidentity"></a>TaxIdentity | 税收身份 | 与居民身份等税务判断有关的人员事实和证据，不能只由国籍或工作地点推定。 |
| <a id="term-m11-taxdeduction"></a>TaxDeduction<br>别名／原称：Deduction、税务扣除信息 | 个税扣除信息 | 按适用期间记录扣除项目、人员选择及证据来源，实际采用由税务规则判断。 |
| <a id="term-m11-taxledger"></a>TaxLedger | 税务累计账 | 按明确累计口径保存已确认所得、税额及冲销流水的独立账本，不等同于工资累计。 |
| <a id="term-m11-taxassessment"></a>TaxAssessment<br>别名／原称：税务计算评估 | 个税计算结果 | 在固定输入和累计基准上形成的不可变个税计算结果，保存依赖版本；计算结果不自动确认所得。 |
| <a id="term-m11-taxfiling"></a>TaxFiling<br>别名／原称：Filing、税务申报 | 个税扣缴申报 | 向税务渠道提交的个税扣缴申报及更正记录，保存范围、版本和回执；接受申报不等于已缴税，也不据此扩展为个人年度汇算业务。 依据：[cn-tax-terms](#source-cn-tax-terms)。 |
| <a id="term-m11-taxmovement"></a>TaxMovement | 税款资金流水 | 已证实的缴税、退税等资金事实，独立于申报、指令和技术回执状态。 |
| <a id="term-m11-taxcommitment"></a>TaxCommitment<br>别名／原称：税务确认承诺 | 所得确认控制记录 | 协调一组有先后依赖的所得明细及确认、取消状态的控制记录，约束后续执行许可；不是纳税承诺书或纳税申报。 |
| <a id="term-m11-incomeitemplan"></a>IncomeItemPlan<br>别名／原称：所得成员计划 | 所得明细处理计划 | 所得确认控制记录中各笔所得的处理顺序、来源及分配安排，固定计划版本。 |
| <a id="term-m11-dispatchpermit"></a>DispatchPermit<br>别名／原称：税务执行许可、支付执行许可 | 涉税付款执行许可 | 本系统税务域对特定用途、所得确认锚点或返还分类签发的受限执行许可；不是税务机关许可，也不等同于付款审批。 |
| <a id="term-m11-taxpolicy"></a>TaxPolicy | 税务规则 | 税务域维护的可执行所得分类、扣除及计算规则版本，引用政策来源证据。 |
| <a id="term-m11-incomeevidence"></a>IncomeEvidence | 所得来源证据 | 税务域采用的工资、福利等所得输入快照及其性质依据，区别于已确认税务流水。 |
| <a id="term-m11-paymenttreatmentdecision"></a>PaymentTreatmentDecision<br>别名／原称：付款税务用途认定 | 付款涉税处理认定 | 确认某笔付款是否属于非所得超收返还等用途的决定，保存允许分配、依据和撤销链。 |
| <a id="term-m11-taxremittanceaccount"></a>TaxRemittanceAccount<br>别名／原称：应缴税款账户 | 应缴税款台账 | 税务域管理来源唯一税债、更正、应付和资金占用的内部台账，与员工工资应付分别管理；不是实际缴税银行账户。 |
| <a id="term-m11-taxremittanceinstruction"></a>TaxRemittanceInstruction<br>别名／原称：税款缴付指令 | 税款缴纳指令 | 针对已确认税款义务形成的资金执行请求，可直接执行或排他委托财务办理。 依据：[cn-tax-terms](#source-cn-tax-terms)。 |
| <a id="term-m11-settlementauthorization"></a>SettlementAuthorization | 税款结算授权 | 绑定特定税款缴付内容与权限范围的批准记录，内容变更需重新核验授权。 |
| <a id="term-m11-settlementinvestigation"></a>SettlementInvestigation | 税款结算核查 | 对未知或冲突缴付结果保留原身份并开展查询核对的记录，不以重复付款代替调查。 |

<a id="domain-m12"></a>

## M12 社会保险与住房公积金

业务依据与详细契约：[M12模块说明](modules/m12.md)。

| 术语 | 中文名称 | 定义 |
| --- | --- | --- |
| <a id="term-m12-social-insurance"></a>Social Insurance | 社会保险 | 按适用制度管理参保、应缴义务、申报和缴费的业务范围，具体险种及参数按属地和期间确定。 |
| <a id="term-m12-housing-fund"></a>Housing Fund | 住房公积金 | 按适用制度管理住房公积金账户、缴存义务、申报和实缴的业务范围，与各社保险种分别配置。 |
| <a id="term-m12-agencyaccount"></a>AgencyAccount | 社保公积金经办账户 | 主体在经办机构办理特定险种或公积金业务的账户及属地身份，不是个人工资收款账户。 |
| <a id="term-m12-contributionenrollment"></a>ContributionEnrollment<br>别名／原称：Enrollment | 参保与缴存登记 | 人员参加某险种或住房公积金的资格、账户及有效期间记录。 |
| <a id="term-m12-obligationaccount"></a>ObligationAccount | 参缴义务账户 | 按规范责任范围稳定标识一项参缴义务，汇总应缴全量版本及有符号变动。 |
| <a id="term-m12-obligationversion"></a>ObligationVersion | 应缴义务版本 | 对同一义务完整应缴金额及依据的重述版本；与前一版本比较形成差额，不能直接累加全量。 |
| <a id="term-m12-contributionfiling"></a>ContributionFiling | 参缴申报 | 向社保或公积金渠道提交的申报批次与逐行内容，保留采用的义务版本。 |
| <a id="term-m12-linereceipt"></a>LineReceipt<br>别名／原称：参缴申报行回执 | 参缴申报明细回执 | 针对原申报行和版本的接受、拒绝或未知结果；不代表机构已收款。 |
| <a id="term-m12-contributionsettlement"></a>ContributionSettlement<br>别名／原称：参缴资金结算 | 参缴实缴与退款记录 | 经办机构实缴或退款的已证实资金事实，与申报接受状态独立。 |
| <a id="term-m12-employeereceivable"></a>EmployeeReceivable<br>别名／原称：个人应收缴费款 | 应收个人缴费款 | 单位需从人员扣收或收回的个人负担金额子账，保存余额和扣收占用。 |
| <a id="term-m12-deductionreservation"></a>DeductionReservation | 代扣预留 | 为某次工资代扣保留的个人应收额度，工资确认时原子核销，不能被多次扣收。 |
| <a id="term-m12-contributioncorrection"></a>ContributionCorrection<br>别名／原称：Correction | 参缴义务更正 | 按原义务账户和版本形成应缴差额及后继的操作，保留原申报与资金历史。 |
| <a id="term-m12-collectionallocation"></a>CollectionAllocation<br>别名／原称：缴费扣收分配 | 个人缴费扣收分配 | 将已成立的工资扣收或其他收款来源分配到明确义务账户及当时义务版本的记录，历史分配不随更正迁移；待扣收额度由DeductionReservation预留，尚不构成实际收款分配。 依据：[ddd-contracts](#source-ddd-contracts)。 |
| <a id="term-m12-agencypayableaccount"></a>AgencyPayableAccount<br>别名／原称：经办机构应付账户 | 应付经办机构缴款台账 | 管理单位应向经办机构缴款的应付、占用和核销，并按义务分配的内部台账；与应收个人缴费款及义务全量分别记录，不是经办机构的银行账户。 |
| <a id="term-m12-contributionpolicy"></a>ContributionPolicy | 参缴规则 | 按险种、地区和有效期间维护资格、比例及计算规则的配置，引用对应政策证据。 |
| <a id="term-m12-baseassessment"></a>BaseAssessment | 缴费基数评估 | 依据明确期间和来源收入形成的缴费基数判断结果，保留版本和依据。 |
| <a id="term-m12-personal-social-insurance-number"></a>Personal Social Insurance Number | 个人社保编号 | 用于社保经办业务识别个人的外部编号，保存其经办机构及适用范围；与单位经办账户、个人银行账户及公积金账号分别记录。 依据：[cr2018](#source-cr2018)。 原文定位：华润PDF第245页（DS_CRH0_HR_030002）。 |
| <a id="term-m12-personal-housing-fund-account-number"></a>Personal Housing Fund Account Number | 个人公积金账号 | 用于识别个人住房公积金缴存账户的外部编号；与单位经办账户、个人社保编号及工资收款账户分别记录。 依据：[cr2018](#source-cr2018)。 原文定位：华润PDF第245页（DS_CRH0_HR_030002）。 |

<a id="domain-m13"></a>

## M13 企业福利

业务依据与详细契约：[M13模块说明](modules/m13.md)。

| 术语 | 中文名称 | 定义 |
| --- | --- | --- |
| <a id="term-m13-benefits"></a>Benefits | 企业福利 | 企业提供的保障、待遇和服务安排，涵盖计划、资格、参与及费用，与工资核算分域管理。 |
| <a id="term-m13-benefitplan"></a>BenefitPlan | 福利计划 | 定义保障内容、适用对象、期间及费用分担等规则的配置。 |
| <a id="term-m13-benefitoption"></a>BenefitOption<br>别名／原称：Option | 福利选项 | 某福利计划下可供选择的保障档次或组合，不等于个人已经参选。 |
| <a id="term-m13-benefitenrollment"></a>BenefitEnrollment | 福利参与记录 | 人员或家属在有效期间选择并参与计划选项的记录，包含资格和退出依据。 |
| <a id="term-m13-lifeevent"></a>LifeEvent | 生活事件 | 婚姻、出生等可能改变福利资格或选择机会的事件；是否影响计划由规则判断。 |
| <a id="term-m13-benefitcost"></a>BenefitCost | 福利费用 | 按计划及参与情况确认的单位和个人费用事实，向工资或成本域提供明确用途来源。 |
| <a id="term-m13-premium"></a>Premium | 保费 | 保险或保障服务的应付费用和账单金额，与个人待遇申请或理赔金额分别管理。 |
| <a id="term-m13-benefitclaim"></a>BenefitClaim | 福利待遇申请 | 人员提出报销或待遇给付的申请及证据、决定，区分企业承担和保险商直接理赔。 |
| <a id="term-m13-benefitsupplierpayable"></a>BenefitSupplierPayable<br>别名／原称：福利供应商应付 | 应付福利供应商款 | 针对单位应付供应商的保费义务管理来源、占用和净已付的内部账户，不混入员工报销应付，也不代表供应商银行账户。 |
| <a id="term-m13-suppliersettlementinstruction"></a>SupplierSettlementInstruction | 福利供应商结算指令 | 福利域针对供应商应付形成的直接或财务委托执行请求，保持互斥执行路径。 |
| <a id="term-m13-suppliermovement"></a>SupplierMovement | 福利供应商资金流水 | 已证实的供应商付款及退款事实，供原应付账户逐项核销。 |

<a id="domain-m14"></a>

## M14 工资支付与收付款账户

业务依据与详细契约：[M14模块说明](modules/m14.md)。

| 术语 | 中文名称 | 定义 |
| --- | --- | --- |
| <a id="term-m14-payment-method"></a>Payment Method | 支付方式 | 银行、现金等工资支付方式及各自必要信息；与具体银行账户和付款指令分别定义。 |
| <a id="term-m14-payeraccount"></a>PayerAccount | 机构付款账户 | 机构用于执行工资或其他已授权付款的账户配置，统一控制账户准入而不接管各业务义务余额。 |
| <a id="term-m14-paymentuse"></a>PaymentUse | 账户支付用途 | 某账户在指定期间允许参与的支付用途和分配配置；有账户身份不等于允许付款。这里的业务支付用途及分配配置与银行开户用途分类分别维护。 依据：[cr2018](#source-cr2018)。 原文定位：华润PDF第45页（DS_CRH0_BS_010006）。 |
| <a id="term-m14-payerconfiguration"></a>PayerConfiguration | 付款方配置 | 付款主体、账户、方式和渠道等运营配置，引用组织法律身份及其允许付款用途；付款主体与用人、用工和扣缴角色分别确定，配置存在不等于资金已付出。 |
| <a id="term-m14-payableaccount"></a>PayableAccount<br>别名／原称：工资应付结算账户 | 工资应付结算台账 | 统一记录一项工资应付的金额变动、支付、退回和扣回，并管理付款占用的权威内部结算台账；与实际收付款银行账户分开。 |
| <a id="term-m14-settlemententry"></a>SettlementEntry | 工资结算分录 | 原应付账户中按业务来源记载的付款、银行退回、员工归还或工资扣回等变动；区分现金与非现金。 |
| <a id="term-m14-paymentreservation"></a>PaymentReservation<br>别名／原称：Reservation | 付款额度占用 | 某付款执行前在原应付中保留的可付额度，结果未知时不能随意释放。 |
| <a id="term-m14-paymentinstruction"></a>PaymentInstruction | 付款指令 | 固定付款主体、收款人、金额、方式和账户快照的业务请求，具有独立授权与执行状态。 |
| <a id="term-m14-paymentauthorization"></a>PaymentAuthorization<br>别名／原称：Authorization | 付款授权 | 绑定付款载荷及审批权限范围的批准记录，金额或账户改变需要新授权。 |
| <a id="term-m14-paymentattempt"></a>PaymentAttempt | 付款执行尝试 | 对同一付款指令的一次技术执行记录，不能把重试视为新业务付款。 |
| <a id="term-m14-paymentreceipt"></a>PaymentReceipt<br>别名／原称：Receipt | 付款渠道回执 | 银行等渠道返回的执行结果观察，去重核实后用于确认资金事实。 |
| <a id="term-m14-cashmovement"></a>CashMovement<br>别名／原称：Movement | 工资付款资金流水 | 工资付款指令的已证实出款及银行退回记录；员工主动归还使用独立入款证据。 |
| <a id="term-m14-recoveryaccount"></a>RecoveryAccount<br>别名／原称：超付追收账户 | 工资超付追收台账 | 管理工资超付的追收决定、占用及分配，额度从原应付结算账推导，不产生另一份可付余额。 |
| <a id="term-m14-recoveryallocation"></a>RecoveryAllocation<br>别名／原称：Allocation | 追收分配 | 将员工归还或后期工资扣回关联到明确追收事项和原应付的分配记录。 |
| <a id="term-m14-payerexecutionhold"></a>PayerExecutionHold<br>别名／原称：付款账户执行暂停 | 付款账户支付限制 | 付款账户上按渠道范围生效的系统执行限制，用于共同阻断工资和机构付款准入；不是银行账户冻结，不删除或撤回已经发生的执行事实。 |
| <a id="term-m14-claimref"></a>ClaimRef | 付款账户执行占用引用 | 付款账户登记的各来源域执行占用引用，用于账户级准入协调，业务金额仍归来源域。 |
| <a id="term-m14-bank-account-name"></a>Bank Account Name<br>别名／原称：户名、账户名称 | 银行账户名称 | 开户时登记的账户持有人名称，业务中常称户名；与银行账号、人员姓名及组织显示名称分别保存。 依据：[cr2018](#source-cr2018)。 原文定位：华润PDF第45页（DS_CRH0_BS_010001）。 |
| <a id="term-m14-bank-branch"></a>Bank Branch | 开户行 | 办理银行账户开户的银行营业机构，通常指具体网点；与银行品牌、账户名称及银行账号分开。 依据：[cr2018](#source-cr2018)。 原文定位：华润PDF第45页（DS_CRH0_BS_010002）。 |
| <a id="term-m14-paying-entity"></a>Paying Entity | 付款主体 | 相应付款业务中承担资金付出角色的组织，按付款用途、机构账户和授权独立确定；不从用人单位、用工单位或受托经办方自动推导。实际资金执行由核实后的付款证据确认。<br>来源说明：这是项目付款角色概念，不新增聚合，也不自动赋予扣缴或参缴责任。 |

<a id="domain-m15"></a>

## M15 人工成本与财务衔接

业务依据与详细契约：[M15模块说明](modules/m15.md)。

| 术语 | 中文名称 | 定义 |
| --- | --- | --- |
| <a id="term-m15-costing"></a>Costing | 人工成本核算 | 确认人员相关成本的来源、归属及分摊，并形成财务衔接依据的业务过程。 |
| <a id="term-m15-costfact"></a>CostFact | 人工成本事实 | 来自工资、单位缴费或福利等领域的已确认成本记录，保存来源身份和版本。 |
| <a id="term-m15-costallocation"></a>CostAllocation | 成本分摊 | 将一项成本按部门、成本中心或项目等归属分配的完整集合，统一检查总量。 |
| <a id="term-m15-accounting-structure"></a>Accounting Structure | 会计核算结构 | 用于会计输出的科目及成本归属维度结构；具体映射由成本域版本化维护。 |
| <a id="term-m15-accountingmapping"></a>AccountingMapping | 会计映射 | 把业务成本项目和归属映射到会计科目、成本中心及项目等的规则。 |
| <a id="term-m15-accountingperiod"></a>AccountingPeriod<br>别名／原称：Period | 会计期间 | 组织会计确认和过账的期间及开放、关账中、已关闭等状态。 |
| <a id="term-m15-accountingbatch"></a>AccountingBatch | 会计输出批次 | M15持有冻结JournalRequest、逐行回执和执行核查状态的会计输出聚合根；准备及领取与Period共享期间守卫，外部结果未知时保留原批次和请求身份，不另建批次重发。 依据：[ddd-contracts](#source-ddd-contracts)。 |
| <a id="term-m15-journalrequest"></a>JournalRequest | 会计分录请求 | AccountingBatch拥有的会计分录内容及其冻结请求版本；请求内容不代表外部已经接受或过账，批次的回执及执行核查状态另行保留。 依据：[ddd-contracts](#source-ddd-contracts)。 |
| <a id="term-m15-accountingreceipt"></a>AccountingReceipt<br>别名／原称：Receipt | 会计回执 | 财务系统对原分录请求返回的接受、拒绝或过账等结果，需按行和版本核对。 |
| <a id="term-m15-costcorrection"></a>CostCorrection | 成本更正 | 对原成本或分摊形成差额、冲销及后继会计处理的记录，不覆盖旧财务事实。 |
| <a id="term-m15-labor-cost-item"></a>Labor Cost Item | 人工成本科目 | 按管理目的对人员相关成本归类的科目或分类项，可与会计科目建立映射；层级、组成和有效版本由采用的成本口径确定，不自动等同工资项目分类。 依据：[cr2018](#source-cr2018)。 原文定位：华润PDF第245页（DS_CRH0_HR_030001）。 |
| <a id="term-m15-accounting-account"></a>Accounting Account | 会计科目 | 财务核算中用于分类确认和归集会计事项的科目；人工成本域按会计映射引用其明确版本，科目层级及可过账规则由财务配置确定。 依据：[cr2018](#source-cr2018)。 原文定位：华润PDF第50页（DS_CRH0_FI_040006）。 |

<a id="domain-m16"></a>

## M16 人才档案、技能与资格

业务依据与详细契约：[M16模块说明](modules/m16.md)。

| 术语 | 中文名称 | 定义 |
| --- | --- | --- |
| <a id="term-m16-contenttype"></a>ContentType<br>别名／原称：Content Type、人才内容类型 | 人才档案内容类型 | 定义技能、教育、证书等内容类别及属性的配置，供档案栏目引用。 |
| <a id="term-m16-contentitem"></a>ContentItem<br>别名／原称：Content Item、Item、人才内容项 | 人才档案内容项 | 某内容类型下可复用的具体条目，例如一项技能或资格名称，具有稳定身份。 |
| <a id="term-m16-ratingmodel"></a>RatingModel<br>别名／原称：Rating Model、量表 | 评分模型 | 定义等级、分值及其含义的可复用评分规则，可包含量表；版本变化不重释旧评分。 |
| <a id="term-m16-talentprofile"></a>TalentProfile<br>别名／原称：Profile | 人才档案 | 按档案类型组织人员能力或职务岗位要求的结构化资料，与个人身份及联系方式资料分开。 |
| <a id="term-m16-personprofile"></a>PersonProfile | 人员能力档案 | 记录人员技能、资格等内容和证据的人才档案，不等同于个人基本资料。 |
| <a id="term-m16-requirementprofile"></a>RequirementProfile<br>别名／原称：JobProfile | 任职要求档案 | 记录职务或岗位所需能力及资格的档案，供匹配和差距分析使用。其中任职资格描述所需知识、经验和技能等要求，与已认定的人员资格事实分开。 依据：[cr2018](#source-cr2018)。 原文定位：华润PDF第246页（DS_CRH0_HR_040006）。 |
| <a id="term-m16-profilesection"></a>ProfileSection<br>别名／原称：Profile Section、Section、章节、档案内容区段 | 档案栏目 | 档案中按内容类型组织字段和条目的栏目，具有读写权限约束。 |
| <a id="term-m16-qualification"></a>Qualification | 人员资格 | 人员在某项目或期间达到特定资格要求的事实，需有规则或授权判断，学习完成不自动成立。 |
| <a id="term-m16-qualificationassessment"></a>QualificationAssessment<br>别名／原称：Assessment | 资格评价证据 | 自评、经理、专家、证书等针对资格形成的评价及有效期依据，区别于最终资格决定。 |
| <a id="term-m16-questionnaire"></a>Questionnaire | 问卷 | 可复用的问题、选项和结构定义，实际回答由具体评价业务持有。 |
| <a id="term-m16-questionnaireresponse"></a>QuestionnaireResponse<br>别名／原称：Response、问卷回答 | 问卷作答记录 | 在特定人员和评价上下文中提交的回答及其版本，由来源评价对象保存。 |
| <a id="term-m16-qualificationdecision"></a>QualificationDecision | 资格认定决定 | 依据证据和规则确认、拒绝、撤销或更新资格的决定，保存确切版本及理由。 |
| <a id="term-m16-profiletype"></a>ProfileType<br>别名／原称：Profile Type | 档案类型 | 定义档案适用对象及内容区段结构的配置，实例明确采用某一版本。 |
| <a id="term-m16-sectiondefinition"></a>SectionDefinition<br>别名／原称：档案区段定义 | 档案栏目定义 | 档案类型中的可复用栏目结构、内容类型引用和约束；与档案中的具体内容分开。 |
| <a id="term-m16-job-qualification-requirements"></a>Job Qualification Requirements | 任职资格 | 从事某职务或岗位所需的知识、经验、技能、素质和行为等要求；用于招聘与任职匹配，区别于人员已经达到要求的资格事实。 依据：[cr2018](#source-cr2018)。 原文定位：华润PDF第246页（DS_CRH0_HR_040006）。 |
| <a id="term-m16-educational-attainment"></a>Educational Attainment | 学历 | 反映人员接受教育的层次及相应完成状况的资料；与学位、教育形式分别记录，不以学历名称直接代替学位。 依据：[cr2018](#source-cr2018)。 原文定位：华润PDF第4页（DS_CRH0_PB_010003）。 |
| <a id="term-m16-academic-degree"></a>Academic Degree | 学位 | 由具备相应授予资格的机构按学业或学术成就等授予的称号；与学历及教育形式分开，名誉等具体类别由相应内容配置说明。 依据：[cr2018](#source-cr2018)。 原文定位：华润PDF第4页（DS_CRH0_PB_010004）。 |
| <a id="term-m16-education-mode"></a>Education Mode | 教育形式 | 人员接受教育的方式，例如全日制或非全日制；与所获学历层次和学位分开，具体分类采用明确来源及版本。 依据：[cr2018](#source-cr2018)。 原文定位：华润PDF第244页（DS_CRH0_HR_010017）。 |

<a id="domain-m17"></a>

## M17 目标与绩效

业务依据与详细契约：[M17模块说明](modules/m17.md)。

| 术语 | 中文名称 | 定义 |
| --- | --- | --- |
| <a id="term-m17-goal-library"></a>Goal Library | 目标库 | 可复用目标定义的集合，为目标计划和个人目标提供来源，不保存个人完成进度。 |
| <a id="term-m17-goaldefinition"></a>GoalDefinition | 绩效目标定义 | 目标库中的可复用目标内容和衡量要求，区别于分配给个人后的目标实例。 |
| <a id="term-m17-goalplan"></a>GoalPlan<br>别名／原称：Goal Plan | 绩效目标计划 | 规定目标适用期间、范围和可选目标等的配置，由个人目标分配明确采用。 |
| <a id="term-m17-reviewperiod"></a>ReviewPeriod<br>别名／原称：Review Period、绩效评估周期 | 绩效考核周期 | 组织某轮目标与绩效评价的时间范围，不等同于工资期间。它表示具体评价期间，考核频率表示重复间隔，两者分别维护。 依据：[cr2018](#source-cr2018)。 原文定位：华润PDF第270页（DS_CHA0_HR_050001）。 |
| <a id="term-m17-performancetemplate"></a>PerformanceTemplate<br>别名／原称：Performance Template | 绩效模板 | 定义评价内容、章节、权重及参与规则的配置，引用流程、文档类型和评分模型。 |
| <a id="term-m17-performancedocument"></a>PerformanceDocument<br>别名／原称：Performance Document、绩效文档 | 绩效评价文档 | 针对个人及评价周期建立的评价业务对象，固定目标、模板等版本并保存阶段、反馈及结果。 依据：[cr2018](#source-cr2018)。 原文定位：华润PDF第246页（DS_CRH0_HR_050001）。 |
| <a id="term-m17-stageinstance"></a>StageInstance | 绩效阶段实例 | 某份绩效文档中按流程创建的阶段及状态，完成依据由绩效域确认。 |
| <a id="term-m17-feedback"></a>Feedback | 绩效反馈 | 评价参与者就目标或表现给出的意见及证据，区别于最终评分和申诉处理。 |
| <a id="term-m17-appeal"></a>Appeal | 绩效申诉 | 对绩效过程或结果提出复核请求及其处理记录，保留原结果与后继依据。 |
| <a id="term-m17-calibration"></a>Calibration | 绩效校准 | 根据明确准则对跨人员或团队评价进行一致性复核及调整的过程和依据。 |
| <a id="term-m17-goalassignment"></a>GoalAssignment | 个人绩效目标分配 | 把目标与计划落实到某人及周期的目标对象，拥有具体目标、权重和进度。 |
| <a id="term-m17-goalinstance"></a>GoalInstance | 个人绩效目标 | 某人实际承担的一个目标实例，可引用目标库或自建，记录个人化内容及权重。 |
| <a id="term-m17-goalprogress"></a>GoalProgress<br>别名／原称：Progress | 目标进度 | 某个人目标在指定时间的完成情况及证据，不直接等于最终绩效评价。 |
| <a id="term-m17-performanceprocessdefinition"></a>PerformanceProcessDefinition<br>别名／原称：Process Flow | 绩效流程定义 | 定义评价任务、顺序和完成条件的可复用配置，实际状态由绩效文档保存。 |
| <a id="term-m17-performancedocumenttype"></a>PerformanceDocumentType<br>别名／原称：DocumentType | 绩效文档类型 | 区分不同评价用途及适用规则的配置类型，供绩效模板和文档引用。 |
| <a id="term-m17-performance-score"></a>Performance Score<br>别名／原称：考核得分 | 绩效考核得分 | 人员在明确考核周期及评分规则下形成的数值结果，保留采用的规则和结果版本；与评分模型本身及分类等级分开。 依据：[cr2018](#source-cr2018)。 原文定位：华润PDF第246页（DS_CRH0_HR_050001）。 |
| <a id="term-m17-performance-rating"></a>Performance Rating<br>别名／原称：考核等级 | 绩效考核等级 | 人员绩效评价形成的分类结果，等级含义及与分值的关系由采用的评分模型确定；与得分及评分模型本身分开。 依据：[cr2018](#source-cr2018)。 原文定位：华润PDF第246页（DS_CRH0_HR_050002）。 |

<a id="domain-m18"></a>

## M18 学习与发展

业务依据与详细契约：[M18模块说明](modules/m18.md)。

| 术语 | 中文名称 | 定义 |
| --- | --- | --- |
| <a id="term-m18-learningcatalog"></a>LearningCatalog | 学习目录 | 组织可供查找和分配的课程及其他学习项目的目录，区别于一次课程开设或个人报名。 |
| <a id="term-m18-catalog-item"></a>Catalog Item | 学习目录项 | 学习目录中可被检索和引用的条目，具体可指课程或其他学习对象，不等同于人才内容项。 |
| <a id="term-m18-course"></a>Course | 课程 | 定义学习目标、内容和完成要求的学习项目，可有多个不同方式的开设安排。 |
| <a id="term-m18-learningoffering"></a>LearningOffering<br>别名／原称：Offering、开班 | 课程开设安排 | 课程以特定方式、时间或资源提供学习的实例，可含面授、自学或混合学习；目标模型在此管理名额和候补。 |
| <a id="term-m18-learningenrollment"></a>LearningEnrollment | 学习报名 | 人员参与某一具体课程开设安排的记录，具有报名和名额状态，与学习要求及完成记录分开。 |
| <a id="term-m18-seatreservation"></a>SeatReservation<br>别名／原称：座位 | 学习名额占用 | 某人在课程开设安排中已确认占用的名额，取消或转班时按规则释放。 |
| <a id="term-m18-waitlistentry"></a>WaitlistEntry | 学习候补登记 | 满员时记录的候补意向和排序，尚未占用正式名额；递补成功后形成正式报名。 |
| <a id="term-m18-learningrecord"></a>LearningRecord<br>别名／原称：Learning Record | 学习记录 | 人员的学习进度、完成证据及考试证书等记录，历史更正保留原版本。计划时长与实际培训时长分开；参与时长及人次须使用明确的计量口径。 依据：[cr2018](#source-cr2018)。 原文定位：华润PDF第246页（DS_CRH0_HR_060001）；华润PDF第246页（DS_CRH0_HR_060005）。 |
| <a id="term-m18-developmentplan"></a>DevelopmentPlan | 个人发展计划 | 围绕个人发展目标组织学习、辅导及实践等行动的计划，单次课程完成不等于计划完成。 |
| <a id="term-m18-developmentgoal"></a>DevelopmentGoal | 个人发展目标 | 某人希望达成的能力或职业发展目标，与绩效目标分别管理。 |
| <a id="term-m18-developmentaction"></a>DevelopmentAction<br>别名／原称：Action | 发展行动 | 为实现发展目标安排的学习、辅导、实践等具体行动及完成标准。 |
| <a id="term-m18-learningpath"></a>LearningPath<br>别名／原称：Learning Path | 学习路径 | 组织课程或学习项目顺序、前置条件及完成要求的配置。 |
| <a id="term-m18-learningaudience"></a>LearningAudience<br>别名／原称：Audience、学习适用受众 | 学习适用人群 | 某学习项目面向的人群及适用条件，适用不自动赋予访问权限。 |
| <a id="term-m18-learningassignment"></a>LearningAssignment<br>别名／原称：学习要求、学习指派 | 学习任务 | 对某人的学习要求、截止日期及重修周期，可先于课程开设和报名存在。 |
| <a id="term-m18-developmentgoaldefinition"></a>DevelopmentGoalDefinition | 发展目标定义 | 可供个人发展计划选用的标准目标内容，配置变更不直接改写已采用的个人目标。 |
| <a id="term-m18-waitlist-promotion"></a>Waitlist Promotion<br>别名／原称：候补转正 | 候补递补 | 有可用名额后，重新核验资格和容量并将候补转为正式报名的操作。 |
| <a id="term-m18-learningevent"></a>LearningEvent<br>别名／原称：Learning Event | 学习活动场次 | **参考概念。** 参考映射中的具体学习活动或事件概念；目标以课程开设安排承载相关活动，不把所有活动与课程开设身份视为相同。 依据：[reference-model](#source-reference-model)。 |
| <a id="term-m18-planned-learning-duration"></a>Planned Learning Duration | 计划学习时长 | 某次培训或学习安排的预计持续时间，须标明单位；与实际培训时长分别记录，不自动采用企业特定的小时与天转换阈值。 依据：[cr2018](#source-cr2018)。 原文定位：华润PDF第246页（DS_CRH0_HR_060001）。 |
| <a id="term-m18-actual-training-duration"></a>Actual Training Duration | 实际培训时长 | 某次培训实际发生的持续时间，保留计量单位和来源；与计划时长、按人员累计的参与时长及人次分别表达。 依据：[cr2018](#source-cr2018)。 原文定位：华润PDF第246页（DS_CRH0_HR_060005）。 |

<a id="domain-m19"></a>

## M19 人才盘点、人才池与继任

业务依据与详细契约：[M19模块说明](modules/m19.md)。

| 术语 | 中文名称 | 定义 |
| --- | --- | --- |
| <a id="term-m19-talentreviewsession"></a>TalentReviewSession<br>别名／原称：Talent Review | 人才盘点会议 | 在明确范围和周期内开展的人才评价活动及其参与者、证据快照和结论。 |
| <a id="term-m19-talentpool"></a>TalentPool<br>别名／原称：Talent Pool、人才库 | 人才池 | 按发展或继任等目的管理的一组人才，成员加入和退出有依据及有效期间。 |
| <a id="term-m19-talentpoolmembership"></a>TalentPoolMembership<br>别名／原称：Membership | 人才池成员关系 | 某人与某人才池之间的成员身份、期间及加入退出依据，不等同于任职关系。 |
| <a id="term-m19-successionplan"></a>SuccessionPlan<br>别名／原称：Succession Plan | 继任计划 | 围绕职务、岗位或在任者建立的后备人选与发展安排，候选入选不自动获得任命。 |
| <a id="term-m19-successorcandidate"></a>SuccessorCandidate<br>别名／原称：Candidate | 继任候选人 | 被提名进入某继任计划的人选，可为内部人员或经招聘身份引用的外部候选；提名本身不预留编制。 |
| <a id="term-m19-incumbent"></a>Incumbent | 在任者 | 当前承担某一任职的人，作为继任主题时需明确其任职和组织范围。 |
| <a id="term-m19-readiness"></a>Readiness<br>别名／原称：准备度 | 继任准备度 | 对候选人何时具备承担目标职责条件的判断，不等同于已批准任命。 |
| <a id="term-m19-readinessassessment"></a>ReadinessAssessment | 继任准备度评估 | 针对候选人及日期形成的准备度判断和证据版本，后续变化保留原评估。 |
| <a id="term-m19-talentreviewtemplate"></a>TalentReviewTemplate | 人才盘点模板 | 配置盘点维度和评价方式的可复用模板，会议实例固定采用版本。 |
| <a id="term-m19-talentreviewdimension"></a>TalentReviewDimension<br>别名／原称：Dimension | 人才盘点评价维度 | 盘点使用的绩效、潜力等评价角度及评分尺度引用。 |

<a id="domain-m20"></a>

## M20 员工与经理服务及HR任务

业务依据与详细契约：[M20模块说明](modules/m20.md)。

| 术语 | 中文名称 | 定义 |
| --- | --- | --- |
| <a id="term-m20-servicerequest"></a>ServiceRequest | 人力资源服务请求 | 员工或经理提出的咨询及办理请求，保存分派和答复；请求关闭不替代业务事实生效。 |
| <a id="term-m20-taskprojection"></a>TaskProjection | 业务待办视图 | 从来源域状态形成的可重建任务列表，实际办理返回来源领域。 |
| <a id="term-m20-journey"></a>Journey | 员工历程任务清单 | 围绕入职、晋升等场景组织跨模块步骤的业务清单；清单状态与各步骤的业务事实分别管理。 |
| <a id="term-m20-journeyinstance"></a>JourneyInstance | 员工历程实例 | 按固定模板版本为对象创建的任务清单，保存步骤和完成证据；模板改版不覆盖已有实例。 |
| <a id="term-m20-stepinstance"></a>StepInstance | 历程步骤实例 | 员工历程中按模板证据条件判断的步骤。来源事实可判定时自动更新，依据不足、矛盾或需人判断时转待复核。 |
| <a id="term-m20-communication"></a>Communication | 员工沟通内容 | 面向明确受众发布的通知或沟通正文及受众快照，区别于渠道送达事实。 |
| <a id="term-m20-deliveryprojection"></a>DeliveryProjection | 消息送达视图 | 从通知或集成设施的回执形成的送达状态投影，业务服务域无权另造送达事实。 |
| <a id="term-m20-payslipview"></a>PayslipView | 工资单授权视图 | 按工资单发布状态及当前访问权限展示的只读视图，发布权仍归工资域。 |
| <a id="term-m20-journeytemplate"></a>JourneyTemplate<br>别名／原称：清单模板 | 员工历程模板 | 定义跨业务步骤、依赖及证据要求的可复用清单配置。 |
| <a id="term-m20-stepdefinition"></a>StepDefinition | 历程步骤定义 | 模板中一个步骤的内容、前置条件及所需完成证据。 |

<a id="domain-x01"></a>

## X01 对象标识、有效期、修订与采用历史

业务依据与详细契约：[X01模块说明](modules/x01.md)。

| 术语 | 中文名称 | 定义 |
| --- | --- | --- |
| <a id="term-x01-identity"></a>Identity<br>别名／原称：稳定身份 | 稳定对象标识 | 在业务对象生命周期及修订中保持不变的标识，区别于名称、业务编号和版本号。 |
| <a id="term-x01-effectiverange"></a>EffectiveRange<br>别名／原称：EffectiveRange Types | 有效期间 | 业务事实生效的日期区间，本设计使用前闭后开区间并明确粒度及时区。 |
| <a id="term-x01-dated-slice"></a>Dated Slice | 有效期切片 | 稳定对象在一段业务有效期间内的状态记录，与系统记录时间及修订次数分开。<br>来源说明：M02 BU 维护中的“生效记录”是场景用语，可包含未来记录，并非只指已经生效的记录。查看日期早于 BU 首条时，统一显示“未生效”，原“待生效”只作历史对照；它是按查看日期得到的读取结果，不是新增的持久状态或审批状态。 依据：[chnhcm-m02-terms-011](#source-chnhcm-m02-terms-011)、[chnhcm-m02-dated-maintenance-117](#source-chnhcm-m02-dated-maintenance-117)。 |
| <a id="term-x01-effective-date-query"></a>Effective Date Query | 按生效日期查询 | 根据指定业务日期读取当时有效事实的查询方式，不等同于查询当时系统已经知道的信息。 |
| <a id="term-x01-update"></a>Update | 有效期更新 | 通过新的生效日期段记录业务变化，保留旧期间的事实。 |
| <a id="term-x01-correction"></a>Correction | 数据更正 | 纠正既有事实或记录错误并保留修订证据；是否影响下游由各域政策判断。<br>来源说明：M02普通名称更正保留日期/分类，改日使用专用修改面；022允许显式更正业务分类，可与日期/名称或其他关联项联合办理。首条只改CREATE内原因，非首条可纠正类型及原因；原件、旧来源和未选后继独立快照保持，无处理原因。各域资格独立，不扩大其他领域范围。 依据：[chnhcm-m02-terms-011](#source-chnhcm-m02-terms-011)、[chnhcm-record-maintenance-020](#source-chnhcm-record-maintenance-020)、[chnhcm-bu-lifecycle-021](#source-chnhcm-bu-lifecycle-021)、[chnhcm-joint-timeline-022](#source-chnhcm-joint-timeline-022)。 |
| <a id="term-x01-end-date"></a>End Date | 结束日期 | 标识事实有效期间终点的日期语义，需明确包含或不包含该日及外部格式转换。 |
| <a id="term-x01-future-slice-deletion"></a>Record Deletion<br>别名／原称：未来切片撤销、撤销未来生效变更、Future Slice Deletion、删除未来记录 | 删除记录 | 从当前已记录信息的记录集合中移除符合本域条件的记录效力，并重新核验受影响期间和引用；保留删除前原件、完整期间及删除关系，不物理抹除历史。<br>来源说明：M02统一“删除此记录”，非首条普通/停用/重新启用含过去/当天/未来可删，首条始终保护。单条删除留下孤立恢复或停用期间资料则拒绝；022允许显式成对删除等有限联合项，按最终整线和引用一次校验，不隐式级联。各目标专用权和准确版本仍须通过，预览全部期间/今日状态；旧来源及原采用保持。删除继承原分类，无处理原因，不增加撤销、恢复已删记录或整BU删除；旧别名只供检索。 依据：[chnhcm-m02-terms-011](#source-chnhcm-m02-terms-011)、[chnhcm-record-maintenance-020](#source-chnhcm-record-maintenance-020)、[chnhcm-bu-lifecycle-021](#source-chnhcm-bu-lifecycle-021)、[chnhcm-joint-timeline-022](#source-chnhcm-joint-timeline-022)。 |
| <a id="term-x01-date-interval-overlap"></a>Date Interval Overlap | 有效期间重叠 | 两段事实的有效范围相交；是否合法取决于对象的唯一性及兼容规则。 |
| <a id="term-x01-effective-sequence"></a>Effective Sequence<br>别名／原称：EFFSEQ、Same-day effective sequence | 同日生效序号 | **参考概念。** 参考系统用于区分同一有效日期内业务先后状态的序号；本设计未采用其日内状态查询或计量能力。 依据：[reference-model](#source-reference-model)。 |
| <a id="term-x01-parent-child-date-alignment"></a>Parent-child Date Alignment<br>别名／原称：父子有效期对齐 | 父子有效期约束 | 子对象或引用关系的有效期间满足父对象存在及适用范围的约束；不要求父子起止日期完全相同。 |
| <a id="term-x01-audit-timestamp"></a>Audit Timestamp | 审计记录时间 | 系统记录某操作或变化的时间戳，不直接代表业务生效时间或同日业务先后。 |
| <a id="term-x01-revision"></a>Revision | 修订版本 | 对象记录的系统变更版本，用于审计及并发检查；不能代替业务有效日期或同日生效序号。<br>来源说明：M02普通更正保留日期，改日及显式分类纠错以新修订保存新日期/类型/原因，recordId保持；有效类型按所选修订解释，初始kind只作元数据。一次联合修订中每条变化记录至多一个合并修订，区间/继承派生变化同样留证；后继独立资料不自动传播，旧来源不改写。 依据：[chnhcm-m02-terms-011](#source-chnhcm-m02-terms-011)、[chnhcm-record-maintenance-020](#source-chnhcm-record-maintenance-020)、[chnhcm-bu-lifecycle-021](#source-chnhcm-bu-lifecycle-021)、[chnhcm-joint-timeline-022](#source-chnhcm-joint-timeline-022)。 |
| <a id="term-x01-adoption"></a>Adoption | 版本采用 | 消费者明确将某来源及其版本用于指定业务用途的事实，不等于来源已发布或通知已送达。 |
| <a id="term-x01-adopted-history"></a>Adopted History | 采用历史 | 消费者历次引用的来源、规则和结果版本及依据，支持解释历史结果。 |
| <a id="term-x01-result-version"></a>Result Version | 结果版本 | 某业务计算或决定输出的确定版本，后继更正需保留与原版本的联系。 |
| <a id="term-x01-event-envelope"></a>Event Envelope | 事件信封 | 携带源对象、版本、事件身份和协议版本的通用消息描述，业务内容仍归来源域。 |
| <a id="term-x01-impact-lookup-protocol"></a>Impact Lookup Protocol | 影响查询协议 | 按来源及版本定位消费者和受影响事项的公共协议，索引及业务处理由各域负责。 |
| <a id="term-x01-decisionref-envelope"></a>DecisionRef Envelope | 决定引用信封 | 标识目标域、对象版本及决定身份的共享引用结构，不持有另一份决定事实。 |
| <a id="term-x01-concurrent-editing"></a>Concurrent Editing | 并发编辑 | 多个操作同时修改同一事实的情况，应按预期版本或其他约束检测冲突。 |
| <a id="term-x01-stale-authorization"></a>Stale Authorization | 失效授权 | 因对象、政策、权限或期限变化而不再满足本次执行条件的旧授权，不能继续用作执行许可。 |
| <a id="term-x01-idempotency"></a>Idempotency | 幂等 | 同一业务请求重复执行仍产生同一个业务效果的性质，需要稳定业务键和明确冲突规则。 |
| <a id="term-x01-cas"></a>CAS<br>别名／原称：Compare-and-Swap | 比较并交换 | 仅当当前值或版本与预期一致时才提交更新的并发控制方式。 |
| <a id="term-x01-impactcase"></a>ImpactCase | 来源变更影响事项 | 已采用结果受影响、依据不足或有专项追溯要求时，由消费者建立的评估及处理记录。工资域使用工资追溯事项。 依据：[ddd-contracts](#source-ddd-contracts)。 |
| <a id="term-x01-impactresolution"></a>ImpactResolution | 影响处置结论 | 消费者完成复核后记录的更正、无需更改或其他处置及依据，区别于收到事件的技术确认。 依据：[ddd-contracts](#source-ddd-contracts)。 |
| <a id="term-x01-business-effective-time"></a>Business Effective Time<br>别名／原称：businessAt | 业务生效时间 | 事实在业务上成立的日期或时刻，与政策公布、数据录入及系统提交时间分别记录。<br>来源说明：M02按日称“生效日期”，用于新建、资料变更、停用、重新启用及记录详情。新增与改日统一允许过去/当天/未来，仍核对企业、首条、日期占用、完整启停顺序和引用；当前日期只作默认读取及办理说明。停用提示“自该日起停用”，重新启用提示“自该日起重新启用”，不另设日期事实。普通更正保留原日，改日同一身份追加修订，删除不输入另一生效日；旧来源仍按原日期解释。businessAt称查看日期，不与写入日期混用。 依据：[ddd-contracts](#source-ddd-contracts)、[chnhcm-m02-terms-011](#source-chnhcm-m02-terms-011)、[chnhcm-record-maintenance-020](#source-chnhcm-record-maintenance-020)、[chnhcm-bu-lifecycle-021](#source-chnhcm-bu-lifecycle-021)。 |
| <a id="term-x01-known-at-time"></a>Known-at Time<br>别名／原称：knownAt、已知截点 | 已知信息截点 | 已退役的设计术语，仅保留稳定词条身份供历史资料检索；原指按任意系统时刻还原已记录信息。现行产品不提供 knownAt 或等价记录时间截点查询；按业务日期读取当前资料，历史通过确切来源版本或已采用版本引用定位。 依据：[ddd-contracts](#source-ddd-contracts)、[chnhcm-m02-terms-011](#source-chnhcm-m02-terms-011)。 |
| <a id="term-x01-adopted-reference"></a>Adopted Reference<br>别名／原称：adoptedRef、采用引用 | 已采用版本引用 | 用于定位消费者当次采用的确切来源版本或不可变快照，以解释原业务结果；引用不等于采用事实本身，采用事实由消费者维护。不支持解析时明确说明，不回退到最新版本。 依据：[ddd-contracts](#source-ddd-contracts)、[chnhcm-m02-terms-011](#source-chnhcm-m02-terms-011)。 |
| <a id="term-x01-bounded-context"></a>Bounded Context | 限界上下文 | 具有独立业务语言、模型和责任边界的领域范围，跨边界通过明确合同协作。 依据：[ddd-contracts](#source-ddd-contracts)。 |
| <a id="term-x01-aggregate"></a>Aggregate | 聚合 | 为共同维护业务不变量而形成的一组对象，由聚合根控制写入和一致性边界。 依据：[ddd-contracts](#source-ddd-contracts)。 |
| <a id="term-x01-aggregate-root"></a>Aggregate Root | 聚合根 | 聚合对外提供身份和写入口的对象，负责维护聚合内部约束。 依据：[ddd-contracts](#source-ddd-contracts)。 |
| <a id="term-x01-entity"></a>Entity | 实体 | 以稳定业务身份区分并追踪生命周期的对象，其属性变化不改变该身份。 依据：[ddd-contracts](#source-ddd-contracts)。 |
| <a id="term-x01-value-object"></a>Value Object | 值对象 | 按属性值而非独立生命周期身份比较的描述对象，例如金额或日期范围。 依据：[ddd-contracts](#source-ddd-contracts)。 |
| <a id="term-x01-projection"></a>Projection | 投影 | 从权威事实派生的查询视图，可按来源重建，不取得独立业务写权限。 依据：[ddd-contracts](#source-ddd-contracts)。 |
| <a id="term-x01-invariant"></a>Invariant | 业务不变量 | 在指定一致性边界内每次有效提交都必须成立的约束，例如分摊总额或可付额度限制。 依据：[ddd-contracts](#source-ddd-contracts)。 |
| <a id="term-x01-domain-event"></a>Domain Event | 领域事件 | 领域确认发生的业务变化及来源版本通知，区别于技术尝试和未知外部观察。 依据：[ddd-contracts](#source-ddd-contracts)。 |
| <a id="term-x01-public-contract"></a>Public Contract<br>别名／原称：公开业务合同 | 对外业务契约 | 领域向其他模块提供的命令、查询、输入输出及行为约束，调用方不得绕过写入口；这里的契约指软件协作约定，不是劳动或劳务合同。 依据：[ddd-contracts](#source-ddd-contracts)、[software-contract](#source-software-contract)。 |
| <a id="term-x01-anti-corruption-layer"></a>Anti-Corruption Layer<br>别名／原称：ACL | 防腐层 | 把外部或上游模型转换为本领域概念的适配边界，避免直接混用不同领域语义。 依据：[ddd-contracts](#source-ddd-contracts)。 |
| <a id="term-x01-record-identifier"></a>Record Identifier<br>别名／原称：记录身份 | 记录标识 | 区分同一业务对象下某条记录的稳定标识，与对象标识、生效日期及修订版本分别维护。M02 普通更正、改日及显式分类纠错保留原记录标识，新增生效记录形成新记录标识；不能把操作者的认证身份当作记录标识。<br>来源说明：M02更正、改日及显式分类纠错均保留recordId；新增含重新启用使用新身份，删除后同日再建也是新身份。首条历史补全通过原首条前移/更正和显式后继新增，首条指针不变且不可删除。当前与历史各按其修订日期和类型解析。 依据：[chnhcm-m02-terms-011](#source-chnhcm-m02-terms-011)、[chnhcm-record-maintenance-020](#source-chnhcm-record-maintenance-020)、[chnhcm-bu-lifecycle-021](#source-chnhcm-bu-lifecycle-021)、[chnhcm-joint-timeline-022](#source-chnhcm-joint-timeline-022)。 |
| <a id="term-x01-submission-identifier"></a>Submission Identifier<br>别名／原称：提交身份 | 提交标识 | 识别同一次业务提交的稳定标识，绑定调用主体、作用域、业务动作及完整业务内容。同标识同内容重试返回原结果，同标识异内容拒绝；它不同于调用者身份、业务对象标识及每次传输尝试的标识。具体接口字段由技术规格确定。 依据：[chnhcm-m02-terms-011](#source-chnhcm-m02-terms-011)。 |
| <a id="term-x01-source-version"></a>Source Version<br>别名／原称：sourceVersion、源版本 | 来源版本 | 来源域已提交事实的确切不可变版本，配合来源对象标识定位该版本的内容和有效范围。与记录修订、审计时间及消费者采用引用分别解释；后续更正或停用不改写旧版本，存在来源版本不证明已经投递或被采用。各域合同确定封装及可重建范围。 依据：[chnhcm-m02-terms-011](#source-chnhcm-m02-terms-011)。 |
| <a id="term-x01-current-recorded-information"></a>Current Knowledge<br>别名／原称：当前知识 | 当前已记录信息 | 按本次查询可见的最新已提交记录解释指定业务日期的事实，是未指定确切来源版本时的查询口径。可包含已登记的过去及未来事实，不等于今日有效事实，也不等于消费者曾采用的版本；仍执行当前授权。 依据：[chnhcm-m02-terms-011](#source-chnhcm-m02-terms-011)。 |
| <a id="term-x01-business-today"></a>Current Date<br>别名／原称：业务今日、今日业务日期、Business Today | 当前日期 | 服务端按指定业务时区计算的当天日期，用于默认查询及日期资格判断。M02 首批采用 Asia/Shanghai，并在办理期间跨午夜时重新校验。用户调整查看日期或生效日期不改变当前日期，不以浏览器日期代替。文中以 D 表示。<br>来源说明：按 CHNHCM-015 的四字优先原则采用“当前日期”；原称仅作检索别名，保留词条 ID 及锚点。Current Date 是项目英文说明名，不指定数据库时间函数或声称是 Oracle 统一界面译名。 依据：[chnhcm-m02-terms-011](#source-chnhcm-m02-terms-011)。 |
| <a id="term-x01-operation-type"></a>Operation Type | 操作类型 | 真实业务变化的分类，回答发生什么变化；M02 首批为新建、资料变更、停用、重新启用。与对象类型、记录处理及状态分别表达；更正、改日或删除记录不另造一个真实业务类型。具体目录、合法组合及状态守卫由所属领域维护。<br>来源说明：项目公共用语，不指定跨域配置目录。M02新业务按入口确定类型，普通更正/改日/删除从准确目标修订继承；022显式分类纠错允许首条改原因及非首条CHANGE/DEACTIVATE/REACTIVATE互纠，类型保存于新修订，旧来源仍解释原分类。“原操作类型”为只读对照。目录和状态规则归M02，M04人事业务类型仍归M04；英文为项目说明名。117将M02显示称谓“业务状态”改为“状态”，内涵不变；共用目录在R1接入时将CHANGE显示为“数据更改”，新原因按117；本轮不改变现行运行目录或旧分类快照。 依据：[chnhcm-m02-terms-011](#source-chnhcm-m02-terms-011)、[chnhcm-record-maintenance-020](#source-chnhcm-record-maintenance-020)、[chnhcm-bu-lifecycle-021](#source-chnhcm-bu-lifecycle-021)、[chnhcm-joint-timeline-022](#source-chnhcm-joint-timeline-022)、[chnhcm-m02-dated-maintenance-117](#source-chnhcm-m02-dated-maintenance-117)。 |
| <a id="term-x01-operation-reason"></a>Operation Reason | 操作原因 | 解释业务变化为何发生、从属于相应操作类型的受控代码值。采用时固化代码和名称快照，后续目录变化不改写历史；与记录处理、字段差异和操作者等审计事实分别表达。<br>来源说明：M02新业务选择固定业务原因，普通更正/改日/删除继承原代码及名称快照，不要求旧选项仍可选。显式分类纠错选择新适用原因并保存新快照，旧分类/名称保留，未改分类不重验候选；首条原因允许纠错。不新增处理原因、办理依据、备注或自由文本，不建设通用平台，M04目录仍归本域。 依据：[chnhcm-m02-terms-011](#source-chnhcm-m02-terms-011)、[chnhcm-record-maintenance-020](#source-chnhcm-record-maintenance-020)、[chnhcm-bu-lifecycle-021](#source-chnhcm-bu-lifecycle-021)、[chnhcm-joint-timeline-022](#source-chnhcm-joint-timeline-022)。 |
| <a id="term-x01-record-action"></a>Record Action | 记录处理 | 对业务事实进行登记或修正的方式，与事实所表达的操作类型分开。M02首批包含新建、新增、更正、改日、删除；停用及重新启用通过新增对应记录表达，普通更正保留日期，改日形成同一记录的新日期修订，删除按允许条件移除该来源下的效力，全部保留历史；普通处理默认继承原分类，显式分类纠错按本域规则保存新分类。<br>来源说明：M02追加/插入/补登记均为新增；停用/重新启用分别表达DEACTIVATE/REACTIVATE。普通更正/改日/删除默认继承分类，显式分类纠错为更正的专用修改面；联合修订把明确新增/更新/删除项分组一次提交，不另造业务类型。013映射技术命令和项修改面，自动保留主体、时间、目标及前后结果，无处理原因；本轮只是后续实施合同。 依据：[chnhcm-m02-terms-011](#source-chnhcm-m02-terms-011)、[chnhcm-record-maintenance-020](#source-chnhcm-record-maintenance-020)、[chnhcm-bu-lifecycle-021](#source-chnhcm-bu-lifecycle-021)、[chnhcm-joint-timeline-022](#source-chnhcm-joint-timeline-022)。 |

<a id="domain-x02"></a>

## X02 共享配置、适用范围、默认值与政策版本

业务依据与详细契约：[X02模块说明](modules/x02.md)。

| 术语 | 中文名称 | 定义 |
| --- | --- | --- |
| <a id="term-x02-sharedreference"></a>SharedReference<br>别名／原称：ReferenceHandle | 共享对象引用 | 描述所属域、对象类型、稳定身份及版本的引用值，不建立另一份业务对象。 |
| <a id="term-x02-applicabilitypolicy"></a>ApplicabilityPolicy<br>别名／原称：适用范围政策 | 适用范围规则 | 按对象、用途和上下文规定允许及排除范围的版本化规则，区别于访问权限。 |
| <a id="term-x02-business-applicability"></a>Business Applicability | 业务适用性 | 某对象或规则在指定组织、日期和用途下能否被使用的业务判断，不等于用户被授权操作。 |
| <a id="term-x02-defaultrule"></a>DefaultRule | 默认值规则 | 根据业务上下文提出候选值的可复用规则，发布规则不自动修改既有业务事实。 |
| <a id="term-x02-defaultproposal"></a>DefaultProposal | 默认值建议 | 带来源、规则版本和期限的候选值，使用方接受前仍需核验业务条件及权限。 |
| <a id="term-x02-default-selection"></a>Default Selection | 默认值选取 | 在当前上下文中按规则选择并明确采用建议值的过程，不代表无条件自动生效。 |
| <a id="term-x02-policyversion"></a>PolicyVersion | 政策版本 | 按辖区、期间和修订标识的一组政策依据与参数，供领域规则引用。 |
| <a id="term-x02-policyevidence"></a>PolicyEvidence<br>别名／原称：Evidence | 政策来源证据 | 记录政策官方来源、发布日期、适用条件及核验状态的证据，不直接执行领域计算。 |
| <a id="term-x02-reference-data-set"></a>Reference Data Set | 参考数据集 | **参考概念。** Oracle 用于组织参考数据共享范围的机制；本设计采用对象级适用策略承接共享问题。 依据：[reference-model](#source-reference-model)。 |
| <a id="term-x02-common-set"></a>Common Set | 公共参考数据集 | **参考概念。** Oracle 中供各业务单元共享参考数据的公共集合；不等同于本设计未配置适用条件的对象。 依据：[reference-model](#source-reference-model)。 |
| <a id="term-x02-setid"></a>SetID | 集合标识 | **参考概念。** PeopleSoft 参考机制中用于区分一组共享控制数据的标识；本项目明确不采纳此机制。 依据：[reference-model](#source-reference-model)。 |
| <a id="term-x02-tableset"></a>TableSet | 表集合共享机制 | **参考概念。** PeopleSoft 用于组织表组及共享控制数据适用关系的机制；不等同于Oracle Fusion参考数据集。 依据：[reference-model](#source-reference-model)。 |
| <a id="term-x02-legislative-data-group"></a>Legislative Data Group<br>别名／原称：LDG | 法规数据组 | **参考概念。** Oracle 按法规及相关薪资处理上下文组织数据的参考概念；本设计通过法规上下文及政策适用承接。 依据：[reference-model](#source-reference-model)。 |
| <a id="term-x02-country-extension"></a>Country Extension | 国家扩展 | **参考概念。** Oracle 针对国家或地区提供的功能及法规扩展；本设计以中国政策包和领域规则版本表达适用内容。 依据：[reference-model](#source-reference-model)。 |
| <a id="term-x02-user-defined-table"></a>User-Defined Table<br>别名／原称：UDT | 用户定义表 | **参考概念。** Oracle 可配置的规则查表数据机制；本设计采用有主责、版本和适用条件的查表数据。 依据：[reference-model](#source-reference-model)。 |
| <a id="term-x02-eligibility"></a>Eligibility | 参与资格 | 按特定业务计划、人员事实及有效期间判断能否参与或享受待遇的条件或结果；缺勤、参缴和福利域分别作出判断。 |
| <a id="term-x02-code-list"></a>Code List | 代码表 | 围绕某类数据组织可选代码项及其含义的集合，须明确来源、版本和适用范围；不等同参考数据集或SetID共享机制。 依据：[cr2018](#source-cr2018)。 原文定位：华润PDF第4页（DS_CRH0_PB_010001）；华润PDF第6页（CD_CRH0_0101）。 |
| <a id="term-x02-code-item"></a>Code Item | 代码项 | 代码表中的一个代码值及其名称、含义等说明；身份需结合所属代码表理解，代码值不自动作为全局唯一标识。 依据：[cr2018](#source-cr2018)。 原文定位：华润PDF第6页（CD_CRH0_0101）；华润PDF第6页（CD_CRH0_0101）。 |

<a id="domain-x03"></a>

## X03 身份、权限与隐私

业务依据与详细契约：[X03模块说明](modules/x03.md)。

| 术语 | 中文名称 | 定义 |
| --- | --- | --- |
| <a id="term-x03-principal"></a>Principal | 访问主体 | 发起访问的用户账号或服务身份，可关联人员但不等同于人员业务身份。 |
| <a id="term-x03-roledefinition"></a>RoleDefinition | 角色定义 | 以领域公开操作描述职责的可复用权限定义，可按受控规则继承。 |
| <a id="term-x03-rolegrant"></a>RoleGrant | 角色授权 | 把角色、数据范围、字段、用途及期限绑定为完整授权单元的授予记录。 |
| <a id="term-x03-datascope"></a>DataScope | 数据访问范围 | 依据人员、工作关系、任职或结果等对象限定可访问数据的策略范围。 |
| <a id="term-x03-fieldpolicy"></a>FieldPolicy | 字段访问策略 | 规定敏感字段可见、可修改或可导出的条件，与对象级访问范围共同判断。 |
| <a id="term-x03-processingpurpose"></a>ProcessingPurpose | 个人信息处理目的 | 说明某类数据访问或使用的业务目的及处理依据，影响授权和数据最小化。 |
| <a id="term-x03-retentionpolicy"></a>RetentionPolicy<br>别名／原称：Retention、数据保留政策 | 数据保留策略 | 规定资料保存、删除及争议保全例外的规则，需要结合具体目的和义务判断。 |
| <a id="term-x03-accessdecision"></a>AccessDecision | 访问判定结果 | 当前请求内的允许或拒绝及字段权限结果；不永久保存完整输入、Grant或匹配过程，不能用于后续请求放行。 |
| <a id="term-x03-exportaudit"></a>ExportAudit | 导出审计 | 已有必要导出操作的最小事件，记录操作者、时间、对象类型、字段名、行数、结果和来源域导出身份；不保存全量目标清单或授权匹配轨迹。 |
| <a id="term-x03-capabilityref"></a>CapabilityRef | 操作能力引用 | 对所属领域公开操作的稳定引用，供角色定义使用，不另造领域写入口。 |
| <a id="term-x03-job-role"></a>Job Role | 职务角色 | **参考概念。** Oracle 以岗位职责组织功能权限的安全角色概念，与业务对象 Job 不同。 依据：[reference-model](#source-reference-model)。 |
| <a id="term-x03-functional-permission"></a>Functional Permission | 功能权限 | 执行某类功能或操作的许可，仍需同时满足数据范围和其他限制。 |
| <a id="term-x03-hcm-data-role"></a>HCM Data Role | HCM数据角色 | **参考概念。** Oracle 将职责角色与数据范围组合的安全概念；本设计通过角色授权完整绑定职责及范围。 依据：[reference-model](#source-reference-model)。 |
| <a id="term-x03-security-profile"></a>Security Profile | 安全配置文件 | **参考概念。** Oracle 用于限定可访问人员、组织等数据范围的配置；本设计对应数据范围和字段策略。 依据：[reference-model](#source-reference-model)。 |
| <a id="term-x03-person-level-access"></a>Person-level Access | 人员级访问 | 对人员基本对象的访问许可，不能自动扩展为该人员所有工作关系或敏感结果的权限。 |
| <a id="term-x03-relationship-level-access"></a>Relationship-level Access | 工作关系级访问 | 限定到人员某段工作关系的数据访问范围，可与人员级及任职级权限不同。 |
| <a id="term-x03-authentication"></a>Authentication | 身份认证 | 确认访问主体身份的过程；认证成功不等于拥有某个业务操作权限。 |
| <a id="term-x03-execution-permission"></a>Execution Permission | 执行许可 | 针对特定业务动作及对象状态允许执行的权限条件，与适用性和业务批准分别核验。 |
| <a id="term-x03-approval"></a>Approval<br>别名／原称：核准 | 审批 | 由规定责任方对业务提案作出批准或拒绝决定的过程，批准不等同于业务已经生效。 |
| <a id="term-x03-decisionrequirement"></a>DecisionRequirement | 业务决定要求 | 领域规定某操作所需证据、职责、政策版本及职责分离条件的描述。 |
| <a id="term-x03-decisionrecord"></a>DecisionRecord | 业务决定记录 | 目标域保存的批准、拒绝或规则判断及其对象版本、依据、有效期和撤销关系，不归任务设施维护。 |

<a id="domain-x04"></a>

## X04 任务、作业与人工核对

业务依据与详细契约：[X04模块说明](modules/x04.md)。

| 术语 | 中文名称 | 定义 |
| --- | --- | --- |
| <a id="term-x04-technicaltaskprojection"></a>TechnicalTaskProjection<br>别名／原称：TaskProjection | 技术任务视图 | 用于调度和查看执行进展的任务投影，不拥有员工历程或业务清单状态。 |
| <a id="term-x04-job"></a>Job | 技术作业 | 由任务设施调度执行的技术工作单元，与人事结构中的职务 Job 无关。 |
| <a id="term-x04-attempt"></a>Attempt | 作业执行尝试 | 同一技术作业的一次执行及检查点记录，重试不产生新的业务身份。 |
| <a id="term-x04-delivery"></a>Delivery | 事件投递 | 把源域事件发送到指定消费者的技术过程及记录，送达不代表业务已采用。 |
| <a id="term-x04-processingevidence"></a>ProcessingEvidence | 处理证据 | 记录消费者确认、失败及重放位置等技术证据，不能代替业务结果。 |
| <a id="term-x04-notificationrequest"></a>NotificationRequest | 通知请求 | 来源域要求向指定受众发送某内容的技术请求，内容和受众仍归业务域。 |
| <a id="term-x04-deliveryreceipt"></a>DeliveryReceipt | 通知送达回执 | 渠道返回的通知送达结果，不代表收件人已完成业务办理。 |
| <a id="term-x04-outbox"></a>Outbox | 事务发件箱 | 与源业务事实在同一事务记录的待发布事件或执行意图，供后续可靠投递。 |
| <a id="term-x04-inbox"></a>Inbox | 事务收件箱 | 消费者用于去重和记录事件处理状态的持久记录，与本地业务更新保持一致。 |
| <a id="term-x04-acknowledgement"></a>Acknowledgement<br>别名／原称：Ack | 处理确认 | 消费者对特定消息或步骤返回的确认信息，不当然证明金额已采用或业务已完成。 |
| <a id="term-x04-consumer"></a>Consumer<br>别名／原称：消费者 | 数据使用方 | 接收并采用其他领域数据的系统或模块；技术消息处理语境也可称消费者。 |

<a id="domain-x05"></a>

## X05 导入导出与外部集成

业务依据与详细契约：[X05模块说明](modules/x05.md)。

| 术语 | 中文名称 | 定义 |
| --- | --- | --- |
| <a id="term-x05-endpoint"></a>Endpoint | 集成端点 | 标识目标系统、环境和调用位置的配置，凭据应在安全存储中管理。 |
| <a id="term-x05-mappingversion"></a>MappingVersion | 接口映射版本 | 定义来源与目标字段、代码和格式转换的版本化配置，不能替代业务规则。 |
| <a id="term-x05-importbatch"></a>ImportBatch | 导入批次 | 对一组外部数据进行接收、校验及采用跟踪的批次，包含逐项结果和错误。 |
| <a id="term-x05-observation"></a>Observation | 外部观察记录 | 对外部返回数据或状态的不可变记录，只有经所属领域核验采用才成为本地业务事实。 |
| <a id="term-x05-executionintentref"></a>ExecutionIntentRef | 外部执行意图引用 | 指向来源域已提交执行意图的引用，适配器不能据此自行生成新的业务授权。 |
| <a id="term-x05-channelattempt"></a>ChannelAttempt | 渠道执行尝试 | 适配器执行一次外部调用的技术记录，保留原业务身份、请求和结果关联。 |
| <a id="term-x05-integrationreceipt"></a>IntegrationReceipt<br>别名／原称：Receipt | 接口回执 | 外部系统对某请求返回的状态或结果，需去重、核验并交回所属业务域处理。 |
| <a id="term-x05-reconciliation"></a>Reconciliation | 对账与核对 | 把内部记录与外部回执或资金事实逐项比较，识别差异、冲突及未知结果的过程。 |
| <a id="term-x05-oauth"></a>OAuth | OAuth授权协议 | 用于受控授权访问资源的协议；本身不等同于用户身份认证或业务审批。 |
| <a id="term-x05-availableflag"></a>AvailableFlag | 目录可用标志 | **参考概念。** FSCR参考目录中表示条目可用性的标志，不直接构成China HCM业务能力，也不证明已获得权限或存在可执行的实施路径。 依据：[reference-model](#source-reference-model)。 |
| <a id="term-x05-csv"></a>CSV | 逗号分隔值文件 | 用行列文本交换数据的文件格式。FSCR参考目录中的CSV支持标志是能力声明；格式或文件存在不代表字段语义、导入能力及执行许可已经满足。 依据：[reference-model](#source-reference-model)。 |
| <a id="term-x05-execution-mode"></a>execution_mode | 执行模式 | **参考概念。** FSCR技能声明的建议、辅助、自动或混合执行方式，须结合路由、前置检查和审批要求理解；它与China HCM机构结算的direct/externalFinance执行模式是不同概念。 依据：[reference-model](#source-reference-model)、[ddd-contracts](#source-ddd-contracts)。 |
| <a id="term-x05-unknown-outcome"></a>Unknown Outcome<br>别名／原称：unknown | 执行结果未知 | 请求可能已外发但尚不能确认结果的状态，应保持原身份查询核对，不能直接当失败重发。 |

<a id="domain-x06"></a>

## X06 报表、指标与事实追溯

业务依据与详细契约：[X06模块说明](modules/x06.md)。

| 术语 | 中文名称 | 定义 |
| --- | --- | --- |
| <a id="term-x06-metricdefinition"></a>MetricDefinition | 指标口径定义 | 规定指标业务含义、统计日期、来源、去重和计量单位的版本化定义。按指标类型声明统计时点或期间、人员范围，并在适用时说明分子分母及平均方法；人数、人次及按人或按次平均分别定义。 依据：[cr2018](#source-cr2018)。 原文定位：华润PDF第281页（ID_CRH0_HR_010001）；华润PDF第282页（ID_CRH0_HR_010008）；华润PDF第299页（ID_CHA0_HR_060011）。 |
| <a id="term-x06-authorizedprojection"></a>AuthorizedProjection | 授权分析视图 | 按当前访问范围从领域事实构建的可重建分析结果，保留来源版本及权限依赖。 |
| <a id="term-x06-reportsnapshot"></a>ReportSnapshot | 报表快照 | 按指定口径和来源版本形成的报告结果，重新读取仍需检查当前权限。 |
| <a id="term-x06-lineage"></a>Lineage | 数据来源追溯 | 从报表或指标定位到采用的源事实、规则及版本的关系，说明结果如何形成。 |
| <a id="term-x06-exportrecord"></a>ExportRecord | 报表导出记录 | 保存导出字段、范围、用途和审计依据的记录，文件下载仍受当前授权约束。 |
| <a id="term-x06-headcount"></a>Headcount | 人数 | 在约定人员范围、统计日期或期间及身份口径下去重计量的人员数量；与任职数、已占编制分别统计。 依据：[cr2018](#source-cr2018)。 原文定位：华润PDF第281页（ID_CRH0_HR_010001）；华润PDF第298页（ID_CHA0_HR_060002）。 |
| <a id="term-x06-period-end-headcount"></a>Period-end Headcount | 期末人数 | 在指定报告期末时点、人员范围和去重规则下计量的人员数量；不能代替整个期间的平均人数。 依据：[cr2018](#source-cr2018)。 原文定位：华润PDF第281页（ID_CRH0_HR_010001）。 |
| <a id="term-x06-average-headcount"></a>Average Headcount | 平均人数 | 按明确统计期间、人员范围和平均方法计算的人员数量指标；日平均、期初期末平均等方法必须写入采用的口径版本。 依据：[cr2018](#source-cr2018)。 原文定位：华润PDF第282页（ID_CRH0_HR_010008）。 |
| <a id="term-x06-participation-count"></a>Participation Count | 人次 | 按明确参与事件及有效性规则计量的次数；同一人员参与不同有效事件可重复计数，与按人员去重的人数分开。 依据：[cr2018](#source-cr2018)。 原文定位：华润PDF第299页（ID_CHA0_HR_060010）。 |
| <a id="term-x06-total-labor-cost"></a>Total Labor Cost | 人工成本总额 | 按明确组织、期间、人员范围及成本科目口径汇总的人工成本金额；与单笔人工成本事实、工资应付及发薪金额分开，组成以采用的口径版本为准。 依据：[cr2018](#source-cr2018)。 原文定位：华润PDF第284页（ID_CRH0_HR_030001）。 |

## 命名依据与版本边界

定义首先依据当前模块和目标契约，参考概念通过07保留与来源模型的差异。Oracle中国官网已核对的中文名称与项目根据业务语境选用的中文分别记录；当前英文概念文档不构成官方中文UI译名证明。本表未核验完整的26D简体中文界面术语包，也未把旧PeopleSoft译名作为Fusion当前界面标准。

2026-10-30复核官方Readiness及当前书目，相关HR、Payroll、Time/Absence、Talent/Learning发布入口显示26D；概念页来自当前累计文档渠道。项目定义不宣称与Oracle数据库对象或全部行为等价，中国政策适用证据另见[04](04-china-localization.md)。

<a id="source-reference-model"></a>

- [HCM 核心概念基准](../../reference/not-imported.md#source-a9f68bab28ef)：来源模型及概念导航。

<a id="source-readiness"></a>

- [Oracle HCM Readiness](https://docs.oracle.com/en/cloud/saas/readiness/hcm.html)：相关 HR、Payroll、Time/Absence、Talent/Learning 入口显示 26D；非租户版本证明。 核对日期：2026-10-30。

<a id="source-hr-books"></a>

- [Oracle Human Resources 当前书目](https://docs.oracle.com/en/cloud/saas/human-resources/books.html)：当前累计文档入口；无季度路径不等于不可变26D快照。 核对日期：2026-10-30。

<a id="source-talent-books"></a>

- [Oracle Talent Management 当前书目](https://docs.oracle.com/en/cloud/saas/talent-management/books.html)：来源模型及概念导航。 核对日期：2026-10-30。

<a id="source-cn-analytics"></a>

- [Oracle 中国 HCM Analytics](https://www.oracle.com/cn/fusion-data-intelligence/hcm-analytics/)：中文页面使用职务系列；只核对名称，不宣称全部UI标签一致。 核对日期：2026-10-30。

<a id="source-cn-recruiting"></a>

- [Oracle 中国 Recruiting](https://www.oracle.com/cn/human-capital-management/recruiting/)：中文页面使用求职申请、录用通知书；方案对象与正式文件区分。 核对日期：2026-10-30。

<a id="source-assignments"></a>

- [Oracle Assignments](https://docs.oracle.com/en/cloud/saas/human-resources/fawhr/assignments.html)：当前累计概念资料，支持主要任职层级；中文为项目选用。 核对日期：2026-10-30。

<a id="source-rates"></a>

- [Oracle Rate Definitions](https://docs.oracle.com/en/cloud/saas/human-resources/faigp/overview-of-rate-definitions.html)：Rate含金额标准、单价和比例；中文按本项目语境选用。 核对日期：2026-10-30。

<a id="source-ddd-contracts"></a>

- [China HCM DDD 与状态合同](10-ddd-and-state-contracts.md)：来源模型及概念导航。

<a id="source-cn-tax-terms"></a>

- [国家税务总局：中华人民共和国个人所得税法](https://www.chinatax.gov.cn/n810219/n810744/n3752930/n3752974/c3970366/content.html)：核对扣缴义务人、扣缴申报和缴纳税款等用语；本轮不据此变更政策计算规则。 核对日期：2026-10-30。

<a id="source-software-contract"></a>

- [Apache ServiceComb：使用服务契约](https://servicecomb.apache.org/cn/docs/users/use-service-contract/)：仅用于核对软件服务/接口使用“契约”的中文表达，不据此引入实现框架。 核对日期：2026-10-30。

<a id="source-cr2018"></a>

- [华润集团数据标准（第一批），2018-03-27](#cr2018-review)：用户提供的企业标准；采纳通用名称和概念区分，定义按项目语境整理。逐章评估保留原件散列、PDF页及行定位，不把企业代码和旧政策视作现行全国标准。 核对日期：2026-10-30。

<a id="source-cn-labor-contract-law-009"></a>

- [中华人民共和国劳动合同法（第二、七、十、五十八、五十九条）](https://fgk.chinatax.gov.cn/zcfgk/c100009/c5193025/content.html)：国家税务总局法规库标注全文有效；本轮核对用人单位、用工建立关系及派遣双方角色，不扩张为全部劳动法实施结论。 核对日期：2026-10-30。

<a id="source-cn-labor-contract-regulations-009"></a>

- [中华人民共和国劳动合同法实施条例（第三、四条）](https://xzfg.moj.gov.cn/front/law/detail?LawID=284)：司法部行政法规库；区分合伙等组织、取得证照的分支机构与受委托签约。 核对日期：2026-10-30。

<a id="source-oracle-legal-employer-009"></a>

- [Oracle: What is a legal employer?](https://docs.oracle.com/en/cloud/saas/human-resources/faigh/what-s-a-legal-employer.html)：经当前书目定位的累计文档渠道；Legal Employer 在工作关系层记录。HR最新Readiness入口为26D，不将产品字段直接等同中国法律角色。 核对日期：2026-10-30。

<a id="source-oracle-work-relationships-009"></a>

- [Oracle: Work Relationships](https://docs.oracle.com/en/cloud/saas/human-resources/fawhr/work-relationships.html)：当前累计文档；Employee、Contingent Worker、Nonworker 等是产品类型，Nonworker可记录非工作关联。 核对日期：2026-10-30。

<a id="source-oracle-assignments-zh-009"></a>

- [Oracle 简中：分配](https://docs.oracle.com/zh-cn/cloud/saas/human-resources/fawhr/assignments.html)：同页出现合法雇主与法定雇主；证明译名用例，不证明完整界面译名唯一。 核对日期：2026-10-30。

<a id="source-chnhcm-m02-terms-011"></a>

- [CHNHCM-011 M02 首批操作分类、日期、记录与版本术语](../../specs/m02/011-m02-business-requirements.md)：2026-10-31登记分类、日期及记录处理用语，统一删除并采用处理继承分类、自动留痕；2026-11-01按020允许已生效普通/停用删除及所有记录改日，首条禁止删除但允许改日。M02新业务选择原因，普通更正/改日/删除继承原分类且无处理原因；普通入口不允许分类改写；022专用分类纠错允许新分类，整对象删除仍不纳入。现行业务合同由011持有，不引入办理依据或可选说明。英文为项目说明名，不宣称Oracle官方译名或已实现接口；范围采纳不等于运行验收。 现按021纳入多次启停和重新启用的独立资料快照，统一三类日期新增，删除/改日按完整时间线和引用类别验证。 022进一步纳入有限联合修订、首条历史补全及显式分类纠错。 核对日期：2026-11-01。

<a id="source-chnhcm-record-maintenance-020"></a>

- [CHNHCM-020 M02 已生效记录删除与生效日期更正](../../adr/020-effective-record-delete-and-redate.md)：用户明确允许已生效记录删除和改日，并选择首条删除保护、允许首条改日；旧来源、修订及原采用保持，仅调整合同，记录维护尚未实施。 核对日期：2026-11-01。

<a id="source-chnhcm-bu-lifecycle-021"></a>

- [CHNHCM-021 M02 多次启停与记录维护规则](../../adr/021-bu-lifecycle-and-record-maintenance.md)：2026-11-01用户采纳独立重新启用、多次启停及完整时间线修订，并明确新增资料变更/停用/重新启用均允许过去、当天和未来。首条删除保护、旧来源及原采用保持；仅更新合同，运行能力待实施。 核对日期：2026-11-01。

<a id="source-chnhcm-joint-timeline-022"></a>

- [CHNHCM-022 M02 有限联合修订与历史补全](../../adr/022-joint-timeline-revision.md)：用户采纳同一BU有限显式项按最终合法候选一次保存、固定首条历史补全及分类纠错；普通处理仍默认继承分类，旧来源及后继独立快照保持。仅合同，运行待实施。 核对日期：2026-11-01。

<a id="source-chnhcm-m02-dated-maintenance-117"></a>

- [CHNHCM-117 M02共用按日维护与部门边界](../../adr/117-m02-dated-maintenance-and-department-boundary.md)：R0将M02显示称谓统一为状态，含义不变；新数据更改/原因目录为R1切换目标，部门独立表及最小路径由后续阶段实现。 核对日期：2026-11-04。

## 中文命名建议的取舍

2026-10-30对[外部命名建议](https://chatgpt.com/share/6ae44b0d-0eb8-83ee-9445-a6373e3f9f00)逐项评估。建议基于402词条旧版，本次以已有整合修订的405词条及当前模块契约为准。分领域标题仅优化本表导航，不改变模块编号、职责或原图标题；同源对象链接按本表更新。原有独立评审与冻结快照仍只证明各自输入。

- 优先消除业务误解：收付款方向、内部台账与银行账户、参数与结果、可计量额度与广义权益、软件契约与人员合同。
- 保留大陆常用名称、俗称和简称，包括法律实体、参缴、个税、HR及常用架构术语；不以更长、更正式作为更名理由。
- 名称不扩大或缩小业务范围。规则与策略按用途选词，政策与制度仍表示依据；Plan、Account和Record不作全局机械替换。
- 继续使用“术语、中文名称、定义”三列；现有别名和语境承载检索需要，不增加尚无具体页面依据的显示名称字段。中文更名保留旧名检索、稳定ID和锚点；场景限定词不自动登记为全域同义词。
- 2026-10-31 增补（[CHNHCM-015](../../governance/015-terminology-naming.md)）：中文名优先采用双字或四字，一般不超过六个字；超过四字一般为含义明确的组合词汇。例外限于有明确权威来源或使用惯例的名称，须在词条来源及说明中记录依据、适用语境和保留理由；不以缩短名称改变业务含义，也不将既有长名称自动认定为例外。

### 分领域标题

| 领域 | 本表标题 | 处理依据 |
| --- | --- | --- |
| [M03](#domain-m03) | 职务、职级、岗位与编制 | 采纳岗位，与Position首选中文一致。 |
| [M04](#domain-m04) | 用工、任职与劳动合同 | 暂保留劳动合同；当前模块没有明确扩展劳务合同的独立规则，不能仅凭通用PersonContract定义扩大范围。 |
| [M14](#domain-m14) | 工资支付与收付款账户 | 采纳收付款账户，明确“银行用途”所指；内部结算台账另在词条中区分。 |
| [M20](#domain-m20) | 员工与经理服务及HR任务 | 补“与”消除“员工经理”连读，保留经理和HR常用称呼，不强制扩写。 |
| [X01](#domain-x01) | 对象标识、有效期、修订与采用历史 | 对象标识与访问认证身份分开；保留有效期、采用历史的简洁表达。 |
| [X02](#domain-x02) | 共享配置、适用范围、默认值与政策版本 | 补成默认值，直接点明配置对象。 |
| [X03](#domain-x03) | 身份、权限与隐私 | 以顿号分开身份与权限；身份管理不缩成只有身份认证。 |

### 词条建议

| 涉及词条（当前首选中文） | 处理 | 理由 |
| --- | --- | --- |
| [BankAccountIdentity（个人银行账户）](17-terminology-glossary.md#term-m01-bankaccountidentity) / [Name Style（姓名结构）](17-terminology-glossary.md#term-m01-name-style) | 采纳 | 把账户稳定标识留在定义中；姓名结构与显示格式分开，消除“账户身份”和“样式”的误读。 |
| [PersonNumberAlias（人员编号历史别名）](17-terminology-glossary.md#term-m01-personnumberalias) | 保留 | 外部建议依据旧版定义；当前已明确只登记永久旧号，不是当前编号的期间使用记录，故不改为“人员编号使用记录”。 |
| [OrganizationHierarchy（组织架构）](17-terminology-glossary.md#term-m02-organizationhierarchy) / [HierarchyVersion（组织架构版本）](17-terminology-glossary.md#term-m02-hierarchyversion) / [PositionHierarchy（岗位层级关系）](17-terminology-glossary.md#term-m03-positionhierarchy) / [PositionHierarchyVersion（岗位层级关系版本）](17-terminology-glossary.md#term-m03-positionhierarchyversion) | 采纳 | 组织架构保留多用途及有效版本；岗位层级关系与岗位等级、实际汇报关系分开。 |
| [ContractProcess（合同办理记录）](17-terminology-glossary.md#term-m04-contractprocess) / [BusinessActionRecord（人事业务办理记录）](17-terminology-glossary.md#term-m04-businessactionrecord) / [EmploymentActionCatalog（人事业务配置目录）](17-terminology-glossary.md#term-m04-employmentactioncatalog) / [ActionDefinition（人事业务类型）](17-terminology-glossary.md#term-m04-actiondefinition) / [ReasonDefinition（人事业务原因）](17-terminology-glossary.md#term-m04-reasondefinition) / [HireHandoff（入职信息交接）](17-terminology-glossary.md#term-m05-hirehandoff) | 采纳 | 区分人事业务办理、配置类型、系统操作日志及入职信息交接，保留人员合同的法律业务含义。 |
| [WorkPattern（排班循环模式）](17-terminology-glossary.md#term-m06-workpattern) / [TimeEvent（原始考勤与工时事件）](17-terminology-glossary.md#term-m07-timeevent) / [Device（考勤采集设备）](17-terminology-glossary.md#term-m07-device) / [TimeSourceDefinition（考勤与工时来源配置）](17-terminology-glossary.md#term-m07-timesourcedefinition) | 调整后采纳 | 明确排班循环和考勤/工时来源，设备用简洁的“考勤采集设备”；定义继续覆盖打卡以外的采集，不扩写成冗长全称。 |
| [ScheduleBook（任职排班登记簿）](17-terminology-glossary.md#term-m06-schedulebook) | 保留 | “登记簿”与“台账”均能表达权威登记，没有仅为用词偏好更换的必要。 |
| [AbsencePlan（休假与缺勤方案）](17-terminology-glossary.md#term-m08-absenceplan) / [AbsenceEnrollment（休假与缺勤方案参与记录）](17-terminology-glossary.md#term-m08-absenceenrollment) | 采纳 | 此处是可复用规则方案，与个人准备何时休假分开；不据此把全表的Plan一律改为方案。 |
| [EntitlementAccount（休假额度账户）](17-terminology-glossary.md#term-m08-entitlementaccount) / [GrantLot（休假额度发放批次）](17-terminology-glossary.md#term-m08-grantlot) / [EntitlementEntry（休假额度变动流水）](17-terminology-glossary.md#term-m08-entitlemententry) / [ConsumptionAllocation（休假额度扣减分配）](17-terminology-glossary.md#term-m08-consumptionallocation) / [NoEntitlement（仅登记缺勤模式）](17-terminology-glossary.md#term-m08-noentitlement) / [Entitlement（休假权益）](17-terminology-glossary.md#term-m08-entitlement) | 部分采纳 | 额度账户、发放及扣减使用可计量的“额度”；保留广义“休假权益”。仅登记缺勤不等于无休假权或无薪；扣减保留“分配”，无需另加“分摊记录”。 |
| [RunType（工资核算类型）](17-terminology-glossary.md#term-m10-runtype) / [PayrollInput（计薪输入）](17-terminology-glossary.md#term-m10-payrollinput) / [ElementEntry（工资项目输入记录）](17-terminology-glossary.md#term-m10-elemententry) / [Calculation Value（计薪参数值）](17-terminology-glossary.md#term-m10-calculation-value) / [Time Definition（计薪日期与期间规则）](17-terminology-glossary.md#term-m10-time-definition) / [PayslipPublication（工资单发布记录）](17-terminology-glossary.md#term-m10-payslippublication) | 调整后采纳 | 区分核算类型、输入记录、参数和发布记录；“计薪输入”已清楚，不强制增加“数据”后缀。 |
| [WithholdingRegistration（扣缴义务人登记）](17-terminology-glossary.md#term-m11-withholdingregistration) / [TaxDeduction（个税扣除信息）](17-terminology-glossary.md#term-m11-taxdeduction) / [TaxAssessment（个税计算结果）](17-terminology-glossary.md#term-m11-taxassessment) / [TaxFiling（个税扣缴申报）](17-terminology-glossary.md#term-m11-taxfiling) / [IncomeItemPlan（所得明细处理计划）](17-terminology-glossary.md#term-m11-incomeitemplan) / [PaymentTreatmentDecision（付款涉税处理认定）](17-terminology-glossary.md#term-m11-paymenttreatmentdecision) / [TaxRemittanceInstruction（税款缴纳指令）](17-terminology-glossary.md#term-m11-taxremittanceinstruction) | 采纳 | 使用扣缴义务人及个税扣缴用语，明确计算结果与所得明细；系统登记、所得确认、申报接受和实缴仍分别记录，不扩展个人年度汇算范围。 |
| [EmployeeReceivable（应收个人缴费款）](17-terminology-glossary.md#term-m12-employeereceivable) / [AgencyPayableAccount（应付经办机构缴款台账）](17-terminology-glossary.md#term-m12-agencypayableaccount) / [BenefitSupplierPayable（应付福利供应商款）](17-terminology-glossary.md#term-m13-benefitsupplierpayable) / [TaxRemittanceAccount（应缴税款台账）](17-terminology-glossary.md#term-m11-taxremittanceaccount) / [PayableAccount（工资应付结算台账）](17-terminology-glossary.md#term-m14-payableaccount) / [RecoveryAccount（工资超付追收台账）](17-terminology-glossary.md#term-m14-recoveryaccount) | 采纳 | 明确单位向个人应收、向机构或供应商应付的方向；与真实资金账户相邻的内部账使用台账，未把所有Account统一改名。 |
| [LineReceipt（参缴申报明细回执）](17-terminology-glossary.md#term-m12-linereceipt) / [ContributionSettlement（参缴实缴与退款记录）](17-terminology-glossary.md#term-m12-contributionsettlement) / [CollectionAllocation（个人缴费扣收分配）](17-terminology-glossary.md#term-m12-collectionallocation) / [PayerExecutionHold（付款账户支付限制）](17-terminology-glossary.md#term-m14-payerexecutionhold) | 调整后采纳 | 行回执改为明细回执，资金记录明确实缴/退款及个人扣收方向；保留“参缴”合称。支付限制按渠道生效，不改写成银行冻结或已经撤回付款。 |
| [CostFact（人工成本事实）](17-terminology-glossary.md#term-m15-costfact) / [JournalRequest（会计分录请求）](17-terminology-glossary.md#term-m15-journalrequest) / [AccountingReceipt（会计回执）](17-terminology-glossary.md#term-m15-accountingreceipt) | 保留 | “事实”“请求”“回执”在架构词典中含义清楚，定义已说明确认和处理边界；无需统一增加“已确认”“提交”“财务系统处理”等限定。 |
| [ContentType（人才档案内容类型）](17-terminology-glossary.md#term-m16-contenttype) / [ContentItem（人才档案内容项）](17-terminology-glossary.md#term-m16-contentitem) / [ProfileSection（档案栏目）](17-terminology-glossary.md#term-m16-profilesection) / [SectionDefinition（档案栏目定义）](17-terminology-glossary.md#term-m16-sectiondefinition) / [QuestionnaireResponse（问卷作答记录）](17-terminology-glossary.md#term-m16-questionnaireresponse) / [LearningAudience（学习适用人群）](17-terminology-glossary.md#term-m18-learningaudience) / [LearningAssignment（学习任务）](17-terminology-glossary.md#term-m18-learningassignment) | 采纳 | 档案栏目和问卷作答记录更自然；学习任务表达持续要求，分配任务是操作，不改变其可早于报名存在的边界。 |
| [Identity（稳定对象标识）](17-terminology-glossary.md#term-x01-identity) / [Record Deletion（删除记录）](17-terminology-glossary.md#term-x01-future-slice-deletion) / [Parent-child Date Alignment（父子有效期约束）](17-terminology-glossary.md#term-x01-parent-child-date-alignment) / [Public Contract（对外业务契约）](17-terminology-glossary.md#term-x01-public-contract) | 采纳 | 区分稳定对象标识与认证身份；未来变更撤销保留审计；父子有效期不要求完全相等；软件契约与人员合同分开。 |
| [Dated Slice（有效期切片）](17-terminology-glossary.md#term-x01-dated-slice) / [Effective Date Query（按生效日期查询）](17-terminology-glossary.md#term-x01-effective-date-query) / [Known-at Time（已知信息截点）](17-terminology-glossary.md#term-x01-known-at-time) | 保留日期用语；退役截点 | 有效期切片是常用技术简称，当前区分业务日期、确切来源及采用版本；Known-at Time 仅保留历史定位，不能作为现行查询能力。 |
| [StaffingPolicy（任职与编制管理规则）](17-terminology-glossary.md#term-m03-staffingpolicy) / [WorkingTimePolicy（工时制度规则）](17-terminology-glossary.md#term-m06-workingtimepolicy) / [PrimarySelectionPolicy（主要关系与任职选择规则）](17-terminology-glossary.md#term-m04-primaryselectionpolicy) / [PayrollImpactPolicy（工资变更影响规则）](17-terminology-glossary.md#term-m10-payrollimpactpolicy) / [ApplicabilityPolicy（适用范围规则）](17-terminology-glossary.md#term-x02-applicabilitypolicy) / [RetentionPolicy（数据保留策略）](17-terminology-glossary.md#term-x03-retentionpolicy) | 调整后采纳 | 可执行判断用规则，系统控制用策略；主选择及工资影响名称保留必要对象，不堆叠“确定/判定”等同义限定。 |
| [TimePolicy（工时处理规则）](17-terminology-glossary.md#term-m07-timepolicy) / [ConversionProrationPolicy（计薪换算与分摊规则）](17-terminology-glossary.md#term-m10-conversionprorationpolicy) / [PolicyVersion（政策版本）](17-terminology-glossary.md#term-x02-policyversion) / [PolicyEvidence（政策来源证据）](17-terminology-glossary.md#term-x02-policyevidence) | 同类一致处理 | 工时处理及新增计薪换算/分摊对象同属可执行规则；政策版本和政策证据仍表达制度依据，保留原名。计薪换算和分摊保留完整期分母及舍入规则。 |
| [Legal Entity（法律实体）](17-terminology-glossary.md#term-m02-legal-entity) / [Legal Employer（用人单位）](17-terminology-glossary.md#term-m02-legal-employer) | 保留法律实体；大陆劳动关系语境采用用人单位 | CHNHCM-009更新：人员类型与组织责任是不同维度，支持非员工不是保留法定雇主为首选中文的理由。用人单位、派遣接收用工单位和非劳动关联分别定义；原称保留为Oracle对照及检索别名。 |
| [StaffingQuota（编制额度）](17-terminology-glossary.md#term-m03-positioncapacity) / [PositionReservation（招聘编制预留）](17-terminology-glossary.md#term-m03-positionreservation) / [PositionOccupancy（任职占编）](17-terminology-glossary.md#term-m03-positionoccupancy) / [GradeLadder（职级阶梯）](17-terminology-glossary.md#term-m03-gradeladder) | 按现行合同修订 | 部门和岗位额度按整数名额核定，人员编制计划另记目标与预算。职级阶梯表述进阶关系。 |
| [CompensationPlan（薪酬计划）](17-terminology-glossary.md#term-m09-compensationplan) | 保留 | 薪酬计划沿用企业熟悉的名称。 |
| [PayrollDefinition（工资核算配置）](17-terminology-glossary.md#term-m10-payrolldefinition) / [PayrollRun（工资核算批次）](17-terminology-glossary.md#term-m10-payrollrun) | 采纳 | 工资核算配置比“工资定义”清楚；当前runId可组织一次核算业务批次，技术重试不另建业务批次，未改结果及累计的独立归属。 |
| [TaxCommitment（所得确认控制记录）](17-terminology-glossary.md#term-m11-taxcommitment) / [DispatchPermit（涉税付款执行许可）](17-terminology-glossary.md#term-m11-dispatchpermit) | 采纳 | 控制记录保留整组所得顺序、确认、取消及许可约束；明确是系统税务域的许可，不是纳税承诺书、税务机关许可或付款审批。 |
| [PerformanceDocument（绩效评价文档）](17-terminology-glossary.md#term-m17-performancedocument) / [Journey（员工历程任务清单）](17-terminology-glossary.md#term-m20-journey) / [JourneyInstance（员工历程实例）](17-terminology-glossary.md#term-m20-journeyinstance) / [JourneyTemplate（员工历程模板）](17-terminology-glossary.md#term-m20-journeytemplate) / [StepInstance（历程步骤实例）](17-terminology-glossary.md#term-m20-stepinstance) / [StepDefinition（历程步骤定义）](17-terminology-glossary.md#term-m20-stepdefinition) | 保留 | 绩效文档和员工历程在当前语境可理解；定义已说明业务记录及跨模块清单性质，不为假设中的页面统一改成表单或办理清单。 |
| [PayableSourceHead（工资应付来源版本链头）](17-terminology-glossary.md#term-m10-payablesourcehead) / [ResultSeriesHead（工资结果版本链头）](17-terminology-glossary.md#term-m10-resultserieshead) | 采纳技术名称 | 补明版本链头的职责；不把它们改成普通记录，也不预先把“当前确认工资结果”等场景显示名设为全域同义词。 |
| [Person（人员）](17-terminology-glossary.md#term-m01-person) / [Job（职务）](17-terminology-glossary.md#term-m03-job) / [Position（岗位）](17-terminology-glossary.md#term-m03-position) / [Assignment（任职）](17-terminology-glossary.md#term-m04-assignment) / [Primary Assignment（主要任职）](17-terminology-glossary.md#term-m04-primary-assignment) / [JobFamily（职务系列）](17-terminology-glossary.md#term-m03-jobfamily) / [Offer（录用方案）](17-terminology-glossary.md#term-m05-offer) | 按建议保留 | 人员、职务、岗位、任职、主要任职、职务系列和录用方案各有既定范围；“员工”“主岗”“录用通知书”等只适用于具体场景。 |
| [CompensationAward（个人薪酬核定）](17-terminology-glossary.md#term-m09-compensationaward) / [GradeRate（职级薪酬标准）](17-terminology-glossary.md#term-m09-graderate) / [Rate（计薪标准）](17-terminology-glossary.md#term-m09-rate) / [PayrollBalance（工资累计）](17-terminology-glossary.md#term-m10-payrollbalance) / [RetroCase（工资追溯事项）](17-terminology-glossary.md#term-m10-retrocase) / [Retro Action（追溯处理）](17-terminology-glossary.md#term-m10-retro-action) / [LearningOffering（课程开设安排）](17-terminology-glossary.md#term-m18-learningoffering) | 按建议保留 | 不将核定改为授予、累计改为余额、追溯改为补发、课程开设安排改为培训班；原名已保留实际业务范围。 |
| [RatingModel（评分模型）](17-terminology-glossary.md#term-m16-ratingmodel) / [Qualification（人员资格）](17-terminology-glossary.md#term-m16-qualification) / [Readiness（继任准备度）](17-terminology-glossary.md#term-m19-readiness) / [Adoption（版本采用）](17-terminology-glossary.md#term-x01-adoption) / [Adopted History（采用历史）](17-terminology-glossary.md#term-x01-adopted-history) / [Bounded Context（限界上下文）](17-terminology-glossary.md#term-x01-bounded-context) / [Aggregate（聚合）](17-terminology-glossary.md#term-x01-aggregate) / [Aggregate Root（聚合根）](17-terminology-glossary.md#term-x01-aggregate-root) / [Value Object（值对象）](17-terminology-glossary.md#term-x01-value-object) / [Idempotency（幂等）](17-terminology-glossary.md#term-x01-idempotency) / [Anti-Corruption Layer（防腐层）](17-terminology-glossary.md#term-x01-anti-corruption-layer) | 按建议保留 | 业务边界及成熟技术术语保持，采用不等于接收或同步；不因专业名称较少口语化而机械替换。 |
| [ContributionEnrollment（参保与缴存登记）](17-terminology-glossary.md#term-m12-contributionenrollment) / [ContributionFiling（参缴申报）](17-terminology-glossary.md#term-m12-contributionfiling) / [ObligationAccount（参缴义务账户）](17-terminology-glossary.md#term-m12-obligationaccount) / [ContributionPolicy（参缴规则）](17-terminology-glossary.md#term-m12-contributionpolicy) | 保留简称并注明语境 | 本表以“参缴”统称社保参保与公积金缴存，不在每个词条重复长名称；具体业务仍按社保或公积金使用对应称谓，申报与资金到账分开。 |

<a id="cr2018-review"></a>

## 华润数据标准逐章评估

2026-10-30对用户提供的《华润集团数据标准（第一批）》逐章核对。原件编制日期为2018-03-27，共323个PDF页；正文印刷页码＝PDF页码－2。

原件SHA256：`6b5d6bf936acc2efdb26144b699b231bbeb0c851aabc3a58a5bbeb99a4a55250`。

本地原材料目录：`/home/vivobook/projects/reference-materials/china-resources-data-standard-2018`，含原件、逐页原文、表格坐标、章节索引及SQLite/HTML检索。资料目录不参与本仓库生成或校验；本表保存冻结的来源身份与行定位。

- 先将完整PDF拆成逐页原文、带坐标的表格、22个章节入口及本地检索；323页、320张表、3628条带序号记录均保留。标准项实际提取1458条，其中人力基础83条、指标129条与原概述核对一致。
- 原文有16个跨表重用的标准编码，故以原件SHA256＋页/表/行定位；市场基础数据概述69条而明细连续1—60，差异已保留，未补造。该差异不影响人力部分数量核对。
- 逐章记录采纳与保留。新增项是帮助理解当前设计的通用业务概念，英文检索名称由项目整理，不宣称为原文或Oracle官方译名；不据词条新增业务对象、聚合、字段必填、审批流或计量算法。
- 法律实体、职务系列、岗位编制、薪酬计划、参缴、HR及成熟技术简称继续保留；旧参考法规、企业代码、科目树和统计公式均不自动成为当前大陆规则。
- 原文培训时长的8小时约定、个人社保与公积金合写字段、社保类型中的公积金项、线上线下人数相加及按人次平均等逐项判断；只采纳有用名称和概念区别。
- 本轮材料评估建立在此前405词条之上；此前命名建议、R24/R25等冻结评审的结论仍只覆盖各自输入，本次生成与材料定位校验不冒用其独立评审结论。

| 领域 | 采纳与调整 | 保留与适用边界 | 原文位置 |
| --- | --- | --- | --- |
| [M01](#domain-m01) | 补充员工编号，明确业务编号与稳定人员标识。<br>[Employee Number（员工编号）](17-terminology-glossary.md#term-m01-employee-number) | 保留人员、个人资料及人员编号历史别名；原文“人员ID”是员工编号常用名，8位流水规则及其与personId的等同关系不采纳。 | PDF第243页：DS_CRH0_HR_010001（p0243-t01-r005） |
| [M02](#domain-m02) | 补充统一社会信用代码。<br>[Unified Social Credit Code（统一社会信用代码）](17-terminology-glossary.md#term-m02-unified-social-credit-code) | 保留法律实体、业务单元和地点；按CHNHCM-009在大陆劳动关系语境采用用人单位，并区分用工单位及关联单位。工作地区、发薪地区和付款主体分别记录，不能自动决定税费或用人责任。 | PDF第5页：DS_CRH0_PB_030002（p0005-t01-r010）<br>PDF第243页：DS_CRH0_HR_010007（p0243-t01-r011）<br>PDF第243页：DS_CRH0_HR_010008（p0243-t01-r012） |
| [M03](#domain-m03) | 保留现有首选中文。 | 华润职类、职种和职位序列是企业分类实例。岗位编制的单位和使用按本仓编制管理合同确定。 | PDF第245页：DS_CRH0_HR_020001（p0245-t01-r006）<br>PDF第245页：DS_CRH0_HR_020002（p0245-t01-r007）<br>PDF第271页：DS_CHA0_HR_020003（p0271-t01-r006） |
| [M04](#domain-m04) | 补充入职日期、工龄起算日期。<br>[Hire Date（入职日期）](17-terminology-glossary.md#term-m04-hire-date) / [Service Start Date（工龄起算日期）](17-terminology-glossary.md#term-m04-service-start-date) | 集团加入日、单位加入日及工龄起点分别表达；员工状态不替代任职状态或关系性质。原文“劳动合同用工单位”保留来源措辞，项目按实际事实区分用人、派遣接收及签约角色；非劳动关联使用相应角色。续订可在办理语境称续签。 | PDF第243页：DS_CRH0_HR_010010（p0243-t01-r014）<br>PDF第243页：DS_CRH0_HR_010011（p0243-t01-r015）<br>PDF第270页：DS_CHA0_HR_010006（p0270-t01-r010）<br>PDF第245页：DS_CRH0_HR_010024（p0245-t01-r005） |
| [M05](#domain-m05) | 补充招聘计划、招聘渠道。<br>[Recruitment Plan（招聘计划）](17-terminology-glossary.md#term-m05-recruitment-plan) / [Recruitment Channel（招聘渠道）](17-terminology-glossary.md#term-m05-recruitment-channel) | 招聘计划与单项招聘需求分开；候选人可包括主动寻访对象，保留候选人、求职申请和录用方案。任职资格归要求语境。 | PDF第245页：DS_CRH0_HR_040002（p0245-t01-r013）<br>PDF第245页：DS_CRH0_HR_040003（p0245-t01-r014）<br>PDF第246页：DS_CRH0_HR_040007（p0246-t01-r007） |
| [M06](#domain-m06) | 补充工时制及工时制度别名。<br>[Working Time System（工时制）](17-terminology-glossary.md#term-m06-working-time-system) | 班次、排班循环、具体排班与工时制度规则分别维护；原文制度类别不直接成为当前政策配置。 | PDF第244页：DS_CRH0_HR_010022（p0244-t01-r015） |
| [M07](#domain-m07) | 保留现有考勤和工时术语。 | 计时工资、加班工资属于报酬及成本分类，不能据此替换工时记录、校验、确认或发布机制。 | PDF第255页：CD_CRH0_2301（p0255-t01-r018）<br>PDF第256页：CD_CRH0_2301（p0256-t01-r005） |
| [M08](#domain-m08) | 保留现有休假、缺勤及额度术语。 | 原文在工资说明中列举休假支付场景，未定义额度账户、结转、资格待遇或取消机制，不作相应更名。 | PDF第255页：CD_CRH0_2301（p0255-t01-r018） |
| [M09](#domain-m09) | 补充基本工资、津贴补贴、绩效工资和绩效奖金。<br>[Basic Salary（基本工资）](17-terminology-glossary.md#term-m09-basic-salary) / [Allowances and Subsidies（津贴补贴）](17-terminology-glossary.md#term-m09-allowances-and-subsidies) / [Performance Pay（绩效工资）](17-terminology-glossary.md#term-m09-performance-pay) / [Performance Bonus（绩效奖金）](17-terminology-glossary.md#term-m09-performance-bonus) | 保留薪酬计划、个人薪酬核定及计薪标准；不照搬华润基本工资的归并范围。绩效奖金原定义为残句，仅采纳用语，定义结合项目边界整理。 | PDF第255页：CD_CRH0_2301（p0255-t01-r009）<br>PDF第255页：CD_CRH0_2301（p0255-t01-r010）<br>PDF第255页：CD_CRH0_2301（p0255-t01-r019）<br>PDF第256页：CD_CRH0_2301（p0256-t01-r019） |
| [M10](#domain-m10) | 补明工资项目分类与人工成本科目的区别。<br>[Element Classification（工资项目分类）](17-terminology-glossary.md#term-m10-element-classification) | 保留工资项目、核算结果、累计、追溯及发布术语；工资核算结果不改为工资总额或总薪酬。 | PDF第245页：DS_CRH0_HR_030001（p0245-t01-r009） |
| [M11](#domain-m11) | 保留现有个税及执行许可术语。 | 原文个税主要见于财务现金流分类，不能作为扣缴、所得确认、税务控制记录或许可机制的等价定义。 | PDF第228页：ID_CRH0_FI_050573（p0228-t01-r009）<br>PDF第228页：ID_CRH0_FI_050574（p0228-t01-r010） |
| [M12](#domain-m12) | 将合写的个人编号拆成个人社保编号和个人公积金账号。<br>[Personal Social Insurance Number（个人社保编号）](17-terminology-glossary.md#term-m12-personal-social-insurance-number) / [Personal Housing Fund Account Number（个人公积金账号）](17-terminology-glossary.md#term-m12-personal-housing-fund-account-number) | 保留参保、缴存和参缴简称；不照搬原“社保类型”代码表把公积金并入其中的分类，单位经办账户和个人编号分开。 | PDF第245页：DS_CRH0_HR_030002（p0245-t01-r010）<br>PDF第245页：DS_CRH0_HR_030003（p0245-t01-r011）<br>PDF第260页：CD_CRH0_2302（p0260-t01-r004） |
| [M13](#domain-m13) | 保留企业福利、福利计划和福利参与记录。 | 家属商业保险可作为业务参考实例；原文公费/自费不足以替代保障方案及福利选项，也不据此引入现行个税处理结论。 | PDF第270页：DS_CHA0_HR_010002（p0270-t01-r006）<br>PDF第270页：DS_CHA0_HR_010003（p0270-t01-r007）<br>PDF第270页：DS_CHA0_HR_010004（p0270-t01-r008） |
| [M14](#domain-m14) | 补充银行账户名称（户名）、开户行，澄清账户支付用途。<br>[Bank Account Name（银行账户名称）](17-terminology-glossary.md#term-m14-bank-account-name) / [Bank Branch（开户行）](17-terminology-glossary.md#term-m14-bank-branch) / [PaymentUse（账户支付用途）](17-terminology-glossary.md#term-m14-paymentuse) | 保留机构付款账户、内部结算台账及付款账户支付限制；银行账户性质、开户用途、银行冻结及系统执行限制分别解释。 | PDF第45页：DS_CRH0_BS_010001（p0045-t01-r005）<br>PDF第45页：DS_CRH0_BS_010002（p0045-t01-r006）<br>PDF第45页：DS_CRH0_BS_010004（p0045-t01-r008）<br>PDF第45页：DS_CRH0_BS_010006（p0045-t01-r010） |
| [M15](#domain-m15) | 补充人工成本科目、会计科目。<br>[Labor Cost Item（人工成本科目）](17-terminology-glossary.md#term-m15-labor-cost-item) / [Accounting Account（会计科目）](17-terminology-glossary.md#term-m15-accounting-account) | 保留成本事实、成本分摊、会计映射和回执；成本归类可映射财务科目，不照搬企业科目层数或可过账层级。 | PDF第245页：DS_CRH0_HR_030001（p0245-t01-r009）<br>PDF第50页：DS_CRH0_FI_040006（p0050-t01-r007） |
| [M16](#domain-m16) | 补充任职资格、学历、学位、教育形式；在任职要求档案中澄清要求。<br>[Job Qualification Requirements（任职资格）](17-terminology-glossary.md#term-m16-job-qualification-requirements) / [Educational Attainment（学历）](17-terminology-glossary.md#term-m16-educational-attainment) / [Academic Degree（学位）](17-terminology-glossary.md#term-m16-academic-degree) / [Education Mode（教育形式）](17-terminology-glossary.md#term-m16-education-mode) | 保留人员资格及其认定事实；任职资格是要求。学历、学位和教育形式分别记录，旧代码值不作为当前统一枚举。 | PDF第246页：DS_CRH0_HR_040006（p0246-t01-r006）<br>PDF第4页：DS_CRH0_PB_010003（p0004-t01-r007）<br>PDF第4页：DS_CRH0_PB_010004（p0004-t01-r008）<br>PDF第244页：DS_CRH0_HR_010017（p0244-t01-r010） |
| [M17](#domain-m17) | 周期首选名调整为绩效考核周期；补绩效文档别名、绩效考核得分和等级。<br>[ReviewPeriod（绩效考核周期）](17-terminology-glossary.md#term-m17-reviewperiod) / [PerformanceDocument（绩效评价文档）](17-terminology-glossary.md#term-m17-performancedocument) / [Performance Score（绩效考核得分）](17-terminology-glossary.md#term-m17-performance-score) / [Performance Rating（绩效考核等级）](17-terminology-glossary.md#term-m17-performance-rating) | 考核周期与考核频率分开；得分与等级不等同评分模型，不采用原文仅限不参与结果分布场景的等级定义。绩效模板、反馈、校准和申诉保留。 | PDF第246页：DS_CRH0_HR_050001（p0246-t01-r009）<br>PDF第246页：DS_CRH0_HR_050002（p0246-t01-r010）<br>PDF第270页：DS_CHA0_HR_050001（p0270-t01-r013） |
| [M18](#domain-m18) | 补计划学习时长、实际培训时长；学习记录说明计划和实际分别计量。<br>[Planned Learning Duration（计划学习时长）](17-terminology-glossary.md#term-m18-planned-learning-duration) / [Actual Training Duration（实际培训时长）](17-terminology-glossary.md#term-m18-actual-training-duration) / [LearningRecord（学习记录）](17-terminology-glossary.md#term-m18-learningrecord) | 保留课程开设安排、学习任务和发展计划；8小时转换阈值属企业规则，总培训时间没有足够计量说明，不直接改成培训人时。 | PDF第246页：DS_CRH0_HR_060001（p0246-t01-r011）<br>PDF第246页：DS_CRH0_HR_060005（p0246-t01-r015）<br>PDF第298页：ID_CHA0_HR_060003（p0298-t01-r011） |
| [M19](#domain-m19) | 保留人才盘点会议、人才池及继任术语。 | 测评可用于招聘、晋升、选拔、发展和盘点，不能将人才测评设为人才盘点的全域别名；原文不足以调整继任准备度。 | PDF第247页：DS_CRH0_HR_080002（p0247-t01-r012） |
| [M20](#domain-m20) | 保留HR简称及现有服务、待办、员工历程名称。 | 通知、催办和具体申办事项可作实例，但不把所有服务请求改为审批申请；共享服务中心任务不等于员工历程步骤。 | PDF第248页：DS_CRH0_HR_080006（p0248-t01-r005）<br>PDF第301页：ID_CRH0_HR_100008（p0301-t01-r010） |
| [X01](#domain-x01) | 保留稳定对象标识、有效期、修订和采用历史。 | 任命和生效/失效日期是业务日期例子；原文没有等价的已知信息截点、修订或确切版本采用定义。 | PDF第247页：DS_CRH0_HR_070002（p0247-t01-r006）<br>PDF第247页：DS_CRH0_HR_070003（p0247-t01-r007）<br>PDF第247页：DS_CRH0_HR_070004（p0247-t01-r008）<br>PDF第247页：DS_CRH0_HR_070005（p0247-t01-r009） |
| [X02](#domain-x02) | 补充代码表、代码项；将代码值放在代码项定义中。<br>[Code List（代码表）](17-terminology-glossary.md#term-x02-code-list) / [Code Item（代码项）](17-terminology-glossary.md#term-x02-code-item) | 代码表、参考数据集和SetID不作同义替换；来源标准编码跨表重用，须结合章节、页和行定位，代码值不作全局唯一身份。 | PDF第4页：DS_CRH0_PB_010001（p0004-t01-r005）<br>PDF第6页：CD_CRH0_0101（p0006-t01-r004）<br>PDF第6页：CD_CRH0_0101（p0006-t01-r005） |
| [X03](#domain-x03) | 保留身份认证、权限、访问范围和隐私术语。 | 原文分类、责任部门、敏感资料或必填说明不构成本项目收集要求、字段可见性或角色授权。 | PDF第4页：DS_CRH0_PB_010002（p0004-t01-r006）<br>PDF第244页：DS_CRH0_HR_010019（p0244-t01-r012） |
| [X04](#domain-x04) | 保留技术作业、执行尝试及消息技术术语。 | 原文任务是共享服务中心业务任务，不据此调整Job、Attempt、Outbox、Inbox或处理确认的技术含义。 | PDF第302页：ID_CRH0_HR_100009（p0302-t01-r007） |
| [X05](#domain-x05) | 保留导入、映射、回执、CSV和执行模式名称。 | 附录字段格式记法不是CSV、导入能力或执行许可；原材料已完整索引，不把每个格式代码增加为业务词条。 | PDF第323页（无标准项编码） |
| [X06](#domain-x06) | 补人数、期末人数、平均人数、人次和人工成本总额，并补齐指标口径维度。<br>[Headcount（人数）](17-terminology-glossary.md#term-x06-headcount) / [Period-end Headcount（期末人数）](17-terminology-glossary.md#term-x06-period-end-headcount) / [Average Headcount（平均人数）](17-terminology-glossary.md#term-x06-average-headcount) / [Participation Count（人次）](17-terminology-glossary.md#term-x06-participation-count) / [Total Labor Cost（人工成本总额）](17-terminology-glossary.md#term-x06-total-labor-cost) / [MetricDefinition（指标口径定义）](17-terminology-glossary.md#term-x06-metricdefinition) | 不机械采用企业公式：线上与线下人数求和须核对去重，按登录人次平均与按人员平均分开；平均人数的定义与具体平均方法须一致，成本总额不替代成本事实。 | PDF第281页：ID_CRH0_HR_010001（p0281-t01-r007）<br>PDF第282页：ID_CRH0_HR_010008（p0282-t01-r007）<br>PDF第284页：ID_CRH0_HR_030001（p0284-t01-r009）<br>PDF第298页：ID_CHA0_HR_060002（p0298-t01-r010）<br>PDF第299页：ID_CHA0_HR_060011（p0299-t01-r011）<br>PDF第299页：ID_CHA0_HR_060010（p0299-t01-r010） |

## 维护与完整性检查

人工维护源为[data/terminology-glossary.json](data/terminology-glossary.json)，由[生成脚本](scripts/build_modules.py)生成本表及其他文档中的双语引用。术语ID及显式锚点在改名时保持不变；组合对象可绑定多个词条，不能将组合字符串当成一个概念。新增模块对象或参考映射时必须同步补齐词条及绑定。

每个词条记录所属语境、首选中文、简明定义、别名及来源。对象绑定逐项检查本域名称，参考绑定检查条目身份和词条存在；检查覆盖不代表机器已经证明定义正确。应人工复核语义边界及命名依据，尤其是参考概念的替代关系。历史归档及冻结评审记录保持原有措辞。

本仓大陆组织角色修订见[CHNHCM-009](../../adr/009-mainland-organization-terms.md)；只更新本轮明确的语义范围，源R25及后续其他命名增量的评审状态仍见[来源说明](../../reference/README.md)。

```bash
python3 -B docs/architecture/target/scripts/build_modules.py --dry-run
python3 -B docs/architecture/target/scripts/build_modules.py
python3 -B docs/architecture/target/scripts/build_modules.py --check
python3 -B scripts/check-repository.py --check-refs
```
