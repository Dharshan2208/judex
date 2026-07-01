package cleanup

import (
	"context"
	"time"

	"github.com/Dharshan2208/judex/internal/logutil"
	"github.com/Dharshan2208/judex/internal/store"
)

func Start(s *store.RedisStore, ttl time.Duration) {
	ctx := context.Background()
	logutil.Info(ctx, "cleanup started",
		"ttl", ttl,
		"interval", time.Minute,
	)

	go func() {
		for {
			time.Sleep(time.Minute)
			removed := s.Cleanup(ctx, ttl)

			if removed > 0 {
				logutil.Info(ctx, "cleanup completed",
					"removed_jobs", removed,
				)
			} else {
				logutil.Debug(ctx, "cleanup ran, no jobs removed",
					"ttl", ttl,
				)
			}
		}
	}()
}
