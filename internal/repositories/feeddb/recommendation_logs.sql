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
