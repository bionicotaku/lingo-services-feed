package controllers_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"testing"
	"time"

	feedv1 "github.com/bionicotaku/lingo-services-feed/api/feed/v1"
	controllers "github.com/bionicotaku/lingo-services-feed/internal/controllers"
	"github.com/bionicotaku/lingo-services-feed/internal/models/vo"
	"github.com/bionicotaku/lingo-services-feed/internal/services"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type stubFeedService struct {
	response *vo.FeedResponse
	err      error
	input    services.GetFeedInput
}

func (s *stubFeedService) GetFeed(_ context.Context, input services.GetFeedInput) (*vo.FeedResponse, error) {
	s.input = input
	return s.response, s.err
}

func TestFeedHandler_GetFeed_Success(t *testing.T) {
	service := &stubFeedService{
		response: &vo.FeedResponse{
			Items: []vo.FeedItem{
				{VideoID: "v1", Title: "Video 1", ReasonCode: "mock"},
			},
			GeneratedAt: time.Now(),
		},
	}
	handler := controllers.NewFeedHandler(service, controllers.NewBaseHandler(controllers.HandlerTimeouts{}), log.NewStdLogger(io.Discard))

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-apigateway-api-userinfo", encodeUserInfo(t, map[string]any{"sub": "user-1"})))
	resp, err := handler.GetFeed(ctx, &feedv1.GetFeedRequest{Limit: 5})
	require.NoError(t, err)
	require.Len(t, resp.GetItems(), 1)
	require.Equal(t, "v1", resp.GetItems()[0].GetVideoId())
	require.Equal(t, "user-1", service.input.UserID)
	require.Equal(t, 5, service.input.Limit)
}

func TestFeedHandler_GetFeed_InvalidMetadata(t *testing.T) {
	service := &stubFeedService{}
	handler := controllers.NewFeedHandler(service, controllers.NewBaseHandler(controllers.HandlerTimeouts{}), log.NewStdLogger(io.Discard))

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-apigateway-api-userinfo", "invalid-base64"))
	_, err := handler.GetFeed(ctx, &feedv1.GetFeedRequest{Limit: 1})
	require.Error(t, err)
	st, _ := status.FromError(err)
	require.Equal(t, codes.Unauthenticated, st.Code())

	ctx = metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-apigateway-api-userinfo", encodeUserInfo(t, map[string]any{})))
	_, err = handler.GetFeed(ctx, &feedv1.GetFeedRequest{Limit: 1})
	require.Error(t, err)
	st, _ = status.FromError(err)
	require.Equal(t, codes.Unauthenticated, st.Code())
}

func TestFeedHandler_GetFeed_RecommendationUnavailable(t *testing.T) {
	service := &stubFeedService{err: services.ErrRecommendationUnavailable}
	handler := controllers.NewFeedHandler(service, controllers.NewBaseHandler(controllers.HandlerTimeouts{}), log.NewStdLogger(io.Discard))

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-apigateway-api-userinfo", encodeUserInfo(t, map[string]any{"sub": "user-2"})))
	_, err := handler.GetFeed(ctx, &feedv1.GetFeedRequest{Limit: 3})
	require.Error(t, err)
	st, _ := status.FromError(err)
	require.Equal(t, codes.Unavailable, st.Code())
	require.Equal(t, "user-2", service.input.UserID)
}

func encodeUserInfo(t *testing.T, claims map[string]any) string {
	t.Helper()
	payload, err := json.Marshal(claims)
	require.NoError(t, err)
	return base64.RawURLEncoding.EncodeToString(payload)
}
