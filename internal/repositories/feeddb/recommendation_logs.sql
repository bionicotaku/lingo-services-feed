-- name: InsertRecommendationLog :exec
insert into feed.recommendation_logs (
  user_id,
  scene,
  requested,
  returned,
  partial,
  recommendation_source,
  latency_ms,
  missing_ids,
  generated_at
)
values (
  $1,
  $2,
  $3,
  $4,
  $5,
  $6,
  $7,
  coalesce($8, '[]'::jsonb),
  coalesce($9, now())
);
