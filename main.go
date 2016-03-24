//go:generate stringer -type=Token
package main

import (
	"io/ioutil"
	"os"

	"github.com/danwakefield/gosh/variables"
)

func main() {
	fileContents, _ := ioutil.ReadFile(os.Args[1]) // Ignore error
	p := NewParser(string(fileContents))
	n := p.Parse()
	n.Eval(variables.NewScope())
}
