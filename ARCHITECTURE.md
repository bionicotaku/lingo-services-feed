.  Feed Service Detailed Design (v0.1 · 2025-10-29)

> 本文定义 Feed 微服务的 MVP 架构与落地方案，涵盖职责边界、数据模型、契约、任务流程、非功能指标以及迭代路线。目标是在尽量少的自有状态下，通过 gRPC 向推荐系统获取推荐列表，并依赖 Catalog 事件完成本地补水，向 Gateway 暴露统一的 Feed 接口。

---

## 1. 使命与边界

- **核心使命**：为终端用户生成个性化内容列表，对 Feed 层的推荐质量与可用性负责。
- **职责范围**
  - 调用推荐系统 gRPC（MVP 暂用随机抽样模拟），根据用户请求获取排序后的 `video_id` 列表与推荐理由。
  - 维护 `feed.videos_projection`，在返回给用户前补全标题、简介、缩略图、时长等展示字段。
  - 暴露 REST/gRPC 接口（经 Gateway 转发），支持游标分页、Problem Details、ETag。
- **非职责（Post-MVP）**：召回黑名单、兜底策略、曝光回传、缓存加速、实验治理、用户态补水。

---

## 2. 交互总览

| 调用方向 | 协议 | 目的 | 说明 |
| --- | --- | --- | --- |
| Feed → Recommendation | gRPC (`RecommendationService.GetFeed`)\* | 获取用户/场景下的推荐 `video_id` 列表 | MVP 阶段无真实推荐服务，使用本地投影随机抽样模拟；契约预留，后续替换为真实调用 |
| Feed ↔ Gateway | REST/gRPC（Kratos） | 提供 `/api/v1/feed` | Gateway 鉴权、Problem 映射、Trace 透传 |
| Catalog → Feed | Pub/Sub 事件 | 更新本地 `videos_projection` | 消费 `catalog.video.*` Outbox 事件 |
| Feed → Postgres | pgx/sqlc | 读写投影、Inbox 状态 | schema `feed` |
| Feed → OTel / Prometheus | OTLP / HTTP | 日志、追踪、指标 | 与全局观测体系一致 |

> \* 当推荐系统尚未落地时，Feed 将通过读取 `feed.videos_projection` 随机选择 `limit` 个已发布视频模拟推荐结果，并写入统一日志字段 `recommendation_source=mock`。上线真实推荐服务后，只需替换客户端实现即可。

---

## 3. 领域与数据模型

### 3.1 表结构（Postgres `feed` schema）

```text
feed.videos_projection           -- 结构与 profile.videos_projection 保持一致
  video_id            ulid    primary key
  title               text    not null
  description         text
  duration_micros     bigint
  thumbnail_url       text
  hls_master_playlist text
  status              text
  visibility_status   text
  published_at        timestamptz
  version             bigint  not null
  updated_at          timestamptz default now() not null

feed.inbox_events
  event_id       uuid    primary key
  source_service text
  event_type     text
  aggregate_id   text
  payload        bytea
  received_at    timestamptz
  processed_at   timestamptz
  last_error     text

feed.recommendation_logs
  log_id        uuid primary key default gen_random_uuid()
  user_id       text
  request_limit integer not null
  recommendation_source text not null
  recommendation_latency_ms integer
  recommended_items jsonb not null default '[]'::jsonb
  missing_video_ids jsonb not null default '[]'::jsonb
  error_kind    text
  generated_at  timestamptz not null default now()
```

> 投影表中的字段与 `services-profile/ARCHITECTURE.md` 描述的 `profile.videos_projection` 一致，确保两个服务在消费 Catalog 事件时保持相同语义；区别仅在于 schema 前缀。MVP 暂不维护用户态投影或近期已推荐。

### 3.2 内部值对象

