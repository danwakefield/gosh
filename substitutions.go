package main

import (
	"bytes"
	"os"
	"strconv"
	"strings"

	"gopkg.in/logex.v1"

	"github.com/danwakefield/gosh/arith"
	"github.com/danwakefield/gosh/variables"
)

type Substitution interface {
	Sub(*variables.Scope) string
}

type SubSubshell struct {
	N Node
}

func (s SubSubshell) Sub(scp *variables.Scope) (returnString string) {
	logex.Debug("Substituting shell")
	defer func() {
		logex.Debugf("Returned '%s'", returnString)
	}()

	if _, isNoop := s.N.(NodeNoop); isNoop {
		return ""
	}

	out := &bytes.Buffer{}
	// Not sure if we need to capture this exit code for the $? var.
	// Ignore it for now
	_ = s.N.Eval(scp.Copy(), &IOContainer{&bytes.Buffer{}, out, os.Stderr})

	return strings.TrimRight(out.String(), "\n")
}

type VarSubType int

const (
	VarSubNormal VarSubType = iota
	VarSubMinus
	VarSubPlus
	VarSubQuestion
	VarSubAssign
	VarSubTrimRight
	VarSubTrimRightMax
	VarSubTrimLeft
	VarSubTrimLeftMax
	VarSubLength

	// VarSubSubString, VarSubReplace and VarSubReplaceAll
	// are not used. The parser can currently only handle
	// Var subs that use a single arg following the operator symbol
	VarSubSubString
	VarSubReplace
	VarSubReplaceAll
)

type SubVariable struct {
	VarName   string
	SubVal    string `json:",omitempty"` // The text following any sub operator
	CheckNull bool
	SubType   VarSubType
}

func (s SubVariable) Sub(scp *variables.Scope) (returnString string) {
	logex.Debug("Substituting variable")
	logex.Pretty(s)
	defer func() {
		logex.Debugf("Returned '%s'", returnString)
	}()
	v := scp.Get(s.VarName)

	switch s.SubType {
	case VarSubNormal:
		return v.Val
	case VarSubLength:
		// For the values ${#*} and ${#@}
		// the number of positional parameters is returned
		// We need to perform IFS splitting to figure this out
		return strconv.Itoa(len(v.Val))
	}

	varExists := v.Set == true
	// CheckNull means that an empty string is treated as unset
	if s.CheckNull {
		varExists = varExists && v.Val != ""
	}

	switch s.SubType {
	case VarSubAssign:
		if varExists {
			return v.Val
		}
		scp.Set(s.VarName, s.SubVal)
		return s.SubVal
	case VarSubMinus:
		if varExists {
			return v.Val
		}
		return s.SubVal
	case VarSubPlus:
		if varExists {
			return ""
		}
		return s.SubVal
	case VarSubQuestion:
		if varExists {
			return v.Val
		}
		if s.SubVal != "" {
			ExitShellWithMessage(ExitFailure, s.SubVal)
		}
		ExitShellWithMessage(ExitFailure, s.VarName+": Parameter not set")
	case VarSubTrimRight, VarSubTrimRightMax, VarSubTrimLeft, VarSubTrimLeftMax:
		ExitShellWithMessage(ExitFailure, "Trim operations not implemented")
	}

	logex.Panic("Not Reached")
	return ""
}

type SubArith struct {
	Raw string
}

func (s SubArith) Sub(scp *variables.Scope) string {
	logex.Debug("Subtituting arithmetic")
	logex.Pretty(s)
	i, err := arith.Parse(s.Raw, scp)
	if err != nil {
		panic(err)
	}
	return strconv.FormatInt(i, 10)
}
