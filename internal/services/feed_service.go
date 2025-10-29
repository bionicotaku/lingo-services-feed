package services

import (
	"context"
	"errors"

	"github.com/bionicotaku/lingo-services-feed/internal/models/vo"
	"github.com/go-kratos/kratos/v2/log"
)

// ErrFeedServiceNotImplemented 表示 Feed 业务尚未完成，后续步骤会逐步替换。
var ErrFeedServiceNotImplemented = errors.New("feed service not implemented")

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
	log *log.Helper
}

// NewFeedService 构造 FeedService。
func NewFeedService(logger log.Logger) *FeedService {
	return &FeedService{
		log: log.NewHelper(logger),
	}
}

// GetFeed 返回推荐结果。当前暂未实现，后续章节将逐步完善。
func (s *FeedService) GetFeed(ctx context.Context, input GetFeedInput) (*vo.FeedResponse, error) {
	_ = ctx
	_ = input
	return nil, ErrFeedServiceNotImplemented
}
