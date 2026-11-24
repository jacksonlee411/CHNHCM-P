# a/b列表、日期与启停合并复核

登记信息：[registry.json](../governance/registry.json)的同路径verification_records。关联[111](../plans/111-bu-list-experience-upgrade.md)、[026](../adr/026-date-display-and-input-format.md)、[023](../specs/m02/023-m02-timeline-test-cases.md)。

## 1 对象与基线

b输入1009a93f6d5384b737d0516520b8ff01c8333c87，合入a5e054ebb2e1538a1f83f8e39cebf50d73eccbf7；原共同基线94dd06466ffc44ee233e600d99f3bb82218a143b。首个合并提交ed335d3780eab624afa90dc861dc31b2e23360d7后，继续合入a的cc55b72f55abd32d29e4592013ff762b2e07be33：仅补两个确定冲突测试参数和原失败记录，运行代码不变。保留两支全部原提交，以合并提交整合。远程先a后b，合并以前必须逐个核对最新head三项CI。

## 2 冲突和接线

登记表按文档ID与报告路径取并集；复核元数据指向本记录，旧执行结果不回写。023保留b列表323—342与a启停343—348，合计348项（304现行、44退役），数量不等于业务通过数。原a报告编号勘误及字节快照按023记录保留。

共享查看日期弹窗采用DateField，废弃常驻控件不恢复；列筛选日期也接共享ISO字段。API/Go保持单一权威，集成及浏览器种子补齐所选末条anchorRecordId。Playwright测试入口取并集，旧测试经正常菜单操作日期/状态/排序，适配记录范围分页。

## 3 检查与未覆盖项

make check、make build、make generate（生成物无额外变化）及完整make integration通过；真实DB两包分别537.610秒和293.087秒。输出见[证据目录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-a-b-integration)。

首轮Chromium为9通过、2失败、1中断、143未运行：一处读取日期后改变了待断言的焦点，一处仍从桌面菜单寻找排序；主动停止该轮以修复，未记通过。第二轮完整155项为154通过、1旧分页焦点断言失败，29.8分钟；Playwright执行的是发现时缓存的旧断言，随后当前修订版本定向1项通过，18.3秒。这是154＋1的最新逐例结果，不是单次155项全绿。截图与日志按各轮原样保存。合入cc55b72后，两种来源冲突在b真实接线再次定向通过（2项，24.5秒），最新make check也通过；最终156参数套件由远程CI按最新提交完整验收，本地的155完整轮及后续3项定向分别计证。该增量仅改测试与资料，Go/API/共享组件运行输入未变，原构建、生成及完整DB证据按原范围复用。

测试分页断言按当前合同检查原生控件不可操作时焦点移到结果状态；保留键盘、返回、滚动、权限和历史业务断言。

双方已定稿验证记录按路径逐项一致，b的2188个、a最新cc55b72的1949个验证附件文件逐字节一致（仅导航不比较），见[保护核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-a-b-integration/historical-preservation.json)。完整轮运行输入摘要见[清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-a-b-integration/browser-second-runtime-inputs.json)，1265项已逐一核验，基于1009a93的差异及原焦点断言存于[差异档](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-a-b-integration/browser-second-runtime-delta.tar.gz)，[重建结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-a-b-integration/browser-second-reconstruction.json)可复核，最终远程验收另按PR最新提交核对。

原报告保持原基线及范围；跨列组合筛选、Excel值枚举、Pro、列拖动和固定不纳入，CHNHCM深色#066b68保持。主代理自查，不冒充独立评审或全M02验收。

最终合并输入另见[运行清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-a-b-integration/final-runtime-inputs.json)。本地定向所用临时profile均已自动销毁；日常m02-current-b数据未作为测试写入目标。
