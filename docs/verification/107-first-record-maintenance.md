# CHNHCM-107 首条通用维护验证

登记信息：[registry.json](../governance/registry.json)。适用范围：[107决策](../adr/107-first-record-maintenance.md)、011 SCN016/BR020/AC017、迁移TL-129—131/251及新增TL-302—306。旧报告保持原基线及结果。

## 1 验证对象与受测基线

固定wt-dev-b，基线56945e44d5f3e19a19959b067beeccbcf47c3166：用户要求先提交再实施，已以该提交保存上一轮位置新增/插入及更正补测，作者和提交者均为lijianjun@cubebox。本轮工作区删除首条补全菜单及Web BACKFILL模式、保留选择和自动原日期ADD项，使用普通更正及按需另次新增/插入，规格和测试同步。

无Go生产代码、OpenAPI、客户端类型、迁移或存储变更；通用联合入口与协议保持。受测11份实现/测试文件见[输入清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/107-first-record-maintenance/implementation-inputs.json)及同目录压缩包。当前简化增量未追加提交或推送。

## 2 环境与前置条件

隔离m02-first-record-simple（4546/API8546/DB54646）用于Chromium正常菜单，m02-first-record-simple-ui（4547/API8547/DB54647）用于视口。真实DB/API定点回归使用Testcontainers。角色为管理员及撤联合入口权或新增字段权的主体；所有BU均合成，未保存真实凭据、会话或CSRF，截图遮罩账号。

正常入口为设置→组织配置→业务单元→详情→首条→更多操作→更正记录。原4382/4527没有业务写入、迁移或重建；4382由本工作树Web读取当前页面代码，API无需重启。

## 3 验证过程与结果

| 场景或检查 | 预期及实际结果 | 结论 | 证据 |
| --- | --- | --- | --- |
| 首条直接更正 | 首条前移并改名仅CORRECT，不显示专用菜单/保留选择，不自动新增原日节点；一次保存即正常可查 | 通过 | [正常菜单用例](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/107-first-record-maintenance/browser-case-results.json) |
| 两次独立保存 | 有无后继两种夹具，无联合权亦可CORRECT后ADD；每步独立来源/事件/提交，原日新节点为新ID，首条ID/CREATE保持，旧源与中间源可查 | 通过 | [真实DB/API](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/107-first-record-maintenance/integration-final.txt) |
| 第二步取消、无变化、拒绝及撤权 | 第一步保持；第二步可取消或no_change，非法名称及撤新增字段权拒绝后无事实变化，修正/恢复权后可继续第二步 | 通过 | [DB/API](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/107-first-record-maintenance/integration-final.txt)、[浏览器](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/107-first-record-maintenance/browser-final.txt) |
| 历史、继承及恢复 | 两步原内容重试/resolve分别返回各自原来源；第二步实际保存丢响应仅恢复第二步，后继普通/恢复快照保持，停用随每步前序准确继承 | 通过 | [DB/API](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/107-first-record-maintenance/integration-final.txt)、[浏览器](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/107-first-record-maintenance/browser-final.txt) |
| 静态/单元/类型检查 | make check通过，含65项工具测试、文档/边界、Go单元/vet、TS及差异检查 | 通过 | [检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/107-first-record-maintenance/check-final.txt) |
| 构建 | make build通过；保留既有Vite包体提示 | 通过 | [构建](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/107-first-record-maintenance/build.txt) |
| 定点真实DB/API | 首条两步、通用联合首条、位置新增和七种更正组合等六个测试函数及其参数通过，35.696秒 | 通过 | [集成](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/107-first-record-maintenance/integration-final.txt) |
| 受影响Chromium回归 | 46项单次复跑全部通过，包含首条、菜单/布局、普通更正、通用联合、分类、撤权、冲突及丢响应恢复 | 通过 | [逐例结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/107-first-record-maintenance/browser-case-results.json) |
| 视口及原生缩放 | 桌面、短屏、390/320、320×256及原生200%/400%共7组；首条普通更正、分类、关联操作、取消/保存、真实键盘滚轮可达 | 通过 | [矩阵](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/107-first-record-maintenance/matrix/results.json) |
| 原样本和证据保护 | 4382六样本经正常菜单核对，无首条补全菜单；全部7对象来源头及时间线摘要与105结尾一致；4527样本头保持；1737份既有验证/来源/架构/资产文件原字节保持 | 通过 | [样本](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/107-first-record-maintenance/current-samples/ui-result.json)、[一致性](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/107-first-record-maintenance/consistency.json) |

023保留所有旧ID，新增TL-302—306五例；原TL-132“保留选择尚未确认”因该输入取消而退役并保留原意，不计通过。合计306个编号、301现行、5退役。TL-129—131从强制联合改为单条更正及按需独立新增；通用联合算法及其自愿组合仍回归。

## 4 失败、修复与复验

首次受影响浏览器22项为16通过/6失败，均在测试辅助仍精确匹配专用表单标题：普通更正标题实际含目标日期。修正两个enter辅助方法的标题匹配后，相关原项及新增旅程纳入46项复跑，全部通过；不通过修改产品标题掩盖失败。[首轮输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/107-first-record-maintenance/browser-initial.txt)及遮罩截图保留。

初次静态检查发现新增5.4章节插在第6章之后，以及草稿登记缺results；修正章节位置并登记已实际通过的DB输出，make check复跑通过。首次视口命令误用位置参数，脚本实际从CHNHCM_EVIDENCE_DIR取目录；其已有证据保护拒绝执行，没有覆盖旧证据，按环境变量指定新目录后7组通过。准备阶段裸pnpm不可用，按仓库命令使用corepack pnpm；以上均为执行/测试问题，未改Go产品规则。

## 5 未覆盖范围与限制

本轮按影响执行，未重跑全M02/全部306编号参数、容量、Firefox/WebKit、真实消费者或生产验收。没有Go生产逻辑变更，因此未重复105已完成的完整DB回归，以本批六个相关测试函数验证两步保存和共同守卫。视口矩阵验证当前更正及关联编辑；两步新增/插入与响应恢复另由桌面正常菜单和真实API覆盖，不宣称所有参数与视口的笛卡尔积。

其他联合入口及历史停用再启用专用交互未优化。当前为主代理实施和自查，未称独立评审。所有旧报告原结果保持，不将退役TL-132或旧通过记为本轮完成。

## 6 结论与后续事项

首条通用维护简化及本轮受影响验证完成。第一步更正直接保存，第二步按需独立新增/插入，无专用补全入口或保留开关；权限、当前来源、完整候选和历史保护继续由现有Go命令负责。4382仍可验证，六样本及全部7对象保持。本轮4546/4547测试profile已停止并保留卷，原4382/4527保持。

上一轮56945e4已提交；当前107增量未追加提交、推送、PR或合并。

## 7 证据保存

[证据目录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/107-first-record-maintenance/implementation-inputs.json)保存输入冻结、首次失败和最终输出、逐例浏览器结果、遮罩截图、7组视口及6样本结果、7对象摘要、1737原件清单及自查。原始非敏感输出保留于.local/first-record-maintenance。命令输出采用.txt保存，避免通用忽略日志规则遗漏长期证据；凭据、会话及浏览器鉴权档案均不归档。
