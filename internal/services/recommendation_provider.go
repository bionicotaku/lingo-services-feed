package services

import (
	"context"
	"errors"
)

// RecommendationProvider 抽象推荐系统的调用能力。
type RecommendationProvider interface {
	GetFeed(ctx context.Context, input RecommendationInput) (*RecommendationResult, error)
}

// RecommendationInput 描述推荐请求参数。
type RecommendationInput struct {
	UserID string
	Limit  int
}

// RecommendationResult 包含推荐条目与下一游标。
type RecommendationResult struct {
	Items      []RecommendationItem
	NextCursor string
}

// RecommendationItem 表示推荐返回的单条数据。
type RecommendationItem struct {
	VideoID  string
	Reason   string
	Score    float64
	Metadata map[string]string
}

// ErrRecommendationUnavailable 表示推荐不可用。
var ErrRecommendationUnavailable = errors.New("recommendation unavailable")
