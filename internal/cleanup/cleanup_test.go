package cleanup

import (
	"testing"
	"time"

	"github.com/Dharshan2208/judex/internal/store"
	"github.com/Dharshan2208/judex/tests"
)

func TestStartReturnsImmediately(t *testing.T) {
	_, client := tests.NewMiniRedisClient(t)
	s := store.NewRedisStore(client)

	done := make(chan struct{})
	go func() {
		Start(s, time.Minute)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("cleanup.Start should return immediately")
	}
}
