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

// ArithNode is an implementation of the methods described
// in Top Down Operator Precedence; Vaughn Pratt; 1973
// also explained in detail in Beautiful Code (2007) by Douglas Crockford
// We have added the Parser arg to allow concurrent parsers the abilty
// to call subexpressions on themselves
type ArithNode interface {
	nud(*Parser) int64
	led(int64, *Parser) int64
	lbp() int
}

func Parse(input string, scp *variables.Scope) (i int64, err error) {
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
	p := &Parser{
		lexer: NewLexer(input),
		scope: scp,
	}
	p.next()
	return p.expression(0), nil
}

type Parser struct {
	lastNode         ArithNode
	lastToken        Token
	lexer            *Lexer
	scope            *variables.Scope
	blockAssignments bool
}

func (p *Parser) expression(rbp int) int64 {
	node := p.lastNode
	p.next()
	left := node.nud(p)
	for rbp < p.lastNode.lbp() {
		node = p.lastNode
		p.next()
		left = node.led(left, p)
	}
	return left
}

func (p *Parser) consume(t Token) {
	if t != p.lastToken {
		panic("Expected '" + t.String() + "'")
	}
	p.next()
}

func (p *Parser) next() {
	tok, val := p.lexer.Lex()
	switch {
	case TokenIsBinaryOp(tok):
		p.lastNode = InfixNode{Tok: tok}
	case TokenIsAssignmentOp(tok) || TokenIs(tok, ArithAssignment):
		p.lastNode = InfixAssignNode{Tok: tok, Val: p.lastNode}
	case TokenIs(tok, ArithAdd, ArithOr):
		p.lastNode = InfixRightNode{Tok: tok}
	case TokenIs(tok, ArithNumber):
		p.lastNode = LiteralNode{Val: val.(int64)}
	case TokenIs(tok, ArithVariable):
		p.lastNode = VariableNode{Val: val.(string)}
	case TokenIs(tok, ArithBinaryNot, ArithNot, ArithLeftParen):
		p.lastNode = PrefixNode{Tok: tok}
	case TokenIs(tok, ArithEOF):
		p.lastNode = EOFNode{}
	case TokenIs(tok, ArithQuestionMark):
		p.lastNode = TernaryNode{}
	case TokenIs(tok, ArithRightParen, ArithColon):
		p.lastNode = NoopNode{Tok: tok}
	default:
		panic(ErrUnknownToken)
	}
	p.lastToken = tok
}

func (p *Parser) getVariable(name string) int64 {
	v := p.scope.Get(name)
	// We dont care if the variable if unset or empty they both
	// count as a zero
	if v.Val == "" {
		return 0
	}
	// ParseInt figures out the base of the variable itself for
	// hex and octal which are the only constants we currently support.
	// Bash adds a base constant type allowing any base from 1 - 64 which
	// would have to be implemented here somehow
	i, err := strconv.ParseInt(v.Val, 0, 64)
	if err != nil {
		panic("Variable '" + name + "' cannot be used as a number: " + err.Error())
	}
	return i
}

func (p *Parser) setVariable(name string, val int64) {
	if !p.blockAssignments {
		p.scope.Set(name, strconv.FormatInt(val, 10))
	}
}

// IsArithBinaryOp checks if a token operates on two values.
// E.g a + b, a << b
func TokenIsBinaryOp(a Token) bool {
	return a <= ArithAdd && a >= ArithLessEqual
}

// IsArithAssignmentOp checks if a token assigns to the lefthand variable.
// E.g a += b, a <<= b
func TokenIsAssignmentOp(a Token) bool {
	return a <= ArithAssignAdd && a >= ArithAssignBinaryAnd
}

