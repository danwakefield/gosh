//go:generate stringer -type=NodeType,Token -output=stringer.go
package main

import (
	"io/ioutil"
	"os"

	"github.com/danwakefield/gosh/variables"
)

var GlobalScope *variables.Scope

func init() {
	// GlobalScope = variables.NewScope()
	// env := os.Environ()
	// for _, e := range env {
	// 	GlobalScope.SetString(e)
	// }
	// logex.DebugLevel = 0
}

func main() {
	fileContents, _ := ioutil.ReadFile(os.Args[1]) // Ignore error
	p := NewParser(string(fileContents))
	n := p.Parse()
	n.Eval(variables.NewScope())
}
