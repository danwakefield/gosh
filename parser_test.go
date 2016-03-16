package main

import (
	"fmt"
	"testing"

	"github.com/danwakefield/gosh/variables"
)

func TestParse(t *testing.T) {
	p := NewParser(`if true then
	echo 'foo'
	fi`)

	n := p.Parse()

	s := variables.NewScope()
	nc := n.Eval(s)

	fmt.Printf("Exitcode: %d\n", nc)
}
