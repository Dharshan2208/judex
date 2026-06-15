package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"log"

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
		User:         "1000",
		Cmd:          command,
		AttachStdout: true,
		AttachStderr: true,
		WorkingDir:   "/workspace",
	}

	execResp, err := s.Manager.cli.ContainerExecCreate(ctx, s.Container.ID, execConfig)
	if err != nil {
		return Result{Error: err}
	}

	log.Printf(
		"Executing in container=%s workdir=/workspace cmd=%v",
		s.Container.ID,
		command,
	)

	attachResp, err := s.Manager.cli.ContainerExecAttach(ctx, execResp.ID, container.ExecStartOptions{})
	if err != nil {
		return Result{Error: err}
	}
	defer attachResp.Close()

	var stdout, stderr bytes.Buffer
	// stdcopy helps split the multiplexed stream from Docker back into stdout and stderr
	if _, err := stdcopy.StdCopy(&stdout, &stderr, attachResp.Reader); err != nil {
		return Result{Error: err}
	}

	inspectResp, err := s.Manager.cli.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return Result{Error: err}
	}

	status := "success"
	if inspectResp.ExitCode != 0 {
		status = "failed"
	}

	return Result{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
		Status: status,
	}
}

func (s *Sandbox) UploadCode(ctx context.Context, filename string, content string) error {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	if err := tw.WriteHeader(&tar.Header{
		Name: filename,
		Mode: 0o666,
		Size: int64(len(content)),
	}); err != nil {
		return err
	}

	if _, err := tw.Write([]byte(content)); err != nil {
		return err
	}

	if err := tw.Close(); err != nil {
		return err
	}

	return s.Manager.cli.CopyToContainer(
		ctx,
		s.Container.ID,
		"/workspace",
		bytes.NewReader(buf.Bytes()),
		container.CopyToContainerOptions{},
	)
}
