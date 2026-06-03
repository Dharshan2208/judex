package executor

import (
	"log"
	"time"

	"github.com/Dharshan2208/code-compiler/internal/sandbox"
)

type CppExecutor struct{}

func (c CppExecutor) Execute(file string, workspace string) Result {
	sb := sandbox.Sandbox{}

	// log.Printf("cpp compile started: file=%s workspace=%s", file, workspace)
	start := time.Now()
	compileResult := sb.Run(
		"compiler-cpp",
		workspace,
		[]string{
			"g++",
			"main.cpp",
			"-o",
			"app",
		},
	)

	if compileResult.Error != nil {
		if compileResult.Stderr == "execution timeout" {
			log.Printf("cpp compile timed out: file=%s workspace=%s", file, workspace)

			return Result{
				Stderr: compileResult.Stderr,
				Status: "timeout",
			}
		}

		log.Printf("cpp compile failed: file=%s workspace=%s stderr=%q", file, workspace, compileResult.Stderr)

		return Result{
			Stdout: compileResult.Stdout,
			Stderr: compileResult.Stderr,
			Status: "compile_error",
		}
	}

	log.Printf("cpp compile completed: file=%s workspace=%s", file, workspace)
	log.Printf("cpp run started: file=%s workspace=%s", file, workspace)

	runResult := sb.Run(
		"compiler-cpp",
		workspace,
		[]string{
			"./app",
		},
	)

	elapsed := time.Since(start)
	if runResult.Error != nil {
		if runResult.Stderr == "execution timeout" {
			log.Printf("cpp run timed out: file=%s workspace=%s", file, workspace)

			return Result{
				Stderr: runResult.Stderr,
				Status: "timeout",
			}
		}

		log.Printf("cpp run failed: file=%s workspace=%s stderr=%q", file, workspace, runResult.Stderr)

		return Result{
			Stdout:        runResult.Stdout,
			Stderr:        runResult.Stderr,
			Status:        "runtime_error",
			ExecutionTime: elapsed.Milliseconds(),
		}
	}

	log.Printf("cpp run completed: file=%s workspace=%s", file, workspace)

	return Result{
		Stdout:        runResult.Stdout,
		Stderr:        runResult.Stderr,
		Status:        "success",
		ExecutionTime: elapsed.Milliseconds(),
	}
}
