package repositories_test

import (
	context "context"
	"io"
	"testing"
	"time"

	"github.com/bionicotaku/lingo-services-feed/internal/repositories"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestFeedVideoProjectionRepositoryIntegration(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dsn, terminate := startPostgres(ctx, t)
	defer terminate()

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(func() { pool.Close() })

	applyMigrations(ctx, t, pool)

	repo := repositories.NewFeedVideoProjectionRepository(pool, log.NewStdLogger(io.Discard))

	videoID := uuid.New()
	updatedAt := time.Now().UTC()
	publishedAt := updatedAt.Add(-time.Hour)
	duration := int64(90_000_000)

	input := repositories.UpsertFeedVideoProjectionInput{
		VideoID:           videoID,
		Title:             "Feed Title",
		Description:       stringPtr("Feed Desc"),
		DurationMicros:    &duration,
		ThumbnailURL:      stringPtr("https://example.com/feed-thumb.jpg"),
		HLSMasterPlaylist: stringPtr("https://example.com/feed-master.m3u8"),
		Status:            stringPtr("ready"),
		VisibilityStatus:  stringPtr("public"),
		PublishedAt:       &publishedAt,
		Version:           1,
		UpdatedAt:         &updatedAt,
	}

	require.NoError(t, repo.Upsert(ctx, nil, input))

	record, err := repo.Get(ctx, nil, videoID)
	require.NoError(t, err)
	require.Equal(t, "Feed Title", record.Title)
	require.Equal(t, int64(1), record.Version)
	require.Equal(t, "Feed Desc", derefString(record.Description))
	require.Equal(t, int64(90_000_000), derefInt64(record.DurationMicros))

	input.Title = "Feed Title Updated"
	input.Version = 2
	require.NoError(t, repo.Upsert(ctx, nil, input))

	list, err := repo.ListByIDs(ctx, nil, []uuid.UUID{videoID})
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "Feed Title Updated", list[0].Title)
	require.Equal(t, int64(2), list[0].Version)
}
