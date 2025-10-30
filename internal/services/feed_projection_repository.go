package services

import (
    "context"

    "github.com/bionicotaku/lingo-services-feed/internal/models/po"
    "github.com/bionicotaku/lingo-utils/txmanager"
    "github.com/google/uuid"
)

// FeedProjectionRepository 抽象 Feed 投影仓储访问能力。
type FeedProjectionRepository interface {
    ListByIDs(ctx context.Context, sess txmanager.Session, ids []uuid.UUID) ([]*po.FeedVideoProjection, error)
    ListRandomIDs(ctx context.Context, sess txmanager.Session, limit int) ([]uuid.UUID, error)
}
