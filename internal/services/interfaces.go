package services

import (
	"context"

	"github.com/bionicotaku/lingo-services-feed/internal/models/vo"
)

// FeedServiceInterface 抽象 Feed 获取用例，便于测试替换。
type FeedServiceInterface interface {
	GetFeed(ctx context.Context, input GetFeedInput) (*vo.FeedResponse, error)
}
