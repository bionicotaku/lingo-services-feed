-- ============================================
-- Feed Schema 初始化（基于 Feed ARCHITECTURE.md）
-- ============================================

create schema if not exists feed;

-- 通用触发器：更新 updated_at 字段
create or replace function feed.tg_set_updated_at()
returns trigger
language plpgsql
as $$
begin
  new.updated_at := now();
  return new;
end;
$$;
comment on function feed.tg_set_updated_at() is '触发器：在 UPDATE 时自动写入 updated_at';

-- ============================================
-- 1) 视频投影表：feed.videos_projection
-- ============================================
create table if not exists feed.videos_projection (
  video_id            uuid primary key,                                    -- 视频主键（来自 catalog）
  title               text not null default '',                            -- 标题
  description         text,                                                -- 简介
  duration_micros     bigint,                                              -- 时长（微秒）
  thumbnail_url       text,                                                -- 缩略图
  hls_master_playlist text,                                                -- HLS Master URL
  status              text,                                                -- 视频状态
  visibility_status   text,                                                -- 可见性状态
  published_at        timestamptz,                                         -- 发布时间
  version             bigint not null default 0,                           -- 版本号（来自事件）
  updated_at          timestamptz not null default now()                   -- 最近更新时间
);

comment on table feed.videos_projection is 'Feed 用于补水的只读投影，与 catalog.video 聚合对齐';
comment on column feed.videos_projection.video_id is '视频主键';
comment on column feed.videos_projection.version is '事件版本号，便于幂等更新';

create index if not exists feed_videos_projection_updated_idx
  on feed.videos_projection (updated_at desc);
comment on index feed.feed_videos_projection_updated_idx is '按更新时间排序，便于监控投影延迟';

-- ============================================
-- 2) Inbox 事件表：feed.inbox_events
-- ============================================
create table if not exists feed.inbox_events (
  event_id       uuid primary key,                       -- 来源事件 ID（幂等）
  source_service text not null,                          -- 事件来源服务，例如 catalog
  event_type     text not null,                          -- 事件名称
  aggregate_type text,                                   -- 聚合根类型
  aggregate_id   text,                                   -- 聚合根主键
  payload        bytea not null,                         -- 原始事件载荷
  received_at    timestamptz not null default now(),     -- 接收时间
  processed_at   timestamptz,                            -- 处理完成时间
  last_error     text                                    -- 最近一次错误原因
);

comment on table feed.inbox_events is 'Feed Inbox 表：记录已消费的外部事件，保障幂等';
comment on column feed.inbox_events.source_service is '事件来源服务名称';

create index if not exists feed_inbox_events_processed_idx
  on feed.inbox_events (processed_at);
comment on index feed.feed_inbox_events_processed_idx is '按处理状态筛选 Inbox 记录';

-- ============================================
-- 3) 推荐日志表（可选）：feed.recommendation_logs
-- ============================================
create table if not exists feed.recommendation_logs (
  log_id        uuid primary key default gen_random_uuid(), -- 日志主键
  user_id       text,                                       -- 用户标识（脱敏/匿名）
  scene         text not null,                               -- 场景
  requested     integer not null,                            -- 请求数量
  returned      integer not null,                            -- 实际返回数量
  partial       boolean not null default false,              -- 是否补水不完整
  recommendation_source text not null,                       -- 推荐来源（mock/random/real）
  latency_ms    integer,                                     -- 推荐调用耗时
  missing_ids   jsonb default '[]'::jsonb,                   -- 未补水的 video_id 列表
  generated_at  timestamptz not null default now()           -- 生成时间
);

comment on table feed.recommendation_logs is 'Feed 推荐调用日志（MVP 可选，用于观测）';
comment on column feed.recommendation_logs.missing_ids is '未补水的视频 ID 列表（JSON 数组）';

