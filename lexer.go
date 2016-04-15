package main

import (
	"bytes"
	"errors"
	"fmt"
	"unicode/utf8"

	"gopkg.in/logex.v1"

	"github.com/danwakefield/gosh/char"
)

const (
	EOFRune              rune = -1
	SentinalEscape       rune = 201
	SentinalSubstitution rune = 202
)

var (
	ErrQuotedString = errors.New("Unterminated quoted string")
)

type StateFn func(*Lexer) StateFn

type LexItem struct {
	Tok    Token
	Pos    int
	LineNo int
	Val    string `json:",omitempty"`
	Quoted bool
	Subs   []Substitution `json:",omitempty"`
}

type Lexer struct {
	Input         string
	InputLen      int
	LineNo        int
	LastPos       int
	Pos           int
	BackupWidth   int
	State         StateFn
	ItemChan      chan LexItem
	Buf           bytes.Buffer
	Backslash     bool
	Quoted        bool
	Subs          []Substitution
	SubReturnFunc StateFn

	Parser         *Parser
	IgnoreNewlines bool
	CheckAlias     bool
	CheckKeyword   bool
}

func NewLexer(input string, p *Parser) *Lexer {
	l := &Lexer{
		Input:          input,
		InputLen:       len(input),
		ItemChan:       make(chan LexItem),
		Subs:           []Substitution{},
		LineNo:         1,
		IgnoreNewlines: false,
		CheckAlias:     true,
		CheckKeyword:   true,
		Parser:         p,
	}
	go l.run()
	return l
}

func (l *Lexer) emit(t Token) {
	l.ItemChan <- LexItem{
		Tok:    t,
		Pos:    l.LastPos,
		LineNo: l.LineNo,
		Val:    l.Buf.String(),
		Quoted: l.Quoted,
		Subs:   l.Subs,
	}
	l.LastPos = l.Pos
	l.Quoted = false
	l.Subs = []Substitution{}
	l.Buf.Reset()
}

func (l *Lexer) next() rune {
	if l.Pos >= l.InputLen {
		l.Pos++
		return EOFRune
	}
	var r rune
	var w int
	for {
		r, w = utf8.DecodeRuneInString(l.Input[l.Pos:])
		l.Pos += w
		l.BackupWidth = w
		if r != '\x01' {
			break
		}
	}
	return r
}

func (l *Lexer) backup() {
	l.Pos -= l.BackupWidth
	l.BackupWidth = 0
}

func (l *Lexer) hasNext(r rune) bool {
	if r == l.next() {
		return true
	}
	l.backup()
	return false
}

func (l *Lexer) run() {
	for l.State = lexStart; l.State != nil; {
		l.State = l.State(l)
	}
	close(l.ItemChan)
}

func (l *Lexer) ignore() {
	l.LastPos = l.Pos
}

func (l *Lexer) NextLexItem() (li LexItem) {
	defer func() {
		l.CheckAlias = false
		l.IgnoreNewlines = false
		l.CheckKeyword = false

		logex.Pretty(li, li.Tok.String())
	}()
	li = <-l.ItemChan

	if l.IgnoreNewlines {
		for li.Tok == TNewLine {
			li = <-l.ItemChan
		}
	}

	if li.Tok != TWord || li.Quoted {
		return li
	}

	if l.CheckKeyword {
		t, found := KeywordLookup[li.Val]
		if found {
			li = LexItem{Tok: t, Pos: li.Pos, LineNo: li.LineNo, Val: li.Val}
			return li
		}
	}

	if l.CheckAlias {
		// Expand Alias.
	}

	return li
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
			l.ignore()
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
				l.LineNo++
				continue
			}
			l.backup()
			l.Backslash = true
			l.Quoted = true
			return lexWord
		case '\n':
			l.emit(TNewLine)
			l.LineNo++
		case '&':
			if l.hasNext('&') {
				l.emit(TAnd)
			} else {
				l.emit(TBackground)
			}
		case '|':
			if l.hasNext('|') {
				l.emit(TOr)
			} else {
				l.emit(TPipe)
			}
		case ';':
			if l.hasNext(';') {
				l.emit(TEndCase)
			} else {
				l.emit(TSemicolon)
			}
		case '(':
			l.emit(TLeftParen)
		case ')':
			l.emit(TRightParen)
		}
	}
}

func lexWord(l *Lexer) StateFn {

OuterLoop:
	for {
		c := l.next()

		if l.Backslash {
			if c == EOFRune {
				l.backup()
				l.Buf.WriteRune('\\')
				break
			}
			if c == '\\' {
				l.Buf.WriteRune('\\')
				continue
			}
			l.Buf.WriteRune(SentinalEscape)
			l.Buf.WriteRune(c)
			l.Backslash = false
			continue
		}

		switch c {
		case '\n', '\t', ' ', '<', '>', '(', ')', ';', '&', '|', EOFRune:
			l.backup()
			break OuterLoop
		case '\'':
			l.Quoted = true
			return lexSingleQuote
		case '"':
			l.Quoted = true
			return lexDoubleQuote
		case '`':
			return lexBackQuote
		case '$':
			l.SubReturnFunc = lexWord
			return lexSubstitution
		default:
			l.Buf.WriteRune(c)
		}
	}

	l.emit(TWord)
	return lexStart
}

