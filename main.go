//go:generate stringer -type=NodeType,Token -output=stringer.go
package main

import (
	"os"

	"github.com/danwakefield/gosh/variables"
)

var GlobalScope *variables.Scope

func init() {
	GlobalScope = variables.NewScope()
	env := os.Environ()
	for _, e := range env {
		GlobalScope.SetString(e)
	}
}

func main() {
}
