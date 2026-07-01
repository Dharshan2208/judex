package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"time"

	"github.com/Dharshan2208/judex/internal/logutil"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
)

type Result struct {
	Stdout string
	Stderr string
	Status string
	Error  error
}

type Sandbox struct {
	Container *WarmContainer
	Manager   *PoolManager
}

func (s *Sandbox) Execute(ctx context.Context, command []string) Result {
	execConfig := container.ExecOptions{
		User: "1000",
		Env: []string{
			"HOME=/tmp",
			"GOCACHE=/var/cache/go-cache",
		},
		Cmd:          command,
		AttachStdout: true,
		AttachStderr: true,
		WorkingDir:   "/workspace",
	}

	t0 := time.Now()
	execResp, err := s.Manager.cli.ContainerExecCreate(ctx, s.Container.ID, execConfig)
	if err != nil {
		logutil.Error(ctx, "failed to create exec config",
			"container_id", s.Container.ID,
			"command", command,
			"error", err,
		)
		return Result{Error: err}
	}
	logutil.Debug(ctx, "exec create done",
		"container_id", s.Container.ID,
		"duration", time.Since(t0),
	)

	logutil.Debug(ctx, "executing in container",
		"container_id", s.Container.ID,
		"workdir", "/workspace",
		"command", command,
	)

	t1 := time.Now()
	attachResp, err := s.Manager.cli.ContainerExecAttach(ctx, execResp.ID, container.ExecStartOptions{})
	if err != nil {
		logutil.Error(ctx, "failed to attach to exec",
			"container_id", s.Container.ID,
			"exec_id", execResp.ID,
			"error", err,
		)
		return Result{Error: err}
	}
	logutil.Debug(ctx, "exec attach done",
		"container_id", s.Container.ID,
		"duration", time.Since(t1),
	)
	defer attachResp.Close()

	var stdout, stderr bytes.Buffer
	t2 := time.Now()
	if _, err := stdcopy.StdCopy(&stdout, &stderr, attachResp.Reader); err != nil {
		logutil.Error(ctx, "failed to copy exec output",
			"container_id", s.Container.ID,
			"exec_id", execResp.ID,
			"error", err,
		)
		return Result{Error: err}
	}
	logutil.Debug(ctx, "command execution done",
		"container_id", s.Container.ID,
		"duration", time.Since(t2),
	)

	t3 := time.Now()
	inspectResp, err := s.Manager.cli.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		logutil.Error(ctx, "failed to inspect exec",
			"container_id", s.Container.ID,
			"exec_id", execResp.ID,
			"error", err,
		)
		return Result{Error: err}
	}
	logutil.Debug(ctx, "exec inspect done",
		"container_id", s.Container.ID,
		"duration", time.Since(t3),
	)

	status := "success"
	if inspectResp.ExitCode != 0 {
		status = "failed"
		logutil.Warn(ctx, "command exited with non-zero",
			"container_id", s.Container.ID,
			"exit_code", inspectResp.ExitCode,
		)
	}

	return Result{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
		Status: status,
	}
}

// UploadCode streams source code into the container as a tar archive.
func (s *Sandbox) UploadCode(ctx context.Context, filename string, content string) error {
	logutil.Debug(ctx, "uploading code to container",
		"container_id", s.Container.ID,
		"filename", filename,
		"size", len(content),
	)

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	if err := tw.WriteHeader(&tar.Header{
		Name: filename,
		Mode: 0o666,
		Size: int64(len(content)),
	}); err != nil {
		logutil.Error(ctx, "tar header write failed",
			"container_id", s.Container.ID,
			"filename", filename,
			"error", err,
		)
		return err
	}

	if _, err := tw.Write([]byte(content)); err != nil {
		logutil.Error(ctx, "tar content write failed",
			"container_id", s.Container.ID,
			"filename", filename,
			"error", err,
		)
		return err
	}

	if err := tw.Close(); err != nil {
		logutil.Error(ctx, "tar close failed",
			"container_id", s.Container.ID,
			"filename", filename,
			"error", err,
		)
		return err
	}

	err := s.Manager.cli.CopyToContainer(
		ctx,
		s.Container.ID,
		"/workspace",
		bytes.NewReader(buf.Bytes()),
		container.CopyToContainerOptions{},
	)
	if err != nil {
		logutil.Error(ctx, "copy to container failed",
			"container_id", s.Container.ID,
			"filename", filename,
			"error", err,
		)
		return err
	}

	logutil.Debug(ctx, "code uploaded successfully",
		"container_id", s.Container.ID,
		"filename", filename,
	)
	return nil
}
