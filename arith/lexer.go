//go:generate stringer -type=Token
package arith

import (
	"errors"
	"strconv"
	"unicode/utf8"

	"github.com/danwakefield/gosh/char"
)

var (
	ErrHexConstant     = errors.New("Invalid Hex Constant")
	ErrOctalConstant   = errors.New("Invalid Octal Constant")
	ErrDecimalConstant = errors.New("Invalid Decimal Constant")
)

type LexError struct {
	X   string
	Err error
}

func (e LexError) Error() string {
	return "Error parsing '" + e.X + "' :" + e.Err.Error()
}

// Lexer ...
type Lexer struct {
	input         string
	pos           int
	inputLen      int
	lastRuneWidth int
}

func NewLexer(s string) *Lexer {
	return &Lexer{
		input:    s,
		inputLen: len(s),
	}
}

// next returns the next available rune from the input string.
func (l *Lexer) next() rune {
	if l.pos >= l.inputLen {
		l.lastRuneWidth = 0
		return EOFRune
	}
	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.lastRuneWidth = w
	l.pos += w
	return r
}

// backup reverses a call to next idempotently
func (l *Lexer) backup() {
	l.pos -= l.lastRuneWidth
	l.lastRuneWidth = 0
}

// peek returns the next rune from the input
// state of the lexer is preserved
func (l *Lexer) peek() rune {
	lrw := l.lastRuneWidth
	r := l.next()
	l.backup()
	l.lastRuneWidth = lrw
	return r
}

func (l *Lexer) hasNext(r rune) bool {
	if r == l.next() {
		return true
	}
	l.backup()
	return false
}

// hasNextFunc uses the supplied func to check the validity of the next
// character from the input
func (l *Lexer) hasNextFunc(fn func(rune) bool) bool {
	if fn(l.next()) {
		return true
	}
	l.backup()
	return false
}

// Lex returns the next Token in the input string and an interface value.
// The interface will also contain a value dependant on the Token
// If Token == ArithNumber then interface will be an int64
// If Token == ArithVariable then interface will be a string
// If Token == ArithError then interface will be an error
func (l *Lexer) Lex() (Token, interface{}) {
	var t Token
	var checkAssignmentOp bool
	var startPos, endPos int

	c := l.next()

	// Ignore whitespace
	for {
		if c == ' ' || c == '\n' || c == '\t' {
			c = l.next()
		} else {
			break
		}
	}

	if c == EOFRune {
		return ArithEOF, nil
	}

	if char.IsDigit(c) {
		return lexDigit(l, c)
	}

	// Finds variable names.
	if char.IsFirstInVarName(c) {
		startPos = l.pos - l.lastRuneWidth
		endPos = l.pos
		for {
			if l.hasNextFunc(char.IsInVarName) {
				endPos++
			} else {
				break
			}
		}
		return ArithVariable, l.input[startPos:endPos]
	}

	switch c {
	case '>':
		switch l.next() {
		case '>':
			t = ArithRightShift
			checkAssignmentOp = true
		case '=':
			t = ArithGreaterEqual
		default:
			t = ArithGreaterThan
			l.backup()
		}
	case '<':
		switch l.next() {
		case '<':
			t = ArithLeftShift
			checkAssignmentOp = true
		case '=':
			t = ArithLessEqual
		default:
			t = ArithLessThan
			l.backup()
		}
	case '|':
		if l.hasNext('|') {
			t = ArithOr
		} else {
			t = ArithBinaryOr
			checkAssignmentOp = true
		}
	case '&':
		if l.hasNext('&') {
			t = ArithAnd
		} else {
			t = ArithBinaryAnd
			checkAssignmentOp = true
		}
	case '*':
		t = ArithMultiply
		checkAssignmentOp = true
	case '/':
		t = ArithDivide
		checkAssignmentOp = true
	case '%':
		t = ArithRemainder
		checkAssignmentOp = true
	case '+':
		t = ArithAdd
		checkAssignmentOp = true
	case '-':
		t = ArithSubtract
		checkAssignmentOp = true
	case '^':
		t = ArithBinaryXor
		checkAssignmentOp = true
	case '!':
		if l.hasNext('=') {
			t = ArithNotEqual
		} else {
			t = ArithNot
		}
	case '=':
		if l.hasNext('=') {
			t = ArithEqual
		} else {
			t = ArithAssignment
		}
	case '(':
		t = ArithLeftParen
	case ')':
		t = ArithRightParen
	case '~':
		t = ArithBinaryNot
	case '?':
		t = ArithQuestionMark
	case ':':
		t = ArithColon
	default:
		t = ArithError
	}

	if checkAssignmentOp {
		if l.hasNext('=') {
			t += ArithAssignDiff
		}
	}

	return t, nil
}

func lexDigit(l *Lexer, c rune) (Token, interface{}) {
	if c == '0' { // Special case for Hex (0xff) and Octal (0777) constants
		if l.hasNext('x') || l.hasNext('X') {
			return lexHexConstant(l)
		} else if l.hasNextFunc(char.IsOctalDigit) {
			return lexOctalConstant(l)
		}
		// Simple Zero constant
		return ArithNumber, int64(0)
	}
	startPos := l.pos - l.lastRuneWidth
	endPos := l.pos
	for {
		if l.hasNextFunc(char.IsDigit) {
			endPos++
		} else {
			if char.IsFirstInVarName(l.peek()) {
				return ArithError, LexError{
					X:   l.input[startPos : endPos+1],
					Err: ErrDecimalConstant,
				}
			}
			break
		}
	}
	parsedVal, err := strconv.ParseInt(l.input[startPos:endPos], 10, 64)
	if err != nil {
		panic("Not Reached: Broken Decimal Constant")
	}
	return ArithNumber, parsedVal
}

func lexHexConstant(l *Lexer) (Token, interface{}) {
	startPos := l.pos
	endPos := l.pos
	for {
		//Find the end of the constant
		if l.hasNextFunc(char.IsHexDigit) {
			endPos++
		} else {
			//Check if the number is invalid.
			//We already know the next rune is not a hex digit
			if char.IsInVarName(l.peek()) {
				return ArithError, LexError{
					X:   l.input[startPos-2 : endPos+1],
					Err: ErrHexConstant,
				}
			}
			break
		}
	}
	parsedVal, err := strconv.ParseInt(l.input[startPos:endPos], 16, 64)
	if err != nil {
		panic("Not Reached: Broken Hex Constant")
	}
	return ArithNumber, parsedVal
}

func lexOctalConstant(l *Lexer) (Token, interface{}) {
	startPos := l.pos - l.lastRuneWidth
	endPos := l.pos
	for {
		if l.hasNextFunc(char.IsOctalDigit) {
			endPos++
		} else {
			if char.IsInVarName(l.peek()) {
				return ArithError, LexError{
					X:   l.input[startPos-1 : endPos+1],
					Err: ErrOctalConstant,
				}
			}
			break
		}
	}
	parsedVal, err := strconv.ParseInt(l.input[startPos:endPos], 8, 64)
	if err != nil {
		panic("Not Reached: Broken Octal Constant")
	}
	return ArithNumber, parsedVal
}
