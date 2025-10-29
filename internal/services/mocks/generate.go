package mocks

//go:generate go run github.com/golang/mock/mockgen -destination=mock_video_projection_repository.go -package=mocks github.com/bionicotaku/lingo-services-feed/internal/services VideoProjectionRepository
//go:generate go run github.com/golang/mock/mockgen -destination=mock_profile_users_repository.go -package=mocks github.com/bionicotaku/lingo-services-feed/internal/services ProfileUsersRepository
//go:generate go run github.com/golang/mock/mockgen -destination=mock_watch_logs_repository.go -package=mocks github.com/bionicotaku/lingo-services-feed/internal/services WatchLogsRepository
//go:generate go run github.com/golang/mock/mockgen -destination=mock_watch_stats_repository.go -package=mocks github.com/bionicotaku/lingo-services-feed/internal/services WatchStatsRepository
//go:generate go run github.com/golang/mock/mockgen -destination=mock_outbox_enqueuer.go -package=mocks github.com/bionicotaku/lingo-services-feed/internal/services OutboxEnqueuer
//go:generate go run github.com/golang/mock/mockgen -destination=mock_engagements_repository.go -package=mocks github.com/bionicotaku/lingo-services-feed/internal/services EngagementsRepository
//go:generate go run github.com/golang/mock/mockgen -destination=mock_engagement_stats_repository.go -package=mocks github.com/bionicotaku/lingo-services-feed/internal/services EngagementStatsRepository
//go:generate go run github.com/golang/mock/mockgen -destination=mock_video_stats_repository.go -package=mocks github.com/bionicotaku/lingo-services-feed/internal/services VideoStatsRepository
