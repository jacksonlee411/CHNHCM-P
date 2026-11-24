# CHNHCM-125 部门路径阶段目标采纳与同样本复评

登记信息：[registry.json](../governance/registry.json)的governance.verification_records。上游为[120技术合同](../specs/m02/120-dated-maintenance-technical-contract.md#department-path-capacity-target)、[125统一清单](../plans/125-r6-capacity-remediation.md#r125-path-stage-target)及[300关闭记录](300-closeout.md)。本批完成阶段目标采纳、判定器同步及同样本复评，原测量保持。

## 1 采纳决定与实际基线

2026-11-15用户针对20会话路径阶段目标1000ms建议明确“采纳建议，请进行”。正式采用：单会话阶段／长期均300ms；20会话阶段1000ms、长期300ms。适用于部门路径文字筛选及其子树范围交集，既有两种完整形状、角色、暖冷工况、并发和规定样本数量保持。完整决定与未变约束见[用户采纳](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/125-path-stage-target-adoption/decision.json)。

工作树为wt-dev-b，HEAD为979473b918c64063f8cae67723d3a5837417e8c3。开始时已有116／131／133和300关闭的未提交成果；本批保留原成果，新增差异仅为当前合同／导航／登记、容量目标与复评工具。没有新增领域、SQL、存储或前端行为修改。受测工具摘要见[输入](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/125-path-stage-target-adoption/driver-inputs.json)。300编号归属仍为wt-dev-d，本次由wt-dev-b接续目标采纳，不修改号段归属。

## 2 合同与判定器同步

120第7节为权威目标入口，125第8.2／8.9节与300第6.2节路由该决定。两个查询入口共用tests/capacity/path_targets.py：query131及其remediation133继承者按路径动作取阶段阈值；tree300按scope-path动作取同一阈值，清单与结果均登记策略和输入摘要。未来路径结果同时输出阶段目标／通过与长期目标／通过。普通列表、树节点／搜索／定位、选择／恢复、父候选和保存不跟随本次调整。

原120／125／300审阅证据仍可从[原审阅对应](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/125-path-stage-target-adoption/previous-review-links.json)追溯。本次主代理核对范围仅为用户采纳决定与上述消费者一致性，不将文档修订或边界测试解释为容量达标。原53容器／47卷的运行保持；本批没有启动业务库／API／浏览器，资源创建与清理不适用，结束时仍核对原资源清单。

## 3 原样本双目标复评

只复评125最终查询轮和300最终关闭轮中已有完整200次样本的路径组，均为关闭诊断后的原正式测量；单组P95按原最近秩第190个样本重算，保留所有错误与分母。每组原结果和输入摘要分别登记，不合并不同范围、角色、机器时段或P95。新增业务操作及HTTP请求均为0；原1600个样本用于本次判据复核，不冒称新跑1600次容量请求。

| 原测量／角色／会话 | 原正式样本 | P95(ms) | 本阶段目标(ms) | 阶段结果 | 长期目标(ms)／结果 |
| --- | ---: | ---: | ---: | --- | --- |
| 125查询首页／admin／1 | 200 | 568.935 | 300 | 未达 | 300／未达 |
| 125查询首页／admin／20 | 200 | 2732.418 | 1000 | 未达 | 300／未达 |
| 300子树交集／admin／1 | 200 | 624.414 | 300 | 未达 | 300／未达 |
| 300子树交集／admin／20 | 200 | 2913.780 | 1000 | 未达 | 300／未达 |
| 300子树交集／reader／1 | 200 | 804.713 | 300 | 未达 | 300／未达 |
| 300子树交集／reader／20 | 200 | 3551.027 | 1000 | 未达 | 300／未达 |
| 300子树交集／sparse-reader／1 | 200 | 1666.454 | 300 | 未达 | 300／未达 |
| 300子树交集／sparse-reader／20 | 200 | 4666.429 | 1000 | 未达 | 300／未达 |

8组原样本均零HTTP／语义错误，但阶段与长期各0组通过。20会话4组仍为2732.418—4666.429ms，明显高于1000ms；单会话4组仍按300ms判定。目标采纳没有新增容量关闭项，125容量P1及C125-01/11保持。原报告中的300ms数值失败保留，复评和原样本摘要见[双目标结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/125-path-stage-target-adoption/reassessment.json)。

原176单元中的16个路径责任单元另存[目标覆盖](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/125-path-stage-target-adoption/responsibility-overlay.json)：8个20会话单元的阶段阈值变为1000ms，其余8个单会话仍300ms；其他160个入口单元保持。覆盖只同步判据，不给历史结果自动授予当前构建通过资格，冷态／合成及其原样本数量继续按125核对复用或补验。

## 4 检查、保护与接续

6项容量策略测试通过，覆盖1000ms临界值、单会话300ms、零错误要求、长期目标未达、原失败及样本深拷贝保护，以及query131／tree300同一目标策略；其他入口的目标由既有反例共同核对。3个查询判定入口的命令行装载、82项文档脚本测试及仓库检查、git diff --check均通过，见[检查结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/125-path-stage-target-adoption/checks.json)。本批只改容量目标与资料，不重复启动业务库、完整DB/API、浏览器或容量压测。

本批没有新增产品性能优化；后续由125优先定位路径首屏必要材料、范围遍历、授权及页装配成本，保留同输入复现与正式独占计时。300维持阶段关闭，9个明显未达组、36个共用待验及原宽深合成／写入责任继续接续。其他近目标已接受组不追加微调，原R65仅记录、10000条退出、20条代表及固定环境决定保持。

初态12178份文件中，12166份保持摘要一致，修改仅限现行合同、登记、导航与路径判定工具；全部10075份原验证文件保持，含116／131／133／300关闭和125原测量。结束快照中的原53容器／47卷身份、状态、标签和挂载一致，HEAD、索引及stash保持，见[保护核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/125-path-stage-target-adoption/protection.json)。首次资源比较断言失败但未留存差异明细，后续独立只读采样及复验一致；未尝试修改任何资源，也不推断期间不存在外部瞬时活动。首错与依赖文件尚未生成的定稿读取中断由[执行偏差](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/125-path-stage-target-adoption/execution-issues.json)保留。

本次目标采纳实施完成，125容量P1继续开放；未提交、推送或更新固定环境。原300关闭记录中“尚未采纳”属于该批时点事实，本次由现行120合同及本记录接续，不回写旧证据。
