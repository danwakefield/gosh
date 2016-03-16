package main

import (
	"bytes"
	"errors"
	"unicode/utf8"

	"github.com/danwakefield/gosh/char"
)

const (
	EOFRune             rune = -1
	SentinalEscape      rune = 201
	SentinalVariable    rune = 202
	SentinalEndVariable rune = 203
	SentinalBackquote   rune = 204
	SentinalArith       rune = 206
	SentinalEndArith    rune = 207
	SentinalQuote       rune = 210

	VarSubCheckNull rune = (iota + 1)
	VarSubNormal
	VarSubMinus
	VarSubPlus
	VarSubQuestion
	VarSubAssign
	VarSubTrimRight
	VarSubTrimRightMax
	VarSubTrimLeft
	VarSubTrimLeftMax
	VarSubLength
	VarSubSeperator
)

var (
	ErrQuotedString = errors.New("Unterminated quoted string")
)

type StateFn func(*Lexer) StateFn

type LexItem struct {
	Tok    Token
	Pos    int
	LineNo int
	Val    string
}

type Lexer struct {
	input              string
	inputLen           int
	lineNo             int
	lastPos            int
	pos                int
	backupWidth        int
	state              StateFn
	itemChan           chan LexItem
	buf                bytes.Buffer
	backslash          bool
	quoted             bool
	variableReturnFunc StateFn

	CheckNewline bool
	CheckAlias   bool
	CheckKeyword bool
}

func NewLexer(input string) *Lexer {
	l := &Lexer{
		input:        input,
		inputLen:     len(input),
		itemChan:     make(chan LexItem),
		lineNo:       1,
		CheckNewline: false,
		CheckAlias:   true,
		CheckKeyword: true,
	}
	go l.run()
	return l
}

func (l *Lexer) emit(t Token) {
	l.itemChan <- LexItem{
		Tok:    t,
		Pos:    l.lastPos,
		LineNo: l.lineNo,
		Val:    l.buf.String(),
	}
	l.lastPos = l.pos
	l.buf.Reset()
}

func (l *Lexer) next() rune {
	if l.pos >= l.inputLen {
		l.pos++
		return EOFRune
	}
	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += w
	l.backupWidth = w
	return r
}

func (l *Lexer) backup() {
	l.pos -= l.backupWidth
	l.backupWidth = 0
}

func (l *Lexer) hasNext(r rune) bool {
	if r == l.next() {
		return true
	}
	l.backup()
	return false
}

func (l *Lexer) run() {
	for l.state = lexStart; l.state != nil; {
		l.state = l.state(l)
	}
	close(l.itemChan)
}

func (l *Lexer) ignore() {
	l.lastPos = l.pos
}

func (l *Lexer) NextLexItem() LexItem {
	tok := <-l.itemChan

	if l.CheckNewline {
		for tok.Tok == TNewLine {
			tok = <-l.itemChan
		}
	}

	if tok.Tok != TWord || l.quoted {
		return tok
	}

	if l.CheckKeyword {
		t, found := KeywordLookup[tok.Val]
		if found {
			return LexItem{Tok: t, Pos: tok.Pos, LineNo: tok.LineNo, Val: tok.Val}
		}
	}

	// Expand Alias.

	return tok
}

func lexStart(l *Lexer) StateFn {
	for {
		c := l.next()

		switch c {
		default:
			l.backup()
			return lexWord
		case EOFRune:
			l.emit(TEOF)
			return nil
		case ' ', '\t': // Ignore Whitespace
			continue
		case '#': // Consume comments upto EOF or a newline
			for {
				c = l.next()
				if c == '\n' || c == EOFRune {
					l.ignore()
					l.backup()
					break
				}
			}
		case '\\': // Line Continuation or escaped character
			if l.hasNext('\n') {
				l.lineNo++
				continue
			}
			l.backup()
			l.backslash = true
			l.quoted = true
			return lexWord
		case '\n':
			l.emit(TNewLine)
			l.lineNo++
		case '&':
			if l.hasNext('&') {
				l.emit(TAnd)
			}
			l.emit(TBackground)
		case '|':
			if l.hasNext('|') {
				l.emit(TOr)
			}
			l.emit(TPipe)
		case ';':
			if l.hasNext(';') {
				l.emit(TEndCase)
			}
			l.emit(TSemicolon)
		case '(':
			l.emit(TLeftParen)
		case ')':
			l.emit(TRightParen)
		}
	}
}

