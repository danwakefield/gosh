package main

import "fmt"

type ArithNode interface {
	Nud() int64
	Led(int64) int64
	Lbp() int
}

var parser *ArithParser

type ArithParser struct {
	lastNode ArithNode
	y        *ArithLexer
}

func Parse(s string) int64 {
	ap := &ArithParser{y: NewArithLexer(s)}
	ap.Next()
	parser = ap
	return parser.Expression(0)
}

func (ap *ArithParser) Expression(rbp int) int64 {
	n := ap.lastNode
	ap.Next()
	left := n.Nud()
	for rbp < ap.lastNode.Lbp() {
		n = ap.lastNode
		ap.Next()
		left = n.Led(left)
	}
	return left
}

func (ap *ArithParser) Consume(t ArithToken) {
	// TODO(Fix)
	ap.Next()
}

func (ap *ArithParser) Next() {
	l := ap.y.Lex()
	switch {
	case l.T == ArithNumber:
		ap.lastNode = LiteralNode{Val: l.Val.(int64)}
	case IsArithBinaryOp(l.T):
		ap.lastNode = InfixNode{T: l.T}
	case l.T == ArithBinaryNot || l.T == ArithNot || l.T == ArithLeftParen:
		ap.lastNode = PrefixNode{T: l.T}
	case l.T == ArithAdd || l.T == ArithOr:
		ap.lastNode = InfixRightNode{T: l.T}
	case l.T == ArithEOF:
		ap.lastNode = EOFNode{}
	default:
		fmt.Printf("%T - %#v\n%s - %T - %#v", l, l, l.T, l.T, l.T)
		panic("")
	}
}

type EOFNode struct{}

func (n EOFNode) Nud() int64      { return 0 }
func (n EOFNode) Led(int64) int64 { panic("Not Reached: EOFNode Led Call") }
func (n EOFNode) Lbp() int        { return -1 }

type LiteralNode struct {
	Val int64
}

func (n LiteralNode) Nud() int64      { return n.Val }
func (n LiteralNode) Led(int64) int64 { panic("Not Reached: LiteralNode Led Call") }
func (n LiteralNode) Lbp() int        { return 0 }

var (
	InfixNudFunctions = map[ArithToken]func(InfixNode) int64{
		ArithAdd:      func(InfixNode) int64 { return parser.Expression(130) },
		ArithSubtract: func(InfixNode) int64 { return -parser.Expression(130) },
	}
	PrefixNudFunctions = map[ArithToken]func(PrefixNode) int64{
		ArithBinaryNot: func(n PrefixNode) int64 { return -parser.Expression(n.Lbp()) - 1 },
		ArithNot:       func(n PrefixNode) int64 { return BoolToShell(parser.Expression(n.Lbp()) != ShellTrue) },
		ArithLeftParen: func(n PrefixNode) int64 {
			e := parser.Expression(0)
			parser.Consume(ArithRightParen)
			return e
		},
	}
	InfixLedFunctions = map[ArithToken]func(InfixNode) int64{
		ArithLessEqual:    func(n InfixNode) int64 { return BoolToShell(n.left <= n.right) },
		ArithGreaterEqual: func(n InfixNode) int64 { return BoolToShell(n.left >= n.right) },
		ArithLessThan:     func(n InfixNode) int64 { return BoolToShell(n.left < n.right) },
		ArithGreaterThan:  func(n InfixNode) int64 { return BoolToShell(n.left > n.right) },
		ArithEqual:        func(n InfixNode) int64 { return BoolToShell(n.left == n.right) },
		ArithNotEqual:     func(n InfixNode) int64 { return BoolToShell(n.left != n.right) },
		ArithBinaryAnd:    func(n InfixNode) int64 { return n.left & n.right },
		ArithBinaryOr:     func(n InfixNode) int64 { return n.left | n.right },
		ArithBinaryXor:    func(n InfixNode) int64 { return n.left ^ n.right },
		ArithLeftShift:    func(n InfixNode) int64 { return LeftShift(n.left, n.right) },
		ArithRightShift:   func(n InfixNode) int64 { return RightShift(n.left, n.right) },
		ArithRemainder:    func(n InfixNode) int64 { return n.left % n.right },
		ArithMultiply:     func(n InfixNode) int64 { return n.left * n.right },
		ArithDivide:       func(n InfixNode) int64 { return n.left / n.right },
		ArithSubtract:     func(n InfixNode) int64 { return n.left - n.right },
		ArithAdd:          func(n InfixNode) int64 { return n.left + n.right },
	}
	InfixRightLedFunctions = map[ArithToken]func(InfixRightNode) int64{
		ArithAnd: func(n InfixRightNode) int64 { return BoolToShell((n.left == ShellTrue) && (n.right == ShellTrue)) },
		ArithOr:  func(n InfixRightNode) int64 { return BoolToShell((n.left == ShellTrue) || (n.right == ShellTrue)) },
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

type InfixNode struct {
	T           ArithToken
	left, right int64
}

func (n InfixNode) Nud() int64 {
	f, ok := InfixNudFunctions[n.T]
	if !ok {
		panic("Nud function called on " + string(n.T))
	}
	return f(n)
}
func (n InfixNode) Led(left int64) int64 {
	n.left = left
	n.right = parser.Expression(n.Lbp())
	f, ok := InfixLedFunctions[n.T]
	if !ok {
		panic("Led function called on " + string(n.T))
	}
	return f(n)
}
func (n InfixNode) Lbp() int { return LbpValues[n.T] }

type InfixRightNode struct {
	T           ArithToken
	left, right int64
}

func (n InfixRightNode) Nud() int64 {
	panic("Nud function called on right associative " + string(n.T))
}
func (n InfixRightNode) Led(left int64) int64 {
	n.left = left
	n.right = parser.Expression(n.Lbp() - 1)
	f, ok := InfixRightLedFunctions[n.T]
	if !ok {
		panic("Led function called on right associative " + string(n.T))
	}
	return f(n)
}
func (n InfixRightNode) Lbp() int { return LbpValues[n.T] }

type PrefixNode struct {
	T ArithToken
}

func (n PrefixNode) Nud() int64 {
	f, ok := PrefixNudFunctions[n.T]
	if !ok {
		panic("Nud function called on prefix " + string(n.T))
	}
	return f(n)
}

func (n PrefixNode) Led(left int64) int64 { panic("Led function called on prefix " + string(n.T)) }
func (n PrefixNode) Lbp() int             { return LbpValues[n.T] }
