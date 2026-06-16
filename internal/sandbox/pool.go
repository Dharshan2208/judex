package sandbox

import (
	"context"
	"fmt"
	"log"
	"sync"

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
		pm.pools[lang] = make(chan *WarmContainer, capacity)
		for range capacity {
			c, err := pm.createWarmContainer(context.Background(), lang, image)
			if err != nil {
				return nil, fmt.Errorf("failed to create warm container for %s: %w", lang, err)
			}
			pm.pools[lang] <- c
		}
	}

	return pm, nil
}

func (pm *PoolManager) createWarmContainer(ctx context.Context, lang, image string) (*WarmContainer, error) {
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
		// we using tmpfs for /workspace and /tmp for speed and easy cleanup
		// Tmpfs: map[string]string{
		// 	"/workspace": "rw,size=64m",
		// 	"/tmp":       "rw,size=64m",
		// },
	}

	resp, err := pm.cli.ContainerCreate(ctx, config, hostConfig, nil, nil, "")
	if err != nil {
		return nil, err
	}

	if err := pm.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return nil, err
	}

	return &WarmContainer{ID: resp.ID, Language: lang, Image: image}, nil
}

func (pm *PoolManager) Acquire(ctx context.Context, lang string) (*WarmContainer, error) {
	pool, ok := pm.pools[lang]

	if !ok {
		return nil, fmt.Errorf("unsupported language %s", lang)
	}

	select {
	case container := <-pool:
		// todo
		// check if container healthy via sdk
		return container, nil

	case <-ctx.Done():
		return nil, ctx.Err()

	}
}

func (pm *PoolManager) Release(ctx context.Context, container *WarmContainer) {
	if err := pm.Sanitize(ctx, container); err != nil {
		// if fails then just kiiling it and replacing it
		log.Printf("Sanitisation falied for container %s, replacing it with :%v", container.ID, err)
		pm.replaceContainer(ctx, container)
		return
	}

	pm.pools[container.Language] <- container
}

func (pm *PoolManager) Sanitize(ctx context.Context, c *WarmContainer) error {
	// killing all user processes and also wiping workspace
	// running this as root to kill evything user started

	execConfig := container.ExecOptions{
		User: "root",
		Cmd:  []string{"sh", "-c", "pkill -u 1000 || true; rm -rf /workspace/* /tmp/*"},
	}

	exec, err := pm.cli.ContainerExecCreate(ctx, c.ID, execConfig)
	if err != nil {
		return err
	}

	return pm.cli.ContainerExecStart(ctx, exec.ID, container.ExecStartOptions{})
}

func (pm *PoolManager) replaceContainer(ctx context.Context, c *WarmContainer) {
	pm.cli.ContainerKill(ctx, c.ID, "SIGKILL")
	pm.cli.ContainerRemove(ctx, c.ID, container.RemoveOptions{Force: true})

	newC, err := pm.createWarmContainer(ctx, c.Language, c.Image)
	if err != nil {
		log.Printf("CRITICAL : Failed to relace the container : %v", err)
		return
	}

	pm.pools[c.Language] <- newC
}

func ptrInt64(i int64) *int64 { return &i }
