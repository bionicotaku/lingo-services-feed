package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bionicotaku/lingo-services-feed/internal/models/po"
	"github.com/bionicotaku/lingo-services-feed/internal/repositories/feeddb"
	"github.com/bionicotaku/lingo-services-feed/internal/repositories/mappers"
	"github.com/bionicotaku/lingo-utils/txmanager"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// FeedRecommendationLogRepository 负责推荐调用日志持久化。
type FeedRecommendationLogRepository struct {
	db      *pgxpool.Pool
	queries *feeddb.Queries
	log     *log.Helper
}

// NewFeedRecommendationLogRepository 构造仓储实例。
func NewFeedRecommendationLogRepository(db *pgxpool.Pool, logger log.Logger) *FeedRecommendationLogRepository {
	return &FeedRecommendationLogRepository{
		db:      db,
		queries: feeddb.New(db),
		log:     log.NewHelper(logger),
	}
}

// Insert 写入推荐日志。
func (r *FeedRecommendationLogRepository) Insert(ctx context.Context, sess txmanager.Session, logEntry po.FeedRecommendationLog) error {
	queries := r.queries
	if sess != nil {
		queries = queries.WithTx(sess.Tx())
	}
	recommended := logEntry.RecommendedItems
	if recommended == nil {
		recommended = []po.RecommendedItemLog{}
	}
	recommendedPayload, err := json.Marshal(recommended)
	if err != nil {
		return fmt.Errorf("marshal recommended_items: %w", err)
	}
	missing := logEntry.MissingVideoIDs
	if missing == nil {
		missing = []string{}
	}
	missingPayload, err := json.Marshal(missing)
	if err != nil {
		return fmt.Errorf("marshal missing_video_ids: %w", err)
	}
	var generatedAt *time.Time
	if !logEntry.GeneratedAt.IsZero() {
		gt := logEntry.GeneratedAt.UTC()
		generatedAt = &gt
	}
	params := feeddb.InsertRecommendationLogParams{
		UserID:                  mappers.ToPgText(logEntry.UserID),
		RequestLimit:            logEntry.RequestLimit,
		RecommendationSource:    logEntry.RecommendationSource,
		RecommendationLatencyMs: mappers.ToPgInt4(logEntry.RecommendationLatencyMS),
		RecommendedItems:        recommendedPayload,
		MissingVideoIds:         missingPayload,
		ErrorKind:               mappers.ToPgText(logEntry.ErrorKind),
		GeneratedAt:             mappers.ToPgTimestamptzPtr(generatedAt),
	}
	if err := queries.InsertRecommendationLog(ctx, params); err != nil {
		r.log.WithContext(ctx).Errorw("msg", "insert feed recommendation log failed", "error", err)
		return fmt.Errorf("insert feed recommendation log: %w", err)
	}
	return nil
}

// GetByID 按 log_id 查询推荐日志。
func (r *FeedRecommendationLogRepository) GetByID(ctx context.Context, sess txmanager.Session, id uuid.UUID) (*po.FeedRecommendationLog, error) {
	queries := r.queries
	if sess != nil {
		queries = queries.WithTx(sess.Tx())
	}
	row, err := queries.GetRecommendationLog(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get recommendation log: %w", err)
	}
	return mappers.FeedRecommendationLogFromRow(row)
}

// ListRecommendationLogsParams 描述推荐日志的查询条件。
type ListRecommendationLogsParams struct {
	UserID *string
	Source *string
	Since  *time.Time
	Until  *time.Time
	Limit  int
}

// List 返回满足条件的推荐日志，按时间倒序排序。
func (r *FeedRecommendationLogRepository) List(ctx context.Context, sess txmanager.Session, params ListRecommendationLogsParams) ([]*po.FeedRecommendationLog, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = 50
	}
	queries := r.queries
	if sess != nil {
		queries = queries.WithTx(sess.Tx())
	}
	dbParams := feeddb.ListRecommendationLogsParams{
		UserID:   textFromPtr(params.UserID),
		Source:   textFromPtr(params.Source),
		Since:    timestamptzFromPtr(params.Since),
		Until:    timestamptzFromPtr(params.Until),
		RowLimit: int32(limit),
	}
	rows, err := queries.ListRecommendationLogs(ctx, dbParams)
	if err != nil {
		return nil, fmt.Errorf("list recommendation logs: %w", err)
	}
	result := make([]*po.FeedRecommendationLog, 0, len(rows))
	for _, row := range rows {
		entry, mapErr := mappers.FeedRecommendationLogFromRow(row)
		if mapErr != nil {
			return nil, mapErr
		}
		result = append(result, entry)
	}
	return result, nil
}

func textFromPtr(ptr *string) pgtype.Text {
	if ptr == nil || *ptr == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *ptr, Valid: true}
}

func timestamptzFromPtr(ptr *time.Time) pgtype.Timestamptz {
	if ptr == nil || ptr.IsZero() {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: ptr.UTC(), Valid: true}
}