| 名称 | 字段 | 描述 |
| --- | --- | --- |
| `RecommendationItem` | `VideoID`, `Reason`, `Score`, `Metadata` | 推荐模块返回的原始条目 |
| `VideoCard` | `VideoID`, `Title`, `Description`, `ThumbnailURL`, `DurationMicros`, `ReasonLabel`, `Score`, `VisibilityStatus`, `HLSMasterPlaylist`, `PublishedAt` | 返回给客户端的补水结果 |
| `FeedResponse` | `Items []VideoCard`, `NextCursor`, `Partial`, `GeneratedAt` | API 响应体 |

---

## 4. 服务结构（Kratos + MVC）

```
services-feed/
├── cmd/grpc/                 # 主服务入口（Kratos gRPC/HTTP）
├── cmd/tasks/catalog_inbox/  # 可选：独立运行投影消费者
├── configs/                  # 配置（YAML + .env）
├── internal/
│   ├── controllers/http      # REST Handler（Problem、ETag、游标）
│   ├── controllers/grpc      # FeedService gRPC Server
│   ├── services              # FeedService（协调推荐调用与补水）
│   ├── repositories          # VideosProjectionRepo、InboxRepo
│   ├── clients               # Recommendation gRPC Client
│   ├── infrastructure        # Config、logger、pgxpool、OTel、wire provider
│   ├── tasks                 # CatalogInboxConsumer（订阅 catalog.video.*）
│   └── views                 # DTO 构造、reason 文案映射、分页工具
├── api/proto/feed/v1         # gRPC 契约（buf 管理）
├── api/openapi               # REST 契约（spectral 校验）
├── migrations                # feed schema 迁移脚本
└── ARCHITECTURE.md           # 本文
```

---

## 5. 契约设计

### 5.1 gRPC：`feed.v1.FeedService`

```proto
service FeedService {
  rpc GetFeed(GetFeedRequest) returns (GetFeedResponse);
}

message GetFeedRequest {
  // 请求条目数量，默认 10，最大 100。
  int32 limit = 1;
}

message FeedItem {
  string video_id = 1;
  string reason_code = 2;
  string reason_label = 3;
  double score = 4;
  string title = 5;
  string description = 6;
  string thumbnail_url = 7;
  int64  duration_micros = 8;
  string visibility_status = 9;
  string hls_master_playlist = 10;
  string published_at = 11;
}

message GetFeedResponse {
  repeated FeedItem items = 1;
  string next_cursor = 2;
  bool partial = 3;
  string etag = 4;
  string generated_at = 5;
}
```

### 5.2 REST `/api/v1/feed`

- **请求参数**：`limit`（默认 10，上限 100）。
- **必备 Header**：`X-Apigateway-Api-Userinfo`（由 Gateway 解码的用户信息，缺失或解析失败时返回 `401 Unauthorized`）。
- **响应字段**：`items`（结构同 gRPC）、`paging.next_cursor`、`partial`、`generated_at`、`etag`。
- **Problem Details**（示例类型）：
  - `feed.errors.recommendation_unavailable`（503）—— 推荐 gRPC 超时或失败。
  - `feed.errors.projection_unavailable`（500）—— 投影查询异常。
  - 429 限流与 4xx 参数错误保留扩展。

---

## 6. 推荐调用与补水流程

1. **Controller**：解析请求 → 校验 `limit` → 设定 `ctx` 超时（总 600ms）。
2. **Service**：
   - 若配置中启用了真实推荐客户端：调用 gRPC（超时 200ms），传递 `user_id`、`limit`，获取 `{video_id, reason_code, score, next_cursor}`。
   - 若使用模拟模式：调用 `MockRecommendationProvider.RandomPick(ctx, limit)`，从 `feed.videos_projection` 随机抽取已发布视频，产生默认 `reason_code="mock.random"`、`score=0`、空游标；生成 `recommendation_source="mock"` 日志字段。
   - 批量读取 `feed.videos_projection`，获取标题、简介、缩略图、时长、可见性、播放清单等，并记录推荐日志（原始推荐列表、补水缺失 video_id、耗时等）。
   - 若某些记录缺失或版本落后（事件版本小于当前版本），剔除并标记 `partial=true`，记录缺失数量指标。
   - 调用 `views.ReasonMapper` 将 reason_code 映射为可读标签。
   - 生成 `ETag`（如对 `video_id`+`version` 拼接后 Hash）。
