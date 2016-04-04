package main

import (
	"os"
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

	return strings.Join(x, "")
}

type IOContainer struct {
	In  *os.File
	Out *os.File
	Err *os.File
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
func (n NodeList) Eval(scp *variables.Scope, io *IOContainer) ExitStatus {
	returnExit := ExitSuccess

	for _, x := range n {
		returnExit = x.Eval(scp, io)
	}

	return returnExit
}

type NodeBinary struct {
	Left, Right Node
	IsAnd       bool
}

func (n NodeBinary) Eval(scp *variables.Scope, io *IOContainer) ExitStatus {
	var runRight bool

	leftExit := n.Left.Eval(scp, io)
	if n.IsAnd {
		runRight = leftExit == ExitSuccess
	} else {
		runRight = leftExit != ExitSuccess
	}

	if runRight {
		return n.Right.Eval(scp, io)
	}

	return leftExit
}

// NodeNegate is used to flip the ExitStatus of the contained Node
type NodeNegate struct {
	N Node
}

func (n NodeNegate) Eval(scp *variables.Scope, io *IOContainer) ExitStatus {
	ex := n.N.Eval(scp, io)
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

func (n NodeLoop) Eval(scp *variables.Scope, io *IOContainer) ExitStatus {
	var runBody bool
	returnExit := ExitSuccess

	for {
		condExit := n.Condition.Eval(scp, io)
		if n.IsWhile {
			runBody = condExit == ExitSuccess
		} else { // Until
			runBody = condExit != ExitSuccess
		}

		if runBody {
			returnExit = n.Body.Eval(scp, io)
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

func (n NodeFor) Eval(scp *variables.Scope, io *IOContainer) ExitStatus {
	returnExit := ExitSuccess

	expandedArgs := make([]string, len(n.Args))
	for i, arg := range n.Args {
		// This will need to be changed when IFS splitting is coded.
		// Append each split as a seperate item
		expandedArgs[i] = arg.Expand(scp)
	}

	for _, arg := range expandedArgs {
		scp.Set(n.LoopVar, arg)
		returnExit = n.Body.Eval(scp, io)
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

func (n NodeIf) Eval(scp *variables.Scope, io *IOContainer) ExitStatus {
	logex.Debug("Entered if")
	if n.Condition == nil {
		return n.Body.Eval(scp, io)
	}

	runBody := (*n.Condition).Eval(scp, io)
	if runBody == ExitSuccess {
		return n.Body.Eval(scp, io)
	}

	if n.Else != nil {
		return n.Else.Eval(scp, io)
	}

	return ExitSuccess
}

type NodeCommand struct {
	Assign []string
	Args   []Arg
	LineNo int
}

func (n NodeCommand) Eval(scp *variables.Scope, io *IOContainer) ExitStatus {
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
	cmd.Stdin = io.In
	cmd.Stdout = io.Out
	cmd.Stderr = io.Err

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

func (n NodeCaseList) Eval(scp *variables.Scope, io *IOContainer) ExitStatus {
	return n.Body.Eval(scp, io)
}

func (n NodeCaseList) Matches(s string, scp *variables.Scope) bool {
	for _, p := range n.Patterns {
		// Apply FNMatch here.
		expandedPattern := p.Expand(scp)
		if fnmatch.Match(expandedPattern, s, 0) {
			return true
		}
	}
	return false
}

type NodeCase struct {
	Expr  Arg
	Cases []NodeCaseList
}

func (n NodeCase) Eval(scp *variables.Scope, io *IOContainer) ExitStatus {
	expandedExpr := n.Expr.Expand(scp)

	for _, c := range n.Cases {
		if c.Matches(expandedExpr, scp) {
			return c.Eval(scp, io)
		}
	}
	return ExitSuccess
}

type NodePipe struct {
	Background bool
	Commands   NodeList
}

func (n NodePipe) Eval(scp *variables.Scope, io *IOContainer) ExitStatus {
	returnExit := ExitSuccess

	return returnExit
}
