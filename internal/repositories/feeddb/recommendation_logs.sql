-- name: InsertRecommendationLog :exec
insert into feed.recommendation_logs (
  user_id,
  request_limit,
  recommendation_source,
  recommendation_latency_ms,
  recommended_items,
  missing_video_ids,
  error_kind,
  generated_at
)
values (
  sqlc.arg(user_id),
  sqlc.arg(request_limit),
  sqlc.arg(recommendation_source),
  sqlc.arg(recommendation_latency_ms),
  coalesce(sqlc.arg(recommended_items), '[]'::jsonb),
  coalesce(sqlc.arg(missing_video_ids), '[]'::jsonb),
  sqlc.arg(error_kind),
  coalesce(sqlc.arg(generated_at), now())
);

-- name: GetRecommendationLog :one
select
  log_id,
  user_id,
  request_limit,
  recommendation_source,
  recommendation_latency_ms,
  recommended_items,
  missing_video_ids,
  error_kind,
  generated_at
from feed.recommendation_logs
where log_id = sqlc.arg(log_id);

-- name: ListRecommendationLogs :many
select
  log_id,
  user_id,
  request_limit,
  recommendation_source,
  recommendation_latency_ms,
  recommended_items,
  missing_video_ids,
  error_kind,
  generated_at
from feed.recommendation_logs
where
  (sqlc.narg(user_id)::text is null or user_id = sqlc.narg(user_id)) and
  (sqlc.narg(source)::text is null or recommendation_source = sqlc.narg(source)) and
  (sqlc.narg(since)::timestamptz is null or generated_at >= sqlc.narg(since)) and
  (sqlc.narg(until)::timestamptz is null or generated_at < sqlc.narg(until))
order by generated_at desc
limit sqlc.arg(row_limit);
