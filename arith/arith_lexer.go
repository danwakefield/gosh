//go:generate stringer -type=ArithToken
package arith

import (
	"errors"
	"strconv"
	"strings"
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

func (le LexError) Error() string {
	return "Error parsing '" + le.X + "' :" + le.Err.Error()
}

type ArithToken int

const (
	EOFRune = -1

	ArithError ArithToken = iota
	ArithAssignment
	ArithNot
	ArithAnd
	ArithOr
	ArithNumber
	ArithVariable

	// These tokens are all binary operations requiring two arguments
	// (E.g 1+2)
	ArithLessEqual
	ArithGreaterEqual
	ArithLessThan
	ArithGreaterThan
	ArithEqual
	ArithNotEqual

	// These binary operations also have assignment equivalents
	ArithBinaryAnd
	ArithBinaryOr
	ArithBinaryXor
	ArithLeftShift
	ArithRightShift
	ArithRemainder
	ArithMultiply
	ArithDivide
	ArithSubtract
	ArithAdd

	// These tokens perform assignment to a variable as well as an
	// operation (E.g  x+=1)
	ArithAssignBinaryAnd
	ArithAssignBinaryOr
	ArithAssignBinaryXor
	ArithAssignLeftShift
	ArithAssignRightShift
	ArithAssignRemainder
	ArithAssignMultiply
	ArithAssignDivide
	ArithAssignSubtract
	ArithAssignAdd

	ArithLeftParen
	ArithRightParen
	ArithBinaryNot
	ArithQuestionMark
	ArithColon

	ArithEOF

	// ArithAssignDiff is used to turn an Arith token into its ArithAssign equivalent.
	ArithAssignDiff ArithToken = ArithAssignBinaryAnd - ArithBinaryAnd
)

// ArithLexer ...
type ArithLexer struct {
	input         string
	pos           int
	inputLength   int
	lastRuneWidth int
}

func NewArithLexer(s string) *ArithLexer {
	return &ArithLexer{
		input:       s,
		inputLength: len(s),
	}
}

// next returns the next available rune from the input string.
// returns EOFRune
func (al *ArithLexer) next() rune {
	if al.pos >= al.inputLength {
		al.lastRuneWidth = 0
		return EOFRune
	}
	r, w := utf8.DecodeRuneInString(al.input[al.pos:])
	al.lastRuneWidth = w
	al.pos += w
	return r
}

// backup undoes a call to next.
// Only works once per invocation of call, multiple calls are idempotent
func (al *ArithLexer) backup() {
	al.pos -= al.lastRuneWidth
	al.lastRuneWidth = 0
}

// peek returns the next rune from the input
// state of the lexer is preserved
func (al *ArithLexer) peek() rune {
	lrw := al.lastRuneWidth
	r := al.next()
	al.backup()
	al.lastRuneWidth = lrw
	return r
}

// hasNext checks that the next character of the input is one of the
// characters in the string s
func (al *ArithLexer) hasNext(s string) bool {
	if strings.IndexRune(s, al.next()) >= 0 {
		return true
	}
	al.backup()
	return false
}

// hasNextFunc uses the supplied func to check the validity of the next
// character from the input
func (al *ArithLexer) hasNextFunc(fn func(rune) bool) bool {
	if fn(al.next()) {
		return true
	}
	al.backup()
	return false
}

// Lex returns the next ArithToken in the input string and an interface value.
// The interface will also contain a value dependant on the ArithToken
// If ArithToken == ArithNumber then interface will be an int64
// If ArithToken == ArithVariable then interface will be a string
//
// In the future it may be possible that
// If ArithToken == ArithError then interface will be an error
func (al *ArithLexer) Lex() (ArithToken, interface{}) {
	var t ArithToken
	var checkAssignmentOp bool
	var startPos, endPos int

	c := al.next()

	// Ignore whitespace
	for {
		if c == ' ' || c == '\n' || c == '\t' {
			c = al.next()
		} else {
			break
		}
	}

	if c == EOFRune {
		return ArithEOF, nil
	}

	// Finds Numeric constants.
	if char.IsDigit(c) {
		// Special case for Hex (0xff) and Octal (0777) constants
		if c == '0' {
			// Hex constants
			if al.hasNext("Xx") {
				startPos = al.pos
				endPos = al.pos
				for {
					//Find the end of the constant
					if al.hasNextFunc(char.IsHexDigit) {
						endPos++
					} else {
						//Check if the number is invalid.
						//We already know the next rune is not a hex digit
						if char.IsInVarName(al.peek()) {
							return ArithError, LexError{
								X:   al.input[startPos-2 : endPos+1],
								Err: ErrHexConstant,
							}
						}
						break
					}
				}
				parsedVal, err := strconv.ParseInt(al.input[startPos:endPos], 16, 64)
				if err != nil {
					panic("Not Reached: Broken Hex Constant")
				}
				return ArithNumber, parsedVal
			}
			// Octal constants
			if al.hasNextFunc(char.IsOctalDigit) {
				startPos = al.pos - al.lastRuneWidth
				endPos = al.pos
				for {
					if al.hasNextFunc(char.IsOctalDigit) {
						endPos++
					} else {
						if char.IsInVarName(al.peek()) {
							return ArithError, LexError{
								X:   al.input[startPos-1 : endPos+1],
								Err: ErrOctalConstant,
							}
						}
						break
					}
				}
				parsedVal, err := strconv.ParseInt(al.input[startPos:endPos], 8, 64)
				if err != nil {
					panic("Not Reached: Broken Octal Constant")
				}
				return ArithNumber, parsedVal
			}

			// Simple Zero constant
			return ArithNumber, int64(0)
		}
		startPos = al.pos - al.lastRuneWidth
		endPos = al.pos
		for {
			if al.hasNextFunc(char.IsDigit) {
				endPos++
			} else {
				if char.IsFirstInVarName(al.peek()) {
					return ArithError, LexError{
						X:   al.input[startPos : endPos+1],
						Err: ErrDecimalConstant,
					}
				}
				break
			}
		}
		parsedVal, err := strconv.ParseInt(al.input[startPos:endPos], 10, 64)
		if err != nil {
			panic("Not Reached: Broken Decimal Constant")
		}
		return ArithNumber, parsedVal
	}

	// Finds variable names.
	if char.IsFirstInVarName(c) {
		startPos = al.pos - al.lastRuneWidth
		endPos = al.pos
		for {
			if al.hasNextFunc(char.IsInVarName) {
				endPos++
			} else {
				break
			}
		}
		return ArithVariable, al.input[startPos:endPos]
	}

	switch c {
	case '>':
		switch al.next() {
		case '>':
			t = ArithRightShift
			checkAssignmentOp = true
		case '=':
			t = ArithGreaterEqual
		default:
			t = ArithGreaterThan
			al.backup()
		}
	case '<':
		switch al.next() {
		case '<':
			t = ArithLeftShift
			checkAssignmentOp = true
		case '=':
			t = ArithLessEqual
		default:
			t = ArithLessThan
			al.backup()
		}
	case '|':
		if al.hasNext("|") {
			t = ArithOr
		} else {
			t = ArithBinaryOr
			checkAssignmentOp = true
		}
	case '&':
		if al.hasNext("&") {
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
		if al.hasNext("=") {
			t = ArithNotEqual
		} else {
			t = ArithNot
		}
	case '=':
		if al.hasNext("=") {
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
		if al.hasNext("=") {
			t += ArithAssignDiff
		}
	}

	return t, nil
}
