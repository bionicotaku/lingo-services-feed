package services_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bionicotaku/lingo-services-feed/internal/models/po"
	"github.com/bionicotaku/lingo-services-feed/internal/repositories"
	"github.com/bionicotaku/lingo-services-feed/internal/services"
	"github.com/docker/go-connections/nat"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	testPool      *pgxpool.Pool
	testContainer testcontainers.Container
	stdLogger     = log.NewStdLogger(io.Discard)
)

func TestMain(m *testing.M) {
	ctx := context.Background()
	code := 0
	if err := startPostgres(ctx); err != nil {
		code = 1
	} else {
		code = m.Run()
	}
	if testPool != nil {
		testPool.Close()
	}
	if testContainer != nil {
		termCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = testContainer.Terminate(termCtx)
	}
	os.Exit(code)
}

func startPostgres(ctx context.Context) error {
	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_PASSWORD": "postgres",
			"POSTGRES_USER":     "postgres",
			"POSTGRES_DB":       "feed",
		},
		WaitingFor: wait.ForSQL("5432/tcp", "pgx", func(host string, port nat.Port) string {
			return "postgres://postgres:postgres@" + host + ":" + port.Port() + "/feed?sslmode=disable&search_path=feed"
		}).WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return err
	}
	testContainer = container

	host, err := container.Host(ctx)
	if err != nil {
		return err
	}
	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		return err
	}

	dsn := "postgres://postgres:postgres@" + host + ":" + port.Port() + "/feed?sslmode=disable&search_path=feed"
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return err
	}
	testPool = pool
	return applyMigrations(ctx, pool)
}

func applyMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	migrationsDir := filepath.Join("..", "..", "migrations")
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		return err
	}
	for _, path := range files {
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if _, err := pool.Exec(ctx, string(content)); err != nil {
			return err
		}
	}
	return nil
}

func resetDatabase(t *testing.T) {
	t.Helper()
	_, err := testPool.Exec(context.Background(), `
		TRUNCATE TABLE
			feed.recommendation_logs,
			feed.videos_projection
		RESTART IDENTITY
	`)
	require.NoError(t, err)
}

func newFeedService(provider services.RecommendationProvider) *services.FeedService {
	videoRepo := repositories.NewFeedVideoProjectionRepository(testPool, stdLogger)
	logRepo := repositories.NewFeedRecommendationLogRepository(testPool, stdLogger)
	return services.NewFeedService(provider, videoRepo, logRepo, stdLogger)
}

type stubRecommendationProvider struct {
	items     []services.RecommendationItem
	err       error
	source    string
	lastInput services.RecommendationInput
}

func (s *stubRecommendationProvider) GetFeed(_ context.Context, input services.RecommendationInput) (*services.RecommendationResult, error) {
	s.lastInput = input
	if s.err != nil {
		return nil, s.err
	}
	items := append([]services.RecommendationItem(nil), s.items...)
	return &services.RecommendationResult{
		Items:  items,
		Source: s.Source(),
	}, nil
}

func (s *stubRecommendationProvider) Source() string {
	if strings.TrimSpace(s.source) == "" {
		return "stub"
	}
	return s.source
}

func TestFeedService_GetFeed_ReturnsItemsAndLogs(t *testing.T) {
	resetDatabase(t)
	ctx := context.Background()

	videoRepo := repositories.NewFeedVideoProjectionRepository(testPool, stdLogger)
	now := time.Now().UTC()

	video1 := uuid.New()
	video2 := uuid.New()

	require.NoError(t, videoRepo.Upsert(ctx, nil, repositories.UpsertFeedVideoProjectionInput{
		VideoID: video1,
		Title:   "Video One",
		Version: 1,
		UpdatedAt: func() *time.Time {
			ts := now
			return &ts
		}(),
	}))
	require.NoError(t, videoRepo.Upsert(ctx, nil, repositories.UpsertFeedVideoProjectionInput{
		VideoID: video2,
		Title:   "Video Two",
		Version: 1,
		UpdatedAt: func() *time.Time {
			ts := now
			return &ts
		}(),
	}))

	provider := &stubRecommendationProvider{
		source: "stub",
		items: []services.RecommendationItem{
			{
				VideoID:  video1.String(),
				Reason:   "reason.a",
				Score:    0.9,
				Metadata: map[string]string{"reason_label": "Label A"},
			},
			{
				VideoID:  video2.String(),
				Reason:   "reason.b",
				Score:    0.7,
				Metadata: map[string]string{"reason_label": "Label B"},
			},
		},
	}

	service := newFeedService(provider)

	resp, err := service.GetFeed(ctx, services.GetFeedInput{UserID: "user-1", Limit: 2})
	require.NoError(t, err)
	require.False(t, resp.Partial)
	require.Len(t, resp.Items, 2)
	require.Equal(t, "Video One", resp.Items[0].Title)
	require.Equal(t, "reason.a", resp.Items[0].ReasonCode)
	require.Equal(t, "Label A", resp.Items[0].ReasonLabel)

	logEntry := fetchLatestRecommendationLog(ctx, t)
	require.Equal(t, int32(2), logEntry.requestLimit)
	require.Equal(t, "stub", logEntry.source)
	require.False(t, logEntry.errorKind.Valid)
	require.Len(t, logEntry.recommendedItems, 2)
	require.Empty(t, logEntry.missingVideoIDs)
}

