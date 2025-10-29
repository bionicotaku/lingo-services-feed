create schema if not exists feed;

create table if not exists feed.videos_projection (
  video_id uuid primary key,
  title text not null default '',
  description text,
  duration_micros bigint,
  thumbnail_url text,
  hls_master_playlist text,
  status text,
  visibility_status text,
  published_at timestamptz,
  version bigint not null default 0,
  updated_at timestamptz not null default now()
);

create table if not exists feed.inbox_events (
  event_id uuid primary key,
  source_service text not null,
  event_type text not null,
  aggregate_type text,
  aggregate_id text,
  payload bytea not null,
  received_at timestamptz not null default now(),
  processed_at timestamptz,
  last_error text
);

create table if not exists feed.recommendation_logs (
  log_id uuid primary key default gen_random_uuid(),
  user_id text,
  scene text not null,
  requested integer not null,
  returned integer not null,
  partial boolean not null default false,
  recommendation_source text not null,
  latency_ms integer,
  missing_ids jsonb default '[]'::jsonb,
  generated_at timestamptz not null default now()
);
