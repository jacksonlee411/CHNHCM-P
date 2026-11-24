# CHNHCM-112 DataGrid Pro未转译源码接入

本文已由[113 Community决定](113-bu-list-community-adoption.md)整体替代，以下保留原Pro接入决策及当时未验事项，均不作为当前实施前置。未转译源码、自足构建、来源保护等继续适用原则已由113承接；当前代码已按111退出Pro并完成Community接线，实际结果见[111实施报告](../verification/111-community-t01-t02.md)。历史许可专项因Pro选型退出而撤销，不记为通过，不再要求注入密钥推进新方案。

登记信息：[registry.json](../governance/registry.json)。适用范围：111 T01中DataGrid Pro的分发与构建入口；上游为[110采用决定](110-bu-list-and-datagrid.md)、[111计划](../plans/111-bu-list-experience-upgrade.md)及用户本次要求。只变更组件依赖的接入方式，BU页面替换仍归T02，代码接线及受限验证见[T02报告](../verification/111-datagrid-pro-t02.md)。

## 1 背景与问题

110最初拟采用官方npm发布包。用户随后要求改为引入未转译源码，并要求无水印、无许可控制台提示。两者分别是构建输入和许可运行结果，不能混为同一个条件；用户确认已取得Pro许可，但密钥尚未配置。

## 2 决策

使用官方mui/mui-x的v8.29.3，固定提交`109f5a0c73075355f755352ad2849d52a01197b5`。Pro运行源码位于`packages/mui-x-data-grid-pro/src`，保留官方TS/TSX原字节，由本仓Vite在开发/生产构建时转译；不再以npm的Pro预编译入口作为应用运行入口。上游测试程序不进入本仓运行包。

React、MUI Core、DataGrid基础层、x-internals和x-license继续使用官方发布依赖；本次不是把整个MUI单仓及所有依赖改为源码构建。原始来源子集、原始package.json及LICENSE保存在独立来源包，提交、逐文件摘要、排除范围及适配清单由`packages/mui-x-data-grid-pro/upstream.json`登记。

本地package.json仅将上游workspace/catalog解析为本仓精确依赖，并指向原始src/index.ts。归档与源码摘要通过显式`pnpm verify:mui-source`核对，供依赖升级及来源审查使用，不在每次开发/生产构建中重复设置前置门禁。构建插件只将上游`__RELEASE_INFO__`占位替换为同版本官方发布包的原releaseInfo；不得用当前日期重算或改变许可适用版本。Watermark、useLicenseVerifier及官方x-license保留，不修改、隐藏或绕过校验，不替换控制台函数。

许可初始化仍使用官方LicenseInfo.setLicenseKey，并在任何Pro组件挂载前执行。密钥由仓外环境`VITE_MUI_X_LICENSE_KEY`提供，不在源文件、日志或仓内环境文件保存；该值按官方机制会进入浏览器端，不能替换成数据库/登录凭据。按用户最新要求，未配置密钥不再阻断业务列表或开发夹具，两者正常挂载Pro；应用不增加许可配置拦截页，也不隐藏或过滤官方水印/控制台许可提示。有值时仍在挂载前通过官方入口配置。无水印/无许可提示的专项验收仍须有效密钥，当前恢复挂载不等于该项通过。

## 3 备选方案与取舍

| 方案 | 结论与代价 |
| --- | --- |
| 继续使用官方Pro预编译发布包 | 由本次源码入口替代，保留此前安装/检查记录 |
| 导入全部MUI单仓、测试和构建工具 | 不纳入；只保留Pro运行源码，基础依赖精确锁定 |
| 本地Pro源码包加Vite转译 | 采用；增加源码摘要与releaseInfo适配责任，实际接口/构建须核验 |
| 删除水印/关闭许可检查或伪造密钥 | 不采用；源码接入不改变官方许可校验，缺密钥不能宣称无水印验收通过 |

## 4 影响与验证

pnpm工作区增加本地源码包，apps/web指向workspace依赖，packages/ui按同版本声明peer并在开发中指向本地源码。保留原商业许可证及NOTICE，不声明取得超出用户已有许可的权利。安装、冻结锁文件、源码摘要、TypeScript、Vite构建及缺密钥实际挂载分别记录；本次没有Go/API/SQL变更，不计BU新页面或业务案例通过。

T01的实际结果由后续附属验证报告持有。许可控制台提示只在官方有效密钥下核验，不通过修改node_modules、上游源码、CSS遮挡或日志过滤实现。

## 5 适用与替代关系

仅替代110/013中Pro作为官方发布包直接接入的技术路径；Pro选择、业务/导航保护、List view和分页目标均保持。110继续持有体验采用，本文持有源码及版本/许可边界，111持有执行进度；此前实施前文档报告原件保持，不回写。

## 6 待决事项

有效密钥的仓外注入位置及无水印/无许可提示的实际浏览器验收尚待完成。用户已确认拥有许可；本仓不补造许可或激活结果。此项不阻止源码依赖、类型及构建检查，普通组件及业务功能验证可继续推进；无水印/无许可提示仍按有效密钥专项关闭。此前不挂载保护及其验证原件保留为历史结果，移除结果另见[本次记录](../verification/111-datagrid-pro-unblock.md)。

2026-11-03按用户进一步清理不必要限制：源码核对改为显式审查命令，前端build不读取本地API/DB profile；开发及preview继续核对实际服务归属，见[清理记录](../verification/111-defensive-design-cleanup.md)。
