package executor

type Result struct {
	Stdout        string
	Stderr        string
	Status        string
	ExecutionTime int64
}

type Executor interface {
	Execute(file string, workspace string) Result
}
