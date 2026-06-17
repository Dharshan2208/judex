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
	logutil.Info("initializing container pool manager with capacity=%d", capacity)

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	pm := &PoolManager{
		cli:      cli,
		pools:    make(map[string]chan *WarmContainer),
		capacity: capacity,
	}

	// Initialize warm pools for each language
	for lang, image := range languages {
		logutil.Info("initializing warm container pool for language=%s image=%s count=%d", lang, image, capacity)
		pm.pools[lang] = make(chan *WarmContainer, capacity)
		for range capacity {
			c, err := pm.createWarmContainer(context.Background(), lang, image)
			if err != nil {
				return nil, fmt.Errorf("failed to create warm container for %s: %w", lang, err)
			}
			logutil.Debug("created warm container: container_id=%s language=%s image=%s", c.ID, lang, image)
			pm.pools[lang] <- c
		}
	}
	logutil.Info("container pool manager initialized successfully")

	return pm, nil
}

func (pm *PoolManager) createWarmContainer(ctx context.Context, lang, image string) (*WarmContainer, error) {
	logutil.Debug("creating warm container: language=%s image=%s", lang, image)

	config := &container.Config{
		Image: image,
		// keeping the containers alvie
		Cmd: []string{"tail", "-f", "/dev/null"},
		// for non root user so 1000
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
		logutil.Error("failed to create docker container: language=%s image=%s error=%v", lang, image, err)
		return nil, err
	}

	if err := pm.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		logutil.Error("failed to start docker container: container_id=%s language=%s image=%s error=%v", resp.ID, lang, image, err)
		return nil, err
	}

	logutil.Debug("docker container started: container_id=%s language=%s image=%s", resp.ID, lang, image)

	return &WarmContainer{ID: resp.ID, Language: lang, Image: image}, nil
}

func (pm *PoolManager) Acquire(ctx context.Context, lang string) (*WarmContainer, error) {
	logutil.Debug("attempting to acquire container: language=%s", lang)
	pool, ok := pm.pools[lang]

	if !ok {
		logutil.Warn("unsupported language for container acquisition: language=%s", lang)
		return nil, fmt.Errorf("unsupported language %s", lang)
	}

	select {
	case container := <-pool:
		// todo
		logutil.Debug("container acquired from pool: container_id=%s language=%s", container.ID, lang)
		// check if container healthy via sdk
		return container, nil

	case <-ctx.Done():
		logutil.Warn("container acquisition cancelled or timed out: language=%s error=%v", lang, ctx.Err())
		return nil, ctx.Err()

	}
}

func (pm *PoolManager) Release(ctx context.Context, container *WarmContainer) {
	logutil.Debug("attempting to release container: container_id=%s language=%s", container.ID, container.Language)
	if err := pm.Sanitize(ctx, container); err != nil {
		// if fails then just kiiling it and replacing it
		logutil.Warn("sanitization failed for container: container_id=%s language=%s error=%v, replacing it", container.ID, container.Language, err)
		pm.replaceContainer(ctx, container)
		return
	}

	logutil.Debug("container sanitized: container_id=%s language=%s", container.ID, container.Language)
	pm.pools[container.Language] <- container
	logutil.Debug("container returned to pool: container_id=%s language=%s", container.ID, container.Language)
}

func (pm *PoolManager) Sanitize(ctx context.Context, c *WarmContainer) error {
	// killing all user processes and also wiping workspace
	// running this as root to kill evything user started

	logutil.Debug("sanitizing container: container_id=%s language=%s", c.ID, c.Language)
	execConfig := container.ExecOptions{
		User: "root",
		Cmd:  []string{"sh", "-c", "pkill -u 1000 || true; rm -rf /workspace/* /tmp/*"},
	}

	exec, err := pm.cli.ContainerExecCreate(ctx, c.ID, execConfig)
	if err != nil {
		logutil.Error("failed to create exec config for sanitization: container_id=%s language=%s error=%v", c.ID, c.Language, err)
		return err
	}

	err = pm.cli.ContainerExecStart(ctx, exec.ID, container.ExecStartOptions{})
	if err != nil {
		logutil.Error("failed to start exec for sanitization: container_id=%s language=%s error=%v", c.ID, c.Language, err)
	}
	return err
}

func (pm *PoolManager) replaceContainer(ctx context.Context, c *WarmContainer) {
	logutil.Warn("replacing container: old_container_id=%s language=%s", c.ID, c.Language)

	// Logging for Docker operations during replacement
	logutil.Debug("killing old container: container_id=%s", c.ID)
	pm.cli.ContainerKill(ctx, c.ID, "SIGKILL")

	logutil.Debug("removing old container: container_id=%s", c.ID)
	pm.cli.ContainerRemove(ctx, c.ID, container.RemoveOptions{Force: true})
	logutil.Info("old container removed: container_id=%s", c.ID)

	newC, err := pm.createWarmContainer(ctx, c.Language, c.Image)
	if err != nil {
		logutil.Error("CRITICAL: failed to replace container: language=%s error=%v", c.Language, err)
		return
	}

	pm.pools[c.Language] <- newC
	logutil.Info("new container added to pool: container_id=%s language=%s", newC.ID, newC.Language)
}

func ptrInt64(i int64) *int64 { return &i }
