package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bionicotaku/lingo-services-feed/internal/models/po"
	"github.com/bionicotaku/lingo-services-feed/internal/models/vo"
	"github.com/bionicotaku/lingo-services-feed/internal/repositories"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
)

// GetFeedInput 描述获取 Feed 所需的参数。
type GetFeedInput struct {
	UserID string
	Limit  int
}

// FeedService 是 Feed MVP 的主用例，后续步骤会注入推荐 Provider 与投影仓储。
type FeedService struct {
	recommendations RecommendationProvider
	projections     *repositories.FeedVideoProjectionRepository
	logs            *repositories.FeedRecommendationLogRepository
	log             *log.Helper
}

// NewFeedService 构造 FeedService。
func NewFeedService(recommendations RecommendationProvider, projections *repositories.FeedVideoProjectionRepository, logs *repositories.FeedRecommendationLogRepository, logger log.Logger) *FeedService {
	return &FeedService{
		recommendations: recommendations,
		projections:     projections,
		logs:            logs,
		log:             log.NewHelper(logger),
	}
}

// GetFeed 返回推荐结果。
func (s *FeedService) GetFeed(ctx context.Context, input GetFeedInput) (*vo.FeedResponse, error) {
	limit := input.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	startedAt := time.Now()
	recResult, err := s.recommendations.GetFeed(ctx, RecommendationInput{
		UserID: input.UserID,
		Limit:  limit,
	})
	latencyMs := millisOrZero(time.Since(startedAt))
	source := s.resolveRecommendationSource(recResult)
	if err != nil {
		s.logRecommendation(ctx, recommendationLogParams{
			UserID:           input.UserID,
			Limit:            limit,
			Source:           source,
			LatencyMs:        latencyMs,
			RecommendedItems: nil,
			MissingVideoIDs:  nil,
			ErrorKind:        errorKindFromError(err),
			GeneratedAt:      time.Now().UTC(),
		})
		return nil, err
	}
	resp := &vo.FeedResponse{
		GeneratedAt: time.Now().UTC(),
	}
	if recResult == nil || len(recResult.Items) == 0 {
		s.logRecommendation(ctx, recommendationLogParams{
			UserID:           input.UserID,
			Limit:            limit,
			Source:           source,
			LatencyMs:        latencyMs,
			RecommendedItems: toRecommendedLogItems(nil),
			MissingVideoIDs:  nil,
			GeneratedAt:      resp.GeneratedAt,
		})
		return resp, nil
	}
	recommendedLogItems := toRecommendedLogItems(recResult.Items)
	videoIDs := make([]uuid.UUID, 0, len(recResult.Items))
	missing := make([]vo.MissingProjection, 0)
	missingIDs := make([]string, 0)
	for _, item := range recResult.Items {
		id, parseErr := uuid.Parse(item.VideoID)
		if parseErr != nil {
			missing = append(missing, vo.MissingProjection{VideoID: item.VideoID, Reason: "invalid video id"})
			missingIDs = append(missingIDs, item.VideoID)
			continue
		}
		videoIDs = append(videoIDs, id)
	}
	projections := map[string]*vo.FeedItem{}
	if len(videoIDs) > 0 {
		records, repoErr := s.projections.ListByIDs(ctx, nil, videoIDs)
		if repoErr != nil {
			s.logRecommendation(ctx, recommendationLogParams{
				UserID:           input.UserID,
				Limit:            limit,
				Source:           source,
				LatencyMs:        latencyMs,
				RecommendedItems: recommendedLogItems,
				MissingVideoIDs:  missingIDs,
				ErrorKind:        "projection_error",
				GeneratedAt:      resp.GeneratedAt,
			})
			return nil, fmt.Errorf("list projections: %w", repoErr)
		}
		for _, record := range records {
			if record == nil {
				continue
			}
			item := vo.FeedItemFromProjection(record)
			projections[record.VideoID] = &item
		}
	}
	items := make([]vo.FeedItem, 0, len(recResult.Items))
	for _, rec := range recResult.Items {
		if feedItem, ok := projections[rec.VideoID]; ok {
			feedItem.ApplyRecommendation(rec.Reason, rec.Metadata, rec.Score)
			items = append(items, *feedItem)
			continue
		}
		missing = append(missing, vo.MissingProjection{VideoID: rec.VideoID, Reason: "projection missing"})
		missingIDs = append(missingIDs, rec.VideoID)
	}
	resp.Items = items
	resp.MissingProjections = missing
	resp.Partial = len(missing) > 0
	s.logRecommendation(ctx, recommendationLogParams{
		UserID:           input.UserID,
		Limit:            limit,
		Source:           source,
		LatencyMs:        latencyMs,
		RecommendedItems: recommendedLogItems,
		MissingVideoIDs:  missingIDs,
		GeneratedAt:      resp.GeneratedAt,
	})
	return resp, nil
}

type recommendationLogParams struct {
	UserID           string
	Limit            int
	Source           string
	LatencyMs        int32
	RecommendedItems []po.RecommendedItemLog
	MissingVideoIDs  []string
	ErrorKind        string
	GeneratedAt      time.Time
}

func (s *FeedService) logRecommendation(ctx context.Context, params recommendationLogParams) {
	if s.logs == nil {
		return
	}
	source := firstNonEmpty(params.Source, s.recommendations.Source())
	entry := po.NewFeedRecommendationLog(po.FeedRecommendationLogParams{
		UserID:                  params.UserID,
		RequestLimit:            params.Limit,
		RecommendationSource:    source,
		RecommendationLatencyMS: params.LatencyMs,
		RecommendedItems:        params.RecommendedItems,
		MissingVideoIDs:         params.MissingVideoIDs,
		ErrorKind:               params.ErrorKind,
		GeneratedAt:             params.GeneratedAt,
	})
	if err := s.logs.Insert(ctx, nil, entry); err != nil {
		s.log.WithContext(ctx).Warnw("msg", "write recommendation log failed", "error", err)
	}
}

func toRecommendedLogItems(items []RecommendationItem) []po.RecommendedItemLog {
	if len(items) == 0 {
		return []po.RecommendedItemLog{}
	}
	logs := make([]po.RecommendedItemLog, 0, len(items))
	for _, item := range items {
		logs = append(logs, po.RecommendedItemLog{
			VideoID: item.VideoID,
			Reason:  item.Reason,
			Score:   item.Score,
			Meta:    cloneStringMap(item.Metadata),
		})
	}
	return logs
}

func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(src))
	for k, v := range src {
		cloned[k] = v
	}
	return cloned
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func millisOrZero(d time.Duration) int32 {
	if d <= 0 {
		return 0
	}
	maxInt32 := int64(1<<31 - 1)
	ms := d / time.Millisecond
	if ms > time.Duration(maxInt32)*time.Millisecond {
		return int32(maxInt32)
	}
	return int32(ms)
}

func (s *FeedService) resolveRecommendationSource(result *RecommendationResult) string {
	if result != nil && result.Source != "" {
		return result.Source
	}
	if s.recommendations != nil {
		return s.recommendations.Source()
	}
	return "unknown"
}

func errorKindFromError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, ErrRecommendationUnavailable) {
		return "recommendation_unavailable"
	}
	return "unknown_error"
}
