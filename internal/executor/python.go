package executor

import (
	"context"
	"time"

	"github.com/Dharshan2208/judex/internal/metrics"
	"github.com/Dharshan2208/judex/internal/sandbox"
)

type PythonExecutor struct{}

func (p PythonExecutor) Execute(ctx context.Context, sb *sandbox.Sandbox) Result {
	start := time.Now()
	defer func() {
		metrics.ExecutionDuration.WithLabelValues("python").Observe(time.Since(start).Seconds())
	}()
	runStart := time.Now()
	res := sb.Execute(ctx, []string{"python3", "/workspace/main.py"})
	metrics.RunDuration.WithLabelValues("python").
		Observe(time.Since(runStart).Seconds())

	elapsed := time.Since(start)

	return Result{
		Stdout:        res.Stdout,
		Stderr:        res.Stderr,
		Status:        res.Status,
		ExecutionTime: elapsed.Milliseconds(),
	}
}
