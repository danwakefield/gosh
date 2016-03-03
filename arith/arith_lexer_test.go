package arith

import (
	"fmt"
	"reflect"
	"testing"
)

func TestTokenIsBinaryOp(t *testing.T) {
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
		got := TokenIsBinaryOp(c.in)
		if got != c.want {
			t.Errorf(
				"%s Should be %v not %v",
				c.in, c.want, got,
			)
		}
	}
}

func TestTokenIsAssignmentOp(t *testing.T) {
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
		got := TokenIsAssignmentOp(c.in)
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
		in      string
		wantTok ArithToken
		wantVal interface{}
	}{
		{"_abcd", ArithVariable, "_abcd"},
		{"5", ArithNumber, int64(5)},
		{"555", ArithNumber, int64(555)},
		{"0", ArithNumber, int64(0)},
		{"0xff", ArithNumber, int64(255)},
		{"077", ArithNumber, int64(63)},
		{"", ArithEOF, nil},
		{"   \n\t  ", ArithEOF, nil},
		{">", ArithGreaterThan, nil},
		{">=", ArithGreaterEqual, nil},
		{">>", ArithRightShift, nil},
		{">>=", ArithAssignRightShift, nil},
		{"<", ArithLessThan, nil},
		{"<=", ArithLessEqual, nil},
		{"<<", ArithLeftShift, nil},
		{"<<=", ArithAssignLeftShift, nil},
		{"|", ArithBinaryOr, nil},
		{"|=", ArithAssignBinaryOr, nil},
		{"||", ArithOr, nil},
		{"&", ArithBinaryAnd, nil},
		{"&=", ArithAssignBinaryAnd, nil},
		{"&&", ArithAnd, nil},
		{"*", ArithMultiply, nil},
		{"*=", ArithAssignMultiply, nil},
		{"/", ArithDivide, nil},
		{"/=", ArithAssignDivide, nil},
		{"%", ArithRemainder, nil},
		{"%=", ArithAssignRemainder, nil},
		{"+", ArithAdd, nil},
		{"+=", ArithAssignAdd, nil},
		{"-", ArithSubtract, nil},
		{"-=", ArithAssignSubtract, nil},
		{"^", ArithBinaryXor, nil},
		{"^=", ArithAssignBinaryXor, nil},
		{"!", ArithNot, nil},
		{"!=", ArithNotEqual, nil},
		{"=", ArithAssignment, nil},
		{"==", ArithEqual, nil},
		{"(", ArithLeftParen, nil},
		{")", ArithRightParen, nil},
		{"~", ArithBinaryNot, nil},
		{"?", ArithQuestionMark, nil},
		{":", ArithColon, nil},
	}

	for _, c := range cases {
		y := NewArithLexer(c.in)
		gotTok, gotVal := y.Lex()
		if c.wantTok != gotTok {
			t.Errorf("'%s' should produce the token \n%s\n not\n%s", c.in, c.wantTok, gotTok)
		}
		if !reflect.DeepEqual(c.wantVal, gotVal) {
			t.Errorf("'%s' should produce the value \n%#v\n not\n%#v", c.in, c.wantVal, gotVal)
		}
	}
}

func TestArithLexerErrors(t *testing.T) {
	cases := []struct {
		in      string
		wantTok ArithToken
		wantVal interface{}
	}{
		{"555a", ArithError, LexError{X: "555a", Err: ErrDecimalConstant}},
		{"0xfi", ArithError, LexError{X: "0xfi", Err: ErrHexConstant}},
		{"0778", ArithError, LexError{X: "0778", Err: ErrOctalConstant}},
	}

	for _, c := range cases {
		y := NewArithLexer(c.in)
		gotTok, gotVal := y.Lex()
		if c.wantTok != gotTok {
			t.Errorf("'%s' should produce the token \n%s\n not\n%s", c.in, c.wantTok, gotTok)
		}

		if !reflect.DeepEqual(c.wantVal, gotVal) {
			t.Errorf("'%s' should produce\n%#v\n not\n%#v", c.in, c.wantVal, gotVal)
		}
	}

}

func TestArithLexerComplex(t *testing.T) {
	type lexPair struct {
		Tok ArithToken
		Val interface{}
	}
	type complexTestCase struct {
		in   string
		want []lexPair
	}

	TC := func(i string, lexems ...lexPair) complexTestCase {
		ctc := complexTestCase{in: i}
		ctc.want = []lexPair{}
		ctc.want = append(ctc.want, lexems...)
		// Append the EOF lexem
		ctc.want = append(ctc.want, lexPair{Tok: ArithEOF})
		return ctc
	}

	cases := []complexTestCase{
		TC(
			"5 >= 4",
			lexPair{Tok: ArithNumber, Val: int64(5)},
			lexPair{Tok: ArithGreaterEqual},
			lexPair{Tok: ArithNumber, Val: int64(4)},
		),
		TC(
			">>= <<= 0xff 067 55 ==",
			lexPair{Tok: ArithAssignRightShift},
			lexPair{Tok: ArithAssignLeftShift},
			lexPair{Tok: ArithNumber, Val: int64(255)},
			lexPair{Tok: ArithNumber, Val: int64(55)},
			lexPair{Tok: ArithNumber, Val: int64(55)},
			lexPair{Tok: ArithEqual},
		),
	}

	for _, c := range cases {
		y := NewArithLexer(c.in)
		for pairCount, want := range c.want {
			gotTok, gotVal := y.Lex()
			if want.Tok != gotTok {
				t.Errorf("'%s' should produce the token \n%s\n as token #%d not\n%s", c.in, want.Tok, pairCount, gotTok)
			}

			if !reflect.DeepEqual(want.Val, gotVal) {
				t.Errorf("'%s' should produce\n%#v\n as value #%d not\n%#v", c.in, want.Val, pairCount, gotVal)
			}
		}
	}
}
