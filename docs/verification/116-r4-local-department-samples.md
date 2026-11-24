# CHNHCM-116 R4本地部门验证样本

登记信息：[registry.json](../governance/registry.json)。依据[116路线图](../plans/116-m02-shared-dated-maintenance-roadmap.md)、[R4实施报告](116-r4-department-backend.md)及[017运行指南](../guides/017-local-bu-development.md)。本次只准备并检查真实本地数据，不扩大为R5页面验收。

## 1 受测基线与环境

2026-11-05，在固定wt-dev-b、HEAD 5fa32750ebaf154fd476ed26dd4a983c11a39ef5及当前未提交R4增量执行。原报告1369个运行/构建/测试输入逐一核对未变；本轮未修改业务代码。环境为持久profile m02-department-b，Web4382、API8382、DB54373；当前模型m02-department-1，迁移履历[1,2,3,4,5]。Go服务从当前源码编译启动，Web/API归属、健康及真实业务请求均已核验。

此前m02-current-b实际模型为m02-timeline-7，有7个BU，不满足部门存储前提。仅执行该profile的down，保留容器、数据卷、仓外配置和原程序；随后用新profile独立空库复用分支端口。旧数据未迁入当前新库，未清空、升级或改写旧库；a工作树及其他任务资源未操作。

## 2 进入与查看

先打开[4382登录入口](http://127.0.0.1:4382)，使用本地管理员admin或查询者reader；公开开发登录约定见017。BU从“切换应用 → 设置 → 组织配置 → 业务单元”进入，可见编码R4DEMO的“华东事业部（验证）”，生效日期2026-01-01。

R4已接通部门API，R5部门菜单/页面仍未实施。登录后可在同一浏览器直接打开[五个部门的JSON列表](http://127.0.0.1:4382/api/v1/departments?q=R4)，或下表编码链接查看详情；JSON是当前验证入口，不是已交付的部门页面。服务端本次执行日为2026-11-05，未来跨日后的状态按实际日期重新计算。

| 编码及详情链接 | 当前展示名称 | 上级 | 本次状态 | 所选记录生效日期 |
| --- | --- | --- | --- | --- |
| [R4HQ](http://127.0.0.1:4382/api/v1/departments/affccf14-563d-4b0c-be51-b21fb8173174) | 华东总部（验证） | 企业直属根部门 | 启用 | 2026-01-01 |
| [R4HR](http://127.0.0.1:4382/api/v1/departments/7c4e04ee-8067-4a17-b4bc-1914733b164c) | 人力资源与行政部（验证） | 华东总部（验证） | 启用 | 2026-07-01 |
| [R4FIN](http://127.0.0.1:4382/api/v1/departments/aa3a4f12-45c1-452e-bd61-13451484da69) | 财务部（验证） | 华东总部（验证） | 启用 | 2026-01-01 |
| [R4DIG](http://127.0.0.1:4382/api/v1/departments/7c5e3d34-04f4-4597-8f8e-b6ffe6c12faf) | 数字化创新部（验证） | 华东总部（验证） | 未生效 | 2027-01-01 |
| [R4PROJ](http://127.0.0.1:4382/api/v1/departments/ac81ed32-7182-45bb-8e60-6f740170876d) | 项目筹备组（验证） | 华东总部（验证） | 停用 | 2026-10-01 |

全部部门归属同一BU `50502a69-1335-4d1c-b279-d3112afb95a8`。三个启用部门及项目组首条为2026-01-01；数字化部门首条为2027-01-01。项目筹备组2026-10-01停用，可读停用前2026-09-30路径。

## 3 历史与结果核验

人力部门首条名称为“人力资源部（验证）”，2026-07-01新增名称变更为“人力资源与行政部（验证）”。[选中首条](http://127.0.0.1:4382/api/v1/departments/7c4e04ee-8067-4a17-b4bc-1914733b164c?selectedRecordId=e4744908-61b6-4889-a259-5015ca7931c4)显示2026-01-01至2026-06-30；[创建时准确来源](http://127.0.0.1:4382/api/v1/departments/7c4e04ee-8067-4a17-b4bc-1914733b164c?sourceVersion=afad3e69-12cb-448a-9fd9-4d4d4923e0ab)保持旧名称及保存时的“华东筹建总部（验证）”路径。上级更正为“华东总部（验证）”后，普通当前路径随之刷新，人力部门头来源仍是自己的名称变更来源，没有因祖先更名被重写。

五次创建均按本人submissionId调用真实resolve，返回准确原始部门及来源；当前所选记录均含操作者摘要。人力部门日志共两次操作，历史准确来源只读。未来首条返回not_effective及M02_NO_FACT；停用部门返回inactive及inactive_before_stop路径，路径日期为2026-09-30。库内5个部门、8个部门来源、8个部门Outbox，与5次创建、1次资料新增、1次停用和1次根部门更正一致。

管理员和查询者分别从真实Chromium登录、正常BU菜单进入，再通过同一会话读取部门API；最终六组浏览器核验通过，未捕获页面错误。查询者可读五个样本和摘要，但没有更正能力。未在持久样本库修改授权，因此本次不单独声明“撤销日志权后摘要仍可读”的补验；该后端能力证据仍由原R4报告持有。

## 4 脚本错误与复核边界

首次样本脚本误将停用成功码写为200，服务按合同返回201并已成功落库，脚本中断。续跑从保存的提交结果恢复，未重建或重放样本；随后脚本将路径模式误写before_stop，真实返回为合同规定的inactive_before_stop，再次中断。两次原始记录分别保留。修正预期后只读复核，并由最终管理员及查询者浏览器请求重新验证停用路径、日期及其他样本。

这些是本次检查脚本的错误，不作为服务故障或单轮全绿。R5菜单/表单/视口、R6容量、跨域消费者及生产交付未测；本轮没有重复全套DB或完整浏览器回归。

## 5 环境交接与收尾

当前m02-department-b继续运行，供用户手工验证及后续R5接入；责任为wt-dev-b实施维护者，用户结束本轮验证或下一次运行环境/版本交接时复查。旧m02-current-b停止并保留；两者配置同端口，不能同时启动。旧环境恢复须先停止新环境并使用匹配旧模型的原程序，不能直接用当前源码执行旧profile的up或restart-api。

```bash
cd /home/vivobook/projects/CHNHCM-wt-dev-b
export CHNHCM_LOCAL_PROFILE=m02-department-b
python3 scripts/local-bu.py verify
python3 scripts/local-bu.py status
```

任务创建的API/浏览器会话均已退出，Chromium已关闭；新数据库为明确留存的持久环境，不做临时清理。未新增其他测试容器。本次只保存合成业务数据及非敏感证据，未保存会话cookie、CSRF令牌或数据库凭据。本轮没有提交、推送、PR、合并或部署。

## 6 证据

- [API请求及准确业务结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-local-department-samples/api-evidence.json)，[首次中断](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-local-department-samples/api-first-attempt.json)，[第二次断言错误](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-local-department-samples/api-second-attempt.json)。最终结果字段明确为修正脚本预期后完成，保留原断言历史。
- [最终浏览器核验](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-local-department-samples/browser-evidence.json)，[管理员真实返回](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-local-department-samples/admin-reads.json)，[查询者真实返回](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-local-department-samples/reader-reads.json)。
- [管理员BU页面](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-local-department-samples/admin-business-unit.png)，[查询者BU页面](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-local-department-samples/reader-business-unit.png)，[登录后部门JSON](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-local-department-samples/admin-department-api.png)。
- [输入一致性及环境保留记录](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/116-r4-local-department-samples/runtime-evidence.json)。当前可用性以verify和实际请求为准。
