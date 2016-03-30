//go:generate stringer -type=Token
package main

import (
	"io/ioutil"
	"os"

	"gopkg.in/logex.v1"

	"github.com/danwakefield/gosh/variables"
)

func init() {
	logex.DebugLevel = 0
}

func main() {
	fileContents, _ := ioutil.ReadFile(os.Args[1]) // Ignore error
	p := NewParser(string(fileContents))
	scp := variables.NewScope()
	ex := ExitSuccess

	for {
		n := p.Parse()
		if n == nil {
			//Newline
			continue
		}
		logex.Pretty(n)
		if _, isEOF := n.(NodeEOF); isEOF {
			os.Exit(int(ex))
		}

		ex = n.Eval(scp)
	}
}
