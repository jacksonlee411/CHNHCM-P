# CHNHCM-100 原生缩放截图修复与补采验证

登记信息：[registry.json](../governance/registry.json) 的 `governance.verification_records`。适用范围：T02 评审发现的原生缩放截图裁切问题。12 张页面图已补采；**截图修复不代表 R01—R09 产品问题修复**。图面使用[校正索引](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-capture-fix/index.html)，原件继续保存。

上游依据：[100 计划](../archive/plans/100-ui-experience-improvement.md)、[012 页面合同](../specs/m02/012-m02-functional-and-interaction-design.md)及[原基线](100-ui-baseline.md)。

## 1 验证对象与受测基线

2026-11-01，固定工作树 `CHNHCM-wt-dev-b`、分支 `wt-dev-b`，产品基线为 `718ef7afcc629ae0ddd2e26a6b63e13b37644d05`。用户要求修复评审问题并提交全部更改；开始时已有 T02 的 100/012 修订、登记、导航、基线报告与附件待提交。本轮在此基础上修复证据采集，保持产品源码、API、010/011/013、既有 012 修订和原数据附件。

[修复前输入](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-capture-fix/inputs-before.json)记录 589 份文件摘要及 main/a 状态；[保护检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-capture-fix/protection.json)确认其中 583 份未变，6 份为本次授权的登记、导航、计划及报告勘误。原基线目录的 65 个清单条目全部匹配；原报告正文的精确副本另存 [ZIP](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-capture-fix/original-baseline-report.zip)及[摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-capture-fix/original-baseline-report.json)，现行报告只在开头追加勘误。

## 2 环境与前置条件

使用 b 独立 Web 4381 / API 8381 / DB 54371；采集前及收尾执行运行归属核验。浏览器为本工作树 Playwright 1.55.1 / Chromium 140.0.7339.186，任务专用 Xvfb，`viewport: null`，中文环境、Asia/Shanghai 时区。100% 内容区域校准为 1280×900，外框固定 1280×1039，使用 X11 原生快捷键放大，未用 CSS zoom、pinch zoom 或改变设备比例模拟页面缩放。

使用原基线的虚构 BU 前缀 `UI100T02_0DE876`，列表查询仍为 21 条。登录凭据只在内存读取；独立 profile 位于仓外临时目录、权限 0700，结束即删除。不保存登录响应、Cookie、CSRF、storageState 或 trace。浏览器外框倍率证明均在输入凭据前截取；登录字段用不透明遮蔽，捕获后检查遮蔽像素再写文件。

## 3 验证过程与结果

### 3.1 截图回归与真实页面补采

| 对象 | 预期与实际 | 结论与证据 |
| --- | --- | --- |
| 100% / 200% / 400% 四角夹具 | 新捕获均为 1280×900，四角红/绿/蓝/黄像素全部存在；输入遮蔽有效，输入值保留，临时遮蔽层已移除，最终恢复 100% | 三组通过；[夹具测量](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-capture-fix/fixture/results.json) |
| 原调用对照 | 同一窗口的原 `page.screenshot` 在 200% / 400% 分别只产出 640×450 / 320×225，未覆盖完整可见区域 | 裁切已复现；[200% 原图](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-capture-fix/fixture/old-200.png)、[400% 原图](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-capture-fix/fixture/old-400.png) |
| 原生 200% 六页 | CSS 640×450、DPR 2；登录、列表、创建、详情、日志、历史滚动后均为完整 1280×900 PNG | 六张通过；[补采测量](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-capture-fix/results.json) |
| 原生 400% 六页 | CSS 320×225、DPR 4；六页均为完整 1280×900 PNG；外框不变，`visualViewport.scale` 为 1 | 六张通过；同一测量及[图面](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-capture-fix/index.html) |
| 登录图面遮蔽 | 两种倍率各识别并遮蔽两个输入框，采样均为不透明洋红；当前窄屏布局隐藏顶栏账号，日志未展开操作者区域 | 最终两张登录图通过；逐图 `maskCount` / `masks` 已记录 |
| 数据保护与恢复 | 从原有虚构数据读取；打开空白新建表单后取消；观察到的 BU 非 GET 请求数为 0；结束恢复 1280×900、DPR 1 | 通过；不将身份登录请求或只读采集当作新业务办理验收 |

列表和创建由正常菜单进入。200% 可从列表打开首条；400% 列表仍为零高，无法点击对象。为补足详情/日志/历史图面，两种倍率均明确采用原对象 URL 的诊断直达访问，再执行日志和历史返回。因此这 12 张图证明截图完整性及当前布局，不能证明全部正常菜单链路在放大后通过。原对象 URL 保留首次创建提示参数，画面中的“已保存”不代表补采发生新的保存。