3. **响应**：返回 `items`、`next_cursor`、`partial`、`generated_at=now()`；写日志和指标。
4. **降级策略**：
   - 推荐 gRPC 失败：直接返回 Problem Details 503。
   - 投影缺失过多：若缺失数 ≥ 50%，可返回 503（可配置），提示稍后重试。
   - 后续可加入 Catalog gRPC 回退（Post-MVP）。

---

## 7. 投影消费与任务

### 7.1 订阅事件

- Topic：`video.events`（Catalog Outbox）。
- 关注事件：`catalog.video.created`、`catalog.video.media_ready`、`catalog.video.ai_enriched`、`catalog.video.visibility_changed`、`catalog.video.processing_failed`。

### 7.2 消费流程

1. 使用 `FOR UPDATE SKIP LOCKED` 批量拉取未处理的事件。
2. 事务内写入 `feed.inbox_events`（`event_id` 幂等）。
3. 对比事件中的 `version` 与当前投影的 `version`，若事件版本更高或记录不存在，则基于事件类型更新 `feed.videos_projection`：
   - `created` → 插入记录，填充基础字段。
   - `media_ready` → 更新 `duration_micros`、`thumbnail_url`、`hls_master_playlist`、`status`。
   - `ai_enriched` → 当前仅刷新 `updated_at`（保持与 Profile 一致，后续若新增字段需同时扩展两侧投影）。
   - `visibility_changed` → 更新 `visibility_status`、`status`、`published_at`。
   - `processing_failed` → 标记 `status=failed`。
4. 提交事务；若失败记录 `last_error`，下一轮重试。

### 7.3 运行模式

- 默认在 `cmd/grpc` 启动时注册后台 goroutine。
- 提供 `cmd/tasks/catalog_inbox` 以独立运行（便于 scale-out 或故障恢复）。

---

## 8. 配置与启动

### 8.1 样例环境变量

```
APP_ENV=dev
PG_DSN=postgres://user:pass@host:5432/feed?sslmode=require
RECOMMENDATION_GRPC_ADDR=127.0.0.1:12000
RECOMMENDATION_TIMEOUT_MS=200
FEED_SCENE_DEFAULT_LIMIT=20
CATALOG_EVENTS_PROJECT_ID=<gcp-project>
CATALOG_EVENTS_SUBSCRIPTION=feed.catalog.events
OTEL_EXPORTER_ENDPOINT=http://localhost:4317
LOG_LEVEL=debug
```

### 8.2 启动命令

```
make run feed         # 启动主服务（gRPC/HTTP）
make run feed-inbox   # 可选：独立运行事件消费者
```

### 8.3 验证步骤

1. 启动 Supabase PG，并运行 Catalog 事件生产脚本以填充 `feed.videos_projection`（模拟模式无需额外推荐服务）。
2. 执行 `make run feed`。
3. `grpcurl -d '{"limit":5}' localhost:8082 feed.v1.FeedService/GetFeed`.
4. `curl -H "Authorization: Bearer <token>" "http://localhost:8080/api/v1/feed?limit=5"`.
5. 验证响应 `partial=false`、返回条目数与 limit 一致，`reason_code="mock.random"`（默认模拟值）。

---

## 9. 非功能需求

| 类别 | 目标 |
| --- | --- |
| 可用性 | 99.5%（MVP 单实例即可） |
| 延迟 | 推荐 RPC P95 < 200ms；整体响应 P95 < 600ms |
| 可靠性 | 推荐失败返回 Problem；补水缺失返回 `partial=true` 并告警 |
| 安全 | Gateway 完成 JWT 验签；Feed 内部仅信任服务身份 |
| 合规 | 投影不落敏感信息；日志脱敏用户标识（仅保留哈希） |
| 观测 | 指标、日志、Trace 三位一体，默认接入 OTel/Prometheus |

---

## 10. 指标与日志

