package builtins

import (
	"github.com/danwakefield/gosh/T"
	"github.com/danwakefield/gosh/variables"
)

func TrueCmd(*variables.Scope, *T.IOContainer, []string) T.ExitStatus {
	return T.ExitSuccess
}
