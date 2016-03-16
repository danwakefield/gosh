//go:generate stringer -type=NodeType,Token -output=stringer.go
package main

import (
	"os"

	"gopkg.in/logex.v1"

	"github.com/danwakefield/gosh/variables"
)

var GlobalScope *variables.Scope

func init() {
	GlobalScope = variables.NewScope()
	env := os.Environ()
	for _, e := range env {
		GlobalScope.SetString(e)
	}
	logex.DebugLevel = 0
}

func main() {
}
