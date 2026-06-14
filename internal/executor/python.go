package executor

import (
	"context"
	"time"

	"github.com/Dharshan2208/judex/internal/sandbox"
)

type PythonExecutor struct{}

func (p PythonExecutor) Execute(ctx context.Context, sb *sandbox.Sandbox) Result {
	start := time.Now()

	res := sb.Execute(ctx,
		[]string{
			"python3",
			"main.py",
		},
	)

	elapsed := time.Since(start)

	if res.Error != nil {
		if res.Stderr == "execution timeout" {
			return Result{
				Stderr: res.Stderr,
				Status: "timeout",
			}
		}

		return Result{
			Stdout:        res.Stdout,
			Stderr:        res.Stderr,
			Status:        "runtime_error",
			ExecutionTime: elapsed.Milliseconds(),
		}
	}

	return Result{
		Stdout:        res.Stdout,
		Stderr:        res.Stderr,
		Status:        "success",
		ExecutionTime: elapsed.Milliseconds(),
	}
}