// TokenIs checks if the first supplied token is equal to any of the other
// supplied tokens.
func TokenIs(toks ...Token) bool {
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

var (
	InfixNudFunctions = map[Token]func(*Parser) int64{
		ArithAdd:      func(p *Parser) int64 { return p.expression(150) },
		ArithSubtract: func(p *Parser) int64 { return -p.expression(150) },
	}
	PrefixNudFunctions = map[Token]func(*Parser) int64{
		ArithBinaryNot: func(p *Parser) int64 { return -p.expression(LbpValues[ArithBinaryNot]) - 1 },
		ArithNot:       func(p *Parser) int64 { return BoolToShell(p.expression(LbpValues[ArithNot]) != ShellTrue) },
		ArithLeftParen: func(p *Parser) int64 {
			e := p.expression(0)
			p.consume(ArithRightParen)
			return e
		},
	}
	InfixLedFunctions = map[Token]func(int64, int64) int64{
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
	InfixRightLedFunctions = map[Token]func(int64, int64) int64{
		ArithAnd: func(l, r int64) int64 { return BoolToShell((l == ShellTrue) && (r == ShellTrue)) },
		ArithOr:  func(l, r int64) int64 { return BoolToShell((l == ShellTrue) || (r == ShellTrue)) },
	}
	LbpValues = map[Token]int{
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

type EOFNode struct{}

func (n EOFNode) nud(*Parser) int64        { panic("Nud called on EOFNode") }
func (n EOFNode) led(int64, *Parser) int64 { panic("Led called on EOFNode") }
func (n EOFNode) lbp() int                 { return -1 }

type NoopNode struct {
	Tok Token
}

func (n NoopNode) nud(*Parser) int64        { panic("Nud called on NoopNode: " + n.Tok.String()) }
func (n NoopNode) led(int64, *Parser) int64 { panic("Led called on NoopNode: " + n.Tok.String()) }
func (n NoopNode) lbp() int                 { return 0 }

type LiteralNode struct {
	Val int64
}

func (n LiteralNode) nud(*Parser) int64        { return n.Val }
func (n LiteralNode) led(int64, *Parser) int64 { panic("Led called on LiteralNode") }
func (n LiteralNode) lbp() int                 { return 0 }

type VariableNode struct {
	Val string
}

func (n VariableNode) nud(p *Parser) int64      { return p.getVariable(n.Val) }
func (n VariableNode) led(int64, *Parser) int64 { panic("Led called on VariableNode") }
func (n VariableNode) lbp() int                 { return 0 }

type InfixAssignNode struct {
	Tok Token
	Val ArithNode
}

func (n InfixAssignNode) nud(*Parser) int64 {
	panic("Nud called on InfixAssignNode: " + n.Tok.String())
}
func (n InfixAssignNode) led(left int64, p *Parser) int64 {
	v, ok := n.Val.(VariableNode)
	if !ok {
		// TODO: Remove panic / panic with recognizable Err so we can
		// catch it in recover
		panic("LHS of assignment '" + n.Tok.String() + "' is not a variable")
	}

	var fn func(int64, int64) int64
	if n.Tok == ArithAssignment {
		fn = InfixLedFunctions[ArithAssignment]
	} else {
		fn, ok = InfixLedFunctions[n.Tok-ArithAssignDiff]
		if !ok {
			panic("No Led function for InfixAssignNode: " + n.Tok.String())
		}
	}

	right := p.expression(0)
	t := fn(left, right)
	p.setVariable(v.Val, t)
	return t
}
func (n InfixAssignNode) lbp() int {
	if n.Tok == ArithAssignment {
		return LbpValues[n.Tok]
	}
	return LbpValues[n.Tok-ArithAssignDiff]
}

type InfixNode struct {
	Tok Token
}

func (n InfixNode) nud(p *Parser) int64 {
	fn, ok := InfixNudFunctions[n.Tok]
	if !ok {
		panic("No Nud function for InfixNode: " + n.Tok.String())
	}
	return fn(p)
}
func (n InfixNode) led(left int64, p *Parser) int64 {
	right := p.expression(n.lbp())
	fn, ok := InfixLedFunctions[n.Tok]
	if !ok {
		panic("No Led function for InfixNode: " + n.Tok.String())
	}
	return fn(left, right)
}
func (n InfixNode) lbp() int { return LbpValues[n.Tok] }

type InfixRightNode struct {
	Tok Token
}

func (n InfixRightNode) nud(*Parser) int64 {
	panic("Nud called on InfixRightNode: " + n.Tok.String())
}
func (n InfixRightNode) led(left int64, p *Parser) int64 {
	right := p.expression(n.lbp() - 1)
	fn, ok := InfixRightLedFunctions[n.Tok]
	if !ok {
		panic("No Led function for InfixRightNode: " + n.Tok.String())
	}
	return fn(left, right)
}
func (n InfixRightNode) lbp() int { return LbpValues[n.Tok] }

type PrefixNode struct {
	Tok Token
}

func (n PrefixNode) nud(p *Parser) int64 {
	fn, ok := PrefixNudFunctions[n.Tok]
	if !ok {
		panic("No Nud function for PrefixNode: " + n.Tok.String())
	}
	return fn(p)
}

func (n PrefixNode) led(int64, *Parser) int64 {
	panic("Led called on PrefixNode: " + n.Tok.String())
}
func (n PrefixNode) lbp() int { return LbpValues[n.Tok] }

type TernaryNode struct {
	condition int64
}

func (n TernaryNode) nud(*Parser) int64 { panic("Nud called on TernaryNode") }
func (n TernaryNode) led(left int64, p *Parser) int64 {
	/* Somewhat confusingly the shell's ternary operator does not work using
	   the shell's True/False semantics.
	   The actual operation is, given (a ? b : c)
	   if (a != 0)
	      return b
	   else
	      return c
	   See the ISO C Standard Section 6.5.15
	*/

	n.condition = left

	p.blockAssignments = true
	// We capture the lexer positions and lastNode before each expression so we can
	// rewind, get the correct value and return to the end of the ternary
	pos1 := p.lexer.pos
	ln1 := p.lastNode
	p.expression(0)

	p.consume(ArithColon)

	pos2 := p.lexer.pos
	ln2 := p.lastNode
	p.expression(0)
	pos3 := p.lexer.pos
	ln3 := p.lastNode

	var returnVal int64
	p.blockAssignments = false
	if n.condition != 0 {
		p.lexer.pos = pos1
		p.lastNode = ln1
		returnVal = p.expression(0)
	} else {
		p.lexer.pos = pos2
		p.lastNode = ln2
		returnVal = p.expression(0)
	}
	p.lexer.pos = pos3
	p.lastNode = ln3

	return returnVal
}
func (n TernaryNode) lbp() int {
	return 20
}
