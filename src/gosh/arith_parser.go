package main

import (
	"fmt"
	"strconv"
)

// ArithNode is an implementation of the symbols described
// in Top Down Operator Precedence; Vaughn Pratt; 1973
type ArithNode interface {
	Nud() int64
	Led(int64) int64
	Lbp() int
}

func Parse(s string) int64 {
	ap := &ArithParser{lexer: NewArithLexer(s)}
	ap.next()
	parser = ap
	return parser.expression(0)
}

var parser *ArithParser

type ArithParser struct {
	lastNode  ArithNode
	lastToken ArithToken
	lexer     *ArithLexer
}

func (ap *ArithParser) expression(rbp int) int64 {
	n := ap.lastNode
	ap.next()
	left := n.Nud()
	for rbp < ap.lastNode.Lbp() {
		n = ap.lastNode
		ap.next()
		left = n.Led(left)
	}
	return left
}

func (ap *ArithParser) consume(t ArithToken) {
	if t != ap.lastToken {
		panic("consume Failed: Expected " + t.String())
	}
	ap.next()
}

func (ap *ArithParser) next() {
	l := ap.lexer.Lex()
	switch {
	case TokenIsBinaryOp(l.T):
		ap.lastNode = InfixNode{T: l.T}
	case TokenIsAssignmentOp(l.T) || TokenIs(l.T, ArithAssignment):
		ap.lastNode = InfixAssignNode{T: l.T, V: ap.lastNode}
	case TokenIs(l.T, ArithNumber):
		ap.lastNode = LiteralNode{Val: l.Val.(int64)}
	case TokenIs(l.T, ArithVariable):
		ap.lastNode = VariableNode{Val: l.Val.(string)}
	case TokenIs(l.T, ArithBinaryNot, ArithNot, ArithLeftParen):
		ap.lastNode = PrefixNode{T: l.T}
	case TokenIs(l.T, ArithAdd, ArithOr):
		ap.lastNode = InfixRightNode{T: l.T}
	case TokenIs(l.T, ArithEOF):
		ap.lastNode = EOFNode{}
	case TokenIs(l.T, ArithQuestionMark):
		ap.lastNode = TernaryNode{}
	case TokenIs(l.T, ArithRightParen, ArithColon):
		ap.lastNode = NoopNode{T: l.T}
	default:
		panic(fmt.Sprintf("%T - %#v\n%s - %T - %#v", l, l, l.T, l.T, l.T))
	}
	ap.lastToken = l.T
}

func (ap *ArithParser) GetVariable(name string) int64 {
	v := GlobalScope.Get(name)
	if v.Val == "" {
		return 0
	}
	i, err := strconv.ParseInt(v.Val, 0, 64)
	if err != nil {
		panic("Variable '" + name + "' cannot be used as a number: " + err.Error())
	}
	return i
}

func (ap *ArithParser) SetVariable(name string, val int64) {
	GlobalScope.Set(name, strconv.FormatInt(val, 10))
}

// IsArithBinaryOp checks if a token operates on two values.
// E.g a + b, a << b
func TokenIsBinaryOp(a ArithToken) bool {
	return a <= ArithAdd && a >= ArithLessEqual
}

// IsArithAssignmentOp checks if a token assigns to the lefthand variable.
// E.g a += b, a <<= b
func TokenIsAssignmentOp(a ArithToken) bool {
	return a <= ArithAssignAdd && a >= ArithAssignBinaryAnd
}

func TokenIs(toks ...ArithToken) bool {
	if len(toks) < 2 {
		return false
	}
	have := toks[0]
	toks = toks[1:]
	for _, t := range toks {
		if have == t {
			return true
		}
	}
	return false
}

type EOFNode struct{}

func (n EOFNode) Nud() int64      { panic("Nud called on EOFNode") }
func (n EOFNode) Led(int64) int64 { panic("Led called on EOFNode") }
func (n EOFNode) Lbp() int        { return -1 }

type NoopNode struct {
	T ArithToken
}

func (n NoopNode) Nud() int64      { panic("Nud called on NoopNode: " + n.T.String()) }
func (n NoopNode) Led(int64) int64 { panic("Led called on NoopNode: " + n.T.String()) }
func (n NoopNode) Lbp() int        { return 0 }

type LiteralNode struct {
	Val int64
}

func (n LiteralNode) Nud() int64      { return n.Val }
func (n LiteralNode) Led(int64) int64 { panic("Led called on LiteralNode") }
func (n LiteralNode) Lbp() int        { return 0 }

type VariableNode struct {
	Val string
}

func (n VariableNode) Nud() int64      { return parser.GetVariable(n.Val) }
func (n VariableNode) Led(int64) int64 { panic("Led called on VariableNode") }
func (n VariableNode) Lbp() int        { return 0 }

