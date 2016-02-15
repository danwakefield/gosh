package main

import (
	"fmt"
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
