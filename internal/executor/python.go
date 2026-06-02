package executor

import (
	"bytes"
	"context"
	"os/exec"
	"time"
)

type PythonExecutor struct{}

func (p PythonExecutor) Execute(file string) Result {
	// Creating a 5 second timer
	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	// Running the command
	cmd := exec.CommandContext(ctx, "python3", file)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	if ctx.Err() == context.DeadlineExceeded {
		return Result{Status: "timeout"}
	}

	if err != nil {
		return Result{
			Stdout: stdout.String(),
			Stderr: stderr.String(),
			Status: "error",
		}
	}

	return Result{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
		Status: "success",
	}
}
