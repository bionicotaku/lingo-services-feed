package po

import (
	"strings"
	"time"
)

// FeedRecommendationLogParams 描述构造推荐日志所需的参数。
type FeedRecommendationLogParams struct {
	UserID                  string
	RequestLimit            int
	RecommendationSource    string
	RecommendationLatencyMS int32
	RecommendedItems        []RecommendedItemLog
	MissingVideoIDs         []string
	ErrorKind               string
	GeneratedAt             time.Time
}

// NewFeedRecommendationLog 基于参数构造 FeedRecommendationLog 实例。
func NewFeedRecommendationLog(params FeedRecommendationLogParams) FeedRecommendationLog {
	items := cloneRecommendedItems(params.RecommendedItems)
	missing := cloneStrings(params.MissingVideoIDs)

	entry := FeedRecommendationLog{
		UserID:                  optionalString(params.UserID),
		RequestLimit:            int32(params.RequestLimit),
		RecommendationSource:    strings.TrimSpace(params.RecommendationSource),
		RecommendationLatencyMS: optionalInt32(params.RecommendationLatencyMS),
		RecommendedItems:        items,
		MissingVideoIDs:         missing,
		GeneratedAt:             params.GeneratedAt,
	}
	if entry.GeneratedAt.IsZero() {
		entry.GeneratedAt = time.Now().UTC()
	}
	if kind := strings.TrimSpace(params.ErrorKind); kind != "" {
		entry.ErrorKind = &kind
	}
	return entry
}

func cloneRecommendedItems(src []RecommendedItemLog) []RecommendedItemLog {
	if len(src) == 0 {
		return []RecommendedItemLog{}
	}
	dst := make([]RecommendedItemLog, len(src))
	for i, item := range src {
		dst[i] = RecommendedItemLog{
			VideoID: item.VideoID,
			Reason:  item.Reason,
			Score:   item.Score,
		}
		if len(item.Meta) > 0 {
			meta := make(map[string]string, len(item.Meta))
			for k, v := range item.Meta {
				meta[k] = v
			}
			dst[i].Meta = meta
		}
	}
	return dst
}

func cloneStrings(src []string) []string {
	if len(src) == 0 {
		return []string{}
	}
	dst := make([]string, len(src))
	copy(dst, src)
	return dst
}

func optionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	v := strings.TrimSpace(value)
	return &v
}

func optionalInt32(value int32) *int32 {
	if value <= 0 {
		return nil
	}
	v := value
	return &v
}
