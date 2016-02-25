package main

import "testing"

func TestArithParserBinops(t *testing.T) {
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
		{"5 <  4", ShellFalse},
		{"3 <  4", ShellTrue},
		{"3 >  4", ShellFalse},
		{"5 >  4", ShellTrue},
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
	}

	for _, c := range cases {
		got, err := Parse(c.in)
		if err != nil {
			t.Errorf("Parse returned an error: %s", err.Error())
		}
		if got != c.want {
			t.Errorf("Parse(%s) should return %d not %d", c.in, c.want, got)
		}
	}
}

func TestArithPrefix(t *testing.T) {
	type TestCase struct {
		in   string
		want int64
	}
	cases := []TestCase{
		{"~4", -5},
		{"~~4", 4},
		{"!1", ShellTrue},
		{"!4", ShellTrue},
		{"!0", ShellFalse},
		{"!!1", ShellFalse},
		{"1+2*3", 7},
		{"1+(2*3)", 7},
		{"(1+2)*3", 9},
	}
	for _, c := range cases {
		got, err := Parse(c.in)
		if err != nil {
			t.Errorf("Parse returned an error: %s", err.Error())
		}
		if got != c.want {
			t.Errorf("Parse(%s) should return %d not %d", c.in, c.want, got)
		}
	}
}

func TestArithParserTernary(t *testing.T) {
	type TestCase struct {
		in   string
		want int64
	}
	cases := []TestCase{
		{"1 ? 3 : 4", 3},
		{"0 ? 3 : 4", 4},
	}
	for _, c := range cases {
		got, err := Parse(c.in)
		if err != nil {
			t.Errorf("Parse returned an error: %s", err.Error())
		}
		if got != c.want {
			t.Errorf("Parse(%s) should return %d not %d", c.in, c.want, got)
		}
	}
}

func TestArithParserAssignment(t *testing.T) {
	type TestCase struct {
		inString string
		inVars   map[string]string
		want     int64
		wantVars map[string]string
	}
	cases := []TestCase{
		{
			"x=2",
			map[string]string{},
			2,
			map[string]string{"x": "2"},
		},
		{
			"x+=2",
			map[string]string{},
			2,
			map[string]string{"x": "2"},
		},
		{
			"x+=2",
			map[string]string{"x": "2"},
			4,
			map[string]string{"x": "4"},
		},
		{
			"x*=4",
			map[string]string{"x": "2"},
			8,
			map[string]string{"x": "8"},
		},
	}

	for _, c := range cases {
		GlobalScope = NewScope()
		for k, v := range c.inVars {
			GlobalScope.Set(k, v)
		}
		got, err := Parse(c.inString)
		if err != nil {
			t.Errorf("Parse returned an error: %s", err.Error())
		}
		if got != c.want {
			t.Errorf("Variable assignment '%s' should evaluate to '%d'", c.inString, c.want)
		}
		for varName, wantVar := range c.wantVars {
			gotVar := GlobalScope.Get(varName)
			if gotVar.Val != wantVar {
				t.Errorf("Variable assignment should modify global scope. '%s' should be '%s' not '%s'", varName, wantVar, gotVar.Val)
			}
		}
	}
}
