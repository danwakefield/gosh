package main

import "math"

const (
	ShellTrue  = 0
	ShellFalse = 1
)

func IsAlpha(r rune) bool {
	return (r <= 'z' && r >= 'a') || (r <= 'Z' && r >= 'A')
}

func IsDigit(r rune) bool {
	return r <= '9' && r >= '0'
}

func IsAlnum(r rune) bool {
	return IsAlpha(r) || IsDigit(r)
}

func IsInName(r rune) bool {
	return r == '_' || IsAlnum(r)
}

func IsFirstInName(r rune) bool {
	return r == '_' || IsAlpha(r)
}

func IsHexDigit(r rune) bool {
	return IsDigit(r) || (r <= 'f' && r >= 'a') || (r <= 'F' && r >= 'A')
}

func IsOctalDigit(r rune) bool {
	return r <= '7' && r >= '0'
}

func LeftShift(a, b int64) int64 {
	c := int64(math.Pow(2, float64(b)))
	if c == 0 {
		return a
	} else if c < 0 {
		panic("Negative Left Shift")
	}
	return a * c
}

func RightShift(a, b int64) int64 {
	c := int64(math.Pow(2, float64(b)))
	if c == 0 {
		return 0
	} else if c < 0 {
		panic("Negative Right Shift")
	}
	return a / c
}

func BoolToShell(b bool) int64 {
	if b {
		return ShellTrue
	}
	return ShellFalse
}
