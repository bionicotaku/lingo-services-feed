package repositories_test

import (
	context "context"
	"io"
	"testing"
	"time"

	"github.com/bionicotaku/lingo-services-feed/internal/models/po"
	"github.com/bionicotaku/lingo-services-feed/internal/repositories"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestFeedInboxRepositoryIntegration(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dsn, terminate := startPostgres(ctx, t)
	defer terminate()

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(func() { pool.Close() })

	applyMigrations(ctx, t, pool)

	repo := repositories.NewFeedInboxRepository(pool, log.NewStdLogger(io.Discard))

	eventID := uuid.New()
	receivedAt := time.Now().Add(-time.Minute).UTC()
	evt := po.FeedInboxEvent{
		EventID:       eventID.String(),
		SourceService: "catalog",
		EventType:     "catalog.video.created",
		AggregateType: stringPtr("video"),
		AggregateID:   stringPtr(uuid.New().String()),
		Payload:       []byte("payload"),
		ReceivedAt:    receivedAt,
	}

	require.NoError(t, repo.InsertInboxEvent(ctx, nil, evt))

	stored, err := repo.Get(ctx, nil, eventID)
	require.NoError(t, err)
	require.Equal(t, evt.SourceService, stored.SourceService)
	require.Equal(t, evt.EventType, stored.EventType)

	processedAt := time.Now().UTC()
	require.NoError(t, repo.MarkProcessed(ctx, nil, eventID, &processedAt))

	retryErr := repo.RecordError(ctx, nil, eventID, "temporary failure")
	require.NoError(t, retryErr)

	retrieved, err := repo.Get(ctx, nil, eventID)
	require.NoError(t, err)
	require.NotNil(t, retrieved.ProcessedAt)
	require.NotNil(t, retrieved.LastError)
}
