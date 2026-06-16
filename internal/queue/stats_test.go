package queue

import (
	"sync"
	"testing"
)

func TestStatsSnapshotConcurrent(t *testing.T) {
	stats := &Stats{}
	const workers = 20
	const loops = 200

	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < loops; j++ {
				stats.IncSubmitted()
				if j%2 == 0 {
					stats.IncCompleted()
				} else {
					stats.IncFailed()
				}
			}
		}()
	}
	wg.Wait()

	sub, comp, fail := stats.Snapshot()
	if sub != workers*loops {
		t.Fatalf("unexpected submitted count %d", sub)
	}
	if comp+fail != sub {
		t.Fatalf("completed+failed mismatch: %d + %d != %d", comp, fail, sub)
	}
}
