package main

import (
	"bytes"
	"io"
	"os/exec"
	"os/user"
	"strings"

	"gopkg.in/logex.v1"

	"github.com/danwakefield/fnmatch"
	"github.com/danwakefield/gosh/T"
	"github.com/danwakefield/gosh/builtins"
	"github.com/danwakefield/gosh/variables"
)

type Arg struct {
	Raw    string
	Quoted bool
	Subs   []Substitution
}

type ExpandFlag int

const (
	NoExpandSubstitutions ExpandFlag = iota
	NoExpandTilde
	NoExpandWordSplit
	NoExpandGlob
)

func (a Arg) Expand(scp *variables.Scope, flags ...ExpandFlag) (returnString string) {
	flagSet := func(e ExpandFlag) bool {
		for _, v := range flags {
			if v == e {
				return true
			}
		}
		return false
	}

	logex.Debugf("Expand '%s'", a.Raw)
	defer func() {
		logex.Debugf("Returned '%s'", returnString)
	}()

	expString := a.Raw

	if !flagSet(NoExpandTilde) {
		expString = a.expandTilde(scp, expString)
	}

	if !flagSet(NoExpandSubstitutions) {
		expString = a.expandSubstitutions(scp, expString)
	}

	// XXX: Do FNmatch pathname expansion here.
	// see `man 7 glob` for details. Key point is that is the expansion
	// has no files it should be returned as is.
	return expString
}

func (a Arg) expandSubstitutions(scp *variables.Scope, s string) string {
	if !strings.ContainsRune(s, SentinalSubstitution) {
		return s
	}
	// Split the Raw string into a []string. Each element would have been
	// immediately followed by a substitution.
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return r == SentinalSubstitution
	})

	x := make([]string, len(a.Subs)+len(fields))
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

func (a Arg) expandTilde(scp *variables.Scope, s string) string {
	if strings.HasPrefix(s, "~") {
		u, err := user.Current()
		if err != nil {
			return s
		}
		return u.HomeDir + s[1:]
	} else {
		return s
	}
}

type Node interface {
	Eval(*variables.Scope, *T.IOContainer) T.ExitStatus
}

type NodeNoop struct{}

func (NodeNoop) Eval(*variables.Scope, *T.IOContainer) T.ExitStatus { return T.ExitSuccess }

// NodeEOF is end of file sentinal node.
type NodeEOF struct {
	NodeNoop
}

type NodeList []Node

