package controllers

import (
	"context"
	"errors"

	feedv1 "github.com/bionicotaku/lingo-services-feed/api/feed/v1"
	"github.com/bionicotaku/lingo-services-feed/internal/models/vo"
	"github.com/bionicotaku/lingo-services-feed/internal/services"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// feedServiceAPI 定义 FeedHandler 依赖的 Service 能力。
type feedServiceAPI interface {
	GetFeed(ctx context.Context, input services.GetFeedInput) (*vo.FeedResponse, error)
}

// FeedHandler 实现 FeedService gRPC 接口。
type FeedHandler struct {
	feedv1.UnimplementedFeedServiceServer

	*BaseHandler
	service feedServiceAPI
	log     *log.Helper
}

// NewFeedHandler 构造 FeedHandler。
func NewFeedHandler(feed feedServiceAPI, base *BaseHandler, logger log.Logger) *FeedHandler {
	if base == nil {
		base = NewBaseHandler(HandlerTimeouts{})
	}
	return &FeedHandler{
		BaseHandler: base,
		service:     feed,
		log:         log.NewHelper(logger),
	}
}

// GetFeed 返回推荐结果。当前尚未实现完整逻辑，后续步骤将替换占位实现。
func (h *FeedHandler) GetFeed(ctx context.Context, req *feedv1.GetFeedRequest) (*feedv1.GetFeedResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	meta := h.ExtractMetadata(ctx)
	if meta.InvalidUserInfo || meta.UserID == "" {
		return nil, status.Error(codes.Unauthenticated, "invalid user info")
	}

	userID := meta.UserID

	input := services.GetFeedInput{
		UserID: userID,
		Limit:  int(req.GetLimit()),
	}

	timeoutCtx, cancel := h.WithTimeout(ctx, HandlerTypeQuery)
	defer cancel()

	res, err := h.service.GetFeed(timeoutCtx, input)
	switch {
	case err == nil:
		return toProtoFeedResponse(res), nil
	case errors.Is(err, services.ErrRecommendationUnavailable):
		return nil, status.Error(codes.Unavailable, err.Error())
	default:
		h.log.WithContext(ctx).Errorw("msg", "get feed failed", "error", err)
		return nil, status.Errorf(codes.Internal, "get feed: %v", err)
	}
}

func toProtoFeedResponse(res *vo.FeedResponse) *feedv1.GetFeedResponse {
	if res == nil {
		return &feedv1.GetFeedResponse{}
	}
	resp := &feedv1.GetFeedResponse{
		NextCursor: res.NextCursor,
		Partial:    res.Partial,
	}
	if !res.GeneratedAt.IsZero() {
		resp.GeneratedAt = timestamppb.New(res.GeneratedAt.UTC())
	}
	resp.Items = make([]*feedv1.FeedItem, 0, len(res.Items))
	for _, item := range res.Items {
		resp.Items = append(resp.Items, toProtoFeedItem(item))
	}
	resp.MissingProjections = make([]*feedv1.MissingProjection, 0, len(res.MissingProjections))
	for _, missing := range res.MissingProjections {
		resp.MissingProjections = append(resp.MissingProjections, &feedv1.MissingProjection{
			VideoId: missing.VideoID,
			Reason:  missing.Reason,
		})
	}
	return resp
}

func toProtoFeedItem(item vo.FeedItem) *feedv1.FeedItem {
	feedItem := &feedv1.FeedItem{
		VideoId:           item.VideoID,
		Title:             item.Title,
		Description:       item.Description,
		DurationMicros:    item.DurationMicros,
		ThumbnailUrl:      item.ThumbnailURL,
		HlsMasterPlaylist: item.HLSMasterPlaylist,
		ReasonCode:        item.ReasonCode,
		ReasonLabel:       item.ReasonLabel,
		Score:             item.Score,
		VisibilityStatus:  item.VisibilityStatus,
		Attributes:        map[string]string{},
	}
	if item.Attributes != nil {
		for k, v := range item.Attributes {
			feedItem.Attributes[k] = v
		}
	}
	if item.PublishedAt != nil && !item.PublishedAt.IsZero() {
		feedItem.PublishedAt = timestamppb.New(item.PublishedAt.UTC())
	}
	return feedItem
}
