# CHNHCM-111 桌面列表行高调整复核

登记信息：[registry.json](../governance/registry.json)的governance.verification_records。依据用户“我是指桌面的行高”及[012列表合同](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-list-pro)，仅调整桌面数据行。

## 1 基线与范围

2026-11-03，wt-dev-b，HEAD为`7e0fee42271e143a037ab685932fee1ecc7c4b3f`，叠加既有未提交T01/T02实现。此前输入见[原快照](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-community-t01-t02/final-inputs.json)，本轮输入及最终检查见[补充检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-desktop-row-height/checks.json)。原报告/来源原件不覆盖；只修改自有桌面行样式及估算，并同步功能合同和进度。

## 2 原因与修改

getRowHeight使用auto，原40px只是估算；桌面单元格padding与MUI规则同优先级而被覆盖，短行实际只有25px。新增min-width:900px下更具体的自有选择器，设单元格min-height:52px、上下padding12px；长名称继续自动增高，桌面估算同步52。表头高度、字号及低于900px的摘要样式不变，未改供应商源码。

## 3 实测结果

真实Community业务单元页面，使用只读合成身份、三个长短名称和合成HTTP；每个视口重新载入并等待字体/实际行渲染。非真实Go/API/DB验收。

| 视口 | 短行修改前→后 | 长名称修改前→后 | 结果 |
| --- | --- | --- | --- |
| 1440×900 | 25→52px | 61.05→85.05px | 增高且无内容裁切 |
| 1000×900 | 25→52px | 101.08→125.08px | 自动换行增高，无裁切 |
| 390×844 | 53.03→53.03px | 173.13→173.13px | 窄屏保持不变 |

原始尺寸见[修改前](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-desktop-row-height/before.json)、[修改后](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-desktop-row-height/after.json)和[对比](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-desktop-row-height/comparison.json)；[桌面前](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-desktop-row-height/before-desktop.png)、[桌面后](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-desktop-row-height/after-desktop.png)及[1000px](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-desktop-row-height/after-desktop-1000.png)保留图面。

## 4 回归与检查

应用类型/构建通过，见[构建](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-desktop-row-height/build.txt)。既有实际网格虚拟行、返回、三页分页、用户操作优先及尺寸切换10项通过，见[结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-desktop-row-height/grid-return.json)。定稿后的资料、差异和输入摘要检查以补充检查文件实际结果为准。

## 5 限制与收尾

仅桌面行高修订及受影响组件验证；未运行业务写入、真实Go/API/DB或七组完整缩放，不改变111的T03后置范围。自有临时静态服务和Chromium均finally关闭，结果stopped=true；未启动BU profile或操作既有服务/容器/卷。

## 6 结论

桌面短行提高至52px，长文自动增高，窄屏不变。原T01/T02证据按其旧输入保留，本轮另行补充，未提交或推送。
