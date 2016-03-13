package char

import "strings"

func IsSpecial(r rune) bool {
	return IsDigit(r) || strings.IndexRune("!#$*-?@", r) != -1
}

func IsAlpha(r rune) bool {
	return (r <= 'z' && r >= 'a') || (r <= 'Z' && r >= 'A')
}

func IsDigit(r rune) bool {
	return r <= '9' && r >= '0'
}

func IsAlnum(r rune) bool {
	return IsAlpha(r) || IsDigit(r)
}

func IsInVarName(r rune) bool {
	return r == '_' || IsAlnum(r)
}

func IsFirstInVarName(r rune) bool {
	return r == '_' || IsAlpha(r)
}

func IsHexDigit(r rune) bool {
	return IsDigit(r) || (r <= 'f' && r >= 'a') || (r <= 'F' && r >= 'A')
}

func IsOctalDigit(r rune) bool {
	return r <= '7' && r >= '0'
}
