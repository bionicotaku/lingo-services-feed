package repositories_test

import (
	context "context"
	"io"
	"testing"
	"time"

	"github.com/bionicotaku/lingo-services-feed/internal/models/po"
	"github.com/bionicotaku/lingo-services-feed/internal/repositories"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestFeedRecommendationLogRepositoryIntegration(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dsn, terminate := startPostgres(ctx, t)
	defer terminate()

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(func() { pool.Close() })

	applyMigrations(ctx, t, pool)

	repo := repositories.NewFeedRecommendationLogRepository(pool, log.NewStdLogger(io.Discard))

	latency := int32(45)
	generatedAt := time.Now().UTC()
	logEntry := po.FeedRecommendationLog{
		UserID:               stringPtr("user-1"),
		Scene:                "home",
		Requested:            10,
		Returned:             8,
		Partial:              true,
		RecommendationSource: "mock.random",
		LatencyMS:            &latency,
		MissingIDs:           []string{"vid-1", "vid-2"},
		GeneratedAt:          generatedAt,
	}

	require.NoError(t, repo.Insert(ctx, nil, logEntry))

	// 插入不同日志确保 JSON/时间默认值处理正确。
	logEntry.MissingIDs = nil
	logEntry.Partial = false
	require.NoError(t, repo.Insert(ctx, nil, logEntry))
}
