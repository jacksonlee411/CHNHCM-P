# CHNHCM-111 T04列筛选评估与复核

登记信息：[registry.json](../governance/registry.json)的governance.verification_records。依据[111阶段计划](../plans/111-bu-list-experience-upgrade.md)及[110列筛选范围](../adr/110-bu-list-and-datagrid.md#6-待决事项)，从[验证模板](../templates/verification.md)编写。本报告持有评估执行事实，当前选择、比较和后置边界由110第2.4/2.5节维护；不是新增业务规格。

## 1 对象、基线与结论

2026-11-03，固定wt-dev-b，HEAD为`19f182cc482bdffa36d827d399f62781e68529f5`，起始工作区/stash干净。当前T01—T03已提交，T04要求比较Community面板、必要适配、Excel形态和API差距，而非直接实施新列条件。

T04评估交付完成：本批维持现行搜索、查看日期与单值状态面板，新列条件及完整Excel候选继续后置；如另批采纳单条件，优先复用公开GridFilterForm及输入器，由应用提供局部草稿、应用/取消和Go合同适配。完整字段/操作符/组合、权限和候选合同须在该批实施前确定。本次不引入Pro，不修改Community强制属性，也不把现有q/status冒充独立列条件。

## 2 依据、环境与适用边界

证据入口为[固定来源与差距记录](../reference/111-column-filter-assessment.json)，包含16个固定源码/现行合同文件摘要和4个官方页面链接。精确组件为本仓Community8.29.3、mui-x提交`109f5a0c73075355f755352ad2849d52a01197b5`。官方v8滚动页本次显示8.29.5，只作公开能力说明，不代替8.29.3源码或运行证明。

官方[过滤说明](https://v8.mui.com/x/react-data-grid/filtering/)区分单条件与多条件；[服务端过滤说明](https://v8.mui.com/x/react-data-grid/filtering/server-side/)要求应用承接回调及查询；[定制说明](https://v8.mui.com/x/react-data-grid/filtering/customization/)描述输入器/操作符定制；[表头过滤说明](https://v8.mui.com/x/react-data-grid/filtering/header-filters/)使用Pro。精确行为同时核对本仓强制属性、sanitizeFilterModel、singleSelect操作符、GridFilterForm/Panel和filterMode分支。

组件实验使用本仓Node/Vite及Chromium，固定源码由独立临时Vite服务转译，未借相邻仓库或外部node_modules。实验仅有3条合成行及人工73计数，没有业务API、鉴权或数据库；不证明全量73条查询或新服务端能力。桌面1200×900及390×844检查只针对输入器和对话框，不是T03七组产品验收。

当前服务端静态核对范围包括OpenAPI、API查询白名单、Service.List校验和当前SQL：只有businessAt/q/status/sort/page/pageSize；未知/重复参数有拒绝分支。q是获权编码或名称的字面包含，状态为单值，businessAt是查看日。T04未重新发起真实业务请求，不能将静态核对记为新增API通过。

## 3 独立组件实验

原始结果见[7项结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-column-filter-evaluation/results.json)，apiRequests和errors均为空，stopped=true。执行的是供应商公开组件及自有最小实验外壳，未修改业务页面或供应商实现。

| 检查 | 预期与实际 | 结果 |
| --- | --- | --- |
| 原生面板事件 | 输入MISSING后模型已变化，面板没有项目应用按钮；server模式仍显示原3行且无业务请求 | 通过，说明不能原样承接应用/取消 |
| 单项多值 | status的isAnyOf在一个item内接受active/inactive两个值，未增加第二个条件 | 通过；不是跨列多条件 |
| 局部草稿与Escape | 公开GridFilterForm编辑草稿AA，主网格模型和已应用条件仍为空；Escape保留搜索D | 通过 |
| 显式应用 | 点击应用后A推进一次，主网格模型反映AA；合成行仍未被服务端重查 | 通过，仅证明状态外壳可复用 |
| 已应用后的取消 | 编辑ZZ再取消，A及主网格继续AA，不增加应用计数 | 通过 |
| 窄屏按需入口 | 390px将同一公开条件表单按列排列；对话框不横向溢出，应用/取消及Escape可达 | 通过 |
| 客户端当前页过滤 | client模式把已加载的3行隐藏，未获取全授权73条匹配或候选 | 通过，作为错误接线策略的边界示例 |

图面见[原生面板](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-column-filter-evaluation/native-desktop.png)、[同项多值](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-column-filter-evaluation/single-item-multivalue.png)及[窄屏草稿](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-column-filter-evaluation/draft-mobile.png)，已复核截图。实验中的多值按钮只准备测试条件，不代表拟交付的产品入口。A/D实验未接真实Go，也未实现业务请求转换、无效条件校验、清除规则或完整键盘/授权流程。

## 4 方案比较、发现与复核

完整比较和API差距在110第2.4/2.5节：直接开放原生面板不采用；公开输入行加应用对话框为后续单条件优先候选；完全自建只用于不适配的必要输入；完整Excel候选后置；本批采用保持现行查询。

可减少的是单条件字段/操作符选择器、文字和枚举输入器的重复开发。仍需应用/取消事务、只反映A的图标/摘要、搜索D事件合并、URL/返回、移动入口和Go查询合同。不能把草稿模型挂到主网格后让表头先显示未应用条件，也不能直接公开GridFilterModel。

API差距的三个具体反例：q=AA可以同时命中编码AA与另一个名称含AA的BU，不能替代独立编码条件；active/inactive多值集合不等于all，后者含未生效；businessAt用于选定查看日的资料，不是effectiveDate的起止区间。任何新空值条件还须区分真实空值与字段授权裁剪。

另发现官方滚动页称缺省组合为Or，而固定8.29.3的getDefaultGridFilterModel实际为And；本次组件快照亦记录固定版本值。该资料不一致已记入来源记录；未来业务组合须显式定案，不继承任何组件默认，也不改变现行q/status的SQL语义。

首次实验在同一个事件中设置受控模型并打开面板，读取到尚未提交的旧模型，导致多值断言失败；改为等待受控模型提交后再打开，7项通过，保留[首轮输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-column-filter-evaluation/probe-first.txt)及后续输出。窄屏截图改为等待对话框透明度稳定后采集，不把动画中间态当作最终图面。均为实验修订，未发现或修复产品运行缺陷。

## 5 后续合同与未实施项

下一批若采纳列条件，由Organization及体验维护者明确字段/操作符、归一化、集合与空值、日期对应记录、组合和请求大小边界，定义过滤字段获权策略及一致计数；同步011—013、OpenAPI、Go DTO、SQL和客户端。数据查询仍由Go权威实现，日期协议仍为YYYY-MM-DD，不把浏览器Date对象直接作为公开条件。

桌面列菜单与窄屏更多操作可以打开同一个按需编辑器，这是候选入口；本批不增加入口。现有状态面板可表达当前单值状态，无需为复用组件再建设重复条件入口。完整Excel候选另需全授权范围、查看日、搜索/分页/计数及排除本列条件的联动规则；不能以当前页distinct代替。

本次没有新增BR/SCN/AC或TL编号，7项合成组件检查不计入12个真实业务案例。未实施新列过滤、候选服务、跨列组合、收藏/分享、导出、分组、headerFilters或偏好平台；T05整体交付仍未关闭，T03报告及证据保持原结果。

## 6 环境收尾与复现

独立Vite服务及Chromium均在finally关闭，执行结果stopped=true；没有启动BU profile/API/DB、容器、Xvfb或配置许可密钥，没有操作既有服务。任务在apps/web/.local下的临时实验入口完成归档后删除，不进入产品路由、构建或正常菜单。归档文件仅为验证资料，不要求用户使用该实验入口。

如需复现，在本仓将[main.tsx](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-column-filter-evaluation/main.tsx)、[index.html](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-column-filter-evaluation/index.html)及[review.mjs](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-column-filter-evaluation/review.mjs)复制到apps/web/.local/t04-filter-evaluation，然后运行：

```bash
node apps/web/.local/t04-filter-evaluation/review.mjs .local/111-t04-repeat
```

输出使用新的任务目录；浏览器/服务由脚本收尾，复现后移除该临时入口。仅复制验证资料，不改生产配置或供应商源码。

## 7 最终检查与结论

定稿后的来源摘要、实现边界、资料关联、治理回归及git diff --check见[最终检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-column-filter-evaluation/checks.json)。工作树、报告状态和T04进度分别记录，不因评估通过把新查询实现记为完成。

T04已完成其评估交付：明确本批维持现行查询，后续单条件优先复用公开输入组件加按需应用外壳，完整Excel候选后置。T05仍待后续；本轮未提交或推送。
