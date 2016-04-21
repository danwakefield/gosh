package builtins

import (
	"github.com/danwakefield/gosh/T"
	"github.com/danwakefield/gosh/variables"
)

func LocalCmd(scp *variables.Scope, ioc *T.IOContainer, args []string) T.ExitStatus {
	// Local should also do assignments, split args on equal sign? already
	// expanded
	_ = "breakpoint"
	for _, a := range args {
		tmp := scp.Get(a)
		if tmp.Set {
			scp.Set(a, tmp.Val, variables.LocalScope)
		} else {
			scp.Set(a, "", variables.LocalScope)
		}
	}
	return T.ExitSuccess
}
