# Feed 服务重构 TODO（模板代码 → MVP 实现）

> 目标：完全对齐 `services-feed/ARCHITECTURE.md`，将当前 Profile 模板替换为 Feed 业务实现，仅对外提供 gRPC 接口（HTTP 由 Gateway 反向代理）。所有阶段必须在 `make fmt && make lint && make test` 通过后才能推进下一阶段。

## 0. 准备与基线
- [ ] **0.1 阅读上下文**  
  - [ ] 学习 `AGENTS.md`、`docs/ai-context/project-structure.md`、`4MVC架构.md`、`services-feed/ARCHITECTURE.md`。  
  - [ ] 输出：整理 Feed 服务核心职责、分层约束、观测与事件模式。
- [ ] **0.2 建立工作分支与追踪**  
  - [ ] 创建本地分支 `feat/feed-mvp` 并同步项目看板。  
  - [ ] 输出：拆解任务负责人、目标完成时间。
- [ ] **0.3 清理模板遗留**  
  - [ ] 移除/归档 Profile 相关脚本与文档引用。  
  - [ ] 确保目录内无多余生成文件或无用二进制。

## 1. 模块与配置重命名
- [ ] **1.1 调整 Go Module 与依赖**  
  - [x] 将 `go.mod` 模块名改为 `github.com/bionicotaku/lingo-services-feed` 并 `go mod tidy`。  
  - [x] 全局替换 `lingo-services-profile` 引用，更新 `go.work`。  
  - [x] 验证：`go build ./...` 成功。
- [ ] **1.2 更新配置协议**  
  - [x] 更新 `configs/conf.proto` 注释，明确默认 schema 为 `feed`。  
  - [x] 运行 `buf generate` 生成新的 `conf.pb.go`。  
  - [x] 调整 `configs/config.yaml` 默认值：`service.group=feed`、`schema=feed`、消息主题、Feature 开关。  
  - [x] 验证：`go test ./internal/infrastructure/configloader`.
- [ ] **1.3 Wire 依赖清洗**  
  - [ ] 重写 `cmd/grpc/wire.go`，仅注入 Feed 相关 Provider。  
  - [ ] 移除 Profile 服务的服务/仓储绑定。  
  - [ ] 重新执行 `wire ./cmd/grpc` 生成代码。

## 2. 契约定义（gRPC）
- [ ] **2.1 设计 feed.v1 Proto**  
  - [ ] 新建 `api/feed/v1/feed.proto`（GetFeed、FeedItem、RecommendationMetadata、Cursor 等）。  
  - [ ] 暂保留旧 `api/profile/v1`，确保生成链路无阻；新 gRPC 接口通过 Wire 绑定到新 Handler 后再回收旧文件。  
  - [ ] 更新 `buf.yaml`、`buf.gen.yaml` 输出路径，执行 `buf lint && buf generate`。  
  - [ ] 确保生成代码通过 `revive`、`staticcheck`。
- [ ] **2.2 Gateway 协调**  
  - [ ] 输出 gRPC ↔ Gateway 映射表（Header 透传、Problem Details 映射、游标与缓存策略）。  
  - [ ] 与 Gateway 团队确认 HTTP→gRPC 反向代理契约与上线窗口。

## 3. 数据层重构
- [ ] **3.1 数据迁移脚本**  
  - [ ] 创建 `migrations/201_create_feed_schema.sql`：`feed.videos_projection`、`feed.inbox_events`、可选 `feed.recommendation_logs`。  
  - [ ] 保留旧迁移用于回溯；在上线前最后阶段统一删除或标记废弃。  
  - [ ] 在本地 Supabase/PG 上验证脚本。
- [ ] **3.2 sqlc Schema & 查询**  
  - [ ] 更新 `sqlc/schema/*.sql` 对应 Feed 表结构。  
  - [ ] 配置 `sqlc.yaml`：批量读取投影、Inbox 幂等插入、状态更新。  
  - [ ] 执行 `sqlc generate`，核对生成 DAO。
- [ ] **3.3 Repository 实现**  
  - [ ] 重写 `internal/repositories`：  
    - [ ] `VideosProjectionRepository`（Upsert/List/Get）、  
    - [ ] `InboxRepository`（幂等写入、状态更新）、  
    - [ ] 可选 `RecommendationLogRepository`。  
  - [ ] 日志与指标前缀统一 `feed.*`，并提供接口抽象。
