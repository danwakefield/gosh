//go:generate stringer -type=ArithToken
package main

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

type ArithToken int

type ArithLexem struct {
	T   ArithToken
	Val interface{}
}

const (
	// DigitRuneOffset can be subtracted from a rune from 0-9
	// to get it as an integer value. Saves conversion to a string
	// then a call to Atoi
	DigitRuneOffset = 48
	EOFRune         = -1

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

	// Used to turn an Arith token into its ArithAssign equivalent
	// by adding to it
	ArithAssignDiff ArithToken = ArithAssignBinaryAnd - ArithBinaryAnd
)

var ArithPrecedence = map[ArithToken]int{
	ArithMultiply:     0,
	ArithDivide:       0,
	ArithRemainder:    0,
	ArithAdd:          1,
	ArithSubtract:     1,
	ArithLeftShift:    2,
	ArithRightShift:   2,
	ArithLessThan:     3,
	ArithLessEqual:    3,
	ArithGreaterThan:  3,
	ArithGreaterEqual: 3,
	ArithEqual:        4,
	ArithNotEqual:     4,
	ArithBinaryAnd:    5,
	ArithBinaryXor:    6,
	ArithBinaryOr:     7,
}

func IsArithBinaryOp(a ArithToken) bool {
	return a <= ArithDivide && a >= ArithLessEqual
}

func IsArithAssignmentOp(a ArithToken) bool {
	return a <= ArithAssignDivide && a >= ArithAssignBinaryAnd
}

// Arith expects a string with all variable expansions performed
// Hexadecimal and octal expansion is done here
func Arith(s string) int64 {
	return 0
}

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

func (al *ArithLexer) Lex() ArithLexem {
	var t ArithToken
	var checkAssignmentOp bool
	var startPos, endPos int

	c := al.Next()

	// Ignore a run of whitespace
	for {
		if c == ' ' || c == '\n' || c == '\t' {
			c = al.Next()
		} else {
			break
		}
	}

	if c == EOFRune {
		return ArithLexem{}
	}

	// Special case for Hex (0xff) and Octal (0777) constants
	if c == '0' {
		if al.HasNext("Xx") {
			startPos = al.pos
			endPos = startPos
			for {
				if al.HasNextFunc(IsHexDigit) {
					endPos++
				} else {
					// Handle the case of invalid hex
					// e.g 0xfii should return an error
					// I think any letter should, anything
					// else could be valid
					break
				}
			}
			hexVal, err := strconv.ParseInt(al.input[startPos:endPos], 16, 64)
			if err != nil {
				panic("Not Reached: Broken Hex Constant")
			}
			return ArithLexem{T: ArithNumber, Val: hexVal}
		}
		if al.HasNextFunc(IsOctalDigit) {
			startPos = al.pos - al.lastRuneWidth
			endPos = al.pos
			for {
				if al.HasNextFunc(IsOctalDigit) {
					endPos++
				} else {
					// Handle invalid constants?
					// e.g 0778
					break
				}
			}
			octVal, err := strconv.ParseInt(al.input[startPos:endPos], 8, 64)
			if err != nil {
				panic("Not Reached: Broken Octal Constant")
			}
			return ArithLexem{T: ArithNumber, Val: octVal}
		}
		return ArithLexem{T: ArithNumber, Val: int64(0)}
	}

	if IsDigit(c) {
		// Take whole number? just like the parsed constants?
		return ArithLexem{T: ArithNumber, Val: int64(c - DigitRuneOffset)}
	}

	if IsFirstInName(c) {
		startPos = al.pos - al.lastRuneWidth
		for {
			if IsInName(al.Next()) {
				endPos = al.pos
			} else {
				al.Backup()
				break
			}
		}
		return ArithLexem{T: ArithVariable, Val: al.input[startPos:endPos]}
	}

	switch c {
	case '>':
		switch al.Next() {
		case '>':
			t = ArithRightShift
			checkAssignmentOp = true
		case '=':
			t = ArithGreaterEqual
		default:
			t = ArithGreaterThan
			al.Backup()
		}
	case '<':
		switch al.Next() {
		case '<':
			t = ArithLeftShift
			checkAssignmentOp = true
		case '=':
			t = ArithLessEqual
		default:
			t = ArithLessThan
			al.Backup()
		}
	case '|':
		if al.HasNext("|") {
			t = ArithOr
		} else {
			t = ArithBinaryOr
			checkAssignmentOp = true
		}
	case '&':
		if al.HasNext("&") {
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
		if al.HasNext("=") {
			t = ArithNotEqual
		} else {
			t = ArithNot
		}
	case '=':
		if al.HasNext("=") {
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
		if al.HasNext("=") {
			t += ArithAssignDiff
		}
	}

	return ArithLexem{T: t}
}

func (al *ArithLexer) Next() rune {
	if al.pos >= al.inputLength {
		al.lastRuneWidth = 0
		return EOFRune
	}
	r, w := utf8.DecodeRuneInString(al.input[al.pos:])
	al.lastRuneWidth = w
	al.pos += w
	return r
}

func (al *ArithLexer) HasNext(s string) bool {
	if strings.IndexRune(s, al.Next()) >= 0 {
		return true
	}
	al.Backup()
	return false
}

func (al *ArithLexer) HasNextFunc(f func(rune) bool) bool {
	if f(al.Next()) {
		return true
	}
	al.Backup()
	return false
}

func (al *ArithLexer) Backup() {
	al.pos -= al.lastRuneWidth
	al.lastRuneWidth = 0
}