var (
	InfixNudFunctions = map[ArithToken]func() int64{
		ArithAdd:      func() int64 { return parser.expression(150) },
		ArithSubtract: func() int64 { return -parser.expression(150) },
	}
	PrefixNudFunctions = map[ArithToken]func() int64{
		ArithBinaryNot: func() int64 { return -parser.expression(LbpValues[ArithBinaryNot]) - 1 },
		ArithNot:       func() int64 { return BoolToShell(parser.expression(LbpValues[ArithNot]) != ShellTrue) },
		ArithLeftParen: func() int64 {
			e := parser.expression(0)
			parser.consume(ArithRightParen)
			return e
		},
	}
	InfixLedFunctions = map[ArithToken]func(int64, int64) int64{
		ArithLessEqual:    func(l, r int64) int64 { return BoolToShell(l <= r) },
		ArithGreaterEqual: func(l, r int64) int64 { return BoolToShell(l >= r) },
		ArithLessThan:     func(l, r int64) int64 { return BoolToShell(l < r) },
		ArithGreaterThan:  func(l, r int64) int64 { return BoolToShell(l > r) },
		ArithEqual:        func(l, r int64) int64 { return BoolToShell(l == r) },
		ArithNotEqual:     func(l, r int64) int64 { return BoolToShell(l != r) },
		ArithBinaryAnd:    func(l, r int64) int64 { return l & r },
		ArithBinaryOr:     func(l, r int64) int64 { return l | r },
		ArithBinaryXor:    func(l, r int64) int64 { return l ^ r },
		ArithLeftShift:    func(l, r int64) int64 { return LeftShift(l, r) },
		ArithRightShift:   func(l, r int64) int64 { return RightShift(l, r) },
		ArithRemainder:    func(l, r int64) int64 { return l % r },
		ArithMultiply:     func(l, r int64) int64 { return l * r },
		ArithDivide:       func(l, r int64) int64 { return l / r },
		ArithSubtract:     func(l, r int64) int64 { return l - r },
		ArithAdd:          func(l, r int64) int64 { return l + r },
		ArithAssignment:   func(l, r int64) int64 { return r },
	}
	InfixRightLedFunctions = map[ArithToken]func(int64, int64) int64{
		ArithAnd: func(l, r int64) int64 { return BoolToShell((l == ShellTrue) && (r == ShellTrue)) },
		ArithOr:  func(l, r int64) int64 { return BoolToShell((l == ShellTrue) || (r == ShellTrue)) },
	}
	LbpValues = map[ArithToken]int{
		ArithRightParen:   20,
		ArithOr:           30,
		ArithAnd:          40,
		ArithNot:          50,
		ArithLessEqual:    60,
		ArithGreaterEqual: 60,
		ArithLessThan:     60,
		ArithGreaterThan:  60,
		ArithEqual:        60,
		ArithNotEqual:     60,
		ArithAssignment:   60,
		ArithBinaryOr:     70,
		ArithBinaryXor:    80,
		ArithBinaryAnd:    90,
		ArithLeftShift:    100,
		ArithRightShift:   100,
		ArithSubtract:     110,
		ArithAdd:          110,
		ArithMultiply:     120,
		ArithDivide:       120,
		ArithRemainder:    120,
		ArithBinaryNot:    130,
		ArithLeftParen:    140,
	}
)

type InfixAssignNode struct {
	T ArithToken
	V ArithNode
}

func (n InfixAssignNode) Nud() int64 { panic("Nud called on InfixAssignNode: " + n.T.String()) }
func (n InfixAssignNode) Led(left int64) int64 {
	v, ok := n.V.(VariableNode)
	var f func(int64, int64) int64
	if !ok {
		panic("LHS of assignment '" + n.T.String() + "' is not a variable")
	}

	if n.T == ArithAssignment {
		f = InfixLedFunctions[ArithAssignment]
	} else {
		f, ok = InfixLedFunctions[n.T-ArithAssignDiff]
		if !ok {
			panic("Led called on InfixAssignNode: " + n.T.String())
		}
	}

	right := parser.expression(0)
	t := f(left, right)
	parser.SetVariable(v.Val, t)
	return t
}
func (n InfixAssignNode) Lbp() int {
	if n.T == ArithAssignment {
		return LbpValues[n.T]
	}
	return LbpValues[n.T-ArithAssignDiff]
}

type InfixNode struct {
	T ArithToken
}

func (n InfixNode) Nud() int64 {
	f, ok := InfixNudFunctions[n.T]
	if !ok {
		panic("Nud called on InfixNode: " + n.T.String())
	}
	return f()
}
func (n InfixNode) Led(left int64) int64 {
	right := parser.expression(n.Lbp())
	f, ok := InfixLedFunctions[n.T]
	if !ok {
		panic("Led called on InfixNode: " + n.T.String())
	}
	return f(left, right)
}
func (n InfixNode) Lbp() int { return LbpValues[n.T] }

type InfixRightNode struct {
	T ArithToken
}

func (n InfixRightNode) Nud() int64 { panic("Nud called on InfixRightNode: " + n.T.String()) }
func (n InfixRightNode) Led(left int64) int64 {
	right := parser.expression(n.Lbp() - 1)
	f, ok := InfixRightLedFunctions[n.T]
	if !ok {
		panic("Led called on InfixRightNode: " + n.T.String())
	}
	return f(left, right)
}
func (n InfixRightNode) Lbp() int { return LbpValues[n.T] }

type PrefixNode struct {
	T ArithToken
}

func (n PrefixNode) Nud() int64 {
	f, ok := PrefixNudFunctions[n.T]
	if !ok {
		panic("Nud called on PrefixNode: " + string(n.T))
	}
	return f()
}

func (n PrefixNode) Led(int64) int64 { panic("Led called on PrefixNode: " + n.T.String()) }
func (n PrefixNode) Lbp() int        { return LbpValues[n.T] }

type TernaryNode struct {
	condition         int64
	valTrue, valFalse int64
}

func (n TernaryNode) Nud() int64 { panic("Nud called on TernaryNode") }
func (n TernaryNode) Led(left int64) int64 {
	// Somewhat confusingly the shell's ternary operator does not work using
	// the shell's True/False semantics.
	// The actual operation is Given (a ? b : c)
	// if (a != 0)
	//	return b
	// else
	//	return c
	// See the ISO C Standard Section 6.5.15
	// This function evaluates both sides of the ternary no matter
	// what the condition is.

	n.condition = left
	n.valTrue = parser.expression(0)
	parser.consume(ArithColon)
	n.valFalse = parser.expression(0)

	if n.condition != 0 {
		return n.valTrue
	}
	return n.valFalse
}
func (n TernaryNode) Lbp() int {
	return 20
}
