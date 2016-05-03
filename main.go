//go:generate stringer -type=Token
package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/logex.v1"

	"github.com/danwakefield/gosh/T"
	"github.com/danwakefield/gosh/variables"
)

func init() {
	logex.DebugLevel = 1
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Please pass a file to run")
		os.Exit(1)
	}
	fileContents, err := ioutil.ReadFile(os.Args[1]) // Ignore error
	if err != nil {
		fmt.Println(err.Error())
	}
	p := NewParser(string(fileContents))
	scp := variables.NewScope()
	scp.SetPositionalArgs(os.Args[1:])
	ex := T.ExitSuccess

	stdIO := &T.IOContainer{In: os.Stdin, Out: os.Stdout, Err: os.Stderr}

	for {
		n := p.Parse()
		if n == nil {
			//Newline
			continue
		}
		if _, isEOF := n.(NodeEOF); isEOF {
			os.Exit(int(ex))
		}

		ex = n.Eval(scp, stdIO)
	}
}
