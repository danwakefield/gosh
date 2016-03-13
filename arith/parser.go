package arith

import (
	"errors"
	"runtime"
	"strconv"

	"github.com/danwakefield/gosh/variables"
)

var (
	ErrUnknownToken = errors.New("Unknown token returned by lex")
)

type ParseError struct {
	Err      error
	Fallback string
}

func (e ParseError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Fallback
}

// ArithNode is an implementation of the symbols described
// in Top Down Operator Precedence; Vaughn Pratt; 1973
type ArithNode interface {
	nud() int64
	led(int64) int64
	lbp() int
}

func ParseArith(input string, scp *variables.Scope) (i int64, err error) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			switch t := r.(type) {
			case string:
				err = ParseError{Fallback: t}
			case error:
				err = ParseError{Err: t}
			}
		}
	}()
	ap := &ArithParser{
		lexer: NewArithLexer(input),
		scope: scp,
	}
	ap.next()
	parser = ap
	return parser.expression(0), nil
}

var parser *ArithParser

type ArithParser struct {
	lastNode  ArithNode
	lastToken ArithToken
	lexer     *ArithLexer
	scope     *variables.Scope
}

func (ap *ArithParser) expression(rbp int) int64 {
	n := ap.lastNode
	ap.next()
	left := n.nud()
	for rbp < ap.lastNode.lbp() {
		n = ap.lastNode
		ap.next()
		left = n.led(left)
	}
	return left
}

func (ap *ArithParser) consume(t ArithToken) {
	if t != ap.lastToken {
		panic("Expected '" + t.String() + "'")
	}
	ap.next()
}

func (ap *ArithParser) next() {
	tok, val := ap.lexer.Lex()
	switch {
	case TokenIsBinaryOp(tok):
		ap.lastNode = InfixNode{Tok: tok}
	case TokenIsAssignmentOp(tok) || TokenIs(tok, ArithAssignment):
		ap.lastNode = InfixAssignNode{Tok: tok, Val: ap.lastNode}
	case TokenIs(tok, ArithAdd, ArithOr):
		ap.lastNode = InfixRightNode{Tok: tok}
	case TokenIs(tok, ArithNumber):
		ap.lastNode = LiteralNode{Val: val.(int64)}
	case TokenIs(tok, ArithVariable):
		ap.lastNode = VariableNode{Val: val.(string)}
	case TokenIs(tok, ArithBinaryNot, ArithNot, ArithLeftParen):
		ap.lastNode = PrefixNode{Tok: tok}
	case TokenIs(tok, ArithEOF):
		ap.lastNode = EOFNode{}
	case TokenIs(tok, ArithQuestionMark):
		ap.lastNode = TernaryNode{}
	case TokenIs(tok, ArithRightParen, ArithColon):
		ap.lastNode = NoopNode{Tok: tok}
	default:
		panic(ErrUnknownToken)
	}
	ap.lastToken = tok
}

func (ap *ArithParser) getVariable(name string) int64 {
	v := ap.scope.Get(name)
	// We dont care if the variable if unset or empty they both
	// count as a zero
	if v.Val == "" {
		return 0
	}
	// ParseInt figures out the format of the variable if is in hex / octal
	// format so we can just perform one conversion.
	i, err := strconv.ParseInt(v.Val, 0, 64)
	if err != nil {
		panic("Variable '" + name + "' cannot be used as a number: " + err.Error())
	}
	return i
}

func (ap *ArithParser) setVariable(name string, val int64) {
	ap.scope.Set(name, strconv.FormatInt(val, 10))
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

// TokenIs checks if the first supplied token is equal to any of the other
// supplied tokens.
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

func (n EOFNode) nud() int64      { panic("Nud called on EOFNode") }
func (n EOFNode) led(int64) int64 { panic("Led called on EOFNode") }
func (n EOFNode) lbp() int        { return -1 }

type NoopNode struct {
	Tok ArithToken
}

func (n NoopNode) nud() int64      { panic("Nud called on NoopNode: " + n.Tok.String()) }
func (n NoopNode) led(int64) int64 { panic("Led called on NoopNode: " + n.Tok.String()) }
func (n NoopNode) lbp() int        { return 0 }

type LiteralNode struct {
	Val int64
}

func (n LiteralNode) nud() int64      { return n.Val }
func (n LiteralNode) led(int64) int64 { panic("Led called on LiteralNode") }
func (n LiteralNode) lbp() int        { return 0 }

type VariableNode struct {
	Val string
}

func (n VariableNode) nud() int64      { return parser.getVariable(n.Val) }
func (n VariableNode) led(int64) int64 { panic("Led called on VariableNode") }
func (n VariableNode) lbp() int        { return 0 }

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
	Tok ArithToken
	Val ArithNode
}

