-- name: InsertInboxEvent :exec
insert into feed.inbox_events (
  event_id,
  source_service,
  event_type,
  aggregate_type,
  aggregate_id,
  payload,
  received_at
)
values ($1, $2, $3, $4, $5, $6, coalesce($7, now()))
on conflict (event_id) do nothing;

-- name: MarkInboxProcessed :exec
update feed.inbox_events
set processed_at = coalesce($2, now()),
    last_error   = null
where event_id = $1;

-- name: RecordInboxError :exec
update feed.inbox_events
set last_error = $2
where event_id = $1;

-- name: GetInboxEvent :one
select
  event_id,
  source_service,
  event_type,
  aggregate_type,
  aggregate_id,
  payload,
  received_at,
  processed_at,
  last_error
from feed.inbox_events
where event_id = $1;
