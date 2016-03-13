package variables

import "github.com/danwakefield/gosh/char"

func IsAssignment(s string) bool {
	rs := []rune(s)
	if !char.IsFirstInVarName(rs[0]) {
		return false
	}
	for _, c := range rs[1:] {
		if !char.IsInVarName(c) {
			return c == '='
		}
	}
	return false
}
