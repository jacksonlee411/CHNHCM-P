# CHNHCM-200 G0范围与固定来源准入

登记信息：[registry.json](../governance/registry.json)。承接[200 T00](../archive/plans/200-geography-and-location-roadmap.md)、[201能力表](../specs/m02/201-country-region-geo-area-and-location-maintenance-contract.md)和[202](../adr/202-master-data-default-effective-date.md)。

## 1 基线与阶段

2026-11-13，wt-dev-c，基线0842699c9ab93acb1ab14d55c05bfdde289eb2e9；起点工作区及索引干净。用户要求先提交再完成第一阶段；前批已提交0842699c，本批按路线图第一阶段G0/T00执行。G1日期代码、G2国家功能、G3区域入库及G4地点页面不属于本阶段。

## 2 来源选择与处置

D01选择已保存的固定版次：大陆MCA 2025-12-31、香港DPO来源、台湾NLSC 2026-10-03观察。使用CubeBox离线原件，不宣称2026-11-13最新；原民政部访问失败不改写。输出大陆3209、香港21、台湾390，共3620区域；大陆源三个交叉根留原件，输出明确排除。414个未匹配旧地址只留来源键清单，不猜官方码、不生成Location。

D02逐项处置如下。本轮再次读取[澳门交通事务局页面](https://www.dsat.gov.mo/dsat/subpage_bus.aspx?a_id=1535957405&nav=1618804227)，页面可读，但工具未检出原核对的具体名称；不能重复声称本轮已核验条目；[观察记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/200-g0/macau-observation.json)保留访问与条目核对的区别。八条全部隔离，官方码保持未知，重复中文的nameEn不作为英文译文采用；本期MO按国家/地区加详细地址交付，区域可空。以后补齐来源后另版准入，不阻断其他地区。

| 来源键尾号 | 原名称 | 本次处置 |
| --- | --- | --- |
| DSAT-locality-01 | 花地瑪堂區 | 保留原件，不发布候选 |
| DSAT-locality-02 | 聖安多尼堂區 | 保留原件，不发布候选 |
| DSAT-locality-03 | 望德堂區 | 保留原件，不发布候选 |
| DSAT-locality-04 | 大堂區 | 保留原件，不发布候选 |
| DSAT-locality-05 | 風順堂區 | 保留原件，不发布候选 |
| DSAT-locality-06 | 嘉模堂區 | 保留原件，不发布候选 |
| DSAT-locality-07 | 聖方濟各堂區 | 保留原件，不发布候选 |
| DSAT-locality-08 | 路氹填海區 | 保留原件，不与路氹城區自动合并，不发布候选 |

D03国家251条使用odoo-country命名空间，源XML键保持，业务代码唯一；XI/XK按来源扩展代码解释，不宣称全部ISO分配码。首批名称、英文名及电话区号沿原值；普通显示资料可按权限维护，代码/来源键由受控维护负责。币种/税号/邮编必填及地址模板不作为本批业务规则。来源键不是产品UUID，实例UUID与原提交映射由后续Go装载负责。

## 3 三对象最小接入结果

| 对象 | 身份、范围及日期 | 维护、错误及历史 | 接入边界 |
| --- | --- | --- | --- |
| CountryRegion | 实例目录；源键与代码；初始1900 | 管理员从基础资料查询/显示资料维护/启停；代码变更受控；错误零部分事实，旧来源保留 | 独立Data/Patch/Policy、SQL/结果；复用dated、submission、共用页面；没有企业资格 |
| GeoArea | 实例目录；scheme/code及唯一父；初始1900 | 基础资料菜单；四地真实层级；无环/同域/父资格；准确采用链、单日路径；MO无核验候选 | 同一日期内核，领域拥有父关系；不接部门Hierarchy及BU资格 |
| Location | 企业内稳定编码；四地地址；初始1900并核企业资格 | 组织配置菜单完整维护/选择导出；取消不写、失权裁剪、未知按原身份恢复；旧地址保留 | 企业范围、独立类型/SQL；复用办理、恢复、导航及既有选择端口 |

权限仍由X03裁决，X02不保存第二份地理事实。201第4.1节为能力唯一表，目录高级普通入口及选择/导出后置；没有新增批准流程。具体API、DDL及所有真实实现路径随各自切片落地，当前没有编造实现或验证结果。

## 4 离线准备与检查

[准备工具](../../scripts/prepare-geography200.py)仅读取仓内固定zip，校验归档及20个原件摘要，转换并核对国家唯一、区域代码唯一、国家范围、父引用闭合及无环，输出可重现gzip和[清单](../../tests/fixtures/geography/v1/manifest.json)。工具是资料准备，不执行Go业务写入，也不替代后续受控Go装载。

完整逐条国家/区域及11项排除/隔离处置保存于[离线输入](../../tests/fixtures/geography/v1/package.json.gz)。原许可和署名仍随[来源原件](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/reference/sources/cubebox-geography-2deaf927.zip)保存。后续准备可运行python3 scripts/prepare-geography200.py --check，无需相邻仓库、Odoo或网络。

实际完成准备复验、独立临时目录逐字节复现、make docs（79项回归）、make boundaries及git diff --check，均通过；见[检查记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/200-g0/checks.json)。独立反例覆盖缺父、环、范围外及重复区域码；有效输入不被修改。架构从business-modules.json生成M02模块文档，修订链登记于local-revisions.json；未改变源归档或原采用摘要。

## 5 限制

本阶段完成的是固定资料及任务准入，runtimePublished为false、Location输入数量为0；没有实际主数据入库、菜单、Go默认适配或业务验收。澳门行政/地址分区仍无可发布记录，此缺口以基础地址范围明确承接，不称澳门区划已齐备。保留所有旧失败，不将文档检查当作产品通过。

## 6 结论与收尾

G0/T00完成：对象归属/接入能力、D01—D03的具体处置、完整离线输入和结构验证已落实。G1—G5保持未开始；下一步为默认日期适配及国家真实切片。无运行资源创建、服务更新或清理操作，既有持久环境保持。

## 7 证据

[输入清单](../../tests/fixtures/geography/v1/manifest.json)、[检查记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/200-g0/checks.json)、准备工具及独立反例随本仓保存；前批报告/来源原件不变。本次提交号由Git记录，受测基线见第1节。
