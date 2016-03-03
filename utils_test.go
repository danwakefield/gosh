package main

import "testing"

func TestLeftShift(t *testing.T) {
	type TestCase struct {
		ina  int64
		inb  int64
		want int64
	}
	cases := []TestCase{
		{1, 1, 2},
		{1, 5, 32},
		{5, 5, 160},
		{20, 20, 20971520},
	}

	for _, c := range cases {
		got := LeftShift(c.ina, c.inb)
		if got != c.want {
			t.Errorf("%d << %d should be %d not %d", c.ina, c.inb, c.want, got)
		}
	}
}

func TestRightShift(t *testing.T) {
	type TestCase struct {
		ina  int64
		inb  int64
		want int64
	}
	cases := []TestCase{
		{2, 1, 1},
		{32, 5, 1},
		{160, 5, 5},
		{20971520, 20, 20},
		{1, 1, 0},
	}

	for _, c := range cases {
		got := RightShift(c.ina, c.inb)
		if got != c.want {
			t.Errorf("%d >> %d should be %d not %d", c.ina, c.inb, c.want, got)
		}
	}
}
