# CHNHCM-104 分类单组字段优化验证

登记信息：[registry.json](../governance/registry.json)的governance.verification_records。适用范围：用户采纳的更正表单“操作类型/操作原因”单组展示、逐字段读权及动态写权保护。上游为[012表单原则](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-object-form)及[统一更正验收](../specs/m02/011-m02-business-requirements.md#m02-ac-031)。

## 1 验证对象与受测基线

2026-11-04，固定工作树CHNHCM-wt-dev-b、分支wt-dev-b、HEAD `cba247d3181fd993b30bfe9b0326e7032d3d6d5e`。开始已有上一轮分类直接编辑的12份已跟踪未提交文件及未跟踪报告/证据，index和stash为空。本轮保留这些工作，[开始摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-classification-single-fields/before-inputs.json)、[开始源码](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-classification-single-fields/before-source.tar.gz)及[开始差异](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-classification-single-fields/before-change.patch.gz)标识实际输入。

最终代码、用例及当前规格共11份文件摘要见[final-inputs.json](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-classification-single-fields/final-inputs.json)，本轮相对开始dirty快照的增量见[optimization.patch.gz](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-classification-single-fields/optimization.patch.gz)。登记、验证导航、本报告和证据本身不进入递归摘要。

上一轮[分类直接编辑报告](104-classification-direct-edit.md)及关联长期证据共26份保持原字节，见[保护核验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-classification-single-fields/evidence-protection.json)；原61项结果只解释上一轮范围，不计本轮重跑。

## 2 环境与前置条件

管理员从设置→组织配置→业务单元正常菜单创建合成样本，选择记录更正；单面权限及字段撤权使用真实本地授权策略和Go/PostgreSQL。本轮独占临时profile `task-single-classification-b`，实际Web55278/API54210/DB56148，见[连接归属](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-classification-single-fields/connection.json)。该临时入口已回收，不作为在线地址。

本轮先完成类型检查与构建，冻结前端源码及用例，再串行执行39项浏览器和七组视口，最后回收环境。Chromium版本记录在每组[视口结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-classification-single-fields/viewports.json)。不向既有持久库写入测试业务数据。

## 3 验证过程与结果

| 检查 | 真实步骤及实际结果 | 证据与边界 |
| --- | --- | --- |
| 类型与构建 | `corepack pnpm build`退出0，tsc和Vite构建通过 | [构建输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-classification-single-fields/build.log.gz)；保留既有依赖指令、源码映射及包体积警告 |
| 单组字段及默认值 | 首条/非首条正常菜单进入更正，每个可读分类标签只出现一次，无原/新重复标签；原值带入，固定类型只读 | [39项逐项结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-classification-single-fields/browser-results.json)，覆盖TL-282及相关参数 |
| 读写权限 | 只有名称/日期/分类写权均可办对应修改；无分类写权时原值单组只读；缺类型或原因之一读权时仅显示另一项，不请求分类候选，并仍可更正名称 | TL-299及本轮两项部分读权参数，真实保存成功且未回传classification |
| 动态撤权 | 撤分类读权清理输入，撤写权保留单组禁用草稿；非首条转停用后撤写权仍显示“停用/业务终止”，移除修改后只读恢复“资料变更/名称变更”，取消无新来源 | TL-300及本轮类型转换草稿参数；准确展示当前草稿，不借原目标类型标签 |
| 差异、异常及历史 | 初值/恢复原值不算修改，仅名称提交不含分类；目录缺原值/503仍能仅改名称；组合保存、非法链、名称保护、改日、冲突、取消、英文清空、旧来源、日志和查回通过 | 本轮一次39项全部通过，不复用上一轮数量；目录/辅助异常为受控HTTP参数，真实写入及来源读取使用Go/PostgreSQL |
| 视口与键盘 | 桌面1440×900、短屏1440×540、390×844、320×900、320×256、真实200%/400%缩放七组通过；每组断言单组标签，检查实际字号/高度/下划线、键盘字段和保存、错误焦点、滚轮及无横向溢出 | [七组结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-classification-single-fields/viewports.json)；矩阵为首条名称/日期/原因组合，非首条类型切换为桌面业务实测 |
| 文档与差异 | 最终`make docs`及`git diff --check`结果见[验证摘要](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-classification-single-fields/validation.json) | 文档结构检查不代替语义或全产品验收 |
| 资源与旧证据 | 临时容器/网络/卷及仓外配置、profile登记均为空；原三个运行持久环境保持；旧26份证据字节一致 | [资源核验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-classification-single-fields/resources.json)及保护核验 |

## 4 失败、修复与复验

本轮构建、39项业务浏览器及七组视口均一次通过，无业务检查失败或中断。整改同时复核到撤写权后的新类型回显原先借用目标标签，现按实际草稿类型显示，由独立新增浏览器参数验证；不改变分类权限或Go校验。

旧轮次的失败、中断、复验及回收重试继续保留在原报告，本轮不覆盖或扩写原结论。

## 5 未覆盖范围与限制

只优化更正表单，日志、来源历史及删除的原分类说明保留其原用途。不改Go领域、API、原因目录、迁移、存储或权限模型，也不根据名称/日期匹配分类。

未运行完整make check、全量Go/独立integration套件、完整浏览器套件、Firefox、生产容量或远程CI，不启动116后续阶段。真实DB/API结论来自本轮正常菜单提交，不能扩大为后端全套验收。视口仅覆盖首条组合更正；主代理实施后自查，非独立评审或人工业务签收。

## 6 结论与后续事项

更正表单现在仅一组“操作类型/操作原因”，默认原值，按权限编辑或只读，无读权逐字段隐藏；不另列原值对照、不增加展开或确认步骤。原值、实际差异及保存后的前后信息仍由现有模型和日志保护。

39项定向业务及七组视口通过，当前优化范围闭合。临时profile已回收，原三个运行持久环境保持。`m02-current-b`继续用于用户体验，由当前工作树实现及验证维护者负责，下次使用或交接时复查；[业务单元实际入口](http://127.0.0.1:4382/settings/organization/business-units)、API8382已只读核验归属和就绪，不据此宣称持久库业务验收。保留已有未提交工作，本轮修改也未提交，不推送、PR、合并或部署。

## 7 证据保存

长期证据在[data/104-classification-single-fields](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-classification-single-fields)：开始快照、受测增量及摘要、构建/浏览器/视口原始输出、逐项结果、资源及旧证据保护记录、最终检查摘要。保留[桌面](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-classification-single-fields/desktop-edit.png)、[窄屏](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-classification-single-fields/narrow-edit.png)、[400%缩放](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/104-classification-single-fields/zoom400-edit.png)代表图；完整原始图和辅助输出在`.local/classification-single-fields/`，不作为唯一长期证据。不保存真实凭据、cookie、CSRF或数据库密码。
