# CHNHCM-500 T00 人员登记与名录规格评审

2026-11-22，T00 完成。三份规格经 Codex 主代理四轮顺序评审定稿，T01 可以据此独立实施。人员登记不依赖用人单位、岗位、录用或任职。正常登记经办人的样本同时具备登记、核对和基本查阅权，能在重新登录后找回该人。[业务及验收](../specs/m01/501-person-registration-requirements.md) · [页面继承](../specs/m01/502-person-directory-interactions.md#m01-page-map) · [技术及容量](../specs/m01/503-person-registration-design.md) · [路线图](../plans/500-personnel-roadmap.md)

## 1 基线与范围

固定工作树 `wt-dev-f`，受测 HEAD 为 `f7d84f192ff518555a60dfe939a78455d8a48597`。开工时工作区、索引和 stash 均为空。15 个现有本地及远程跟踪引用均未登记 501—503。本轮变更是三份规格、本记录、路线图接续、文档导航、资产及文档登记和复核材料。评审人从人事办理、权限、体验与技术四个角度走查，未进行人工业务签收。

实际对照 [M01](../architecture/target/modules/m01.md)、[X03](../architecture/target/modules/x03.md)、[004 领域边界](../architecture/004-repository-layout.md)、[012](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-expression)、[121 接入矩阵](../specs/m02/121-dated-object-admission-matrix.md)、[124 页面准入](../guides/124-dated-object-admission-and-capacity.md#page-standard-admission)、[308 职务页面验收](308-job-page-standard-implementation.md)，并查阅 BU、职务、公共表单/导航/提交恢复及 Access 公开接口。Fusion 的本地目标映射提供 Person 与 Assignment 的界限。CubeBox 固定提交 `074fe044ab41c80abbab41a74aadd2a97b66efcf` 的任职列表及多值个人资料原文已逐字节校验并离线登记为 ASSET-0232/0233。NATIVE-V500-T00 留存 500 的两个 M01 场景作为验收输入。三项资产仍待 T01 产品接线与实测。[本地概念映射](../architecture/target/07-concept-mapping.md) · [来源包](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/reference/sources/cubebox-personnel-074fe044.zip) · [资产登记](../assets/inventory.json) · [原场景摘录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/500-personnel-t00/scenarios.txt)

## 2 逐轮发现与结论

| 稿次与审阅角度 | 发现、修订与复验结论 | 三份规格的组合 SHA-256 |
| --- | --- | --- |
| 一，未任职登记、同名核对与改号 | 要求修改。首稿让核对员可打开候选详情，却未要求该人的基本读权。改号依赖“已有决定”，本地生成路径与截止依据不足。同据换号重送与从原档案更正也易混淆。第二稿限定候选投影和返回去向，有决定及执行权限者从本人一次办理，登记重送比较原始语义。 | `b6ff6934002a0351847a88e3ba472a3aa4a9f8ccb88b61eeb7629baa83ed8e3a` |
| 二，受限名录、资料保留与容量 | 首轮事项复核关闭。另加 20 个写入人员会把基线变为 10020 人。只说“资料到期”不足以区分停止展示、保全和可处置。名录检索也需说清旧号和姓名同时匹配时的去重。第三稿改为同快照副本内使用原有 20 人，明确用途/证据/保全投影、处置与永久占位，复算受限检索的跨页和末页结果。 | `ad1c6090c8a0c02c9576aad6c2a68744f1eed5f7dcdc010102c57bc7e58a700c` |
| 三，跨字段更正、截止和迟到回执 | 第二轮事项复核关闭。第三稿还可能只核对原类型就把电话改为邮箱，普通名录失效后的确切证据读取也缺独立准入。改号材料可能明确长期有效。第四稿要求原/新类型各自完整读写，保全只许从确切来源读指定类别，改号按真实期限和政策办理，确定拒绝后的重新核对用新提交号。 | `8244ff91be5f8063e2e14a5b214717ac5a4ff6f917713151265eb0fbd9d5b057` |
| 四，字段表到页面、命令与验收的逐项复查 | 通过。重走三组角色、两种查回次序和两个截止边界，复核露出字段、菜单可达、候选变化、改号约束及 1 万人可见集合。六项规则与十项验收均有对应办理入口或明确拒绝、服务端主责和拟实测层。T00 没有待决的首批规格项。 | `7820321e999d49e0dca9596c0674f2f1a5aca219d9741fd17a2f7286a1e8daed` |

逐稿包共 12 份原文。每轮按文件名顺序，将 UTF-8 文件名、零字节及文件 SHA-256 的 32 字节原值拼接，再取 SHA-256，即得表中摘要。第四稿与交付的三份规格逐字节一致。[四稿原文](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/500-personnel-t00/reviewed-specs.zip)

## 3 独立预期复算

| 样本 | 纸面走查与可复算结果 |
| --- | --- |
| 尚未任职 | 取消、登记第一人、选原人、资料不足、核实第二人、重送第一张材料，按顺序得到 0、1、1、1、2、2 个登记身份。旧号仍指向原人。同一客户的编号占位与另一客户隔离，姓名更正后重送仍对照原登记内容。 |
| 同名与授权 | 客户甲两人同名，完整名录见 2 人，列明范围的读者见 1 人。仅有部门权限看不到人。甲的完整电话授权与乙的基本读权不产生乙的电话读权。原电话可写、新邮箱只有遮罩权时整笔类型变更被拒。 |
| 截止与撤权 | 北京时间 2027-03-31 23:59:59 普通用途仍有效，下一秒退出名录。2027-05-01 证据期限达到时，独立保全只保留指定核对材料的受限读取。准确来源、受限页头和原回执按当前用途裁剪。 |
| 1 万人名录 | 全范围 10000 人、同名 200 人。受限范围 5000 人、同名 100 人。受限结果以现号排序，第 20、21 位为 2000、2100，第五页首末为 8100、10000。共有 4000 条联系方式、100 个旧号和 2000 份留存资料修订。六条链路乘两档会话、每组 200 次，得到 2400 次正式计时办理。 |

运行 `python3 -I -B docs/verification/data/500-personnel-t00/expected.py` 已复算上述集合、边界及查回先后。样本仅使用 Python 标准库和合成事实。两种权限角色的十四个视口／缩放单元、并发与延迟目标为 T01 产品验收输入。[独立复算程序](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/500-personnel-t00/expected.py) · [页面验收](../specs/m01/502-person-directory-interactions.md#4-产品候选的页面验收) · [容量口径](../specs/m01/503-person-registration-design.md#m01-capacity)

## 4 检查与交接

Python 3.14.4 下运行 `make docs`，137 项脚本测试通过，仓库检查覆盖 422 份 Markdown、13,307 处本地链接、233 项保留资产。另行定位新规格与评审记录的 107 处本地链接，其中 53 处直达锚点或章节。逐项核对三份规格的表格列数、六个规则与十个验收锚点、两份来源原件及四稿摘要。使用临时 Git 索引把新材料纳入 `git diff --check`，本次 16 个变更文件全部通过。

Go 1.26.0 下运行 `make boundaries`，625 个既有 Go 文件的导入边界检查通过。`go test ./scripts/check-boundaries -count=1` 通过。T00 使用标准库复算程序及原件核对，没有启动人员产品环境。

T01 的首个产品结果是经办人登记、重登找回、更正一条联系方式并按当前权限查看原值。实际 X03 人员范围、Go 领域、迁移、API、页面及其 DB/API/浏览器、性能和恢复实测由 T01 交付。本轮收口的是 T00 的合同、独立预期与文档评审。