func lexSubstitution(l *Lexer) StateFn {
	// Upon entering we have only read the '$'
	c := l.next()

	switch {
	default:
		l.Buf.WriteRune('$')
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
}

func lexVariableSimple(l *Lexer) StateFn {
	// When we enter this state we know we have at least one readable
	// char for the varname. That means that any character not valid
	// just terminates the parsing and we dont have to
	// worry about the case of an empty varname
	l.Buf.WriteRune(SentinalSubstitution)
	sv := SubVariable{SubType: VarSubNormal}
	varbuf := bytes.Buffer{}

	c := l.next()
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
	default:
		l.backup()
	}

	sv.VarName = varbuf.String()
	l.Subs = append(l.Subs, sv)
	return l.SubReturnFunc
}
func lexVariableComplex(l *Lexer) StateFn {
	// Upon entering we have read the opening '{'
	l.Buf.WriteRune(SentinalSubstitution)
	sv := SubVariable{}
	varbuf := bytes.Buffer{}

	defer func() {
		// We defer this as there are multiple return points
		sv.VarName = varbuf.String()
		l.Subs = append(l.Subs, sv)
	}()

	if l.hasNext('#') {
		// The NParam Special Var
		if l.hasNext('}') {
			varbuf.WriteRune('#')
			return l.SubReturnFunc
		}

		// Length variable operator
		sv.SubType = VarSubLength
	}

	c := l.next()
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
		return l.SubReturnFunc
	}

	if l.hasNext('}') {
		return l.SubReturnFunc
	}

	// Length operator should have returned since only ${#varname} is valid
	if sv.SubType == VarSubLength {
		logex.Panic(fmt.Sprintf("Line %d: Bad substitution (%s)", l.LineNo, l.Input[l.LastPos:l.Pos]))
	}

	if l.hasNext(':') {
		sv.CheckNull = true
	}

	switch l.next() {
	case '-':
		sv.SubType = VarSubMinus
	case '+':
		sv.SubType = VarSubPlus
	case '?':
		sv.SubType = VarSubQuestion
	case '=':
		sv.SubType = VarSubAssign
	case '#':
		if l.hasNext('#') {
			sv.SubType = VarSubTrimLeftMax
		} else {
			sv.SubType = VarSubTrimLeft
		}
	case '%':
		if l.hasNext('%') {
			sv.SubType = VarSubTrimRightMax
		} else {
			sv.SubType = VarSubTrimRight
		}
	default:
		logex.Panic(fmt.Sprintf("Line %d: Bad substitution (%s)", l.LineNo, l.Input[l.LastPos:l.Pos]))
	}

	// Read until '}'
	// In the future to support Nested vars etc create new sublexer from
	// l.input[l.pos:] and take the first lexitem as the sub val then adjust
	// this lexer's position and trash sublexer
	c = l.next()
	subValBuf := bytes.Buffer{}
	for {
		if c == '}' {
			break
		}
		subValBuf.WriteRune(c)
		c = l.next()
	}
	sv.SubVal = subValBuf.String()

	return l.SubReturnFunc

}
func lexBackQuote(l *Lexer) StateFn {
	logex.Panic("Not Implemented")
	return nil
}
func lexSubshell(l *Lexer) StateFn {
	l.ignore()

	p := NewParser(l.Input[l.Pos:])
	ss := SubSubshell{}
	ss.N = p.list(AllowEmptyNode)

	// We have to explictitly get the next item to prevent race conditions
	// in accessing the lexers Pos and LineNo fields.
	x := p.lexer.NextLexItem()
	l.Pos += x.Pos
	l.LineNo += x.LineNo

	l.Buf.WriteRune(SentinalSubstitution)
	l.Subs = append(l.Subs, ss)

	return l.SubReturnFunc
}
func lexArith(l *Lexer) StateFn {
	// Upon entering this state we have read the '$(('
	l.Buf.WriteRune(SentinalSubstitution)
	arithBuf := bytes.Buffer{}
	sa := SubArith{}

	parenCount := 0
	for {
		c := l.next()
		if parenCount == 0 && c == ')' {
			if l.hasNext(')') {
				break
			}
			// Bash just ignores a closing brakcet with no opening
			// bracket so we will emulate that.
			continue
		}
		if c == '(' {
			parenCount++
		}
		if c == ')' {
			parenCount--
		}
		arithBuf.WriteRune(c)
	}

	sa.Raw = arithBuf.String()
	l.Subs = append(l.Subs, sa)
	return l.SubReturnFunc
}

func lexDoubleQuote(l *Lexer) StateFn {
	// We have consumed the first quote before entering this state.
	for {
		c := l.next()

		switch c {
		case EOFRune:
			panic(ErrQuotedString) //TODO: Dont make this panic
		case '$':
			l.SubReturnFunc = lexDoubleQuote
			return lexSubstitution
		case '"':
			return lexWord
		case '\\':
			c = l.next()
			switch c {
			case '\n': // Ignored
			case '\\', '$', '`', '"':
				l.Buf.WriteRune(c)
			default:
				l.backup()
				l.Buf.WriteRune('\\')
			}
		default:
			l.Buf.WriteRune(c)
		}
	}
}

func lexSingleQuote(l *Lexer) StateFn {
	// We have consumed the first quote before entering this state.
	for {
		c := l.next()

		switch c {
		case EOFRune:
			panic(ErrQuotedString) //TODO: Dont make this panic
		case '\'':
			return lexWord
		default:
			l.Buf.WriteRune(c)
		}
	}
}
