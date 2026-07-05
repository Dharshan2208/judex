package executor

import (
	"context"
	"time"

	"github.com/Dharshan2208/judex/internal/metrics"
	"github.com/Dharshan2208/judex/internal/sandbox"
)

type JavaExecutor struct{}

func (j JavaExecutor) Execute(ctx context.Context, sb *sandbox.Sandbox) Result {
	start := time.Now()
	defer metrics.ExecutionDuration.WithLabelValues("java").
		Observe(time.Since(start).Seconds())

	compileStart := time.Now()
	compileRes := sb.Execute(ctx,
		[]string{
			"javac",
			"/workspace/Main.java",
		},
	)
	metrics.CompileDuration.WithLabelValues("java").
		Observe(time.Since(compileStart).Seconds())

	if compileRes.Status != "success" {
		return Result{
			Stdout: compileRes.Stdout,
			Stderr: compileRes.Stderr,
			Status: "compile_error",
		}
	}

	runStart := time.Now()
	runRes := sb.Execute(ctx,
		[]string{
			"java",
			"-cp",
			"/workspace",
			"Main",
		},
	)
	metrics.RunDuration.WithLabelValues("java").
		Observe(time.Since(runStart).Seconds())

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
