# 发布前浏览器位置冲突补验

登记信息：[registry.json](../governance/registry.json)的同路径verification_records。关联[023案例](../specs/m02/023-m02-timeline-test-cases.md#m02-tl-274)、[105位置新增](../plans/105-positioned-addition-and-coverage.md)。

## 1 对象与失败证据

受测a5e054ebb2e1538a1f83f8e39cebf50d73eccbf7；[PR6的原CI](https://github.com/jacksonlee411/CHNHCM/actions/runs/41626415705)中quality/integration通过，Chromium为136通过、1失败。失败例要求点击保存取得Go409，但自动辅助已先返回来源冲突，正确显示重新核对并禁用保存，见[遮罩截图](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-publication-browser-conflict/source-conflict-masked.png)与[原日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-publication-browser-conflict/ci-failure.txt.gz)。这轮不能记通过。

## 2 修订与复验

同一TL274276分两个确定参数：直接保存参数以受控辅助503覆盖辅助不可用时的最终Go409；自动辅助参数仅延后含目标草稿的真实请求，在并发写入完成后放行，明确断言冲突提示与保存禁用。两者均继续重读并保留原日期/名称/锚点，核对新的上界，非法日由Go400拒绝，改为合法日后保存且历史顺序保持。未关闭自动保护或删减原业务断言。

自有临时profile的两个参数均通过，17.0秒，见[定向输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/021-publication-browser-conflict/targeted.txt.gz)；随后容器、网络、卷及仓外临时配置自动回收。源a5之外只改该测试和本记录/登记/导航，运行代码不变。

## 3 边界与发布门禁

本地定向2项不冒充完整138项通过；后续最新提交的quality/integration/browser必须全部通过才能合并，实际结果以[PR6](https://github.com/jacksonlee411/CHNHCM/pull/6)所指当前head为准。主代理复核，非独立评审或人工批准。
