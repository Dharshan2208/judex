package executor

import (
	"context"
	"time"

	"github.com/Dharshan2208/judex/internal/sandbox"
)

type PythonExecutor struct{}

func (p PythonExecutor) Execute(ctx context.Context, sb *sandbox.Sandbox) Result {
	start := time.Now()

	res := sb.Execute(ctx, []string{"python3", "/workspace/main.py"})

	elapsed := time.Since(start)

	return Result{
		Stdout:        res.Stdout,
		Stderr:        res.Stderr,
		Status:        res.Status,
		ExecutionTime: elapsed.Milliseconds(),
	}
}
