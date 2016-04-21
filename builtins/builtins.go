// Package builtins
package builtins

import (
	"github.com/danwakefield/gosh/T"
	"github.com/danwakefield/gosh/variables"
)

type Builtin func(scp *variables.Scope, ioc *T.IOContainer, args []string) T.ExitStatus

var All = map[string]Builtin{
	"true":  TrueCmd,
	":":     TrueCmd,
	"false": FalseCmd,
	"cd":    CdCmd,
	"local": LocalCmd,
}
