package main

import (
	"io"
	"os/exec"
	"strings"

	"gopkg.in/logex.v1"

	"github.com/danwakefield/fnmatch"
	"github.com/danwakefield/gosh/variables"
)

// ExitStatus TODO
type ExitStatus int

const (
	ExitSuccess        ExitStatus = 0
	ExitFailure        ExitStatus = 1
	ExitNotExecutable  ExitStatus = 126
	ExitUnknownCommand ExitStatus = 127
)

type Arg struct {
	Raw  string
	Subs []Substitution
}

func (a Arg) Expand(scp *variables.Scope) (returnString string) {
	logex.Debugf("Expand '%s'", a.Raw)
	defer func() {
		logex.Debugf("Returned '%s'", returnString)
	}()

	if !strings.ContainsRune(a.Raw, SentinalSubstitution) {
		return a.Raw
	}

	// Split the Raw string into a []string. Each element would have been
	// immediately followed by a substitution.
	fields := strings.FieldsFunc(a.Raw, func(r rune) bool {
		return r == SentinalSubstitution
	})

	x := make([]string, len(a.Subs))
	if len(fields) == 0 {
		// If fields contains nothing after being split the string consists
		// of only consecutive substitutions
		for _, s := range a.Subs {
			x = append(x, s.Sub(scp))
		}
	} else {
		for i, f := range fields {
			x = append(x, f)
			x = append(x, a.Subs[i].Sub(scp))
		}
	}

	// XXX: Do FNmatch pathname expansion here.
	// see `man 7 glob` for details. Key point is that is the expansion
	// has no files it should be returned as is.
	return strings.Join(x, "")
}

type IOContainer struct {
	In  io.Reader
	Out io.Writer
	Err io.Writer
}

type Node interface {
	Eval(*variables.Scope, *IOContainer) ExitStatus
}

// NodeEOF is end of file sentinal node.
type NodeEOF struct{}

// Eval is required to fufill the Node interface but the return value in this
// case is useless. NodeEOF should be checked for seperately to terminate
// execution.
func (NodeEOF) Eval(*variables.Scope, *IOContainer) ExitStatus { return ExitSuccess }

type NodeList []Node

// Eval calls Eval on the Nodes contained in the list and returns the
// ExitStatus of the last command.
func (n NodeList) Eval(scp *variables.Scope, ioc *IOContainer) ExitStatus {
	returnExit := ExitSuccess

	for _, x := range n {
		returnExit = x.Eval(scp, ioc)
	}

	return returnExit
}

// NodeBinary is used to chain nodes conditionally
type NodeBinary struct {
	Left, Right Node
	IsAnd       bool
}

func (n NodeBinary) Eval(scp *variables.Scope, ioc *IOContainer) ExitStatus {
	var runRight bool

	leftExit := n.Left.Eval(scp, ioc)
	if n.IsAnd {
		runRight = leftExit == ExitSuccess
	} else {
		runRight = leftExit != ExitSuccess
	}

	if runRight {
		return n.Right.Eval(scp, ioc)
	}

	return leftExit
}

// NodeNegate is used to flip the ExitStatus of the contained Node
type NodeNegate struct {
	N Node
}

func (n NodeNegate) Eval(scp *variables.Scope, ioc *IOContainer) ExitStatus {
	ex := n.N.Eval(scp, ioc)
	// Any Non-zero ExitStatus is a failure so we only check for success
	if ex == ExitSuccess {
		return ExitFailure
	}
	return ExitSuccess
}

type NodeLoop struct {
	IsWhile   bool
	Condition Node
	Body      Node
}

func (n NodeLoop) Eval(scp *variables.Scope, ioc *IOContainer) ExitStatus {
	var runBody bool
	returnExit := ExitSuccess

	for {
		condExit := n.Condition.Eval(scp, ioc)
		if n.IsWhile {
			runBody = condExit == ExitSuccess
		} else { // Until
			runBody = condExit != ExitSuccess
		}

		if runBody {
			returnExit = n.Body.Eval(scp, ioc)
		} else {
			break
		}
	}

	return returnExit
}

type NodeFor struct {
	LoopVar string
	Args    []Arg
	Body    Node
}