func lexSubstitution(l *Lexer) StateFn {
	// Upon entering we have only read the '$'
	c := l.next()

	switch {
	default:
		l.buf.WriteRune('$')
		l.backup()
		return lexWord
	case c == '(':
		if l.hasNext('(') {
			return lexArith
		}
		return lexSubshell
	case c == '{':
		return lexVariableComplex
	case char.IsFirstInVarName(c), char.IsDigit(c), char.IsSpecial(c):
		l.backup()
		return lexVariableSimple
	}
	return nil
}

func lexVariableSimple(l *Lexer) StateFn {
	l.buf.WriteRune(SentinalVariable)
	l.buf.WriteRune(VarSubNormal)
	defer l.buf.WriteRune(SentinalEndVariable)

	c := l.next()
	switch {
	case char.IsSpecial(c):
		l.buf.WriteRune(c)
	case char.IsDigit(c):
		for {
			l.buf.WriteRune(c)
			c = l.next()
			if !char.IsDigit(c) {
				l.backup()
				break
			}
		}
	case char.IsFirstInVarName(c):
		for {
			l.buf.WriteRune(c)
			c = l.next()
			if !char.IsInVarName(c) {
				l.backup()
				break
			}
		}
	default:
		l.backup()
	}

	return l.variableReturnFunc
}
func lexVariableComplex(l *Lexer) StateFn {
	// Upon entering we have read the opening '{'
	l.buf.WriteRune(SentinalVariable)
	defer l.buf.WriteRune(SentinalEndVariable)
	varSubSentinal := rune(0)

	c := l.next()
	if c == '#' {
		// The NParam Special Var
		if l.hasNext('}') {
			l.buf.WriteRune(VarSubNormal)
			l.buf.WriteRune('#')
			return l.variableReturnFunc
		}

		// Length variable operator
		varSubSentinal = VarSubLength
		c = l.next()
	}

	varbuf := bytes.Buffer{}
	switch {
	case char.IsSpecial(c):
		varbuf.WriteRune(c)
	case char.IsDigit(c):
		for {
			varbuf.WriteRune(c)
			c = l.next()
			if !char.IsDigit(c) {
				l.backup()
				break
			}
		}
	case char.IsFirstInVarName(c):
		for {
			varbuf.WriteRune(c)
			c = l.next()
			if !char.IsInVarName(c) {
				l.backup()
				break
			}
		}
	case c == EOFRune:
		l.backup()
		return l.variableReturnFunc
	}

	if l.hasNext('}') {
		if varSubSentinal != rune(0) {
			varbuf.WriteRune(varSubSentinal)
		} else {
			varbuf.WriteRune(VarSubNormal)
		}
		l.buf.Write(varbuf.Bytes())
		return l.variableReturnFunc
	}

	// We have to check for operators
	c = l.next()

	panic("Bad substitution")

}
func lexBackQuote(l *Lexer) StateFn {
	return nil
}
func lexSubshell(l *Lexer) StateFn {
	return nil
}
func lexArith(l *Lexer) StateFn {
	return nil
}
func lexWord(l *Lexer) StateFn {

OuterLoop:
	for {
		c := l.next()

		if l.backslash {
			if c == EOFRune {
				l.backup()
				l.buf.WriteRune('\\')
				break
			}
			l.buf.WriteRune(SentinalEscape)
			l.buf.WriteRune(c)
			l.backslash = false
			continue
		}

		switch c {
		case '\n', ' ', EOFRune:
			l.backup()
			break OuterLoop
		case '\'':
			l.quoted = true
			return lexSingleQuote
		case '"':
			l.quoted = true
			return lexDoubleQuote
		case '`':
			return lexBackQuote
		case '$':
			l.variableReturnFunc = lexWord
			return lexSubstitution
		default:
			l.buf.WriteRune(c)
		}
	}

	l.emit(TWord)
	return lexStart
}

func lexDoubleQuote(l *Lexer) StateFn {
	l.buf.WriteRune(SentinalQuote)
	defer l.buf.WriteRune(SentinalQuote)
	for {
		c := l.next()

		switch c {
		case EOFRune:
			panic(ErrQuotedString) //TODO: Dont make this panic
		case '\x01':
			continue
		case '"':
			return lexWord
		default:
			l.buf.WriteRune(c)
		}
	}
}

func lexSingleQuote(l *Lexer) StateFn {
	l.buf.WriteRune(SentinalQuote)
	defer l.buf.WriteRune(SentinalQuote)
	// We have consumed the first single quote before entering
	// this state.
	for {
		c := l.next()

		switch c {
		case EOFRune:
			panic(ErrQuotedString) //TODO: Dont make this panic
		case '\x01':
			continue
		case '\'':
			return lexWord
		default:
			l.buf.WriteRune(c)
		}
	}
}
