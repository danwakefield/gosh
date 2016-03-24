package arith_test

import (
	"fmt"
	"reflect"
	"testing"

	A "github.com/danwakefield/gosh/arith"
)

func TestTokenIsBinaryOp(t *testing.T) {
	cases := []struct {
		in   A.Token
		want bool
	}{
		{A.ArithLessEqual, true},
		{A.ArithDivide, true},
		{A.ArithEqual, true},
		{A.ArithVariable, false},
		{A.ArithAssignBinaryAnd, false},
	}

	for _, c := range cases {
		got := A.TokenIsBinaryOp(c.in)
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
		in   A.Token
		want bool
	}{
		{A.ArithAssignBinaryAnd, true},
		{A.ArithAssignDivide, true},
		{A.ArithAssignLeftShift, true},
		{A.ArithDivide, false},
		{A.ArithLeftParen, false},
	}

	for _, c := range cases {
		got := A.TokenIsAssignmentOp(c.in)
		if got != c.want {
			t.Errorf("%s Should be %v not %v", c.in, c.want, got)
		}
	}
}

func TestTokenString(t *testing.T) {
	cases := []struct {
		in   A.Token
		want string
	}{
		{A.ArithVariable, "ArithVariable"},
		{A.ArithDivide, "ArithDivide"},
		{A.ArithAssignLeftShift, "ArithAssignLeftShift"},
		{A.ArithLeftParen, "ArithLeftParen"},
	}

	for _, c := range cases {
		got := fmt.Sprintf("%s", c.in)
		if got != c.want {
			t.Errorf("Token should stringify to %s not %s", c.want, got)
		}
	}
}

func TestTokenAssignDiff(t *testing.T) {
	cases := []struct {
		in   A.Token
		want A.Token
	}{
		{A.ArithBinaryAnd, A.ArithAssignBinaryAnd},
		{A.ArithAdd, A.ArithAssignAdd},
		{A.ArithDivide, A.ArithAssignDivide},
	}

	for _, c := range cases {
		got := c.in + A.ArithAssignDiff
		if got != c.want {
			t.Errorf("%s should be %s not %s", c.in, c.want, got)
		}
	}
}

func TestLexer(t *testing.T) {
	cases := []struct {
		in      string
		wantTok A.Token
		wantVal interface{}
	}{
		{"_abcd", A.ArithVariable, "_abcd"},
		{"5", A.ArithNumber, int64(5)},
		{"555", A.ArithNumber, int64(555)},
		{"0", A.ArithNumber, int64(0)},
		{"0xff", A.ArithNumber, int64(255)},
		{"077", A.ArithNumber, int64(63)},
		{"", A.ArithEOF, nil},
		{"   \n\t  ", A.ArithEOF, nil},
		{">", A.ArithGreaterThan, nil},
		{">=", A.ArithGreaterEqual, nil},
		{">>", A.ArithRightShift, nil},
		{">>=", A.ArithAssignRightShift, nil},
		{"<", A.ArithLessThan, nil},
		{"<=", A.ArithLessEqual, nil},
		{"<<", A.ArithLeftShift, nil},
		{"<<=", A.ArithAssignLeftShift, nil},
		{"|", A.ArithBinaryOr, nil},
		{"|=", A.ArithAssignBinaryOr, nil},
		{"||", A.ArithOr, nil},
		{"&", A.ArithBinaryAnd, nil},
		{"&=", A.ArithAssignBinaryAnd, nil},
		{"&&", A.ArithAnd, nil},
		{"*", A.ArithMultiply, nil},
		{"*=", A.ArithAssignMultiply, nil},
		{"/", A.ArithDivide, nil},
		{"/=", A.ArithAssignDivide, nil},
		{"%", A.ArithRemainder, nil},
		{"%=", A.ArithAssignRemainder, nil},
		{"+", A.ArithAdd, nil},
		{"+=", A.ArithAssignAdd, nil},
		{"-", A.ArithSubtract, nil},
		{"-=", A.ArithAssignSubtract, nil},
		{"^", A.ArithBinaryXor, nil},
		{"^=", A.ArithAssignBinaryXor, nil},
		{"!", A.ArithNot, nil},
		{"!=", A.ArithNotEqual, nil},
		{"=", A.ArithAssignment, nil},
		{"==", A.ArithEqual, nil},
		{"(", A.ArithLeftParen, nil},
		{")", A.ArithRightParen, nil},
		{"~", A.ArithBinaryNot, nil},
		{"?", A.ArithQuestionMark, nil},
		{":", A.ArithColon, nil},
	}

	for _, c := range cases {
		y := A.NewLexer(c.in)
		gotTok, gotVal := y.Lex()
		if c.wantTok != gotTok {
			t.Errorf("'%s' should produce the token \n%s\n not\n%s", c.in, c.wantTok, gotTok)
		}
		if !reflect.DeepEqual(c.wantVal, gotVal) {
			t.Errorf("'%s' should produce the value \n%#v\n not\n%#v", c.in, c.wantVal, gotVal)
		}
	}
}

func TestLexerErrors(t *testing.T) {
	cases := []struct {
		in      string
		wantTok A.Token
		wantVal interface{}
	}{
		{"555a", A.ArithError, A.LexError{X: "555a", Err: A.ErrDecimalConstant}},
		{"0xfi", A.ArithError, A.LexError{X: "0xfi", Err: A.ErrHexConstant}},
		{"0778", A.ArithError, A.LexError{X: "0778", Err: A.ErrOctalConstant}},
	}

	for _, c := range cases {
		y := A.NewLexer(c.in)
		gotTok, gotVal := y.Lex()
		if c.wantTok != gotTok {
			t.Errorf("'%s' should produce the token \n%s\n not\n%s", c.in, c.wantTok, gotTok)
		}

		if !reflect.DeepEqual(c.wantVal, gotVal) {
			t.Errorf("'%s' should produce\n%#v\n not\n%#v", c.in, c.wantVal, gotVal)
		}
	}

}

func TestLexerComplex(t *testing.T) {
	type lexPair struct {
		Tok A.Token
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
		ctc.want = append(ctc.want, lexPair{Tok: A.ArithEOF})
		return ctc
	}

	cases := []complexTestCase{
		TC(
			"5 >= 4",
			lexPair{Tok: A.ArithNumber, Val: int64(5)},
			lexPair{Tok: A.ArithGreaterEqual},
			lexPair{Tok: A.ArithNumber, Val: int64(4)},
		),
		TC(
			">>= <<= 0xff 067 55 ==",
			lexPair{Tok: A.ArithAssignRightShift},
			lexPair{Tok: A.ArithAssignLeftShift},
			lexPair{Tok: A.ArithNumber, Val: int64(255)},
			lexPair{Tok: A.ArithNumber, Val: int64(55)},
			lexPair{Tok: A.ArithNumber, Val: int64(55)},
			lexPair{Tok: A.ArithEqual},
		),
	}

	for _, c := range cases {
		y := A.NewLexer(c.in)
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
