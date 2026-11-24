# CHNHCM-302 详情导航位置与响应整改记录

登记信息：[registry.json](../governance/registry.json)。上游：[302计划](../plans/302-detail-navigation.md)。

## 1 对象与基线

2026-11-15，wt-dev-d，HEAD为d732a05be4e2e448fc9089d71c60163dcac4aafd，开工工作区干净。本批修改共用导航、BU／部门接口及位置统计，不改领域写入、模型5、read-segment-4、封套3或迁移。本地实施完成后，用户授权提交、推送、PR及针对性CI通过后合并；远端发布结果另由PR持有。

## 2 既有登记及实际复现

132第7节、133的OD133-05曾主动取消精确全局序号和总数。301的MSG-12处理辅助导航故障后的局部重试。125/133保留延迟责任，但未登记本次普通切换前串行重读列表的具体缺陷。本次分别登记R302-01（位置／匹配总数）及R302-02（响应）。

45300的tree300-synthetic-d保留87345部门、87350部门来源和1份BU来源；[初态15张表摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/302-detail-navigation/navigation302-runtime-before.json)为数据保护依据。管理员正常菜单进入首项后右上只有箭头、无数字。三次切换到URL变化耗时2923／2942／2947ms，箭头重新可用3796／3814／3818ms；实际详情GET仅44／44／49ms，见[只读浏览器记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/302-detail-navigation/navigation302-before.json)。这是三次诊断，非P95。

## 3 实施及主代理复核

普通部门按当前授权、业务日、树范围及筛选一次排序窄事实；只捕获锚点、目标及两个相邻对象的必要资料并继续Go核验。一次结果提供位置、匹配总数和双向邻居；切换前不再读取列表，返回页按服务端位置及原页大小确定。BU共用消费者同步，保留其既有同名排序。

第一候选使相同脚本的可操作时间降至1938／1914／1921ms，但导航HTTP仍为868—964ms，因此继续核对执行计划。在固定库以全读权参数做只读SQL诊断、与API相同jit=off并关闭EXPLAIN逐节点计时后，普通原SQL／中间层展开／只保留五列／无显式筛选时不计恒零拒绝数分别为1259.083／942.121／757.634／294.714ms。最终采用最后一个候选，不改变当前对象读权、可见字段搜索或显式列／状态／树范围的必要权限判断。首次诊断遗漏API的jit=off，所得5133.460ms不作为产品对照；全部SQL读时诊断保留，均非HTTP或P95。

路径文字筛选由Go完整准确匹配，统计独立加载，不阻断详情及已有前后对象请求；路径列表仍是无总数游标页，原返回定位保持。没有从本页估算总数或沿用批量选择成员。共用控件在桌面及窄屏显示“位置 / 总数”。

主代理复核覆盖当前权、隐藏字段搜索、同名稳定排序、辅助失败及取消、同组件内一次响应移交、详情读取失败与注销清理。切换始终重新请求；没有跨请求权限缓存。仅主代理自审，不宣称独立代理或人工验收。

## 4 验证与首失败

[相关DB/API](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/302-detail-navigation/navigation302-api.log.gz)五个顶层测试通过；[追加专项](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/302-detail-navigation/navigation302-api-security.log.gz)覆盖双角色首中尾位置／总数、名称升降序同名排序、路径准确总数、稀疏读权及撤除名称读权后不匹配，通过。普通Go单元测试通过，见[日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/302-detail-navigation/navigation302-unit.log.gz)。

浏览器第一轮1通过／3超时：路径完整流程在末段点击触及30秒整例预算；admin停在登录准备，未验证导航；reader在返回后脚本滚动被原焦点恢复拉回。第二轮1通过／3超时：路径仍触及30秒，两角色停在脚本滚动。原[第一轮](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/302-detail-navigation/navigation302-browser.log.gz)与[第二轮](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/302-detail-navigation/navigation302-browser-retest.log.gz)保留。新用例改用真实滚轮中断恢复；路径整例含41次创建、140多次HTTP，独立调整为60秒整例预算，接口目标不变、不增加自动重试。

初轮类型检查因React useRef未提供初始值失败，已修正；初轮make check在文档active缺少审定信息处失败，未继续执行其后步骤，见[原日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/302-detail-navigation/navigation302-check.log.gz)。后续[四项同组浏览器](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/302-detail-navigation/navigation302-browser-final.log.gz)通过（37.3／11.1／24.3／33.6秒），[路径统计独立加载、受控503及恢复](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/302-detail-navigation/navigation302-path-independent.log.gz)20.7秒另通过。最终SQL候选又完成[相同五项DB/API](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/302-detail-navigation/navigation302-api-final.log.gz)，追加明确列筛选与状态筛选撤权仍403；[双角色跨页及路径统计三项浏览器](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/302-detail-navigation/navigation302-browser-sql.log.gz)再次通过。最终[make check](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/302-detail-navigation/navigation302-check-final.log.gz)通过；最后只做报告、资产及当前阶段登记，[定稿文档检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/302-detail-navigation/navigation302-docs-final.log.gz)通过。

