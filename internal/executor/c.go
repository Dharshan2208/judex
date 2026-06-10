package executor

import (
	"log"
	"time"

	"github.com/Dharshan2208/code-compiler/internal/sandbox"
)

type CExecutor struct{}

func (c CExecutor) Execute(file string, workspace string) Result {
	sb := sandbox.Sandbox{}

	start := time.Now()

	compileResult := sb.Run(
		"compiler-c",
		workspace,
		[]string{
			"gcc",
			"main.c",
			"-o",
			"app",
		},
	)

	if compileResult.Error != nil {
		if compileResult.Stderr == "execution timeout" {
			log.Printf("c compile timed out: file=%s workspace=%s", file, workspace)

			return Result{
				Stderr: compileResult.Stderr,
				Status: "timeout",
			}
		}

		log.Printf("c compile failed: file=%s workspace=%s stderr=%q", file, workspace, compileResult.Stderr)

		return Result{
			Stdout: compileResult.Stdout,
			Stderr: compileResult.Stderr,
			Status: "compile_error",
		}
	}

	log.Printf("c compile completed: file=%s workspace=%s", file, workspace)
	log.Printf("c run started: file=%s workspace=%s", file, workspace)

	runResult := sb.Run(
		"compiler-c",
		workspace,
		[]string{
			"./app",
		},
	)

	elapsed := time.Since(start)

	if runResult.Error != nil {
		if runResult.Stderr == "execution timeout" {
			log.Printf("c run timed out: file=%s workspace=%s", file, workspace)

			return Result{
				Stderr: runResult.Stderr,
				Status: "timeout",
			}
		}

		log.Printf("c run failed: file=%s workspace=%s stderr=%q", file, workspace, runResult.Stderr)

		return Result{
			Stdout:        runResult.Stdout,
			Stderr:        runResult.Stderr,
			Status:        "runtime_error",
			ExecutionTime: elapsed.Milliseconds(),
		}
	}

	log.Printf("c run completed: file=%s workspace=%s", file, workspace)

	return Result{
		Stdout:        runResult.Stdout,
		Stderr:        runResult.Stderr,
		Status:        "success",
		ExecutionTime: elapsed.Milliseconds(),
	}
}
