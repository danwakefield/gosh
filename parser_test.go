package main

import (
	"fmt"
	"testing"

	"github.com/danwakefield/gosh/variables"
)

func TestParse(t *testing.T) {
	p := NewParser("A=1 B=2 echo 'foo'")

	n := p.Parse()

	if n.NodeType() != NCommand {
		t.Errorf("Parse should have returned a NodeCommand")
	}

	s := variables.NewScope()
	nc := n.Eval(s)

	fmt.Printf("Exitcode: %d\n", nc)
}