浏览器统计延迟／503专项证明：详情和前后按钮先可用，统计成功后显示准确数字；失败局部重试恢复。普通双角色用例断言切换期间没有GET列表，覆盖20→21→20、末项按钮、390／320像素计数可见和返回正确页高亮。

## 5 实际环境与资源

最终构建已更新到tree300-synthetic-d，页面<http://127.0.0.1:45300/settings/organization/departments>，API45301／DB45302。基于本报告HEAD加302工作区改动，API SHA256为a3e1952519a320ece02634ed1f21c2097bf4fa6d35215ae6402d9120dec037f4；1461份编译输入和11份实际HTTP文件匹配，见[构建核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/302-detail-navigation/navigation302-runtime-build-check.json)及[完整摘要清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/302-detail-navigation/navigation302-runtime-fixed-build.json.gz)。

同样三次连续切换的[最终只读实测](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/302-detail-navigation/navigation302-final-perf.json)：URL变化924／920／907ms，箭头可用930／927／913ms；导航HTTP551／548／543ms，详情41／44／47ms。与初态相同脚本的约3.8秒相比，可操作时间约减少76%；脚本含断言轮询观察等待，不能当作纯浏览器渲染耗时或正式P95。导航HTTP仍超过原300ms长期目标，不关闭125容量P1。

[45300双角色十组](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/302-detail-navigation/navigation302-live.json)覆盖首项1／87345、名称降序第二页21／87345及下一项／刷新／返回、末项87345／87345、关键词1／10及单对象树范围1／1，全部通过且业务写入0。原[15张组织表内容和来源](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/302-detail-navigation/navigation302-runtime-after.json)与初态一致，原数据库容器保留；BU来源1、部门来源87350不变。

[资源收尾](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/302-detail-navigation/navigation302-closeout.json)核对五个自有临时profile的容器、卷和网络零残留，观察到的12个Testcontainers均已回收。45300保留供手工验收，责任wt-dev-d，复查节点为本次验收结束或下次302整改。其他持久profile未操作。

## 6 限制与交接

R302-01的位置／匹配总数展示和R302-02的重复读取／重复物化缺陷已关闭；整体容量延迟仍由125持有。未执行全量DB/API、全量浏览器、正式125容量矩阵或远程CI。少量串行测量不关闭125容量P1；路径完整统计成本与普通窄查询分别说明。完整业务历史及此前失败保持。

## 7 证据边界

原失败、两阶段候选、SQL诊断及最终结果分开保存于[data目录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/302-detail-navigation)。新增验证资产登记保留原133导航测试源码；当前实现输入由固定构建完整摘要识别。大范围路径统计仍需完整Go匹配，本批未执行其8.7万对象正式容量矩阵；不能以小范围专项通过扩大为所有查询形状达标。

## 8 发布前与main集成

用户授权提交、推送、PR及针对性CI通过后合并。302产品及本地证据先提交e2ed466d，再合入main的8b22e8f7；保留双方文档、测试和提交历史。main已带国家、行政区划及迁移00001—00012，原部门树索引已统一为00012；本批不另增迁移，也不将合并后的程序接入仍保留原构建的45300。

合并时发现新增国家和行政区划是共用详情导航的直接消费者，仍只支持previous／next；同步current、当前位置／匹配总数及双向邻居。国家沿当前授权列表遍历固定251条目录；区划在现行完整条件查询上一次窗口排序，保持code／name／name_desc与UUID稳定次序。公开接口与生成类型一起更新，不引入前端旧协议回退。新增双角色、三种排序、首中尾及切换的真实HTTP专项，以及两个目录双角色浏览器切换／返回验证。

合并后的[首次make check](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/302-detail-navigation/navigation302-merge-check-first.log.gz)在017两侧同号章节处失败，未执行后续检查；将本批新增章节顺延21，不改原历史。后续本地专项和远程CI分别留痕；远程合并仍要求同一最终提交的quality、integration、browser三项全部成功，针对性用例纳入现有完整分片，不缩减既有门禁。远端实际结果由本次PR持有，少量本地诊断和发布集成都不关闭125容量P1。

合并后的[导航及目录DB/API](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/302-detail-navigation/navigation302-merge-api.log.gz)三项、[部门树DB/API](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/302-detail-navigation/navigation302-merge-tree.log.gz)四项全部通过，覆盖新的目录双角色三排序和原树范围、授权、未来移动及固定选择；[make check](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/302-detail-navigation/navigation302-merge-check.log.gz)通过。完整浏览器清单为343项，[11片划分核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/302-detail-navigation/navigation302-shards.json)为36／28／35／28／43／22／36／22／41／27／25，无空片、遗漏或重复；只读划分核对不等于实际页面通过。

[十项针对性浏览器](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/302-detail-navigation/navigation302-merge-browser.log.gz)同轮全部通过（4.0分钟，不含目录初始化），包括部门双角色跨页、路径完整流程及统计独立恢复、辅助失败局部重试、区划完整筛选返回、国家／区划各双角色共享导航。均沿用原业务断言和零重试；[本批资源核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/302-detail-navigation/navigation302-merge-closeout.json)确认观察到的七个Testcontainers及navigation302-merge-d的临时容器／网络／卷已回收，原固定环境未更新。远端CI尚需在最终PR提交执行并通过后方可合并。
