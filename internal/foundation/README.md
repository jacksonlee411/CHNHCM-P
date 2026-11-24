# internal/foundation

X01 小型共享值与协议；不是所有主数据、审计、采用或通用业务总账的归宿。本批包含 UUID、按日日期、错误及跨域凭据传递协议。

依据与入口：[004 共享层依赖](../../docs/architecture/004-repository-layout.md) · [X01 共享值与协议](../../docs/architecture/target/modules/x01.md)。

请求内StageObserver是默认不安装的小型诊断回调协议，只记录阶段名和单调耗时，不保存业务正文、授权或事务策略。

DefaultEffectiveDate按[202](../../docs/adr/202-master-data-default-effective-date.md)提供配置与主数据首次新建的统一日期值；不是最早合法日、今日或已有记录更正日。对象仍提供自己的有效日期下界，公开写请求必须显式带合法日期。
