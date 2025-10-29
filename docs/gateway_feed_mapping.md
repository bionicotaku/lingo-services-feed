# Feed 服务 Gateway 转发说明（MVP）

> 该文件描述 Gateway 从 HTTP 请求到 Feed gRPC 服务的翻译规则，确保双方契约一致。后续上线前需与 Gateway 团队正式评审确认。

- **目标接口**
  - gRPC：`feed.v1.FeedService/GetFeed`
  - HTTP：`GET /api/v1/feed`
- **请求映射**
  - `Authorization` → Gateway 校验并生成 `x-apigateway-api-userinfo`，随后转发；Feed 服务本地默认开启 `skip_validate=true`。
  - `scene`（query，必填） → `GetFeedRequest.scene`
  - `limit`（query，默认 20，上限 100） → `GetFeedRequest.limit`
  - `cursor`（query，可选） → `GetFeedRequest.cursor`
  - 额外 query/header 映射到 `GetFeedRequest.metadata`（Gateway 维护白名单，例如 A/B 实验标签）。
- **响应映射**
  - gRPC 成功 → HTTP 200，Body 直接透传 JSON（由 Gateway 自动转换）。
  - gRPC `codes.Unimplemented`（当前占位）→ HTTP 501。
  - gRPC `codes.InvalidArgument` → HTTP 400。
  - gRPC `codes.Internal` → HTTP 500。
- **透传 Header**
  - `x-md-*`：保持原样透传，支持 Idempotency-Key / ETag。
  - `x-apigateway-api-userinfo`：Gateway 注入，Feed 解析用户身份。
- **缓存/ETag**
  - 暂不启用服务端 ETag；Gateway 维持短期 CDN 缓存策略时需确认 partial=false 条件。
- **Observability**
  - Gateway 将 traceparent 透传给 Feed；Feed 返回 `trace_id` 供 Gateway 日志关联。

TODO（上线前）：与 Gateway 团队确认匿名请求策略、流控/熔断策略，以及 Problem Details 的 JSON 模板。