func (n NodeFor) Eval(scp *variables.Scope, ioc *IOContainer) ExitStatus {
	returnExit := ExitSuccess

	expandedArgs := make([]string, len(n.Args))
	for i, arg := range n.Args {
		// This will need to be changed when IFS splitting is coded.
		// Append each split as a seperate item
		expandedArgs[i] = arg.Expand(scp)
	}

	for _, arg := range expandedArgs {
		scp.Set(n.LoopVar, arg)
		returnExit = n.Body.Eval(scp, ioc)
	}

	return returnExit
}

// NodeIf is the structure that is used for 'if', 'elif' and 'else'
// as an 'if' or 'elif' Condition is required and Else is optionally nil to
// indicate the end of the if chain.
// as an 'else' Condition is required to be nil.
type NodeIf struct {
	Condition *Node
	Else      *NodeIf
	Body      Node
}

func (n NodeIf) Eval(scp *variables.Scope, ioc *IOContainer) ExitStatus {
	logex.Debug("Entered if")
	if n.Condition == nil {
		return n.Body.Eval(scp, ioc)
	}

	runBody := (*n.Condition).Eval(scp, ioc)
	if runBody == ExitSuccess {
		return n.Body.Eval(scp, ioc)
	}

	if n.Else != nil {
		return n.Else.Eval(scp, ioc)
	}

	return ExitSuccess
}

type NodeCommand struct {
	Assign []string
	Args   []Arg
	LineNo int
}

func (n NodeCommand) Eval(scp *variables.Scope, ioc *IOContainer) ExitStatus {
	logex.Pretty(n)
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

	/* ===== THIS NEEDS TO BE EXTRACTED ====
	* This should be the place that we search for builtins,
	* relative path commands, commands etc.
	* Will also need to be able to add redirections here somehow. */
	env := scp.Environ()
	cmd := exec.Command(expandedArgs[0], expandedArgs[1:]...)
	cmd.Env = env
	cmd.Stdin = ioc.In
	cmd.Stderr = ioc.Err
	cmd.Stdout = ioc.Out

	// This is needed so that pipes will terminate
	if pw, isPipeWriter := ioc.Out.(*io.PipeWriter); isPipeWriter {
		defer func() {
			pw.Close()
		}()
	}

	logex.Info("========ENV======")
	logex.Pretty(env)
	logex.Info("========CMD======")
	logex.Pretty(cmd)
	logex.Info("========EXEC======")
	err := cmd.Run()
	logex.Info("=======EXIT======")
	if err == nil {
		logex.Info("> Success")
		return ExitSuccess
	}
	logex.Error(err)
	logex.Info("> Failure")
	return ExitFailure
	// ===== THIS NEEDS TO BE EXTRACTED ====
}

type NodeCaseList struct {
	Patterns []Arg
	Body     Node
}

func (n NodeCaseList) Eval(scp *variables.Scope, ioc *IOContainer) ExitStatus {
	return n.Body.Eval(scp, ioc)
}

func (n NodeCaseList) Matches(s string, scp *variables.Scope) bool {
	for _, p := range n.Patterns {
		if fnmatch.Match(p.Raw, s, 0) {
			return true
		}
	}
	return false
}

type NodeCase struct {
	Expr  Arg
	Cases []NodeCaseList
}

func (n NodeCase) Eval(scp *variables.Scope, ioc *IOContainer) ExitStatus {
	expandedExpr := n.Expr.Expand(scp)

	for _, c := range n.Cases {
		if c.Matches(expandedExpr, scp) {
			return c.Eval(scp, ioc)
		}
	}
	return ExitSuccess
}

type NodePipe struct {
	Background bool
	Commands   NodeList
}

func (n NodePipe) Eval(scp *variables.Scope, ioc *IOContainer) ExitStatus {
	lastPipeReader, pipeWriter := io.Pipe()

	cmd := n.Commands[0]
	go cmd.Eval(scp, &IOContainer{In: ioc.In, Out: pipeWriter, Err: ioc.Err})

	for _, cmd = range n.Commands[1 : len(n.Commands)-1] {
		pipeReader, pipeWriter := io.Pipe()
		go cmd.Eval(scp, &IOContainer{In: lastPipeReader, Out: pipeWriter, Err: ioc.Err})
		lastPipeReader = pipeReader
	}

	cmd = n.Commands[len(n.Commands)-1]
	if !n.Background {
		return cmd.Eval(scp, &IOContainer{In: lastPipeReader, Out: ioc.Out, Err: ioc.Err})
	}

	go cmd.Eval(scp, &IOContainer{In: lastPipeReader, Out: ioc.Out, Err: ioc.Err})
	return ExitSuccess
}