- [ ] **3.4 事务与连接池配置**  
  - [ ] 更新 `config.yaml` 中 `messaging.schema` 等字段为 `feed`。  
  - [ ] 校准 `txmanager` 默认超时与重试策略，符合读多写少场景。

## 4. 推荐客户端与 Mock Provider
- [ ] **4.1 抽象 RecommendationProvider**  
  - [ ] 在 `internal/services` 新建接口（`GetFeed(ctx, userID, scene, limit, cursor)`）、数据结构与错误类型。  
  - [ ] 提供 Wire 绑定声明。
- [ ] **4.2 Mock 推荐实现**  
  - [ ] 基于 `feed.videos_projection` 随机抽样，附带 `mock.random` reason，支持 deterministic seed。  
  - [ ] 记录指标：`feed_recommendation_latency_ms`、`feed_recommendation_fail_total`。  
  - [ ] 新增配置开关：`features.enable_mock_recommender`。
- [ ] **4.3 真实 gRPC 客户端占位**  
  - [ ] 新建 `internal/clients/recommendation` stub（kratos gRPC client、超时、Tracing）。  
  - [ ] 实现运行期开关选择 Mock / Real。  
  - [ ] TODO：与推荐团队确认 proto 契约、认证方式。

## 5. Service 层实现
- [ ] **5.1 FeedService**  
  - [ ] 实现 `GetFeed`：解析 Metadata → 调用推荐 → 批量补水 → 组装响应 → 记录 partial 与指标 → 可选日志写入。  
  - [ ] 定义错误映射（推荐异常对应 Problem 503、投影缺失返回 partial）。  
  - [ ] 输出指标：`feed_partial_response_total`、`feed_projection_missing_total`。
- [ ] **5.2 ProjectionService**  
  - [ ] 提供 `BatchGet`/`ListByIDs`，处理版本冲突、缺失兜底策略。  
  - [ ] 预留缓存/批量查询扩展接口。
- [ ] **5.3 Cursor 工具**  
  - [ ] 在 `internal/models/vo` 增加 Cursor 编解码（base64 + 校验）。  
  - [ ] 单测覆盖空结果、limit 边界、非法 cursor。
- [ ] **5.4 DTO & Problem**  
  - [ ] 在 `internal/controllers/dto` 创建转换逻辑（PO/VO → Proto）。  
  - [ ] 统一 Problem Details 输出（使用 `pkg/problem`）。

## 6. 控制器与传输层（仅 gRPC）
- [ ] **6.1 gRPC Handler**  
  - [ ] 新建 `internal/controllers/feed_handler.go` 实现 `feed.v1.FeedServiceServer`。  
  - [ ] 解析参数与 Metadata、注入上下文、处理 Idempotency-Key、生成 gRPC 响应。  
  - [ ] 在新 Handler 完整上线并通过测试后，再删除 Profile Handler 及其 Wire 绑定。  
  - [ ] 单测覆盖：参数校验、推荐错误、partial、ETag。
- [ ] **6.2 认证与鉴权**  
  - [ ] 调整 JWT 中间件，解析用户身份（匿名策略根据业务确认）。  
  - [ ] 定义 Gateway 透传 Header 列表与校验逻辑。

## 7. 异步任务与投影同步
- [ ] **7.1 Catalog Inbox Runner**  
  - [ ] 改写 `internal/tasks/catalog_inbox`，写入 `feed.videos_projection` 并记录 `feed.inbox_events`。  
  - [ ] 处理版本校验、幂等策略、指标与日志。  
  - [ ] Testcontainers 集成测试模拟 `catalog.video.*` 事件流。
- [ ] **7.2 Outbox Runner（可选）**  
  - [ ] 若 Feed MVP 不发布事件，直接删除 Profile Outbox 相关代码、配置与任务。  
  - [ ] 如需后续扩展曝光/点击事件，再新增 Feed Outbox 设计与实现。
- [ ] **7.3 Makefile 任务**  
  - [ ] `make run feed`：启动 gRPC 服务。  
  - [ ] `make run feed-inbox`：启动 Inbox 消费者。  
  - [ ] 添加 `.PHONY` 目标与依赖说明。

