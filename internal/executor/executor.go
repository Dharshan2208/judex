package executor

import (
	"context"

	"github.com/Dharshan2208/judex/internal/sandbox"
)

type Result struct {
	Stdout        string
	Stderr        string
	Status        string
	ExecutionTime int64
}

// it has context and sandobx bcox it has all the warm containers
type Executor interface {
	Execute(ctx context.Context, sb *sandbox.Sandbox) Result
}
