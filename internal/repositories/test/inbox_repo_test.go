package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/bionicotaku/lingo-services-feed/internal/repositories/feeddb"
	"github.com/bionicotaku/lingo-utils/outbox/store"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestInboxRepository_InsertAndLifecycle(t *testing.T) {
	resetDatabase(t)

	ctx := context.Background()
	repo := newInboxRepo(t)
	eventID := uuid.New()
	aggregateID := "video-1"
	payload := []byte(`{"event":"created"}`)

	message := store.InboxMessage{
		EventID:       eventID,
		SourceService: "catalog",
		EventType:     "catalog.video.created",
		AggregateType: stringPtr("video"),
		AggregateID:   &aggregateID,
		Payload:       payload,
	}
	err := repo.Insert(ctx, nil, message)
	require.NoError(t, err)

	// Idempotent insert should not fail.
	require.NoError(t, repo.Insert(ctx, nil, message))

	queries := feeddb.New(testPool)
	record, err := queries.GetInboxEvent(ctx, eventID)
	require.NoError(t, err)
	require.Equal(t, "catalog", record.SourceService)
	require.Equal(t, "catalog.video.created", record.EventType)

	processTime := time.Now().UTC().Truncate(time.Microsecond)
	require.NoError(t, repo.MarkProcessed(ctx, nil, eventID, processTime))

	record, err = queries.GetInboxEvent(ctx, eventID)
	require.NoError(t, err)
	require.True(t, record.ProcessedAt.Valid)
	require.WithinDuration(t, processTime, record.ProcessedAt.Time, time.Second)
	require.False(t, record.LastError.Valid)

	require.NoError(t, repo.RecordError(ctx, nil, eventID, "transient failure"))

	record, err = queries.GetInboxEvent(ctx, eventID)
	require.NoError(t, err)
	require.True(t, record.LastError.Valid)
	require.Equal(t, "transient failure", record.LastError.String)
}
