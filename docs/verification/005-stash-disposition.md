# CHNHCM-005 三份M03历史stash处置

登记信息：[registry.json](../governance/registry.json)。本记录承接[005入口整理](005-agents-entry-simplification.md)、[308合同恢复](308-job-page-admission-contract.md)及[308正式发布交接](308-job-page-standard-implementation.md#m03-308-publication)。仅证明快照核对、仓外归档与stash清理，不重新验收产品。

## 1 授权、基线与结果

2026-11-17用户在M03第二批方案编制期间要求优先评估三份stash，随后明确“不要过于保守过度防御，要求处理干净”。执行于`wt-dev-d`，HEAD为`9050fa8c3040d54dc3bbc08605d90d651a77341f`。开始本项时已有309／310登记及本轮证据，后续方案正文在同一工作区继续编制；清理不覆盖这些新工作。

三份stash已逐份核对并移除，当前列表为空。未发现缺失文件、独立索引改动或未交付的产品实现；24个不同差异路径均由后续308整改、发布、文档接续承接。完整快照在仓外归档，三份均经真实`git stash apply --index`恢复验证后才删除原条目。没有在当前工作区执行apply／pop，没有reset、切分支、提交、推送或改运行环境。

## 2 逐份核对

| 开始时条目／完整提交 | 已改跟踪文件／未跟踪文件 | 与HEAD相同／不同文件 | 删除项及结论 |
| --- | --- | --- | --- |
| `stash@{0}`／`34dbb4c9d4a5f716aa39a78d84981d96ed38605c` | 52／287 | 317／22 | 3项删除均保留；308页面整改前代码与合同快照 |
| `stash@{1}`／`6fb1d979d7757d24ef0e30e30b7b871841b44d50` | 4／8 | 8／4 | 无删除；精简AGENTS及入口整理证据 |
| `stash@{2}`／`0cf171c5dd4325e61e56029167657b7897932bd8` | 47／268 | 291／24 | 3项删除均保留；原始M03实施及精简前AGENTS |

三份均以`1e830a4ccb0066085bef24239848b086971e4d73`为基线，属于连续保存的状态，不是三批需要依次恢复的实现。不存在仅在索引中的独立改动。全部666个文件版本（跨快照重复计数）、内容模式和6项删除记录（3个不同路径）均进入恢复核验；不能仅凭文件存在判定承接。

差异逐路径及原新blob见[核对清单](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/005-stash-disposition/assessment.json)，完整差异见[压缩补丁](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/005-stash-disposition/reviewed-differences.patch.gz)。主要承接如下：

- API、Go日志投影、能力、固定选择、公共类型和CI发现由`1d65ecea`及308整改承接；旧日志实现由新changes.go取代。
- Staffing列表／详情／表单按308标准重接，发布候选`3b18f024`包含日志直接维护和筛选后退补修；307原旅程对应新入口并完善等待条件。
- 资产、登记、计划、报告和导航增量为已完成整改、运行及发布接续；没有待恢复的新业务功能。
- 原AGENTS逐字节保存在既有`AGENTS-before.txt`，精简入口仅更新运行交接链接；[核对结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/005-stash-disposition/entry-preservation.json)保留原blob身份。

核对发现当前导航仍有307“实施中”、308“待整改”等旧摘要。本轮同步docs/README、specs/README及verification/README到已有实际结果；历史报告的原时点措辞保持，不将这些旧摘要误当实施债务。

## 3 归档与真实恢复

仓外归档目录为`/home/jason/.local/state/chnhcm/recovery/stashes-20261117-9050fa8c/`，内有`preserved-stashes.tar.gz`、`manifest.json`及`RECOVERY.txt`。归档287022150字节，摘要和三份恢复结果见[归档验证](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/005-stash-disposition/archive-verification.json)。

归档包含原stash提交、索引提交、未跟踪树、完整快照文件及共同基线，以共同基线为明确的浅历史边界；不包含无关的基线之前历史。包内Git仓没有alternate对象依赖或远程取数要求；解压后独立克隆，`fsck --full --strict`通过。逐份检出共同基线后实际apply原SHA及索引，跟踪树、索引、所有快照文件内容／模式和删除项均与原件一致。它是已关闭批次的恢复备份，不是新实施待办。

先有两次归档／校验失败，均未删除stash：[第一次](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/005-stash-disposition/archive-first-failure.json)试图打包超出快照范围的历史，本仓`blob:none`未缓存对象使临时仓无法读取；改为精确快照归档。[第二次](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/005-stash-disposition/archive-second-failure.json)是校验器未按NUL分隔解析Git中文路径，实际apply及树对比已成功但文件核对中断；改用`ls-tree -rz`后从实际归档重新执行全部三份。失败如实保存，不计通过。

## 4 删除与保护

按完整SHA重新定位，只删除这三条已验证条目，不使用全局stash clear。按旧至新删除，随后确认stash为空。删除前后HEAD、分支、索引树、完整跟踪差异摘要、工作区状态及未跟踪文件摘要一致，见[清理回执](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/005-stash-disposition/cleanup.json)。

所有自有临时导出／恢复目录已回收；仅上述一份仓外归档保留，供已关闭批次追溯。4380及原数据、API、容器、卷、备份和其他持久环境没有被本项操作。未读取数据库秘密、登录产品或执行业务写入。

## 5 检查与限制

本轮执行Git快照／真实恢复／保护核对及文档检查，证据归本目录；产品check、DB/API、浏览器、容量或部署均未运行，也无需为stash清理重跑这些产品验收。文档最终检查与309／310同批执行，结果见[最终文档日志](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/309-grade-ladder-plan-review/make-docs-final.log)。

当前工作区仍保留本轮文档成果，不称工作区无修改；“处理干净”指三份stash已关闭、无待恢复实施、没有遗留临时恢复目录。下一项继续[309职级与阶梯方案](../plans/309-m03-grade-ladder.md)，不再把三份stash作为实施前置。