func TestFeedService_GetFeed_PartialLogsMissing(t *testing.T) {
	resetDatabase(t)
	ctx := context.Background()

	videoRepo := repositories.NewFeedVideoProjectionRepository(testPool, stdLogger)
	now := time.Now().UTC()
	video1 := uuid.New()
	video2 := uuid.New()

	require.NoError(t, videoRepo.Upsert(ctx, nil, repositories.UpsertFeedVideoProjectionInput{
		VideoID: video1,
		Title:   "Video One",
		Version: 1,
		UpdatedAt: func() *time.Time {
			ts := now
			return &ts
		}(),
	}))

	provider := &stubRecommendationProvider{
		source: "stub",
		items: []services.RecommendationItem{
			{VideoID: video1.String(), Reason: "reason.a", Score: 0.6},
			{VideoID: video2.String(), Reason: "reason.b", Score: 0.5},
		},
	}

	service := newFeedService(provider)

	resp, err := service.GetFeed(ctx, services.GetFeedInput{UserID: "user-2", Limit: 2})
	require.NoError(t, err)
	require.True(t, resp.Partial)
	require.Len(t, resp.Items, 1)
	require.Equal(t, video1.String(), resp.Items[0].VideoID)
	require.Len(t, resp.MissingProjections, 1)
	require.Equal(t, video2.String(), resp.MissingProjections[0].VideoID)

	logEntry := fetchLatestRecommendationLog(ctx, t)
	require.ElementsMatch(t, []string{video2.String()}, logEntry.missingVideoIDs)
	require.Equal(t, "stub", logEntry.source)
	require.False(t, logEntry.errorKind.Valid)
}

func TestFeedService_GetFeed_EmptyRecommendationLogged(t *testing.T) {
	resetDatabase(t)
	ctx := context.Background()

	provider := &stubRecommendationProvider{source: "stub"}
	service := newFeedService(provider)

	resp, err := service.GetFeed(ctx, services.GetFeedInput{UserID: "user-4", Limit: 5})
	require.NoError(t, err)
	require.False(t, resp.Partial)
	require.Empty(t, resp.Items)

	logEntry := fetchLatestRecommendationLog(ctx, t)
	require.Equal(t, int32(5), logEntry.requestLimit)
	require.Empty(t, logEntry.recommendedItems)
	require.Empty(t, logEntry.missingVideoIDs)
}

func TestFeedService_GetFeed_InvalidVideoIDHandled(t *testing.T) {
	resetDatabase(t)
	ctx := context.Background()

	provider := &stubRecommendationProvider{
		source: "stub",
		items:  []services.RecommendationItem{{VideoID: "not-a-uuid", Reason: "broken", Score: 0.1}},
	}

	service := newFeedService(provider)

	resp, err := service.GetFeed(ctx, services.GetFeedInput{UserID: "user-5", Limit: 1})
	require.NoError(t, err)
	require.True(t, resp.Partial)
	require.Empty(t, resp.Items)
	require.NotEmpty(t, resp.MissingProjections)
	foundInvalid := false
	for _, m := range resp.MissingProjections {
		if m.VideoID == "not-a-uuid" && m.Reason == "invalid video id" {
			foundInvalid = true
		}
	}
	require.True(t, foundInvalid, "expected invalid video id entry")

	logEntry := fetchLatestRecommendationLog(ctx, t)
	require.Contains(t, logEntry.missingVideoIDs, "not-a-uuid")
}

func TestFeedService_GetFeed_DefaultLimitApplied(t *testing.T) {
	resetDatabase(t)
	ctx := context.Background()

	provider := &stubRecommendationProvider{source: "stub"}
	service := newFeedService(provider)

	_, err := service.GetFeed(ctx, services.GetFeedInput{UserID: "user-6", Limit: 0})
	require.NoError(t, err)
	require.Equal(t, 10, provider.lastInput.Limit)
}

func TestFeedService_GetFeed_RecommendationErrorLogged(t *testing.T) {
	resetDatabase(t)
	ctx := context.Background()

	provider := &stubRecommendationProvider{
		source: "stub",
		err:    services.ErrRecommendationUnavailable,
	}

	service := newFeedService(provider)

	_, err := service.GetFeed(ctx, services.GetFeedInput{UserID: "user-3", Limit: 1})
	require.ErrorIs(t, err, services.ErrRecommendationUnavailable)

	logEntry := fetchLatestRecommendationLog(ctx, t)
	require.Equal(t, "stub", logEntry.source)
	require.True(t, logEntry.errorKind.Valid)
	require.Equal(t, "recommendation_unavailable", logEntry.errorKind.String)
	require.Empty(t, logEntry.recommendedItems)
	require.Empty(t, logEntry.missingVideoIDs)
}

type recommendationLogRow struct {
	requestLimit     int32
	source           string
	latency          sql.NullInt32
	recommendedItems []po.RecommendedItemLog
	missingVideoIDs  []string
	errorKind        sql.NullString
}

func fetchLatestRecommendationLog(ctx context.Context, t *testing.T) recommendationLogRow {
	t.Helper()
	row := testPool.QueryRow(ctx, `
		SELECT request_limit,
		       recommendation_source,
		       recommendation_latency_ms,
		       recommended_items,
		       missing_video_ids,
		       error_kind
		FROM feed.recommendation_logs
		ORDER BY generated_at DESC
		LIMIT 1
	`)
	var (
		requestLimit int32
		source       string
		latency      sql.NullInt32
		recommended  []byte
		missing      []byte
		errorKind    sql.NullString
	)
	require.NoError(t, row.Scan(&requestLimit, &source, &latency, &recommended, &missing, &errorKind))

	var recommendedItems []po.RecommendedItemLog
	require.NoError(t, json.Unmarshal(recommended, &recommendedItems))

	var missingIDs []string
	require.NoError(t, json.Unmarshal(missing, &missingIDs))

	return recommendationLogRow{
		requestLimit:     requestLimit,
		source:           source,
		latency:          latency,
		recommendedItems: recommendedItems,
		missingVideoIDs:  missingIDs,
		errorKind:        errorKind,
	}
}
