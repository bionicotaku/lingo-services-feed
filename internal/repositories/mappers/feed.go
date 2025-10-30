// Package mappers 提供数据库行与领域模型之间的转换工具。
package mappers

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/bionicotaku/lingo-services-feed/internal/models/po"
	feeddb "github.com/bionicotaku/lingo-services-feed/internal/repositories/feeddb"

	"github.com/jackc/pgx/v5/pgtype"
)

// FeedVideoProjectionFromRow 将 sqlc 结构转换为领域对象。
func FeedVideoProjectionFromRow(row feeddb.FeedVideosProjection) *po.FeedVideoProjection {
	return &po.FeedVideoProjection{
		VideoID:           row.VideoID.String(),
		Title:             row.Title,
		Description:       textPtr(row.Description),
		DurationMicros:    toInt64Ptr(row.DurationMicros),
		ThumbnailURL:      textPtr(row.ThumbnailUrl),
		HLSMasterPlaylist: textPtr(row.HlsMasterPlaylist),
		Status:            textPtr(row.Status),
		VisibilityStatus:  textPtr(row.VisibilityStatus),
		PublishedAt:       timestampPtr(row.PublishedAt),
		Version:           row.Version,
		UpdatedAt:         mustTimestamp(row.UpdatedAt),
	}
}

// FeedInboxEventFromRow 转换 Inbox 事件。
func FeedInboxEventFromRow(row feeddb.FeedInboxEvent) *po.FeedInboxEvent {
	return &po.FeedInboxEvent{
		EventID:       row.EventID.String(),
		SourceService: row.SourceService,
		EventType:     row.EventType,
		AggregateType: textPtr(row.AggregateType),
		AggregateID:   textPtr(row.AggregateID),
		Payload:       row.Payload,
		ReceivedAt:    mustTimestamp(row.ReceivedAt),
		ProcessedAt:   timestampPtr(row.ProcessedAt),
		LastError:     textPtr(row.LastError),
	}
}

// FeedRecommendationLogFromRow 转换推荐日志。
func FeedRecommendationLogFromRow(row feeddb.FeedRecommendationLog) (*po.FeedRecommendationLog, error) {
	recommended := []po.RecommendedItemLog{}
	if len(row.RecommendedItems) > 0 {
		if err := json.Unmarshal(row.RecommendedItems, &recommended); err != nil {
			return nil, fmt.Errorf("unmarshal recommended_items: %w", err)
		}
	}
	missing := []string{}
	if len(row.MissingVideoIds) > 0 {
		if err := json.Unmarshal(row.MissingVideoIds, &missing); err != nil {
			return nil, fmt.Errorf("unmarshal missing_video_ids: %w", err)
		}
	}
	return &po.FeedRecommendationLog{
		LogID:                   row.LogID.String(),
		UserID:                  textPtr(row.UserID),
		RequestLimit:            row.RequestLimit,
		RecommendationSource:    row.RecommendationSource,
		RecommendationLatencyMS: int4Ptr(row.RecommendationLatencyMs),
		RecommendedItems:        recommended,
		MissingVideoIDs:         missing,
		ErrorKind:               textPtr(row.ErrorKind),
		GeneratedAt:             mustTimestamp(row.GeneratedAt),
	}, nil
}

// ToPgInt4 将 *int32 转换为 pgtype.Int4。
func ToPgInt4(value *int32) pgtype.Int4 {
	if value == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: *value, Valid: true}
}

// ToPgInt8 将 *int64 转换为 pgtype.Int8。
func ToPgInt8(value *int64) pgtype.Int8 {
	if value == nil {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: *value, Valid: true}
}

// ToPgText 将 *string 转换为 pgtype.Text。
func ToPgText(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *value, Valid: true}
}

// ToPgTimestamptzPtr 将 *time.Time 转换为 pgtype.Timestamptz。
func ToPgTimestamptzPtr(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: value.UTC(), Valid: true}
}

func int4Ptr(value pgtype.Int4) *int32 {
	if !value.Valid {
		return nil
	}
	return &value.Int32
}

func toInt64Ptr(value pgtype.Int8) *int64 {
	if !value.Valid {
		return nil
	}
	return &value.Int64
}

func textPtr(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func timestampPtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time.UTC()
	return &t
}

func mustTimestamp(value pgtype.Timestamptz) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time.UTC()
}
