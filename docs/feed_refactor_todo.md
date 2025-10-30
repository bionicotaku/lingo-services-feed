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
- [x] **1.3 Wire 依赖清洗**  
  - [x] 为 Feed 服务新增 `NewFeedService`、`NewFeedHandler` 占位实现，保持“先新增后删除”。  
  - [x] 更新 `cmd/grpc/wire.go` 绑定 Feed/Repository 接口，保留旧 Profile Handler 以便兼容。  
  - [x] 调整 `grpc_server.NewGRPCServer` 注册 Feed gRPC 服务，并重新执行 `wire ./cmd/grpc`、`go build ./cmd/grpc` 验证。

## 2. 契约定义（gRPC）
- [x] **2.1 设计 feed.v1 Proto**  
  - [x] 新建 `api/feed/v1/feed.proto`（GetFeed、FeedItem、MissingProjection 等结构）。  
  - [x] 暂保留旧 `api/profile/v1`，待新 Handler 接入后统一回收。  
  - [x] 运行 `buf lint && buf generate`，生成 `feed.pb.go`/`feed_grpc.pb.go`。  
  - [x] 生成代码通过 `go test ./...`、`revive`、`staticcheck` 后续统一校验。
- [x] **2.2 Gateway 协调**  
  - [x] 在 `docs/gateway_feed_mapping.md` 输出 HTTP→gRPC 映射、Header/Problem Details 策略。  
  - [ ] 与 Gateway 团队确认上线窗口（待召开评审会议）。

## 3. 数据层重构
- [ ] **3.1 数据迁移脚本**  
  - [x] 创建 `migrations/201_create_feed_schema.sql`：`feed.videos_projection`、`feed.inbox_events`、`feed.recommendation_logs`。  
  - [ ] 保留旧迁移用于回溯；上线前最后阶段确认废弃策略。  
  - [x] 通过 Testcontainers 集成测试验证迁移可在 PostgreSQL 中执行。
- [ ] **3.2 sqlc Schema & 查询**  
  - [x] 新增 `sqlc/schema/201_feed_schema.sql`，同步 Feed 表结构。  
  - [x] 更新 `sqlc.yaml`，生成 `internal/repositories/feeddb` 代码；执行 `sqlc generate`。  
  - [x] 编写 Feed 投影/Inbox/日志仓储集成测试覆盖核心 DAO。
- [ ] **3.3 Repository 实现**  
  - [x] 新增 Feed 专用仓储：`FeedVideoProjectionRepository`、`FeedInboxRepository`、`FeedRecommendationLogRepository`（保持 Profile 仓储暂存）。  
  - [x] 编写集成测试验证写入/读取流程；日志指标细节留待业务落地。
- [x] **3.4 事务与连接池配置**  
  - [x] 更新 `config.yaml` 中默认 schema、Feature Flag。  
  - [x] 调整事务默认超时/锁等待/重试次数以适配读多写少。

## 4. 推荐客户端与 Mock Provider
- [ ] **4.1 抽象 RecommendationProvider**  
  - [ ] 在 `internal/services` 新建接口（`GetFeed(ctx, userID, scene, limit, cursor)`）、数据结构与错误类型。  
  - [ ] 提供 Wire 绑定声明。
- [ ] **4.2 Mock 推荐实现**  
  - [x] 基于 `feed.videos_projection` 随机抽样，附带 `mock.random` reason，支持 deterministic seed。  
  - [ ] 记录指标：`feed_recommendation_latency_ms`、`feed_recommendation_fail_total`。  
  - [x] 新增配置开关：`features.enable_mock_recommender`。
- [ ] **4.3 真实 gRPC 客户端占位**  
  - [ ] 新建 `internal/clients/recommendation` stub（kratos gRPC client、超时、Tracing）。  
  - [ ] 实现运行期开关选择 Mock / Real。  
  - [ ] TODO：与推荐团队确认 proto 契约、认证方式。

## 5. Service 层实现
- [ ] **5.1 FeedService**  
  - [x] 实现 `GetFeed`：解析 Metadata → 调用推荐 → 批量补水 → 组装响应 → 输出 partial 状态（指标待补充）；可选日志写入待定。  
  - [x] 定义错误映射（推荐异常对应 Problem 503、投影缺失返回 partial）。  
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
