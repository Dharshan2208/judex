package executor

import (
	"context"
	"testing"

	"github.com/Dharshan2208/judex/internal/sandbox"
	"github.com/agiledragon/gomonkey/v2"
)

func newSandbox() *sandbox.Sandbox { return &sandbox.Sandbox{} }

func TestPythonExecutorPassthrough(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyMethod(reflectTypeSandbox(), "Execute", func(*sandbox.Sandbox, context.Context, []string) sandbox.Result {
		return sandbox.Result{Stdout: "hello\n", Status: "success"}
	})

	res := PythonExecutor{}.Execute(context.Background(), newSandbox())
	if res.Status != "success" || res.Stdout != "hello\n" {
		t.Fatalf("unexpected python result: %+v", res)
	}
}

func TestGoExecutorCompileError(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyMethodSeq(reflectTypeSandbox(), "Execute", []gomonkey.OutputCell{
		{Values: gomonkey.Params{sandbox.Result{Stderr: "syntax error", Status: "failed"}}},
	})

	res := GoExecutor{}.Execute(context.Background(), newSandbox())
	if res.Status != "compile_error" {
		t.Fatalf("expected compile_error, got %+v", res)
	}
}

func TestGoExecutorTimeoutOnCompile(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyMethodSeq(reflectTypeSandbox(), "Execute", []gomonkey.OutputCell{
		{Values: gomonkey.Params{sandbox.Result{Stderr: "execution timeout", Status: "failed"}}},
	})

	res := GoExecutor{}.Execute(context.Background(), newSandbox())
	if res.Status != "timeout" {
		t.Fatalf("expected timeout, got %+v", res)
	}
}

func TestJavaExecutorCompileError(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyMethodSeq(reflectTypeSandbox(), "Execute", []gomonkey.OutputCell{
		{Values: gomonkey.Params{sandbox.Result{Stderr: "javac failed", Status: "failed"}}},
	})

	res := JavaExecutor{}.Execute(context.Background(), newSandbox())
	if res.Status != "compile_error" {
		t.Fatalf("expected compile_error, got %+v", res)
	}
}

func TestCppExecutorRuntimeErrorOnExecStartFailure(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyMethodSeq(reflectTypeSandbox(), "Execute", []gomonkey.OutputCell{
		{Values: gomonkey.Params{sandbox.Result{Status: "success"}}},
		{Values: gomonkey.Params{sandbox.Result{Error: context.DeadlineExceeded, Status: "failed"}}},
	})

	res := CppExecutor{}.Execute(context.Background(), newSandbox())
	if res.Status != "runtime_error" {
		t.Fatalf("expected runtime_error, got %+v", res)
	}
}

func TestCExecutorSuccess(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyMethodSeq(reflectTypeSandbox(), "Execute", []gomonkey.OutputCell{
		{Values: gomonkey.Params{sandbox.Result{Status: "success"}}},
		{Values: gomonkey.Params{sandbox.Result{Stdout: "42\n", Status: "success"}}},
	})

	res := CExecutor{}.Execute(context.Background(), newSandbox())
	if res.Status != "success" || res.Stdout != "42\n" {
		t.Fatalf("unexpected c executor result: %+v", res)
	}
}

func reflectTypeSandbox() interface{} {
	return (*sandbox.Sandbox)(nil)
}
