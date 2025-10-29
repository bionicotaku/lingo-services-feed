package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/bionicotaku/lingo-services-feed/internal/models/po"
	"github.com/bionicotaku/lingo-services-feed/internal/repositories/feeddb"
	"github.com/bionicotaku/lingo-services-feed/internal/repositories/mappers"
	"github.com/bionicotaku/lingo-utils/txmanager"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// FeedInboxRepository 管理 feed.inbox_events。
type FeedInboxRepository struct {
	db      *pgxpool.Pool
	queries *feeddb.Queries
	log     *log.Helper
}

// NewFeedInboxRepository 构造 FeedInboxRepository。
func NewFeedInboxRepository(db *pgxpool.Pool, logger log.Logger) *FeedInboxRepository {
	return &FeedInboxRepository{
		db:      db,
		queries: feeddb.New(db),
		log:     log.NewHelper(logger),
	}
}

// InsertInboxEvent 写入 Inbox 记录。
func (r *FeedInboxRepository) InsertInboxEvent(ctx context.Context, sess txmanager.Session, evt po.FeedInboxEvent) error {
	queries := r.queries
	if sess != nil {
		queries = queries.WithTx(sess.Tx())
	}
	id, err := uuid.Parse(evt.EventID)
	if err != nil {
		return fmt.Errorf("parse event_id: %w", err)
	}
	received := mappers.ToPgTimestamptzPtr(nil)
	if !evt.ReceivedAt.IsZero() {
		ts := evt.ReceivedAt.UTC()
		received = mappers.ToPgTimestamptzPtr(&ts)
	}
	params := feeddb.InsertInboxEventParams{
		EventID:       id,
		SourceService: evt.SourceService,
		EventType:     evt.EventType,
		AggregateType: mappers.ToPgText(evt.AggregateType),
		AggregateID:   mappers.ToPgText(evt.AggregateID),
		Payload:       evt.Payload,
		Column7:       received,
	}
	if err := queries.InsertInboxEvent(ctx, params); err != nil {
		r.log.WithContext(ctx).Errorw("msg", "insert feed inbox event failed", "event_id", evt.EventID, "error", err)
		return fmt.Errorf("insert feed inbox event: %w", err)
	}
	return nil
}

// MarkProcessed 设置事件已处理。
func (r *FeedInboxRepository) MarkProcessed(ctx context.Context, sess txmanager.Session, eventID uuid.UUID, processedAt *time.Time) error {
	queries := r.queries
	if sess != nil {
		queries = queries.WithTx(sess.Tx())
	}
	if err := queries.MarkInboxProcessed(ctx, feeddb.MarkInboxProcessedParams{
		EventID:     eventID,
		ProcessedAt: mappers.ToPgTimestamptzPtr(processedAt),
	}); err != nil {
		return fmt.Errorf("mark feed inbox processed: %w", err)
	}
	return nil
}

// RecordError 写入错误信息。
func (r *FeedInboxRepository) RecordError(ctx context.Context, sess txmanager.Session, eventID uuid.UUID, lastError string) error {
	queries := r.queries
	if sess != nil {
		queries = queries.WithTx(sess.Tx())
	}
	if err := queries.RecordInboxError(ctx, feeddb.RecordInboxErrorParams{
		EventID:   eventID,
		LastError: mappers.ToPgText(&lastError),
	}); err != nil {
		return fmt.Errorf("record feed inbox error: %w", err)
	}
	return nil
}

// Get 返回指定 Inbox 事件。
func (r *FeedInboxRepository) Get(ctx context.Context, sess txmanager.Session, eventID uuid.UUID) (*po.FeedInboxEvent, error) {
	queries := r.queries
	if sess != nil {
		queries = queries.WithTx(sess.Tx())
	}
	row, err := queries.GetInboxEvent(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("get feed inbox event: %w", err)
	}
	return mappers.FeedInboxEventFromRow(row), nil
}
