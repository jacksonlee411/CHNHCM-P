# CHNHCM-111 Community T01与T02实施及定向验证

登记信息：[registry.json](../governance/registry.json)的governance.verification_records。依据[111计划](../archive/plans/111-bu-list-experience-upgrade.md)、[113决定](../adr/113-bu-list-community-adoption.md)、[012](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-list-pro)及[013](../specs/m02/013-m02-technical-design.md#m02-tech-list-pro)，按验证模板编写。

## 1 对象与基线

2026-11-03，固定wt-dev-b，基线`7e0fee42271e143a037ab685932fee1ecc7c4b3f`，起始工作区/stash干净。本次连贯实施T01和T02-A/B/C；受测差异由[最终输入摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t01-t02/final-inputs.json)持有。Go/API/SQL及生成客户端未改，旧验证原件保留。

## 2 环境与前置

Node24.19.0、pnpm10.24.0、React19.2.4、MUI7.3.7、Vite7.3.1及Chromium。管理员/只读及请求故障使用合成HTTP，入口按实际应用的组织配置→业务单元菜单；73条分页和虚拟行用开发夹具，不使用真实Go/API/DB。测试仅启动自有临时静态服务（端口由server.address获取）和临时Chromium，finally均关闭，无持久BU profile或容器启停。

## 3 实施与结果

| 范围 | 实际结果 | 证据 |
| --- | --- | --- |
| T01源码/声明 | 固定提交8.29.3，501项原字节运行文件及500份官方声明；运行由Vite转译TS/TSX，声明仅types，未引入发布JavaScript | [清单](../../packages/mui-x-data-grid/upstream.json)、[核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t01-t02/source-audit.json) |
| T01依赖/构建 | 冻结离线安装、应用类型及生产构建通过；Pro/x-license退出运行图，许可初始化/环境声明/releaseInfo/旧夹具退出 | [安装](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t01-t02/frozen-install.log)、[构建](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t01-t02/build-accepted.log)、[依赖](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t01-t02/dependency-summary.json) |
| T02-A | 三段控制区、白色表格/顶部圆角/细边框及紧凑密度；保留#066b68、文本框＋查询、日期/状态局部草稿及原生分页 | [17项检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t01-t02/controls-accepted.json)、[手机](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t01-t02/summary.png)、[短屏桌面](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t01-t02/desktop-short.png) |
| T02-B | 同一DataGrid字段集合通过受控显隐切摘要列，桌面设置不隐藏手机摘要；无重复表头/标签，20/50/100可达，输入不重挂载 | [实际组件10项](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t01-t02/grid-accepted.json)、[网格](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t01-t02/grid-desktop.png) |
| T02-C | 原生显隐/列宽/全部隐藏恢复，A/D变化保持；实际虚拟行、三页返回、手机文档滚动及手机/桌面/有界平板及短屏断点恢复焦点 | 同上；[延迟渲染钩子7项](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t01-t02/hook-return.json)仅证明钩子分支 |
| 许可运行 | 实际Community无Pro许可水印或许可console消息；不注入密钥，不隐藏/过滤官方输出 | controls-complete中licenseMessages为空，actual-grid-final中errors/warnings为空 |
| 历史与业务保护 | Go/API/客户端、旧来源归档/许可证和报告原字节保持，Pro原接入清单转存原字节 | [保护摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t01-t02/protected-inputs.json)、[112清单](../reference/112-datagrid-pro-upstream.json) |
| 资料及最终检查 | 首轮make docs的65项回归与资料检查、实施后18项辅助/语义复核通过；定稿后的重复检查按实际结果保存 | [复核](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t01-t02/implementation-review.json)、[最终检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t01-t02/final-checks.json) |

## 4 失败、修复与复验

首次类型检查同时发现导入名重复、Pro专有属性及上游material实现与当前React声明的类型冲突。修复自有引用，移除Pro属性；以官方同版.d.ts提供types条件，保留原始TS/TSX运行字节，未修改第三方算法。第二次夹具上下文返回值推断过窄，改为明确接口后应用类型/构建通过。原失败见[typecheck-first](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t01-t02/typecheck-first.log)及[typecheck-second](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t01-t02/typecheck-second.log)。

控制栏首轮在原生菜单关闭动画未结束时打开另一列菜单，测试遇到两个菜单项；改为等待菜单实际退出，没有增加产品等待/阻断。最终17项覆盖原生列显隐/改宽/恢复、A/D、链接只携带A、日期/状态、第二/三页草稿、权限拒绝和原因目录独立录入，未调用业务写入口。

实际网格检查发现排序数组每次重建会触发sortModelChange并重置第三页，修复业务页按sort语义memo及夹具稳定模型；原失败见[actual-return-second](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t01-t02/actual-return-second.json)。页大小控件同时具有原生label引用及自有aria-label，精确可访问名称不一致，改用当前select插槽并保留单一名称。断点切换原行链接节点替换后丢焦点，按对象等待新单元格渲染，已有用户焦点优先；原失败见[断点记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t01-t02/breakpoint-focus-first.json)，首次恢复后动态行高仍可能移动目标行，补充订阅实际测量状态直至用户操作或布局/查询变化，原失败另见[延迟测量](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t01-t02/grid-delivery.json)。最终真实网格10项通过。

## 5 限制与未覆盖

T01及T02-A/B/C完成实施和定向组件检查（17＋10＋7项）；T03真实Go/API/DB业务、TL-323—334整例及七组真实缩放未执行，T04列筛选评估与T05整体交付未关闭。此处34项定向检查不能冒充33个业务案例或全M02验收。

构建保留上游use client指令处理及chunk体积提示，它们不是许可console提示；未为消除提示过滤Rollup、console或修改上游字节。声明是编译器输入，供应商实现本身未按本仓React类型重新类型验收；本仓应用按官方同版公开声明完成类型检查及源码实际构建/运行。

## 6 结论与收尾

当前完成T01及T02-A/B/C，T03—T05继续按111后置。所有任务自有静态服务及Chromium在各检查finally关闭，结果stopped=true；未操作既有4180样板、BU profile、容器和卷。无持久在线入口，不据旧端口宣称业务服务在线。本轮未提交、推送或操作远端。

## 7 证据保存

长期证据在[data/111-community-t01-t02](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t01-t02)，首次失败与最终结果分开保留；完整构建产物和非敏感原始输出在.local/111-community-t01-t02，报告结论不只依赖.local。Pro源码归档/商业许可证保留历史，不改变为MIT；当前Community原件及声明各有独立归档/摘要。
