# CHNHCM-111 移除缺密钥列表阻断

登记信息：[registry.json](../governance/registry.json)的`governance.verification_records`。适用范围：[111 T02](../plans/111-bu-list-experience-upgrade.md)后续修正及[112源码/许可决定](../adr/112-datagrid-pro-source-integration.md)。用户要求移除“未配置许可密钥时列表不可用”的应用自建设置；官方许可行为保持。

## 1 对象与基线

2026-11-03，CHNHCM-wt-dev-b、wt-dev-b，基线`bc32de83764bbd50d5a4e3466c70ee76c5440b43`，开始时工作区与stash为空。受测差异及摘要见[输入清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-datagrid-pro-unblock/input-manifest.json)。历史T01、T02及实施前文档报告/证据不回写。

## 2 实施与环境

移除DataGridPage的缺配置拦截、ListPager及控制栏的available参数、返回钩子的许可门禁和muiLicenseConfigured布尔值。开发Pro夹具同样直接挂载；初始化入口有密钥才调用LicenseInfo.setLicenseKey，其返回值不再决定是否渲染。没有改动官方Pro源码、releaseInfo、x-license、水印或console函数，也没有配置或生成替代密钥。

使用本仓Node24.19.0、pnpm10.24.0、TypeScript5.9.3、Vite7.3.1及Chromium。仅启动任务自有127.0.0.1随机端口静态服务，以合成会话和HTTP响应检查真实Pro组件；无真实登录/API/DB、BU profile或容器操作。原生SDK消息原样保存，不能把无密钥挂载等同于无水印验收。

## 3 检查过程与结果

| 检查 | 实际范围与结果 | 证据 |
| --- | --- | --- |
| 类型与构建 | 通过；应用、共享组件及夹具接线，生产构建完成 | [类型日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-datagrid-pro-unblock/typecheck.log)、[构建日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-datagrid-pro-unblock/build.log) |
| 首轮真实Pro挂载及控制栏 | 通过12项；无密钥下从正常菜单进入，表格、合成名称链接及原生分页1–1 / 1可见，无自建不可用提示 | [首轮结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-datagrid-pro-unblock/browser-first.json) |
| 桌面/窄屏复验 | 检查390px原生List view的名称/元数据及表头退出，再复验12项控制栏；实际结果见汇总 | [最终浏览器结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-datagrid-pro-unblock/browser-final.json) |
| 返回适配 | 对移除许可门禁后的真实应用钩子运行6项延迟渲染夹具检查；实际结果见汇总，不是第三方虚拟化/真实业务验收 | [返回结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-datagrid-pro-unblock/return.json) |
| 源码与文档 | 上游119项摘要、受保护历史/业务代码、make docs、冻结输入及diff检查由汇总记录 | [检查汇总](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-datagrid-pro-unblock/checks.json) |

浏览器JSON保存完整warning、console error及pageerror。缺密钥的官方许可消息和合成403/503网络错误保留，测试只按来源分类断言，没有过滤、替换或关闭console；其他运行时异常应为零。截图可见官方水印，不以遮挡或样式隐藏达成页面可用。

## 4 修复边界

本轮删除应用自建限制，保留受授权API结果、加载/错误状态、草稿及返回适配。分页仍由同一Pro上下文提供；密钥缺失不再改变页面结构或阻断普通功能验证。没有新API、Go、SQL、业务写入或依赖升级，也没有重置持久数据。

## 5 未覆盖范围

HTTP及身份均为合成响应，不证明真实鉴权、Go查询、DB、业务写入或历史保护。未运行TL-323—334整例业务矩阵、七组真实缩放或旧浏览器套件的选择器迁移；本轮12项控制/6项钩子只算定向UI证据。390px截图中原生分页只显示范围及前后页，页大小入口的窄屏可达性需T03单独处理；本次不扩展为完整分页适配验收。有效密钥未提供，无水印/无许可提示专项仍未验；T01该项及T03完整业务验收保持待完成。

## 6 结论与资源收尾

缺密钥列表阻断已移除，普通业务列表及开发夹具按同一入口正常挂载Pro。用户本次指令替代此前因“无水印/无提示”而增加的应用门禁，不代表修改官方许可校验。当前功能及许可验收边界以本记录及111进度为准。

浏览器脚本均在finally关闭浏览器与任务静态服务，并在结果中记录stopped；未触碰既有profile、API、容器或卷。没有保留用于演示的新服务入口。

## 7 证据保存

长期输出位于data/111-datagrid-pro-unblock，本地原始输出位于.local/111-unblock。日志只移除末尾冗余空行以通过diff检查，诊断内容不变。固定源文件与历史报告原字节保护，检查汇总和输入摘要分别记录执行结果与受测对象。
