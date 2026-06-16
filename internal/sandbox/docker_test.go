package sandbox

import (
	"context"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
)

func TestPoolAcquireTimeout(t *testing.T) {
	pm := &PoolManager{
		pools: map[string]chan *WarmContainer{
			"go": make(chan *WarmContainer),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := pm.Acquire(ctx, "go")
	if err == nil {
		t.Fatalf("expected acquire timeout")
	}
}

func TestPoolAcquireUnsupportedLanguage(t *testing.T) {
	pm := &PoolManager{pools: map[string]chan *WarmContainer{}}
	if _, err := pm.Acquire(context.Background(), "rust"); err == nil {
		t.Fatalf("expected unsupported language error")
	}
}

func TestPoolReleaseReturnsContainerToPool(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyMethod((*PoolManager)(nil), "Sanitize", func(*PoolManager, context.Context, *WarmContainer) error {
		return nil
	})

	pm := &PoolManager{
		pools: map[string]chan *WarmContainer{
			"go": make(chan *WarmContainer, 1),
		},
	}
	container := &WarmContainer{ID: "c1", Language: "go", Image: "compiler-go"}
	pm.Release(context.Background(), container)

	select {
	case got := <-pm.pools["go"]:
		if got.ID != "c1" {
			t.Fatalf("unexpected container returned: %+v", got)
		}
	default:
		t.Fatalf("expected container to be returned to pool")
	}
}
