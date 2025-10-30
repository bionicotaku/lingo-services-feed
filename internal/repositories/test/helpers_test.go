package repositories_test

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/bionicotaku/lingo-services-feed/internal/repositories"
	outboxcfg "github.com/bionicotaku/lingo-utils/outbox/config"
	"github.com/docker/go-connections/nat"
	"github.com/go-kratos/kratos/v2/log"
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
	if err := startPostgres(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start postgres container: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()

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
			return fmt.Sprintf("postgres://postgres:postgres@%s:%s/feed?sslmode=disable&search_path=feed", host, port.Port())
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

	dsn := fmt.Sprintf("postgres://postgres:postgres@%s:%s/feed?sslmode=disable&search_path=feed", host, port.Port())

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return err
	}
	testPool = pool

	return applyMigrations(ctx, pool)
}

func applyMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	migrationsDir := filepath.Join("..", "..", "..", "migrations")
	entries, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		return err
	}
	sort.Strings(entries)

	for _, path := range entries {
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if _, execErr := pool.Exec(ctx, string(content)); execErr != nil {
			return fmt.Errorf("apply migration %s: %w", filepath.Base(path), execErr)
		}
	}
	return nil
}

func resetDatabase(t *testing.T) {
	t.Helper()
	_, err := testPool.Exec(context.Background(), `
		TRUNCATE TABLE
			feed.inbox_events,
			feed.recommendation_logs,
			feed.videos_projection
		RESTART IDENTITY
	`)
	require.NoError(t, err)
}

func newVideoProjectionRepo() *repositories.FeedVideoProjectionRepository {
	return repositories.NewFeedVideoProjectionRepository(testPool, stdLogger)
}

func newRecommendationLogRepo() *repositories.FeedRecommendationLogRepository {
	return repositories.NewFeedRecommendationLogRepository(testPool, stdLogger)
}

func newInboxRepo(t *testing.T) *repositories.InboxRepository {
	t.Helper()
	return repositories.NewInboxRepository(testPool, stdLogger, outboxcfg.Config{Schema: "feed"})
}

func stringPtr(value string) *string {
	return &value
}

func timePtr(value time.Time) *time.Time {
	return &value
}
