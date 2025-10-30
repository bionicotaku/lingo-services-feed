package repositories_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/bionicotaku/lingo-services-feed/internal/models/po"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestFeedRecommendationLogRepository_Insert(t *testing.T) {
	resetDatabase(t)

	ctx := context.Background()
	repo := newRecommendationLogRepo()

	generated := time.Now().UTC().Truncate(time.Microsecond)
	latency := int32(42)
	userID := "user-1"
	recommended := []po.RecommendedItemLog{
		{VideoID: "v1", Reason: "mock.random", Score: 0.9},
		{VideoID: "v2", Reason: "mock.random", Score: 0.4},
	}
	entry := po.FeedRecommendationLog{
		UserID:                  &userID,
		RequestLimit:            5,
		RecommendationSource:    "mock",
		RecommendationLatencyMS: &latency,
		RecommendedItems:        recommended,
		MissingVideoIDs:         []string{"v2"},
		GeneratedAt:             generated,
	}

	require.NoError(t, repo.Insert(ctx, nil, entry))

	rows, err := testPool.Query(ctx, `
		select user_id,
		       request_limit,
		       recommendation_source,
		       recommendation_latency_ms,
		       recommended_items::text,
		       missing_video_ids::text,
		       error_kind,
		       generated_at
		from feed.recommendation_logs
	`)
	require.NoError(t, err)
	defer rows.Close()

	require.True(t, rows.Next())

	var dbUser pgtype.Text
	var requestLimit int32
	var source string
	var latencyDB pgtype.Int4
	var recommendedRaw string
	var missingRaw string
	var errorKind pgtype.Text
	var generatedAt time.Time

	require.NoError(t, rows.Scan(&dbUser, &requestLimit, &source, &latencyDB, &recommendedRaw, &missingRaw, &errorKind, &generatedAt))

	require.True(t, dbUser.Valid)
	require.Equal(t, userID, dbUser.String)
	require.Equal(t, int32(5), requestLimit)
	require.Equal(t, "mock", source)
	require.True(t, latencyDB.Valid)
	require.Equal(t, latency, latencyDB.Int32)
	require.False(t, errorKind.Valid)

	var recommendedLogged []po.RecommendedItemLog
	require.NoError(t, json.Unmarshal([]byte(recommendedRaw), &recommendedLogged))
	require.Equal(t, recommended, recommendedLogged)
	var missing []string
	require.NoError(t, json.Unmarshal([]byte(missingRaw), &missing))
	require.ElementsMatch(t, []string{"v2"}, missing)
	require.WithinDuration(t, generated, generatedAt, time.Second)
}
