//go:generate stringer -type=Token
package main

type Token int

const (
	TEOF Token = iota
	TNewLine
	TSemicolon
	TBackground
	TAnd
	TOr
	TPipe
	TLeftParen
	TRightParen
	TEndCase
	TEndBackQuote
	TRedirection
	TWord
	TNot
	TCase
	TDo
	TDone
	TElif
	TElse
	TEsac
	TFi
	TFor
	TIf
	TIn
	TThen
	TUntil
	TWhile
	TBegin
	TEnd
)

var (
	EndListTokens = map[Token]bool{
		TEOF:          true,
		TRightParen:   true,
		TEndCase:      true,
		TEndBackQuote: true,
		TDo:           true,
		TDone:         true,
		TElif:         true,
		TElse:         true,
		TEsac:         true,
		TFi:           true,
		TThen:         true,
		TEnd:          true,
	}

	ParseKeywords = map[string]Token{
		"!":     TNot,
		"case":  TCase,
		"do":    TDo,
		"done":  TDone,
		"elif":  TElif,
		"else":  TElse,
		"esac":  TEsac,
		"fi":    TFi,
		"for":   TFor,
		"if":    TIf,
		"in":    TIn,
		"then":  TThen,
		"until": TUntil,
		"while": TWhile,
		"{":     TBegin,
		"}":     TEnd,
	}
)
