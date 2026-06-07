package executor

import (
	"time"

	"github.com/Dharshan2208/code-compiler/internal/sandbox"
)

type GoExecutor struct{}

func (g GoExecutor) Execute(file string, workspace string) Result {
	sb := sandbox.Sandbox{}

	start := time.Now()

	compileRes := sb.Run(
		"compiler-go",
		workspace,
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

	runRes := sb.Run(
		"compiler-go",
		workspace,
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
