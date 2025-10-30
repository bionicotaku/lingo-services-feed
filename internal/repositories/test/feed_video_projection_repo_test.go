package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/bionicotaku/lingo-services-feed/internal/repositories"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestFeedVideoProjectionRepository_UpsertAndGet(t *testing.T) {
	resetDatabase(t)

	ctx := context.Background()
	repo := newVideoProjectionRepo()

	videoID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)
	desc := "desc"
	duration := int64(120 * time.Second)
	status := "published"
	upsertErr := repo.Upsert(ctx, nil, repositories.UpsertFeedVideoProjectionInput{
		VideoID:           videoID,
		Title:             "Sample Title",
		Description:       &desc,
		DurationMicros:    &duration,
		ThumbnailURL:      stringPtr("https://cdn/mock.jpg"),
		HLSMasterPlaylist: stringPtr("https://cdn/mock.m3u8"),
		Status:            &status,
		VisibilityStatus:  stringPtr("public"),
		PublishedAt:       timePtr(now),
		Version:           1,
		UpdatedAt:         timePtr(now),
	})
	require.NoError(t, upsertErr)

	record, err := repo.Get(ctx, nil, videoID)
	require.NoError(t, err)
	require.Equal(t, videoID.String(), record.VideoID)
	require.Equal(t, "Sample Title", record.Title)
	require.NotNil(t, record.Description)
	require.Equal(t, desc, *record.Description)
	require.NotNil(t, record.DurationMicros)
	require.Equal(t, duration, *record.DurationMicros)
	require.NotNil(t, record.PublishedAt)
	require.WithinDuration(t, now, *record.PublishedAt, time.Second)
	require.Equal(t, int64(1), record.Version)
}

func TestFeedVideoProjectionRepository_ListByIDs(t *testing.T) {
	resetDatabase(t)

	ctx := context.Background()
	repo := newVideoProjectionRepo()

	videoOne := uuid.New()
	videoTwo := uuid.New()

	require.NoError(t, repo.Upsert(ctx, nil, repositories.UpsertFeedVideoProjectionInput{
		VideoID: videoOne,
		Title:   "Video One",
		Version: 1,
	}))
	require.NoError(t, repo.Upsert(ctx, nil, repositories.UpsertFeedVideoProjectionInput{
		VideoID: videoTwo,
		Title:   "Video Two",
		Version: 2,
	}))

	records, err := repo.ListByIDs(ctx, nil, []uuid.UUID{videoTwo, videoOne})
	require.NoError(t, err)
	require.Len(t, records, 2)

	found := map[string]int64{}
	for _, rec := range records {
		found[rec.VideoID] = rec.Version
	}
	require.Equal(t, int64(1), found[videoOne.String()])
	require.Equal(t, int64(2), found[videoTwo.String()])

	empty, err := repo.ListByIDs(ctx, nil, nil)
	require.NoError(t, err)
	require.Nil(t, empty)
}

func TestFeedVideoProjectionRepository_ListRandomIDs(t *testing.T) {
	resetDatabase(t)

	ctx := context.Background()
	repo := newVideoProjectionRepo()

	videoIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	for idx, id := range videoIDs {
		status := "ready"
		require.NoError(t, repo.Upsert(ctx, nil, repositories.UpsertFeedVideoProjectionInput{
			VideoID: id,
			Title:   "Video",
			Status:  &status,
			Version: int64(idx + 1),
		}))
	}

	ids, err := repo.ListRandomIDs(ctx, nil, 2)
	require.NoError(t, err)
	require.Len(t, ids, 2)
	seen := map[uuid.UUID]struct{}{}
	for _, id := range ids {
		seen[id] = struct{}{}
	}
	require.Len(t, seen, 2)

	none, err := repo.ListRandomIDs(ctx, nil, 0)
	require.NoError(t, err)
	require.Nil(t, none)
}
