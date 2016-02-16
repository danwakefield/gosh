//go:generate stringer -type=ArithToken
package main

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

type ArithToken int

// ArithLexem contains an ArithToken and a interface value.
// If ArithLexem.T == ArithNumber then ArithLexem.Val will be an int64
// If ArithLexem.T == ArithVariable then ArithLexem.Val will be a string
//
// In the future it may be possible that
// If ArithLexem.T == ArithError then ArithLexem.Val will be an error
//
// In all other cases ArithLexem.T should be nil
type ArithLexem struct {
	T   ArithToken
	Val interface{}
}

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

	ArithBinaryAnd
	ArithBinaryOr
	ArithBinaryXor
	ArithLeftShift
	ArithRightShift
	ArithRemainder
	ArithMultiply
	ArithAdd
	ArithSubtract
	ArithDivide

	// These tokens perform assignment to a variable as well as an
	// operation (E.g  x+=1)
	ArithAssignBinaryAnd
	ArithAssignBinaryOr
	ArithAssignBinaryXor
	ArithAssignLeftShift
	ArithAssignRightShift
	ArithAssignRemainder
	ArithAssignMultiply
	ArithAssignAdd
	ArithAssignSubtract
	ArithAssignDivide

	ArithLeftParen
	ArithRightParen
	ArithBinaryNot
	ArithQuestionMark
	ArithColon

	ArithEOF

	// ArithAssignDiff is used to turn an Arith token into its ArithAssign
	// equivalent by adding to it
	ArithAssignDiff ArithToken = ArithAssignBinaryAnd - ArithBinaryAnd
)

// IsArithBinaryOp checks if a token operates on two values.
// E.g a + b, a << b
func IsArithBinaryOp(a ArithToken) bool {
	return a <= ArithDivide && a >= ArithLessEqual
}

// IsArithAssignmentOp checks if a token assigns to the lefthand variable.
// E.g a += b, a <<= b
func IsArithAssignmentOp(a ArithToken) bool {
	return a <= ArithAssignDivide && a >= ArithAssignBinaryAnd
}

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
func (al *ArithLexer) hasNextFunc(f func(rune) bool) bool {
	if f(al.next()) {
		return true
	}
	al.backup()
	return false
}

// Lex returns an ArithLexem containing the next ArithToken in the input string.
// The ArithLexem will also contain a value dependant on the ArithToken
// If ArithLexem.T == ArithNumber then ArithLexem.Val will be an int64
// If ArithLexem.T == ArithVariable then ArithLexem.Val will be a string
//
// In the future it may be possible that
// If ArithLexem.T == ArithError then ArithLexem.Val will be an error
func (al *ArithLexer) Lex() ArithLexem {
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
		return ArithLexem{T: ArithEOF}
	}

	// Special case for Hex (0xff) and Octal (0777) constants
	if c == '0' {
		// Hex constants
		if al.hasNext("Xx") {
			startPos = al.pos
			endPos = startPos
			for {
				//Find the end of the constant
				if al.hasNextFunc(IsHexDigit) {
					endPos++
				} else {
					break
				}
			}
			parsedVal, err := strconv.ParseInt(al.input[startPos:endPos], 16, 64)
			if err != nil {
				panic("Not Reached: Broken Hex Constant")
			}
			return ArithLexem{T: ArithNumber, Val: parsedVal}
		}
		// Octal constants
		if al.hasNextFunc(IsOctalDigit) {
			startPos = al.pos - al.lastRuneWidth
			endPos = al.pos
			for {
				if al.hasNextFunc(IsOctalDigit) {
					endPos++
				} else {
					break
				}
			}
			parsedVal, err := strconv.ParseInt(al.input[startPos:endPos], 8, 64)
			if err != nil {
				panic("Not Reached: Broken Octal Constant")
			}
			return ArithLexem{T: ArithNumber, Val: parsedVal}
		}

		// Nothing following the 0 means it just reprsents 0
		return ArithLexem{T: ArithNumber, Val: int64(0)}
	}

	// Finds decimal constants.
	if IsDigit(c) {
		startPos = al.pos - al.lastRuneWidth
		endPos = al.pos
		for {
			if al.hasNextFunc(IsDigit) {
				endPos++
			} else {
				break
			}
		}
		parsedVal, err := strconv.ParseInt(al.input[startPos:endPos], 10, 64)
		if err != nil {
			panic("Not Reached: Broken Decimal Constant")
		}
		return ArithLexem{T: ArithNumber, Val: parsedVal}
	}

	// Finds variable names.
	if IsFirstInName(c) {
		startPos = al.pos - al.lastRuneWidth
		endPos = al.pos
		for {
			if al.hasNextFunc(IsInName) {
				endPos++
			} else {
				break
			}
		}
		return ArithLexem{T: ArithVariable, Val: al.input[startPos:endPos]}
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

	return ArithLexem{T: t}
}
