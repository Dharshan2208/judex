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
		logutil.Error("failed to create exec config for container: container_id=%s command=%v error=%v", s.Container.ID, command, err)
		return Result{Error: err}
	}
	logutil.Debug("ExecCreate duration: %v container_id=%s", time.Since(t0), s.Container.ID)

	logutil.Debug(
		"Executing in container: container_id=%s workdir=/workspace cmd=%v",
		s.Container.ID,
		command,
	)

	t1 := time.Now()
	attachResp, err := s.Manager.cli.ContainerExecAttach(ctx, execResp.ID, container.ExecStartOptions{})
	if err != nil {
		logutil.Error("failed to attach to container exec: container_id=%s exec_id=%s error=%v", s.Container.ID, execResp.ID, err)
		return Result{Error: err}
	}
	logutil.Debug("ExecAttach duration: %v container_id=%s", time.Since(t1), s.Container.ID)
	defer attachResp.Close()

	var stdout, stderr bytes.Buffer
	t2 := time.Now()
	// stdcopy helps split the multiplexed stream from Docker back into stdout and stderr
	if _, err := stdcopy.StdCopy(&stdout, &stderr, attachResp.Reader); err != nil {
		logutil.Error("failed to copy stdout/stderr from container: container_id=%s exec_id=%s error=%v", s.Container.ID, execResp.ID, err)
		return Result{Error: err}
	}
	logutil.Debug("Command execution duration: %v container_id=%s", time.Since(t2), s.Container.ID)

	t3 := time.Now()
	inspectResp, err := s.Manager.cli.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		logutil.Error("failed to inspect container exec: container_id=%s exec_id=%s error=%v", s.Container.ID, execResp.ID, err)
		return Result{Error: err}
	}
	logutil.Debug("ExecInspect duration: %v container_id=%s", time.Since(t3), s.Container.ID)

	status := "success"
	if inspectResp.ExitCode != 0 {
		status = "failed"
		logutil.Warn("command failed in container: container_id=%s exit_code=%d", s.Container.ID, inspectResp.ExitCode)
	}

	return Result{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
		Status: status,
	}
}

func (s *Sandbox) UploadCode(ctx context.Context, filename string, content string) error {
	logutil.Debug("uploading code to container: container_id=%s filename=%s size=%d", s.Container.ID, filename, len(content))
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	if err := tw.WriteHeader(&tar.Header{
		Name: filename,
		Mode: 0o666,
		Size: int64(len(content)),
	}); err != nil {
		logutil.Error("failed to write tar header for code upload: container_id=%s filename=%s error=%v", s.Container.ID, filename, err)
		return err
	}

	if _, err := tw.Write([]byte(content)); err != nil {
		logutil.Error("failed to write code content to tar: container_id=%s filename=%s error=%v", s.Container.ID, filename, err)
		return err
	}

	if err := tw.Close(); err != nil {
		logutil.Error("failed to close tar writer for code upload: container_id=%s filename=%s error=%v", s.Container.ID, filename, err)
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
		logutil.Error("failed to copy code to container: container_id=%s filename=%s error=%v", s.Container.ID, filename, err)
		return err
	}
	logutil.Debug("code uploaded successfully to container: container_id=%s filename=%s", s.Container.ID, filename)
	return nil
}
