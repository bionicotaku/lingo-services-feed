// Package po 定义 Feed 服务的数据持久化结构体。
package po

import "time"

// FeedVideoProjection 表示 Feed 服务持久化的投影数据。
type FeedVideoProjection struct {
	VideoID           string
	Title             string
	Description       *string
	DurationMicros    *int64
	ThumbnailURL      *string
	HLSMasterPlaylist *string
	Status            *string
	VisibilityStatus  *string
	PublishedAt       *time.Time
	Version           int64
	UpdatedAt         time.Time
}

// FeedInboxEvent 记录 Inbox 消费状态。
type FeedInboxEvent struct {
	EventID       string
	SourceService string
	EventType     string
	AggregateType *string
	AggregateID   *string
	Payload       []byte
	ReceivedAt    time.Time
	ProcessedAt   *time.Time
	LastError     *string
}

// FeedRecommendationLog 描述推荐调用日志。
type FeedRecommendationLog struct {
	LogID                   string
	UserID                  *string
	RequestLimit            int32
	RecommendationSource    string
	RecommendationLatencyMS *int32
	RecommendedItems        []RecommendedItemLog
	MissingVideoIDs         []string
	ErrorKind               *string
	GeneratedAt             time.Time
}

// RecommendedItemLog 记录推荐模块原始返回的条目。
type RecommendedItemLog struct {
	VideoID string            `json:"video_id"`
	Reason  string            `json:"reason"`
	Score   float64           `json:"score"`
	Meta    map[string]string `json:"meta,omitempty"`
}
