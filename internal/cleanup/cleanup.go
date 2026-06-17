package cleanup

import (
	"time"

	"github.com/Dharshan2208/judex/internal/logutil"
	"github.com/Dharshan2208/judex/internal/store"
)

func Start(s *store.RedisStore, ttl time.Duration) {
	logutil.Info("cleanup started: ttl=%s interval=%s", ttl, time.Minute)

	go func() {
		for {
			time.Sleep(time.Minute)
			removed := s.Cleanup(ttl)

			if removed > 0 {
				logutil.Info("cleanup completed: removed_jobs=%d", removed)
			} else {
				logutil.Debug("cleanup ran, no jobs removed. ttl=%s", ttl)
			}
		}
	}()
}
