package executor

type Result struct {
	Stdout string
	Stderr string
	Status string
}

type Executor interface {
	Execute(file string, workspace string) Result
}
