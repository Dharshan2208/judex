package sandbox

import (
	"bytes"
	"context"
	"log"
	"os/exec"
	"time"
)

type Result struct {
	Stdout string
	Stderr string
	Error  error
}

type Sandbox struct{}

func (s *Sandbox) Run(image string, workspace string, command []string) Result {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Printf("Sandbox run started: image=%s workspace=%s command=%v", image, workspace, command)

	args := []string{
		"run",
		"--rm",

		"--memory=256m",
		"--cpus=1",
		"--pids-limit=64",

		"--network=none",

		"--read-only",
		"--tmpfs", "/tmp:size=64m",

		"--security-opt=no-new-privileges",
		"--cap-drop=ALL",

		"-v",
		workspace + ":/workspace",

		"-w",
		"/workspace",

		image,
	}

	args = append(args, command...)

	cmd := exec.CommandContext(ctx, "docker", args...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		log.Printf("sandbox run timed out: image=%s workspace=%s", image, workspace)
		return Result{
			Stderr: "execution timeout",
			Error:  ctx.Err(),
		}
	}

	if err != nil {
		log.Printf("sandbox run failed: image=%s workspace=%s error=%v stderr=%q", image, workspace, err, stderr.String())
	} else {
		log.Printf("sandbox run completed: image=%s workspace=%s stdout_bytes=%d stderr_bytes=%d", image, workspace, stdout.Len(), stderr.Len())
	}

	return Result{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
		Error:  err,
	}
}
