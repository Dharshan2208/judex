package cleanup

import (
	"log"
	"time"

	"github.com/Dharshan2208/code-compiler/internal/queue"
)

func Start(s *queue.Store, ttl time.Duration) {
	go func() {
		for {
			time.Sleep(time.Minute)
			removed := s.Cleanup(ttl)

			if removed > 0 {
				log.Printf("[CLEANUP] removed %d jobs", removed)
			}
		}
	}()
}
