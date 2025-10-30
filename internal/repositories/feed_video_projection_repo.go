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

// FeedVideoProjectionRepository 维护 feed.videos_projection 投影。
type FeedVideoProjectionRepository struct {
	db      *pgxpool.Pool
	queries *feeddb.Queries
	log     *log.Helper
}

// NewFeedVideoProjectionRepository 构造仓储实例。
func NewFeedVideoProjectionRepository(db *pgxpool.Pool, logger log.Logger) *FeedVideoProjectionRepository {
	return &FeedVideoProjectionRepository{
		db:      db,
		queries: feeddb.New(db),
		log:     log.NewHelper(logger),
	}
}

// UpsertFeedVideoProjectionInput 描述投影写入参数。
type UpsertFeedVideoProjectionInput struct {
	VideoID           uuid.UUID
	Title             string
	Description       *string
	DurationMicros    *int64
	ThumbnailURL      *string
	HLSMasterPlaylist *string
	Status            *string
	VisibilityStatus  *string
	PublishedAt       *time.Time
	Version           int64
	UpdatedAt         *time.Time
}

// Upsert 写入或更新投影记录。
func (r *FeedVideoProjectionRepository) Upsert(ctx context.Context, sess txmanager.Session, input UpsertFeedVideoProjectionInput) error {
	queries := r.queries
	if sess != nil {
		queries = queries.WithTx(sess.Tx())
	}
	params := feeddb.UpsertVideoProjectionParams{
		VideoID:           input.VideoID,
		Title:             input.Title,
		Description:       mappers.ToPgText(input.Description),
		DurationMicros:    mappers.ToPgInt8(input.DurationMicros),
		ThumbnailUrl:      mappers.ToPgText(input.ThumbnailURL),
		HlsMasterPlaylist: mappers.ToPgText(input.HLSMasterPlaylist),
		Status:            mappers.ToPgText(input.Status),
		VisibilityStatus:  mappers.ToPgText(input.VisibilityStatus),
		PublishedAt:       mappers.ToPgTimestamptzPtr(input.PublishedAt),
		Version:           input.Version,
		Column11:          mappers.ToPgTimestamptzPtr(input.UpdatedAt),
	}
	if err := queries.UpsertVideoProjection(ctx, params); err != nil {
		r.log.WithContext(ctx).Errorw("msg", "upsert feed video projection failed", "video_id", input.VideoID, "error", err)
		return fmt.Errorf("upsert feed video projection: %w", err)
	}
	return nil
}

// Get 返回单个投影。
func (r *FeedVideoProjectionRepository) Get(ctx context.Context, sess txmanager.Session, videoID uuid.UUID) (*po.FeedVideoProjection, error) {
	queries := r.queries
	if sess != nil {
		queries = queries.WithTx(sess.Tx())
	}
	row, err := queries.GetVideoProjection(ctx, videoID)
	if err != nil {
		return nil, fmt.Errorf("get feed video projection: %w", err)
	}
	return mappers.FeedVideoProjectionFromRow(row), nil
}

// ListByIDs 批量读取投影。
func (r *FeedVideoProjectionRepository) ListByIDs(ctx context.Context, sess txmanager.Session, ids []uuid.UUID) ([]*po.FeedVideoProjection, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	queries := r.queries
	if sess != nil {
		queries = queries.WithTx(sess.Tx())
	}
	rows, err := queries.ListVideoProjections(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("list feed video projections: %w", err)
	}
	result := make([]*po.FeedVideoProjection, 0, len(rows))
	for _, row := range rows {
		result = append(result, mappers.FeedVideoProjectionFromRow(row))
	}
	return result, nil
}

// ListRandomIDs 返回随机挑选的 video_id 列表。
func (r *FeedVideoProjectionRepository) ListRandomIDs(ctx context.Context, sess txmanager.Session, limit int) ([]uuid.UUID, error) {
	if limit <= 0 {
		return nil, nil
	}
	queries := r.queries
	if sess != nil {
		queries = queries.WithTx(sess.Tx())
	}
	rows, err := queries.ListRandomVideoIDs(ctx, int32(limit))
	if err != nil {
		return nil, fmt.Errorf("list random feed video ids: %w", err)
	}
	ids := make([]uuid.UUID, len(rows))
	for i, id := range rows {
		ids[i] = id
	}
	return ids, nil
}
