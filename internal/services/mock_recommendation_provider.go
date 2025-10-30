package services

import (
	"context"
	"math/rand"
	"time"

	"github.com/bionicotaku/lingo-services-feed/internal/repositories"
	"github.com/go-kratos/kratos/v2/log"
)

// MockRecommendationProvider 根据本地投影随机返回视频。
type MockRecommendationProvider struct {
	repo *repositories.FeedVideoProjectionRepository
	rng  *rand.Rand
	log  *log.Helper
}

const mockRecommendationSource = "mock"

// Source 返回推荐来源标识。
func (p *MockRecommendationProvider) Source() string {
	return mockRecommendationSource
}

// NewMockRecommendationProvider 构造基于投影表的 Mock 推荐实现。
func NewMockRecommendationProvider(repo *repositories.FeedVideoProjectionRepository, logger log.Logger) *MockRecommendationProvider {
	seed := time.Now().UnixNano()
	return &MockRecommendationProvider{
		repo: repo,
		rng:  rand.New(rand.NewSource(seed)),
		log:  log.NewHelper(logger),
	}
}

// GetFeed 返回随机挑选的视频 ID。
func (p *MockRecommendationProvider) GetFeed(ctx context.Context, input RecommendationInput) (*RecommendationResult, error) {
	limit := input.Limit
	if limit <= 0 {
		limit = 20
	}
	ids, err := p.repo.ListRandomIDs(ctx, nil, limit)
	if err != nil {
		p.log.WithContext(ctx).Errorw("msg", "mock recommendation list ids failed", "error", err)
		return nil, ErrRecommendationUnavailable
	}
	items := make([]RecommendationItem, 0, len(ids))
	for _, id := range ids {
		items = append(items, RecommendationItem{
			VideoID: id.String(),
			Reason:  "mock.random",
			Score:   p.rng.Float64(),
			Metadata: map[string]string{
				"source": mockRecommendationSource,
			},
		})
	}
	return &RecommendationResult{Items: items, Source: mockRecommendationSource}, nil
}

var _ RecommendationProvider = (*MockRecommendationProvider)(nil)
