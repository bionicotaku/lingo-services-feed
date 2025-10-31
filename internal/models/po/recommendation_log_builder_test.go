package po

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewFeedRecommendationLog_PopulatesFields(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)
	recommended := []RecommendedItemLog{
		{
			VideoID: "v1",
			Reason:  "mock.random",
			Score:   0.8,
			Meta: map[string]string{
				"experiment": "exp-1",
			},
		},
	}
	missing := []string{"v2"}

	params := FeedRecommendationLogParams{
		UserID:                  "user-1",
		RequestLimit:            5,
		RecommendationSource:    " mock ",
		RecommendationLatencyMS: 123,
		RecommendedItems:        recommended,
		MissingVideoIDs:         missing,
		ErrorKind:               "projection_error",
		GeneratedAt:             now,
	}

	entry := NewFeedRecommendationLog(params)

	require.NotNil(t, entry.UserID)
	require.Equal(t, "user-1", *entry.UserID)
	require.Equal(t, int32(5), entry.RequestLimit)
	require.Equal(t, "mock", entry.RecommendationSource)
	require.NotNil(t, entry.RecommendationLatencyMS)
	require.Equal(t, int32(123), *entry.RecommendationLatencyMS)
	require.Equal(t, recommended, entry.RecommendedItems)
	require.Equal(t, missing, entry.MissingVideoIDs)
	require.NotNil(t, entry.ErrorKind)
	require.Equal(t, "projection_error", *entry.ErrorKind)
	require.WithinDuration(t, now, entry.GeneratedAt, time.Millisecond)

	// Mutate original slices/maps to ensure cloning occurred.
	recommended[0].Meta["experiment"] = "changed"
	missing[0] = "other"
	require.Equal(t, "exp-1", entry.RecommendedItems[0].Meta["experiment"])
	require.Equal(t, []string{"v2"}, entry.MissingVideoIDs)
}

func TestNewFeedRecommendationLog_Defaults(t *testing.T) {
	params := FeedRecommendationLogParams{
		RequestLimit:         10,
		RecommendationSource: "",
	}

	entry := NewFeedRecommendationLog(params)

	require.Nil(t, entry.UserID)
	require.Equal(t, int32(10), entry.RequestLimit)
	require.Equal(t, "", entry.RecommendationSource)
	require.Nil(t, entry.RecommendationLatencyMS)
	require.Empty(t, entry.RecommendedItems)
	require.Empty(t, entry.MissingVideoIDs)
	require.Nil(t, entry.ErrorKind)
	require.False(t, entry.GeneratedAt.IsZero())
}
