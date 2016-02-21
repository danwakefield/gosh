package main

import "testing"

func TestArithParser(t *testing.T) {
	type TestCase struct {
		in   string
		want int64
	}
	cases := []TestCase{
		{"5 <= 4", ShellFalse},
		{"4 <= 4", ShellTrue},
		{"3 <= 4", ShellTrue},
		{"3 >= 4", ShellFalse},
		{"4 >= 4", ShellTrue},
		{"5 >= 4", ShellTrue},
		{"5 < 4", ShellFalse},
		{"3 < 4", ShellTrue},
		{"3 > 4", ShellFalse},
		{"5 > 4", ShellTrue},
		{"5 == 4", ShellFalse},
		{"4 == 4", ShellTrue},
		{"4 != 4", ShellFalse},
		{"5 != 4", ShellTrue},
		{"5 & 4", 4},
		{"3 & 4", 0},
		{"3 | 4", 7},
		{"4 | 4", 4},
		{"3 ^ 4", 7},
		{"4 ^ 4", 0},
		{"1 << 4", 16},
		{"16 >> 4", 1},
		{"10 % 4", 2},
		{"3 * 4", 12},
		{"12 / 4", 3},
		{"10 - 4", 6},
		{"10 + 4", 14},
		{"-5 + 4", -1},
	}

	for _, c := range cases {
		got := Parse(c.in)
		if got != c.want {
			t.Errorf("Parse(%s) should be %d not %d", c.in, c.want, got)
		}
	}
}
