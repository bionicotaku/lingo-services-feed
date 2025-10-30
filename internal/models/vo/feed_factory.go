package vo

import "github.com/bionicotaku/lingo-services-feed/internal/models/po"

// FeedItemFromProjection 根据投影记录构造 FeedItem。
func FeedItemFromProjection(record *po.FeedVideoProjection) FeedItem {
	if record == nil {
		return FeedItem{Attributes: map[string]string{}}
	}
	item := FeedItem{
		VideoID:           record.VideoID,
		Title:             record.Title,
		Description:       derefString(record.Description),
		DurationMicros:    derefInt64(record.DurationMicros),
		ThumbnailURL:      derefString(record.ThumbnailURL),
		HLSMasterPlaylist: derefString(record.HLSMasterPlaylist),
		VisibilityStatus:  derefString(record.VisibilityStatus),
		Attributes:        map[string]string{},
	}
	if record.PublishedAt != nil {
		item.PublishedAt = record.PublishedAt
	}
	return item
}

// ApplyRecommendation 将推荐元数据合并到 FeedItem 中。
func (item *FeedItem) ApplyRecommendation(reason string, metadata map[string]string, score float64) {
	if item == nil {
		return
	}
	item.ReasonCode = reason
	item.Score = score
	if len(metadata) == 0 {
		return
	}
	if label, ok := metadata["reason_label"]; ok && label != "" {
		item.ReasonLabel = label
	}
	if item.Attributes == nil {
		item.Attributes = make(map[string]string, len(metadata))
	}
	for k, v := range metadata {
		if k == "reason_label" {
			continue
		}
		item.Attributes[k] = v
	}
}

func derefString(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

func derefInt64(ptr *int64) int64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}
