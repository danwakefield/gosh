package main

import (
	"strconv"

	"github.com/danwakefield/gosh/variables"
)

type Substitution interface {
	Sub(*variables.Scope) string
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
	SubVal    string // The text following any sub operator
	CheckNull bool
	SubType   VarSubType
}

func (s SubVariable) Sub(scp *variables.Scope) string {
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

	varExists := v.Set == True
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
	}
}
