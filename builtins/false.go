package builtins

import (
	"github.com/danwakefield/gosh/T"
	"github.com/danwakefield/gosh/variables"
)

func FalseCmd(*variables.Scope, *T.IOContainer, []string) T.ExitStatus {
	return T.ExitFailure
}