func (n InfixAssignNode) nud() int64 { panic("Nud called on InfixAssignNode: " + n.Tok.String()) }
func (n InfixAssignNode) led(left int64) int64 {
	v, ok := n.Val.(VariableNode)
	var fn func(int64, int64) int64
	if !ok {
		panic("LHS of assignment '" + n.Tok.String() + "' is not a variable")
	}

	if n.Tok == ArithAssignment {
		fn = InfixLedFunctions[ArithAssignment]
	} else {
		fn, ok = InfixLedFunctions[n.Tok-ArithAssignDiff]
		if !ok {
			panic("No Led function for InfixAssignNode: " + n.Tok.String())
		}
	}

	right := parser.expression(0)
	t := fn(left, right)
	parser.setVariable(v.Val, t)
	return t
}
func (n InfixAssignNode) lbp() int {
	if n.Tok == ArithAssignment {
		return LbpValues[n.Tok]
	}
	return LbpValues[n.Tok-ArithAssignDiff]
}

type InfixNode struct {
	Tok ArithToken
}

func (n InfixNode) nud() int64 {
	fn, ok := InfixNudFunctions[n.Tok]
	if !ok {
		panic("No Nud function for InfixNode: " + n.Tok.String())
	}
	return fn()
}
func (n InfixNode) led(left int64) int64 {
	right := parser.expression(n.lbp())
	fn, ok := InfixLedFunctions[n.Tok]
	if !ok {
		panic("No Led function for InfixNode: " + n.Tok.String())
	}
	return fn(left, right)
}
func (n InfixNode) lbp() int { return LbpValues[n.Tok] }

type InfixRightNode struct {
	Tok ArithToken
}

func (n InfixRightNode) nud() int64 { panic("Nud called on InfixRightNode: " + n.Tok.String()) }
func (n InfixRightNode) led(left int64) int64 {
	right := parser.expression(n.lbp() - 1)
	fn, ok := InfixRightLedFunctions[n.Tok]
	if !ok {
		panic("No Led function for InfixRightNode: " + n.Tok.String())
	}
	return fn(left, right)
}
func (n InfixRightNode) lbp() int { return LbpValues[n.Tok] }

type PrefixNode struct {
	Tok ArithToken
}

func (n PrefixNode) nud() int64 {
	fn, ok := PrefixNudFunctions[n.Tok]
	if !ok {
		panic("No Nud function for PrefixNode: " + n.Tok.String())
	}
	return fn()
}

func (n PrefixNode) led(int64) int64 { panic("Led called on PrefixNode: " + n.Tok.String()) }
func (n PrefixNode) lbp() int        { return LbpValues[n.Tok] }

type TernaryNode struct {
	condition         int64
	valTrue, valFalse int64
}

func (n TernaryNode) nud() int64 { panic("Nud called on TernaryNode") }
func (n TernaryNode) led(left int64) int64 {
	/* Somewhat confusingly the shell's ternary operator does not work using
	   the shell's True/False semantics.
	   The actual operation is, given (a ? b : c)
	   if (a != 0)
	      return b
	   else
	      return c
	   See the ISO C Standard Section 6.5.15

	   BUG
	   This function evaluates both sides of the ternary no matter
	   what the condition is.
	   This introduces bugs when assignment operators are used alongside
	   the ternary.
	   E.g
	    (0 ? x += 2 : x += 2)
	   will assign x = 4
	   The replaced value will also be 4 so
	    (1 + (0 ? x += 2 : x += 2)) == 5

	    (y ? x = 3 : x = 4)
	   will make x = 4 regardless of the value of y

	   Creating a Q of lexed tokens while blocking assignments then
	   replaying the tokens for the section that should be evaluted
	   should fix this.
	*/

	n.condition = left
	n.valTrue = parser.expression(0)
	parser.consume(ArithColon)
	n.valFalse = parser.expression(0)

	if n.condition != 0 {
		return n.valTrue
	}
	return n.valFalse
}
func (n TernaryNode) lbp() int {
	return 20
}
