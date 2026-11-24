# CHNHCM-111 CubeBox样板桌面行高复核

登记信息：[registry.json](../governance/registry.json)的governance.verification_records。依据用户“请参考cubebox的行高”，维护[012列表合同](../specs/m02/012-m02-functional-and-interaction-design.md#m02-ui-list-pro)。

## 1 基线与范围

2026-11-03，wt-dev-b，HEAD为`7e0fee42271e143a037ab685932fee1ecc7c4b3f`，叠加既有未提交T01/T02及上一轮桌面行高修订。上一轮[52px复核](111-desktop-row-height.md)及其证据保留原结果，本轮替代其桌面高度选择；仅修改自有桌面样式和估算，同步012及111，不改供应商源码或业务合同。

## 2 样板依据与修改

只读访问用户指定的[CubeBox部门列表](../reference/113-department-list-template.json)所记4180网址，在1440×900视口等待字体和实际行渲染后测量。十条可见数据行均为39px，表头33px，单元格左右内边距10px。样板普通列表使用compact密度及rowHeight=56，密度换算后的实际高度为39px；它的普通行不启用自动增高。

[新测量](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-cubebox-desktop-row-height/cubebox.json)、[样板截图](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-cubebox-desktop-row-height/cubebox-desktop.png)及[身份核对](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-cubebox-desktop-row-height/source-identity.json)记录依据。4180的HTML摘要与113原观察完全一致；departments-list.tsx及ui.tsx摘要也与固定来源记录一致。未重建、修改或切换CubeBox仓库。

CHNHCM桌面数据行最小高度从52px改为39px，单元格上下内边距从12px改为6px，桌面虚拟滚动估算同步39。保留既定的长文本换行和自动增高，未引入样板普通行的固定高度裁切。表头仍为36px；低于900px的摘要样式、字号、查询规则、链接及#066b68保持原约定。

## 3 实测结果

使用CHNHCM真实Community页面、只读合成身份和HTTP、两个短名称和一个长名称。每个视口重新载入并等待字体及行渲染；这是组件布局测量，不是真实Go/API/DB业务验收。

| 视口 | 短行修改前→后 | 长名称修改前→后 | 结果 |
| --- | --- | --- | --- |
| 1440×900 | 52→39px | 85.05→73.05px | 普通行对齐样板，长文自动增高 |
| 1000×900 | 52→39px | 125.08→113.08px | 普通行对齐样板，长文无裁切 |
| 390×844 | 53.03→53.03px | 173.13→173.13px | 窄屏完全不变 |

全部测量单元格均无纵向内容溢出，各视口表头高度与修改前一致。见[修改前](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-cubebox-desktop-row-height/before.json)、[修改后](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-cubebox-desktop-row-height/after.json)、[断言结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-cubebox-desktop-row-height/comparison.json)及[CHNHCM桌面截图](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-cubebox-desktop-row-height/chnhcm-desktop.png)。已人工核对样板与本仓1440px图面。

## 4 回归与检查

应用类型和生产源码构建通过，见[构建输出](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-cubebox-desktop-row-height/build.txt)。既有实际网格虚拟行58、显式返回、浏览器返回、第三页分页、延迟响应、用户输入优先、隐藏列/摘要切换、缺失目标回退、手机自然滚动及跨尺寸同对象焦点共10项通过，errors及warnings均为空，见[返回结果](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-cubebox-desktop-row-height/grid-return.json)。

定稿后的make docs、git diff --check及本轮输入保护结果见[最终检查](https://github.com/jacksonlee411/CHNHCM/blob/061d28105966bd10eb1e3b0032cf2976a6448de4/docs/verification/data/111-cubebox-desktop-row-height/checks.json)。原T01/T02与52px复核报告及数据未覆盖。

## 5 验证边界与环境收尾

本轮未运行真实Go/API/DB业务写入或七组完整缩放，不改变111的T03后置范围。未增加长期测试；尺寸断言为本次样式核对，返回回归复用既有脚本。

本次启动的临时静态服务及Chromium均已finally关闭，测量及回归结果stopped=true；未启动BU profile或操作既有容器、卷。4180是既有CubeBox样板服务，本轮仅只读观察，继续按其原用途运行，不接管其维护生命周期。

## 6 结论

桌面普通行已对齐CubeBox部门样板实测39px，长文本保留自动增高，手机摘要及表头不变。功能合同与计划记录同步，既有历史证据保留，未提交或推送。
