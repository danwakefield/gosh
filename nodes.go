package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/danwakefield/gosh/variables"
)

type ExitStatus int

const (
	ExitSuccess        = 0
	ExitFailure        = 1
	ExitUnknownCommand = 127
)

type Node interface {
	NodeType() NodeType
	Eval(*variables.Scope) ExitStatus
}

type NodeType int

const (
	NEOF NodeType = iota
	NCommand
	NPipe
	NRedirection
	NBackground
	NSubshell
	NAnd
	NOr
	NSemicolon
	NIf
	NWhile
	NUntil
	NFor
	NCase
	NFunction
	NRedirTo
	NRedirAppend
	NRedirClobber
	NNot
)

type CommandArg struct {
	Raw string
}

func (c CommandArg) Expand(scp *variables.Scope) string {
	return c.Raw
}

type NodeCommand struct {
	Assign []string
	Args   []CommandArg
	LineNo int
}

func (n NodeCommand) NodeType() NodeType { return NCommand }
func (n NodeCommand) Eval(scp *variables.Scope) ExitStatus {
	// A line with only assignments applies them to the Root Scope
	// We check this first to avoid unnecessary scope Push/Pop 's
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

	fmt.Println("=======ENV===============\n")
	fmt.Println(env)
	fmt.Println("=======RUN===============\n")
	es := cmd.Run()
	// TODO: Real Error Codes
	fmt.Println("=========================\n")
	if es != nil {
		return ExitFailure
	}
	return ExitSuccess
}

type NodeEOF struct{}

func (n NodeEOF) NodeType() NodeType               { return NEOF }
func (n NodeEOF) Eval(*variables.Scope) ExitStatus { return ExitSuccess }

type NodeNegate struct {
	N Node
}

func (n NodeNegate) NodeType() NodeType { return NNot }
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
	Commands   []NodeCommand
}

func (n NodePipe) NodeType() NodeType                   { return NPipe }
func (n NodePipe) Eval(scp *variables.Scope) ExitStatus { return ExitFailure }
