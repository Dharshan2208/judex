package executor

import (
	"bytes"
	"context"
	"os/exec"
	"path/filepath"
	"time"
)

type CppExecutor struct{}

func (c CppExecutor) Execute(file string) Result {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	binaryPath := filepath.Join(filepath.Dir(file), "app")

	// Compilation step
	compileCmd := exec.CommandContext(ctx, "g++", file, "-o", binaryPath)

	var compileErr bytes.Buffer

	compileCmd.Stderr = &compileErr

	err := compileCmd.Run()
	if err != nil {
		return Result{
			Stderr: compileErr.String(),
			Status: "compile_error",
		}
	}

	runCmd := exec.CommandContext(ctx, binaryPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	runCmd.Stdout = &stdout
	runCmd.Stderr = &stderr

	err = runCmd.Run()

	if ctx.Err() == context.DeadlineExceeded {
		return Result{
			Status: "timeout",
		}
	}

	if err != nil {
		return Result{
			Stdout: stdout.String(),
			Stderr: stderr.String(),
			Status: "runtime_error",
		}
	}

	return Result{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
		Status: "success",
	}
}
