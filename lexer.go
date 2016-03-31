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
	input         string
	inputLen      int
	lineNo        int
	lastPos       int
	pos           int
	backupWidth   int
	state         StateFn
	itemChan      chan LexItem
	buf           bytes.Buffer
	backslash     bool
	quoted        bool
	subs          []Substitution
	subReturnFunc StateFn

	CheckNewline bool
	CheckAlias   bool
	CheckKeyword bool
}

func NewLexer(input string) *Lexer {
	l := &Lexer{
		input:        input,
		inputLen:     len(input),
		itemChan:     make(chan LexItem),
		subs:         []Substitution{},
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
		Quoted: l.quoted,
		Subs:   l.subs,
	}
	l.lastPos = l.pos
	l.quoted = false
	l.subs = []Substitution{}
	l.buf.Reset()
}

func (l *Lexer) next() rune {
	if l.pos >= l.inputLen {
		l.pos++
		return EOFRune
	}
	var r rune
	var w int
	for {
		r, w = utf8.DecodeRuneInString(l.input[l.pos:])
		l.pos += w
		l.backupWidth = w
		if r != '\x01' {
			break
		}
	}
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

func (l *Lexer) NextLexItem() (li LexItem) {
	defer func() {
		l.CheckAlias = false
		l.CheckNewline = false
		l.CheckKeyword = false

		logex.Pretty(li, li.Tok.String())
	}()
	li = <-l.itemChan

	if l.CheckNewline {
		for li.Tok == TNewLine {
			li = <-l.itemChan
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

		if l.backslash {
			if c == EOFRune {
				l.backup()
				l.buf.WriteRune('\\')
				break
			}
			if c == '\\' {
				l.buf.WriteRune('\\')
				continue
			}
			l.buf.WriteRune(SentinalEscape)
			l.buf.WriteRune(c)
			l.backslash = false
			continue
		}

		switch c {
		case '\n', '\t', ' ', '<', '>', '(', ')', ';', '&', '|', EOFRune:
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
			l.subReturnFunc = lexWord
			return lexSubstitution
		default:
			l.buf.WriteRune(c)
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
}

func lexVariableSimple(l *Lexer) StateFn {
	// When we enter this state we know we have at least one readable
	// char for the varname. That means that any character not valid
	// in the varname just terminates the parsing and we dont have to
	// worry about the case of an empty varname
	l.buf.WriteRune(SentinalSubstitution)
	sv := SubVariable{SubType: VarSubNormal}
	varbuf := bytes.Buffer{}
	defer func() {
		sv.VarName = varbuf.String()
		l.subs = append(l.subs, sv)
	}()

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

	return l.subReturnFunc
}
func lexVariableComplex(l *Lexer) StateFn {
	// Upon entering we have read the opening '{'
	l.buf.WriteRune(SentinalSubstitution)
	sv := SubVariable{}
	varbuf := bytes.Buffer{}
	defer func() {
		sv.VarName = varbuf.String()
		l.subs = append(l.subs, sv)
	}()

	if l.hasNext('#') {
		// The NParam Special Var
		if l.hasNext('}') {
			varbuf.WriteRune('#')
			return l.subReturnFunc
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
		return l.subReturnFunc
	}

	if l.hasNext('}') {
		return l.subReturnFunc
	}

	// Length operator should have returned since only ${#varname} is valid
	if sv.SubType == VarSubLength {
		logex.Panic(fmt.Sprintf("Line %d: Bad substitution (%s)", l.lineNo, l.input[l.lastPos:l.pos]))
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
		logex.Panic(fmt.Sprintf("Line %d: Bad substitution (%s)", l.lineNo, l.input[l.lastPos:l.pos]))
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

	return l.subReturnFunc

}
func lexBackQuote(l *Lexer) StateFn {
	logex.Panic("Not Implemented")
	return nil
}
func lexSubshell(l *Lexer) StateFn {
	logex.Panic("Not Implemented")
	return nil
}
func lexArith(l *Lexer) StateFn {
	// Upon entering this state we have read the '$(('
	l.buf.WriteRune(SentinalSubstitution)
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
	l.subs = append(l.subs, sa)
	return l.subReturnFunc
}

func lexDoubleQuote(l *Lexer) StateFn {
	// We have consumed the first quote before entering this state.
	for {
		c := l.next()

		switch c {
		case EOFRune:
			panic(ErrQuotedString) //TODO: Dont make this panic
		case '$':
			l.subReturnFunc = lexDoubleQuote
			return lexSubstitution
		case '"':
			return lexWord
		default:
			l.buf.WriteRune(c)
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
			l.buf.WriteRune(c)
		}
	}
}
