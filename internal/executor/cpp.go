package executor

import (
	"github.com/Dharshan2208/code-compiler/internal/sandbox"
)

type CppExecutor struct{}

func (c CppExecutor) Execute(file string, workspace string) Result {
	sb := sandbox.Sandbox{}

	res := sb.Run(
		"compiler-cpp",
		workspace,
		[]string{
			"bash",
			"-c",
			"g++ main.cpp -o app && ./app",
		},
	)

	if res.Error != nil {
		if res.Stderr == "execution timeout" {
			return Result{
				Stderr: res.Stderr,
				Status: "timeout",
			}
		}

		return Result{
			Stdout: res.Stdout,
			Stderr: res.Stderr,
			Status: "compile_or_runtime_error",
		}
	}

	return Result{
		Stdout: res.Stdout,
		Stderr: res.Stderr,
		Status: "success",
	}
}
