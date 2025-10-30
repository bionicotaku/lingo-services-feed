// Package vo 定义向上层返回的 Feed 视图对象。
package vo

import "time"

// FeedItem 表示补水后的推荐卡片。
type FeedItem struct {
	VideoID           string
	Title             string
	Description       string
	DurationMicros    int64
	ThumbnailURL      string
	HLSMasterPlaylist string
	ReasonCode        string
	ReasonLabel       string
	Score             float64
	VisibilityStatus  string
	PublishedAt       *time.Time
	Attributes        map[string]string
}

// MissingProjection 描述补水失败的条目。
type MissingProjection struct {
	VideoID string
	Reason  string
}

// FeedResponse 汇总 Feed 返回的数据。
type FeedResponse struct {
	Items              []FeedItem
	NextCursor         string
	Partial            bool
	GeneratedAt        time.Time
	MissingProjections []MissingProjection
}
