package main

import (
	"fmt"
	"reflect"
	"testing"
)

func TestIsArithBinaryOperation(t *testing.T) {
	cases := []struct {
		in   ArithToken
		want bool
	}{
		{ArithLessEqual, true},
		{ArithDivide, true},
		{ArithEqual, true},
		{ArithVariable, false},
		{ArithAssignBinaryAnd, false},
	}

	for _, c := range cases {
		got := IsArithBinaryOp(c.in)
		if got != c.want {
			t.Errorf(
				"%s Should be %v not %v",
				c.in, c.want, got,
			)
		}
	}
}

func TestIsArithAssignmentOperation(t *testing.T) {
	cases := []struct {
		in   ArithToken
		want bool
	}{
		{ArithAssignBinaryAnd, true},
		{ArithAssignDivide, true},
		{ArithAssignLeftShift, true},
		{ArithDivide, false},
		{ArithLeftParen, false},
	}

	for _, c := range cases {
		got := IsArithAssignmentOp(c.in)
		if got != c.want {
			t.Errorf("%s Should be %v not %v", c.in, c.want, got)
		}
	}
}

func TestArithTokenString(t *testing.T) {
	cases := []struct {
		in   ArithToken
		want string
	}{
		{ArithVariable, "ArithVariable"},
		{ArithDivide, "ArithDivide"},
		{ArithAssignLeftShift, "ArithAssignLeftShift"},
		{ArithLeftParen, "ArithLeftParen"},
	}

	for _, c := range cases {
		got := fmt.Sprintf("%s", c.in)
		if got != c.want {
			t.Errorf("Token should stringify to %s not %s", c.want, got)
		}
	}

}

func TestArithTokenAssignDiff(t *testing.T) {
	cases := []struct {
		in   ArithToken
		want ArithToken
	}{
		{ArithBinaryAnd, ArithAssignBinaryAnd},
		{ArithAdd, ArithAssignAdd},
		{ArithDivide, ArithAssignDivide},
	}

	for _, c := range cases {
		got := c.in + ArithAssignDiff
		if got != c.want {
			t.Errorf("%s should be %s not %s", c.in, c.want, got)
		}
	}
}

func TestArithLexer(t *testing.T) {
	cases := []struct {
		in   string
		want ArithLexem
	}{
		{"_abcd", ArithLexem{T: ArithVariable, Val: "_abcd"}},
		{"0", ArithLexem{T: ArithNumber, Val: int64(0)}},
		{"5", ArithLexem{T: ArithNumber, Val: int64(5)}},
		{"0xff", ArithLexem{T: ArithNumber, Val: int64(255)}},
		{"0xfii", ArithLexem{T: ArithNumber, Val: int64(15)}},
		{"077", ArithLexem{T: ArithNumber, Val: int64(63)}},
		{"", ArithLexem{}},
		{"   \n\t  ", ArithLexem{}},
		{">", ArithLexem{T: ArithGreaterThan}},
		{">=", ArithLexem{T: ArithGreaterEqual}},
		{">>", ArithLexem{T: ArithRightShift}},
		{">>=", ArithLexem{T: ArithAssignRightShift}},
		{"<", ArithLexem{T: ArithLessThan}},
		{"<=", ArithLexem{T: ArithLessEqual}},
		{"<<", ArithLexem{T: ArithLeftShift}},
		{"<<=", ArithLexem{T: ArithAssignLeftShift}},
		{"|", ArithLexem{T: ArithBinaryOr}},
		{"|=", ArithLexem{T: ArithAssignBinaryOr}},
		{"||", ArithLexem{T: ArithOr}},
		{"&", ArithLexem{T: ArithBinaryAnd}},
		{"&=", ArithLexem{T: ArithAssignBinaryAnd}},
		{"&&", ArithLexem{T: ArithAnd}},
		{"*", ArithLexem{T: ArithMultiply}},
		{"*=", ArithLexem{T: ArithAssignMultiply}},
		{"/", ArithLexem{T: ArithDivide}},
		{"/=", ArithLexem{T: ArithAssignDivide}},
		{"%", ArithLexem{T: ArithRemainder}},
		{"%=", ArithLexem{T: ArithAssignRemainder}},
		{"+", ArithLexem{T: ArithAdd}},
		{"+=", ArithLexem{T: ArithAssignAdd}},
		{"-", ArithLexem{T: ArithSubtract}},
		{"-=", ArithLexem{T: ArithAssignSubtract}},
		{"^", ArithLexem{T: ArithBinaryXor}},
		{"^=", ArithLexem{T: ArithAssignBinaryXor}},
		{"!", ArithLexem{T: ArithNot}},
		{"!=", ArithLexem{T: ArithNotEqual}},
		{"=", ArithLexem{T: ArithAssignment}},
		{"==", ArithLexem{T: ArithEqual}},
		{"(", ArithLexem{T: ArithLeftParen}},
		{")", ArithLexem{T: ArithRightParen}},
		{"~", ArithLexem{T: ArithBinaryNot}},
		{"?", ArithLexem{T: ArithQuestionMark}},
		{":", ArithLexem{T: ArithColon}},
	}

	for _, c := range cases {
		y := NewArithLexer(c.in)
		got := y.Lex()
		//fmt.Printf("%#v\n%#v\n", got.Val, c.want.Val)
		if !reflect.DeepEqual(c.want, got) {
			t.Errorf("'%s' should produce\n%#v\n not\n%#v", c.in, c.want, got)
		}
	}
}

func TestArithLexerComplex(t *testing.T) {
	type complexTestCase struct {
		in   string
		want []ArithLexem
	}
	// TC creates a new testcase
	TC := func(i string, lexems ...ArithLexem) complexTestCase {
		ctc := complexTestCase{in: i}
		ctc.want = []ArithLexem{}
		ctc.want = append(ctc.want, lexems...)
		// Append the EOF lexem
		ctc.want = append(ctc.want, ArithLexem{})
		return ctc
	}

	cases := []complexTestCase{
		TC(
			"5 >= 4",
			ArithLexem{T: ArithNumber, Val: int64(5)},
			ArithLexem{T: ArithGreaterEqual},
			ArithLexem{T: ArithNumber, Val: int64(4)},
		),
		TC(
			">>= <<= 0xff 077 5 ==",
			ArithLexem{T: ArithAssignRightShift},
			ArithLexem{T: ArithAssignLeftShift},
			ArithLexem{T: ArithNumber, Val: int64(255)},
			ArithLexem{T: ArithNumber, Val: int64(63)},
			ArithLexem{T: ArithNumber, Val: int64(5)},
			ArithLexem{T: ArithEqual},
		),
	}

	for _, c := range cases {
		y := NewArithLexer(c.in)
		for lexemCount, lexem := range c.want {
			got := y.Lex()
			//fmt.Printf("%#v\n%#v\n", got.Val, c.want.Val)
			if !reflect.DeepEqual(lexem, got) {
				t.Errorf("'%s' should produce\n%#v\n as the %d lexem not\n%#v", c.in, lexem, lexemCount, got)
			}
		}
	}
}
