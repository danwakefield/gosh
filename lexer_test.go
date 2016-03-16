package main

import "testing"

func TestLex(t *testing.T) {
	cases := []struct {
		in  string
		out []LexItem
	}{
		{"", []LexItem{LexItem{Tok: TEOF, Pos: 0, LineNo: 1, Val: ""}}},
		{
			"foo",
			[]LexItem{
				LexItem{Tok: TWord, Pos: 0, LineNo: 1, Val: "foo"},
				LexItem{Tok: TEOF, Pos: 3, LineNo: 1, Val: ""},
			},
		},
		{
			"foo bar",
			[]LexItem{
				LexItem{Tok: TWord, Pos: 0, LineNo: 1, Val: "foo"},
				LexItem{Tok: TWord, Pos: 3, LineNo: 1, Val: "bar"},
				LexItem{Tok: TEOF, Pos: 7, LineNo: 1, Val: ""},
			},
		},
		{
			`foo
			bar`,
			[]LexItem{
				LexItem{Tok: TWord, Pos: 0, LineNo: 1, Val: "foo"},
				LexItem{Tok: TNewLine, Pos: 3, LineNo: 1},
				LexItem{Tok: TWord, Pos: 4, LineNo: 2, Val: "bar"},
				LexItem{Tok: TEOF, Pos: 10, LineNo: 2},
			},
		},
		{
			// both dash and bash concat words despite string boundaries
			"'foo'\"bar\"baz",
			[]LexItem{
				LexItem{Tok: TWord, Pos: 0, LineNo: 1, Val: "foobarbaz", Quoted: true},
				LexItem{Tok: TEOF, Pos: 13, LineNo: 1},
			},
		},
		{
			"foo #blah blah",
			[]LexItem{
				LexItem{Tok: TWord, Pos: 0, LineNo: 1, Val: "foo"},
				LexItem{Tok: TEOF, Pos: 15, LineNo: 1},
			},
		},
		{
			"foo='blah'",
			[]LexItem{
				LexItem{Tok: TWord, Pos: 0, LineNo: 1, Val: "foo=blah", Quoted: true},
				LexItem{Tok: TEOF, Pos: 10, LineNo: 1},
			},
		},
	}

	for _, c := range cases {
		l := NewLexer(c.in)
		for count, expectedLexItem := range c.out {
			got := l.NextLexItem()
			if got != expectedLexItem {
				t.Errorf(
					"Lexing:\n %s\nExpected:\n %#v\nas LexItem %d but got:\n %#v\n",
					c.in, expectedLexItem, count, got,
				)
			}
		}
	}
}
