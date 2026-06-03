package executor

import (
	"time"

	"github.com/Dharshan2208/code-compiler/internal/sandbox"
)

type PythonExecutor struct{}

func (p PythonExecutor) Execute(file string, workspace string) Result {
	sb := sandbox.Sandbox{}

	start := time.Now()

	res := sb.Run(
		"compiler-python",
		workspace,
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
