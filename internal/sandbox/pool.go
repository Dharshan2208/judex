package sandbox

import (
	"context"
	"fmt"
	"sync"

	"github.com/Dharshan2208/judex/internal/logutil"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type WarmContainer struct {
	ID       string
	Image    string
	Language string
}

type PoolManager struct {
	cli      *client.Client
	pools    map[string]chan *WarmContainer
	mu       sync.RWMutex
	capacity int
}

func NewPoolManager(capacity int, languages map[string]string) (*PoolManager, error) {
	ctx := context.Background()
	logutil.Info(ctx, "initializing container pool manager",
		"capacity", capacity,
	)

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	pm := &PoolManager{
		cli:      cli,
		pools:    make(map[string]chan *WarmContainer),
		capacity: capacity,
	}

	for lang, image := range languages {
		logutil.Info(ctx, "initializing warm pool",
			"language", lang,
			"image", image,
			"count", capacity,
		)
		pm.pools[lang] = make(chan *WarmContainer, capacity)
		for range capacity {
			c, err := pm.createWarmContainer(ctx, lang, image)
			if err != nil {
				return nil, fmt.Errorf("failed to create warm container for %s: %w", lang, err)
			}
			logutil.Debug(ctx, "warm container created",
				"container_id", c.ID,
				"language", lang,
				"image", image,
			)
			pm.pools[lang] <- c
		}
	}
	logutil.Info(ctx, "container pool manager initialized")

	return pm, nil
}

func (pm *PoolManager) createWarmContainer(ctx context.Context, lang, image string) (*WarmContainer, error) {
	logutil.Debug(ctx, "creating warm container",
		"language", lang,
		"image", image,
	)

	config := &container.Config{
		Image:      image,
		Cmd:        []string{"tail", "-f", "/dev/null"},
		User:       "1000",
		WorkingDir: "/workspace",
		Tty:        false,
	}

	hostConfig := &container.HostConfig{
		Resources: container.Resources{
			Memory:    256 * 1024 * 1024,
			NanoCPUs:  int64(1e9),
			PidsLimit: ptrInt64(64),
		},
		NetworkMode: "none",
		CapDrop:     []string{"ALL"},
		SecurityOpt: []string{"no-new-privileges"},
	}

	resp, err := pm.cli.ContainerCreate(ctx, config, hostConfig, nil, nil, "")
	if err != nil {
		logutil.Error(ctx, "container create failed",
			"language", lang,
			"image", image,
			"error", err,
		)
		return nil, err
	}

	if err := pm.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		logutil.Error(ctx, "container start failed",
			"container_id", resp.ID,
			"language", lang,
			"image", image,
			"error", err,
		)
		return nil, err
	}

	logutil.Debug(ctx, "container started",
		"container_id", resp.ID,
		"language", lang,
		"image", image,
	)

	return &WarmContainer{ID: resp.ID, Language: lang, Image: image}, nil
}

func (pm *PoolManager) Acquire(ctx context.Context, lang string) (*WarmContainer, error) {
	logutil.Debug(ctx, "attempting to acquire container",
		"language", lang,
	)

	pm.mu.RLock()
	pool, ok := pm.pools[lang]
	pm.mu.RUnlock()

	if !ok {
		logutil.Warn(ctx, "unsupported language for container acquisition",
			"language", lang,
		)
		return nil, fmt.Errorf("unsupported language %s", lang)
	}

	select {
	case container := <-pool:
		logutil.Debug(ctx, "container acquired from pool",
			"container_id", container.ID,
			"language", lang,
		)
		return container, nil

	case <-ctx.Done():
		logutil.Warn(ctx, "container acquisition cancelled or timed out",
			"language", lang,
			"error", ctx.Err(),
		)
		return nil, ctx.Err()
	}
}

func (pm *PoolManager) Release(ctx context.Context, container *WarmContainer) {
	logutil.Debug(ctx, "releasing container",
		"container_id", container.ID,
		"language", container.Language,
	)

	if err := pm.Sanitize(ctx, container); err != nil {
		logutil.Warn(ctx, "sanitization failed, replacing container",
			"container_id", container.ID,
			"language", container.Language,
			"error", err,
		)
		pm.replaceContainer(ctx, container)
		return
	}

	logutil.Debug(ctx, "container sanitized",
		"container_id", container.ID,
		"language", container.Language,
	)
	pm.pools[container.Language] <- container
	logutil.Debug(ctx, "container returned to pool",
		"container_id", container.ID,
		"language", container.Language,
	)
}

func (pm *PoolManager) Sanitize(ctx context.Context, c *WarmContainer) error {
	logutil.Debug(ctx, "sanitizing container",
		"container_id", c.ID,
		"language", c.Language,
	)

	execConfig := container.ExecOptions{
		User: "root",
		Cmd:  []string{"sh", "-c", "pkill -u 1000 || true; rm -rf /workspace/* /tmp/*"},
	}

	exec, err := pm.cli.ContainerExecCreate(ctx, c.ID, execConfig)
	if err != nil {
		logutil.Error(ctx, "failed to create sanitization exec",
			"container_id", c.ID,
			"language", c.Language,
			"error", err,
		)
		return err
	}

	err = pm.cli.ContainerExecStart(ctx, exec.ID, container.ExecStartOptions{})
	if err != nil {
		logutil.Error(ctx, "failed to start sanitization exec",
			"container_id", c.ID,
			"language", c.Language,
			"error", err,
		)
	}
	return err
}

func (pm *PoolManager) replaceContainer(ctx context.Context, c *WarmContainer) {
	logutil.Warn(ctx, "replacing container",
		"old_container_id", c.ID,
		"language", c.Language,
	)

	logutil.Debug(ctx, "killing old container",
		"container_id", c.ID,
	)
	pm.cli.ContainerKill(ctx, c.ID, "SIGKILL")

	logutil.Debug(ctx, "removing old container",
		"container_id", c.ID,
	)
	pm.cli.ContainerRemove(ctx, c.ID, container.RemoveOptions{Force: true})
	logutil.Info(ctx, "old container removed",
		"container_id", c.ID,
	)

	newC, err := pm.createWarmContainer(ctx, c.Language, c.Image)
	if err != nil {
		logutil.Error(ctx, "failed to replace container",
			"language", c.Language,
			"error", err,
		)
		return
	}

	pm.pools[c.Language] <- newC
	logutil.Info(ctx, "new container added to pool",
		"container_id", newC.ID,
		"language", newC.Language,
	)
}

func ptrInt64(i int64) *int64 { return &i }