### 3.2 配套核对

脚本语法、文档结构、差异及证据清单核对见[本轮检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-capture-fix/checks.json)与[文档输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-capture-fix/docs-final.txt)。b 服务归属仍通过，见[运行输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-capture-fix/runtime-final.txt)；其他工作树、stash、原始材料及凭据边界见[保护结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-capture-fix/protection.json)和[收尾检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-capture-fix/safety-and-cleanup.json)。

## 4 失败、修复与复验

本地安装的 Playwright 1.55.1 `screenshotter.js` 将 CSS 视口传给 Chromium 截图实现，`crPage.js` 将其组成 CDP `clip`；本次原生页面缩放环境中该路径实际截取左上区域。修复后的[捕获函数](data/100-ui-capture-fix/capture-viewport.mjs)直接调用 `Page.captureScreenshot`，不传 `clip`，使用 `fromSurface: true`、`captureBeyondViewport: false`，保持当前缩放与窗口。写文件前核对 PNG 像素尺寸与 CSS 尺寸 × DPR，并检查遮蔽像素；目标文件已存在即拒绝覆盖。

首轮补采尺寸已经正确，但复核发现原有精确登录标签选择器受必填星号影响，匹配不到输入框，`maskCount` 为 0。该轮原始 JSON 中的 `passed` 只反映尺寸等当时断言，**不能用于声称遮蔽验证通过**；脚本、图面和数据完整保存在 [capture-run1.zip](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-capture-fix/capture-run1.zip)。当时登录画面尚未输入凭据，未保存密码。修正标签选择器并加入匹配/遮蔽断言后，第二轮重采 12 张；当前索引仅采用第二轮。

首次收尾脚本误要求任务 Xvfb 必须仍有一个活进程，因进程已经退出而断言失败。随后核对原任务会话退出码为 0、无该显示进程及临时 profile，未停止其他服务；该失败及实际收尾结果分别保留于[检查记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-capture-fix/checks.json)和[收尾检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-capture-fix/safety-and-cleanup.json)。

首次基线 50 张图原样保留，其中 12 张缩放页面图标记为裁切失败；倍率证明和 DOM 测量不因该截图问题失效。校正索引由原基线 36 张有效页面图、第二轮 12 张页面图及 2 张新的外框倍率证明组成。原截图脚本保留为历史原件，后续补采使用本报告入口。

## 5 未覆盖范围与限制

本轮未修改产品页面，400% 列表 `clientHeight: 0`、短屏可达性、查询草稿、焦点、对比度、历史标识、日期/日志及表单等 R01—R09 问题仍待 T03/T04。夹具只验证截图采集，不是业务或无障碍验收；本轮只验证上述 Chromium 原生倍率，不推导其他浏览器或屏幕比例。

未重跑完整 `make check`、`make build`、`make integration` 或全量 `make smoke`，也未重做权限变化、保存未知、完整键盘/滚动矩阵、a/b 联运行。相关产品/测试输入未变，旧结果仍只适用于原报告范围。本次未导入 a 的并行更改；100 保持 draft/in_progress，T03—T06 不因此完成，无独立人工审阅批准或远程发布。

## 6 结论与后续事项

评审的截图裁切问题已修复，12 张完整页面图及遮蔽复验已形成，可作为 T03/T04 的修复前图面。本次用户授权的本地提交包含原 T02 文档/证据和本轮修复材料；产品 UI 修复仍是下一实施批次，不在本轮启动。

## 7 证据保存

原始资料位于 `data/100-ui-baseline/`，本轮位于 `data/100-ui-capture-fix/`，后者[清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-capture-fix/manifest.json)记录附件 SHA-256。报告原件、首次补采与最终补采分开保存，不覆盖失败记录。

可复核入口为[四角夹具](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-capture-fix/verify-capture.mjs)和[真实页面补采](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/100-ui-capture-fix/recapture.mjs)。前置条件为本工作树依赖和 Chromium 已安装、独立 Xvfb 显示已准备；真实补采还要求 b 服务归属核验通过且原虚构 BU 数据仍在。以下输出目录必须尚不存在；显示号按本机空闲情况指定，避免复用他人的显示会话：

```bash
DISPLAY=:198 node docs/verification/data/100-ui-capture-fix/verify-capture.mjs .local/t02-capture-regression-new
DISPLAY=:198 node docs/verification/data/100-ui-capture-fix/recapture.mjs .local/t02-capture-pages-new
```

脚本从自身路径定位本工作树和原生缩放辅助脚本；新输出目录必须与本轮原件分开。不要再运行原基线脚本覆盖既有附件，或用截图成功替代正常菜单与业务事实验收。
