package executor

import (
	"context"
	"time"

	"github.com/Dharshan2208/judex/internal/sandbox"
)

type GoExecutor struct{}

func (g GoExecutor) Execute(ctx context.Context, sb *sandbox.Sandbox) Result {
	start := time.Now()

	compileRes := sb.Execute(ctx,
		[]string{
			"sh",
			"-c",
			"GOCACHE=/tmp/go-cache go build -o app main.go",
		},
	)

	if compileRes.Error != nil {
		if compileRes.Stderr == "execution timeout" {
			return Result{
				Stderr: compileRes.Stderr,
				Status: "timeout",
			}
		}

		return Result{
			Stdout: compileRes.Stdout,
			Stderr: compileRes.Stderr,
			Status: "compile_error",
		}
	}

	runRes := sb.Execute(ctx,
		[]string{
			"./app",
		},
	)

	elapsed := time.Since(start)

	if runRes.Error != nil {
		if runRes.Stderr == "execution timeout" {
			return Result{
				Stderr: runRes.Stderr,
				Status: "timeout",
			}
		}

		return Result{
			Stdout:        runRes.Stdout,
			Stderr:        runRes.Stderr,
			Status:        "runtime_error",
			ExecutionTime: elapsed.Milliseconds(),
		}
	}

	return Result{
		Stdout:        runRes.Stdout,
		Stderr:        runRes.Stderr,
		Status:        "success",
		ExecutionTime: elapsed.Milliseconds(),
	}
}
