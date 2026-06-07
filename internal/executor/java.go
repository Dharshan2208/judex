package executor

import (
	"time"

	"github.com/Dharshan2208/code-compiler/internal/sandbox"
)

type JavaExecutor struct{}

func (j JavaExecutor) Execute(file string, workspace string) Result {
	sb := sandbox.Sandbox{}

	start := time.Now()

	compileRes := sb.Run(
		"compiler-java",
		workspace,
		[]string{
			"javac",
			"Main.java",
		},
	)

	if compileRes.Error != nil {
		return Result{
			Stdout: compileRes.Stdout,
			Stderr: compileRes.Stderr,
			Status: "compile_error",
		}
	}

	runRes := sb.Run(
		"compiler-java",
		workspace,
		[]string{
			"java",
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