// Eval calls Eval on the Nodes contained in the list and returns the
// T.ExitStatus of the last command.
func (n NodeList) Eval(scp *variables.Scope, ioc *T.IOContainer) T.ExitStatus {
	returnExit := T.ExitSuccess

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

func (n NodeBinary) Eval(scp *variables.Scope, ioc *T.IOContainer) T.ExitStatus {
	var runRight bool

	leftExit := n.Left.Eval(scp, ioc)
	if n.IsAnd {
		runRight = leftExit == T.ExitSuccess
	} else {
		runRight = leftExit != T.ExitSuccess
	}

	if runRight {
		return n.Right.Eval(scp, ioc)
	}

	return leftExit
}

// NodeNegate is used to flip the T.ExitStatus of the contained Node
type NodeNegate struct {
	N Node
}

func (n NodeNegate) Eval(scp *variables.Scope, ioc *T.IOContainer) T.ExitStatus {
	ex := n.N.Eval(scp, ioc)
	// Any Non-zero T.ExitStatus is a failure so we only check for success
	if ex == T.ExitSuccess {
		return T.ExitFailure
	}
	return T.ExitSuccess
}

type NodeLoop struct {
	IsWhile   bool
	Condition Node
	Body      Node
}

func (n NodeLoop) Eval(scp *variables.Scope, ioc *T.IOContainer) T.ExitStatus {
	var runBody bool
	returnExit := T.ExitSuccess

	for {
		condExit := n.Condition.Eval(scp, ioc)
		if n.IsWhile {
			runBody = condExit == T.ExitSuccess
		} else { // Until
			runBody = condExit != T.ExitSuccess
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

func (n NodeFor) Eval(scp *variables.Scope, ioc *T.IOContainer) T.ExitStatus {
	returnExit := T.ExitSuccess

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
	Condition Node
	Else      *NodeIf
	Body      Node
}

func (n NodeIf) Eval(scp *variables.Scope, ioc *T.IOContainer) T.ExitStatus {
	runBody := n.Condition.Eval(scp, ioc)
	if runBody == T.ExitSuccess {
		return n.Body.Eval(scp, ioc)
	}

	if n.Else != nil {
		return n.Else.Eval(scp, ioc)
	}

	return T.ExitSuccess
}

type NodeCommand struct {
	Assign map[string]Arg
	Args   []Arg
	LineNo int
}

func (n NodeCommand) execExternal(scp *variables.Scope, ioc *T.IOContainer, args []string) T.ExitStatus {
	// This is needed so that pipes will terminate
	if pw, isPipeWriter := ioc.Out.(*io.PipeWriter); isPipeWriter {
		defer func() {
			if err := pw.Close(); err != nil {
				panic(err) // XXX: Print error to stdout and continue?
			}
		}()
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = scp.Environ()
	cmd.Stdin = ioc.In
	cmd.Stderr = ioc.Err
	cmd.Stdout = ioc.Out

	err := cmd.Run()
	if err == nil {
		return T.ExitSuccess
	}
	return T.ExitFailure
}

func (n NodeCommand) execFunction(scp *variables.Scope, ioc *T.IOContainer, args []string) T.ExitStatus {
	return T.ExitSuccess
}

func (n NodeCommand) Eval(scp *variables.Scope, ioc *T.IOContainer) T.ExitStatus {
	logex.Pretty(n)
	// A line with only assignments applies them to the Root Scope
	// We check this first to avoid unnecessary scope Push/Pop's
	if len(n.Args) == 0 {
		for k, v := range n.Assign {
			scp.Set(k, v.Expand(scp))
		}
		return T.ExitSuccess
	}

	// Minimum of len(n.Args) after expansions, Likely
	// that it will be more after globbing though
	expandedArgs := []string{}
	for _, arg := range n.Args {
		expandedArgs = append(expandedArgs, arg.Expand(scp))
	}

	command := expandedArgs[0]
	builtinFunc, builtinFound := builtins.All[command]
	userFunc, userFuncFound := scp.Functions[command]

	if strings.ContainsRune(command, '/') || (!builtinFound && !userFuncFound) {
		scp.Push()
		defer scp.Pop()

		for k, v := range n.Assign {
			scp.Set(k, v.Expand(scp), variables.LocalScope)
		}
		return n.execExternal(scp, ioc, expandedArgs)
	}

	if builtinFound {
		return builtinFunc(scp, ioc, expandedArgs[1:])
	}

	if userFuncFound {
		x := userFunc.(NodeFunction)
		return x.EvalFunc(scp, ioc, expandedArgs[1:])
	}

	return T.ExitUnknownCommand
}

type NodeCaseList struct {
	Patterns []Arg
	Body     Node
}

func (n NodeCaseList) Eval(scp *variables.Scope, ioc *T.IOContainer) T.ExitStatus {
	return n.Body.Eval(scp, ioc)
}

func (n NodeCaseList) Matches(s string, scp *variables.Scope) bool {
	for _, p := range n.Patterns {
		expandedPat := p.Expand(scp)
		if fnmatch.Match(expandedPat, s, 0) {
			return true
		}
	}
	return false
}

type NodeCase struct {
	Expr  Arg
	Cases []NodeCaseList
}

func (n NodeCase) Eval(scp *variables.Scope, ioc *T.IOContainer) T.ExitStatus {
	expandedExpr := n.Expr.Expand(scp)

	for _, c := range n.Cases {
		if c.Matches(expandedExpr, scp) {
			return c.Eval(scp, ioc)
		}
	}
	return T.ExitSuccess
}

type NodePipe struct {
	Background bool
	Commands   NodeList
}

func (n NodePipe) Eval(scp *variables.Scope, ioc *T.IOContainer) T.ExitStatus {
	lastPipeReader, pipeWriter := io.Pipe()

	scp = scp.Copy()

	cmd := n.Commands[0]
	go cmd.Eval(scp, &T.IOContainer{In: &bytes.Buffer{}, Out: pipeWriter, Err: ioc.Err})

	for _, cmd = range n.Commands[1 : len(n.Commands)-1] {
		pipeReader, pipeWriter := io.Pipe()
		go cmd.Eval(scp, &T.IOContainer{In: lastPipeReader, Out: pipeWriter, Err: ioc.Err})
		lastPipeReader = pipeReader
	}

	cmd = n.Commands[len(n.Commands)-1]
	if !n.Background {
		return cmd.Eval(scp, &T.IOContainer{In: lastPipeReader, Out: ioc.Out, Err: ioc.Err})
	}

	go cmd.Eval(scp, &T.IOContainer{In: lastPipeReader, Out: ioc.Out, Err: ioc.Err})
	return T.ExitSuccess
}

type NodeFunction struct {
	Body Node
	Name string
}

func (n NodeFunction) Eval(scp *variables.Scope, ioc *T.IOContainer) T.ExitStatus {
	scp.Functions[n.Name] = n
	return T.ExitSuccess
}

func (n NodeFunction) EvalFunc(scp *variables.Scope, ioc *T.IOContainer, args []string) T.ExitStatus {
	scp.Push()
	defer scp.Pop()
	return n.Body.Eval(scp, ioc)
}
