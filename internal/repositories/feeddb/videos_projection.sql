-- name: UpsertVideoProjection :exec
insert into feed.videos_projection (
  video_id,
  title,
  description,
  duration_micros,
  thumbnail_url,
  hls_master_playlist,
  status,
  visibility_status,
  published_at,
  version,
  updated_at
)
values (
  $1,
  $2,
  $3,
  $4,
  $5,
  $6,
  $7,
  $8,
  $9,
  $10,
  coalesce($11, now())
)
on conflict (video_id) do update
set title               = excluded.title,
    description         = excluded.description,
    duration_micros     = excluded.duration_micros,
    thumbnail_url       = excluded.thumbnail_url,
    hls_master_playlist = excluded.hls_master_playlist,
    status              = excluded.status,
    visibility_status   = excluded.visibility_status,
    published_at        = excluded.published_at,
    version             = excluded.version,
    updated_at          = excluded.updated_at;

-- name: GetVideoProjection :one
select
  video_id,
  title,
  description,
  duration_micros,
  thumbnail_url,
  hls_master_playlist,
  status,
  visibility_status,
  published_at,
  version,
  updated_at
from feed.videos_projection
where video_id = $1;

-- name: ListVideoProjections :many
select
  video_id,
  title,
  description,
  duration_micros,
  thumbnail_url,
  hls_master_playlist,
  status,
  visibility_status,
  published_at,
  version,
  updated_at
from feed.videos_projection
where video_id = any($1::uuid[]);