- **指标**
  - `feed_recommendation_latency_ms`（Histogram，标签：source）
  - `feed_recommendation_fail_total`（Counter，标签：source，error_kind）
  - `feed_projection_lag_seconds`（Gauge，事件消费延迟）
  - `feed_partial_response_total`（Counter，标签：source）
  - `feed_projection_missing_total`（Counter，标签：source）
- **日志字段**
  - `ts`, `level`, `msg`, `trace_id`, `user_id_hash`, `request_limit`, `recommendation_source`, `recommendation_latency_ms`, `missing_video_ids_count`。
- **Trace**
  - 主 span：`Feed.GetFeed`；子 span：`Recommendation.GetFeed`、`VideosProjection.BatchGet`。
  - Attributes：`limit`, `returned`, `partial`, `missing_video_ids_count`, `recommendation_source`

---

## 11. 风险与缓解

| 风险 | 描述 | 缓解措施 |
| --- | --- | --- |
| 推荐服务不可用 | gRPC 超时/错误导致无结果（真实推荐服务上线后） | 快速失败返回 Problem；记录告警；客户端可重试；模拟模式下可作为回退策略 |
| 投影滞后 | 新视频未同步导致补水缺失 | 标记 `partial=true`；监控 lag；必要时触发全量重放 |
| 数据漂移 | Catalog schema 更新未同步 | 依赖 protobuf；禁止复用 tag；升级前同步契约 |
| 消费中断 | Inbox 异常堆积 | 指标告警；提供重放脚本；任务支持断点续跑 |
| 性能瓶颈 | 投影批量查询慢 | Prepared statement、批量查询；后续引入缓存层 |
| 需求扩散 | 需要支持多种推荐场景 | 后续若确有需求再扩展请求参数并版本化 proto |

---

## 12. 开发路线图

1. **契约**：定义 `api/proto/feed/v1/feed.proto` 与 OpenAPI `/api/v1/feed`，通过 `buf lint`、`spectral lint`。
2. **数据层**：编写迁移脚本创建 `feed.videos_projection`、`feed.inbox_events`；生成 sqlc DAO。
3. **推荐客户端**：实现可插拔的推荐提供者接口，包含 gRPC Client stub（预留真实服务）与 `MockRecommendationProvider`（随机抽样投影表）。
4. **Service 实现**：`FeedService.GetFeed`（调用推荐提供者 + 投影补水 + DTO）。
5. **Controller 层**：HTTP/gRPC Handler，集成 Problem、ETag、游标处理。
6. **Inbox 任务**：实现事件消费与投影更新；编写 Testcontainers 集成测试。
7. **观测性**：接入 OTel、Prometheus；实现指标、结构化日志。
8. **验证脚本**：提供随机推荐模拟脚本与 demo 数据，保证 `make run feed` 后可验证。
9. **文档**：更新 `README.md`、补充启动说明、模拟模式说明和示例；维护本架构文档版本。

---

## 13. MVP 完成定义

- `/api/v1/feed` 可返回补水后的推荐列表，默认 limit=10，后续按需扩展 cursor 支持。
- 投影消费正常运行，事件滞后 < 5 秒。
- 推荐调用、投影补水、响应输出均有日志与指标覆盖。
- `make run feed` + 提供的验证脚本可完成端到端 smoke test。

---

## 14. 后续扩展（Post-MVP）

1. **近期已推荐**：新增 `feed.recent_recommendations`，向推荐系统传递召回黑名单。
2. **用户态补水**：订阅 `profile.engagement.*`、`profile.watch.progressed`，在卡片中展示点赞/继续观看信息。
3. **缓存策略**：引入本地 LRU/Redis 缓存，与推荐冷启动兜底组合使用。
4. **事件回传**：发布 `feed.impression` / `feed.click` / `feed.refresh`，支持推荐效果评估。
5. **兜底策略**：整合热门榜、FSRS 到期队列，在推荐为空时兜底。
6. **实验治理**：支持多模型分流、实验标签透传、灰度发布。

---

**版本记录**

- **v0.1（2025-10-29）**：初版——定位推荐 gRPC 拉取 + Catalog 补水的 MVP 功能，明确目录结构、契约、投影、非功能指标与后续迭代方向。
