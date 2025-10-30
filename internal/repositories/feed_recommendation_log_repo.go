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
