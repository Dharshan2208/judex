package executor

import (
	"context"
	"time"

	"github.com/Dharshan2208/judex/internal/sandbox"
)

type CExecutor struct{}

func (c CExecutor) Execute(ctx context.Context, sb *sandbox.Sandbox) Result {
	start := time.Now()

	compileResult := sb.Execute(ctx,
		[]string{
			"gcc",
			"/workspace/main.c",
			"-o",
			"/workspace/app",
		},
	)

	if compileResult.Status != "success" {
		if compileResult.Stderr == "execution timeout" {
			return Result{
				Stderr: compileResult.Stderr,
				Status: "timeout",
			}
		}

		return Result{
			Stdout: compileResult.Stdout,
			Stderr: compileResult.Stderr,
			Status: "compile_error",
		}
	}

	runResult := sb.Execute(ctx,
		[]string{
			"/workspace/app",
		},
	)

	elapsed := time.Since(start)

	if runResult.Error != nil {
		if runResult.Stderr == "execution timeout" {
			return Result{
				Stderr: runResult.Stderr,
				Status: "timeout",
			}
		}

		return Result{
			Stdout:        runResult.Stdout,
			Stderr:        runResult.Stderr,
			Status:        "runtime_error",
			ExecutionTime: elapsed.Milliseconds(),
		}
	}

	return Result{
		Stdout:        runResult.Stdout,
		Stderr:        runResult.Stderr,
		Status:        "success",
		ExecutionTime: elapsed.Milliseconds(),
	}
}
