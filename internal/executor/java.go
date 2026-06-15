package executor

import (
	"context"
	"time"

	"github.com/Dharshan2208/judex/internal/sandbox"
)

type JavaExecutor struct{}

func (j JavaExecutor) Execute(ctx context.Context, sb *sandbox.Sandbox) Result {
	start := time.Now()

	compileRes := sb.Execute(ctx,
		[]string{
			"javac",
			"/workspace/Main.java",
		},
	)

	if compileRes.Status != "success" {
		return Result{
			Stdout: compileRes.Stdout,
			Stderr: compileRes.Stderr,
			Status: "compile_error",
		}
	}

	runRes := sb.Execute(ctx,
		[]string{
			"java",
			"-cp",
			"/workspace",
			"Main",
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
