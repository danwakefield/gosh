package main

import (
	"os"
	"os/exec"
	"strings"

	"gopkg.in/logex.v1"

	"github.com/danwakefield/gosh/variables"
)

type ExitStatus int

const (
	ExitSuccess        ExitStatus = 0
	ExitFailure        ExitStatus = 1
	ExitUnknownCommand ExitStatus = 127
)

type Arg struct {
	Raw  string
	Subs []Substitution
}

func (a Arg) Expand(scp *variables.Scope) string {
	if strings.IndexRune(a.Raw, SentinalSubstitution) == -1 {
		return a.Raw
	}
	logex.Panic("Substitutions not implemented")
	return ""
}

type Node interface {
	Eval(*variables.Scope) ExitStatus
}

type NodeList []Node

func (n NodeList) Eval(scp *variables.Scope) ExitStatus {
	returnExit := ExitSuccess

	for _, x := range n {
		returnExit = x.Eval(scp)
	}

	return returnExit
}

type NodeLoop struct {
	IsWhile   bool
	Condition NodeCommand
	Body      NodeCommand
}

func (n NodeLoop) Eval(scp *variables.Scope) ExitStatus {
	var runBody bool
	returnExit := ExitSuccess

	for {
		condExit := n.Condition.Eval(scp)
		if n.IsWhile {
			runBody = condExit == ExitSuccess
		} else { // Until
			runBody = condExit != ExitSuccess
		}

		if runBody {
			returnExit = n.Body.Eval(scp)
		} else {
			break
		}
	}

	return returnExit
}

type NodeFor struct {
	LoopVar string
	Args    []Arg
	Body    NodeCommand
}

func (n NodeFor) Eval(scp *variables.Scope) ExitStatus {
	returnExit := ExitSuccess

	expandedArgs := make([]string, len(n.Args))
	for i, arg := range n.Args {
		expandedArgs[i] = arg.Expand(scp)
	}

	for _, arg := range expandedArgs {
		scp.Set(n.LoopVar, arg)
		returnExit = n.Body.Eval(scp)
	}

	return returnExit
}

// NodeIf is the structure that is used for 'if', 'elif' and 'else'
// as an 'if' or 'elif' Condition is required and Else is optionally nil to
// indicate the end of the if chain.
// as an 'else' Condition is required to be nil.
type NodeIf struct {
	Condition *NodeCommand
	Else      *NodeIf
	Body      NodeCommand
}

func (n NodeIf) Eval(scp *variables.Scope) ExitStatus {
	if n.Condition == nil {
		return n.Body.Eval(scp)
	}

	runBody := n.Condition.Eval(scp)
	if runBody == ExitSuccess {
		return n.Body.Eval(scp)
	}

	if n.Else != nil {
		return n.Else.Eval(scp)
	}

	return ExitSuccess
}

type NodeCommand struct {
	Assign []string
	Args   []Arg
	LineNo int
}

func (n NodeCommand) Eval(scp *variables.Scope) ExitStatus {
	// A line with only assignments applies them to the Root Scope
	// We check this first to avoid unnecessary scope Push/Pop's
	if len(n.Args) == 0 {
		for _, assign := range n.Assign {
			scp.SetString(assign)
		}
		return ExitSuccess
	}

	scp.Push()
	defer scp.Pop()

	for _, assign := range n.Assign {
		scp.SetString(assign, variables.LocalScope)
	}

	expandedArgs := make([]string, len(n.Args))
	for i, arg := range n.Args {
		expandedArgs[i] = arg.Expand(scp)
	}

	env := scp.Environ()
	cmd := exec.Command(expandedArgs[0], expandedArgs[1:]...)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	logex.Info("========ENV======")
	logex.Pretty(env)
	logex.Info("========CMD======")
	err := cmd.Run()
	logex.Info("=======EXIT======")
	if err == nil {
		logex.Info("> Success")
		return ExitSuccess
	}
	if err == exec.ErrNotFound {
		logex.Info("> Unknown Command")
		return ExitUnknownCommand
	}
	logex.Info("> Failure")
	return ExitFailure

}

type NodeEOF struct{}

func (n NodeEOF) Eval(*variables.Scope) ExitStatus { return ExitSuccess }

type NodeNegate struct {
	N Node
}

func (n NodeNegate) Eval(scp *variables.Scope) ExitStatus {
	es := n.N.Eval(scp)
	// Any Non-zero ExitStatus is a failure so we only check for success
	if es == ExitSuccess {
		return ExitFailure
	}
	return ExitSuccess
}

type NodePipe struct {
	Background bool
	Commands   NodeList
}

func (n NodePipe) Eval(scp *variables.Scope) ExitStatus {
	returnExit := ExitSuccess

	return returnExit
}
