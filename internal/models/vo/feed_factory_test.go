package vo

import (
	"testing"
	"time"

	"github.com/bionicotaku/lingo-services-feed/internal/models/po"
	"github.com/stretchr/testify/require"
)

func TestFeedItemFromProjection(t *testing.T) {
	now := time.Now().UTC()
	title := "Sample Title"
	description := "Sample Description"
	duration := int64(123)
	thumbnail := "https://example.com/thumb.jpg"
	status := "public"
	record := &po.FeedVideoProjection{
		VideoID:           "video-1",
		Title:             title,
		Description:       &description,
		DurationMicros:    &duration,
		ThumbnailURL:      &thumbnail,
		HLSMasterPlaylist: nil,
		VisibilityStatus:  &status,
		PublishedAt:       &now,
	}

	item := FeedItemFromProjection(record)

	require.Equal(t, "video-1", item.VideoID)
	require.Equal(t, title, item.Title)
	require.Equal(t, description, item.Description)
	require.Equal(t, duration, item.DurationMicros)
	require.Equal(t, thumbnail, item.ThumbnailURL)
	require.Equal(t, status, item.VisibilityStatus)
	require.NotNil(t, item.PublishedAt)
	require.WithinDuration(t, now, *item.PublishedAt, time.Second)
	require.NotNil(t, item.Attributes)
}

func TestFeedItemFromProjection_NilRecord(t *testing.T) {
	item := FeedItemFromProjection(nil)
	require.Equal(t, "", item.VideoID)
	require.NotNil(t, item.Attributes)
	require.Empty(t, item.Attributes)
}

func TestFeedItem_ApplyRecommendation(t *testing.T) {
	item := FeedItem{}
	meta := map[string]string{
		"reason_label": "Because you liked X",
		"experiment":   "exp-1",
	}

	item.ApplyRecommendation("algo.mock", meta, 0.87)

	require.Equal(t, "algo.mock", item.ReasonCode)
	require.Equal(t, "Because you liked X", item.ReasonLabel)
	require.Equal(t, 0.87, item.Score)
	require.Equal(t, "exp-1", item.Attributes["experiment"])

	// Ensure reason_label is not duplicated in Attributes.
	_, exists := item.Attributes["reason_label"]
	require.False(t, exists)

	// Applying again with nil metadata should not panic.
	item.ApplyRecommendation("algo.mock", nil, 0.9)
	require.Equal(t, 0.9, item.Score)
}
