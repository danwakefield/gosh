package arith

import "math"

const (
	ShellTrue  = 0
	ShellFalse = 1
)

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