## 8. 观测、日志与告警
- [ ] **8.1 日志规范**  
  - [ ] 默认字段：`service=feed`、`scene`、`recommendation_source`、`returned_count`、`partial`、`missing_ids_hash`。  
  - [ ] 用户标识使用哈希/脱敏。  
  - [ ] 如需新增字段更新 `lingo-utils/gclog` 初始化。
- [ ] **8.2 指标落地**  
  - [ ] 实现架构文档中指标：`feed_recommendation_latency_ms`、`feed_recommendation_fail_total`、`feed_projection_lag_seconds`、`feed_partial_response_total`、`feed_projection_missing_total`。  
  - [ ] 使用 OTel Meter，编写测试验证标签与单位。
- [ ] **8.3 Trace**  
  - [ ] 主 Span：`Feed.GetFeed`；子 Span：`Recommendation.GetFeed`、`Projection.BatchGet`。  
  - [ ] Attributes：`scene`、`limit`、`returned`、`partial`、`recommendation_source`。

## 9. 测试体系
- [ ] **9.1 单元测试**  
  - [ ] Service 层：补水逻辑、cursor 处理、partial。  
  - [ ] Mock 推荐：随机抽样 deterministic、limit、空库。  
  - [ ] Controller：参数校验、Problem 映射、Metadata 注入。
- [ ] **9.2 集成测试**  
  - [ ] Testcontainers：运行迁移、写入 Catalog 事件、调用 Inbox Runner、验证投影。  
  - [ ] 调用 `FeedService.GetFeed` 验证响应与 partial。  
  - [ ] 新增 `make test-feed` 目标。
- [ ] **9.3 性能与回归**  
  - [ ] 基准测试：不同 limit 的补水延迟（P95）。  
  - [ ] 回归脚本：推荐超时 → 返回 Problem，指标上报。

## 10. 文档与交付
- [ ] **10.1 服务 README**  
  - [ ] 更新/创建 `services-feed/README.md`：职责、依赖、启动方式、`grpcurl` 验证步骤。  
  - [ ] 列出必需环境变量、假数据准备、Feature Flag 说明。  
  - [ ] 明确 Gateway HTTP→gRPC 转发入口。
- [ ] **10.2 架构文档回填**  
  - [ ] 更新 `ARCHITECTURE.md`：确认指标名、表结构、任务命令、版本号。  
  - [ ] 标记 `v1.0 Feed MVP` 发布信息。
- [ ] **10.3 运维手册**  
  - [ ] 在 `docs/` 增补投影重建、Inbox 重放、指标告警处理流程。  
  - [ ] 提供脚本或命令示例。
- [ ] **10.4 团队同步**  
  - [ ] 与 Gateway、Catalog、推荐团队评审契约与上线计划。  
  - [ ] 输出会议纪要与行动项。

## 11. 发布与回滚
- [ ] **11.1 Smoke 测试**  
  - [ ] 启动 Supabase、Feed gRPC 服务、Inbox Runner。  
  - [ ] 导入示例 Catalog 数据。  
  - [ ] 使用 `grpcurl` 调用 `feed.v1.FeedService/GetFeed`，验证 partial=false。  
  - [ ] 记录关键指标与日志。
- [ ] **11.2 发布策略**  
  - [ ] 制定阶段性计划：Mock-only → 双写验证 → 切换真实推荐 → 全量上线。  
  - [ ] 输出运行手册：启停流程、健康检查、指标阈值。
- [ ] **11.3 回滚方案**  
  - [ ] 明确网关切回旧接口步骤。  
  - [ ] 保留投影数据供排查，记录恢复脚本。  
  - [ ] 文档化回滚流程与责任人。

## 12. 验收标准
- [ ] `feed.v1.FeedService/GetFeed` gRPC 接口符合文档定义，支持 cursor、ETag、Problem Details。  
- [ ] Gateway HTTP→gRPC 转发契约已验证且仅由 Gateway 暴露 HTTP。  
- [ ] Catalog 事件消费延迟 < 5 秒，partial 响应率 < 1%。  
- [ ] 指标、日志、Trace 在本地 OTel 控制台可见且命名正确。  
- [ ] `make fmt lint test`, `buf lint`, `spectral lint`（如仍需 OpenAPI 校验）、`sqlc generate` 全部通过。  
- [ ] README 与运维文档完备，新同学可在 1 小时内完成本地验证。
