package services

import (
	"context"
	"fmt"
	"time"

	"github.com/bionicotaku/lingo-services-feed/internal/models/po"
	"github.com/bionicotaku/lingo-services-feed/internal/models/vo"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
)

// GetFeedInput 描述获取 Feed 所需的参数。
type GetFeedInput struct {
	UserID string
	Scene  string
	Limit  int
	Cursor string
	Meta   map[string]string
}

// FeedService 是 Feed MVP 的主用例，后续步骤会注入推荐 Provider 与投影仓储。
type FeedService struct {
	recommendations RecommendationProvider
	projections     FeedProjectionRepository
	log             *log.Helper
}

// NewFeedService 构造 FeedService。
func NewFeedService(recommendations RecommendationProvider, projections FeedProjectionRepository, logger log.Logger) *FeedService {
	return &FeedService{
		recommendations: recommendations,
		projections:     projections,
		log:             log.NewHelper(logger),
	}
}

// GetFeed 返回推荐结果。
func (s *FeedService) GetFeed(ctx context.Context, input GetFeedInput) (*vo.FeedResponse, error) {
	limit := input.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	recResult, err := s.recommendations.GetFeed(ctx, RecommendationInput{
		UserID: input.UserID,
		Scene:  input.Scene,
		Limit:  limit,
		Cursor: input.Cursor,
		Meta:   input.Meta,
	})
	if err != nil {
		return nil, err
	}
	resp := &vo.FeedResponse{
		NextCursor:  "",
		GeneratedAt: time.Now().UTC(),
	}
	if recResult != nil {
		resp.NextCursor = recResult.NextCursor
	}
	if recResult == nil || len(recResult.Items) == 0 {
		return resp, nil
	}
	videoIDs := make([]uuid.UUID, 0, len(recResult.Items))
	missing := make([]vo.MissingProjection, 0)
	for _, item := range recResult.Items {
		id, parseErr := uuid.Parse(item.VideoID)
		if parseErr != nil {
			missing = append(missing, vo.MissingProjection{VideoID: item.VideoID, Reason: "invalid video id"})
			continue
		}
		videoIDs = append(videoIDs, id)
	}
	projections := map[string]*vo.FeedItem{}
	if len(videoIDs) > 0 {
		records, repoErr := s.projections.ListByIDs(ctx, nil, videoIDs)
		if repoErr != nil {
			return nil, fmt.Errorf("list projections: %w", repoErr)
		}
		for _, record := range records {
			if record == nil {
				continue
			}
			projections[record.VideoID] = projectionToFeedItem(record)
		}
	}
	items := make([]vo.FeedItem, 0, len(recResult.Items))
	for _, rec := range recResult.Items {
		if feedItem, ok := projections[rec.VideoID]; ok {
			feedItem.ReasonCode = rec.Reason
			feedItem.ReasonLabel = rec.Metadata["reason_label"]
			feedItem.Score = rec.Score
			mergeAttributes(feedItem, rec.Metadata)
			items = append(items, *feedItem)
			continue
		}
		missing = append(missing, vo.MissingProjection{VideoID: rec.VideoID, Reason: "projection missing"})
	}
	resp.Items = items
	resp.MissingProjections = missing
	resp.Partial = len(missing) > 0
	return resp, nil
}

func projectionToFeedItem(record *po.FeedVideoProjection) *vo.FeedItem {
	item := &vo.FeedItem{
		VideoID:           record.VideoID,
		Title:             record.Title,
		Description:       derefString(record.Description),
		DurationMicros:    derefInt64(record.DurationMicros),
		ThumbnailURL:      derefString(record.ThumbnailURL),
		HLSMasterPlaylist: derefString(record.HLSMasterPlaylist),
		VisibilityStatus:  derefString(record.VisibilityStatus),
		Attributes:        map[string]string{},
	}
	if record.PublishedAt != nil {
		item.PublishedAt = record.PublishedAt
	}
	return item
}

func mergeAttributes(item *vo.FeedItem, meta map[string]string) {
	if item == nil || meta == nil {
		return
	}
	if item.Attributes == nil {
		item.Attributes = make(map[string]string, len(meta))
	}
	for k, v := range meta {
		if k == "reason_label" {
			if item.ReasonLabel == "" {
				item.ReasonLabel = v
			}
			continue
		}
		item.Attributes[k] = v
	}
}

func derefString(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

func derefInt64(ptr *int64) int64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}
