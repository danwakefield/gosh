package main

import (
	"bytes"
	"errors"
	"os"
	"unicode/utf8"

	"github.com/danwakefield/gosh/char"
	"github.com/danwakefield/kisslog"
)

const (
	EOFRune              rune = -1
	SentinalEscape       rune = 201
	SentinalSubstitution rune = 202
)

var (
	ErrQuotedString = errors.New("Unterminated quoted string")
)

type LexItem struct {
	Tok    Token
	Pos    int
	LineNo int
	Val    string `json:",omitempty"`
	Quoted bool
	Subs   []Substitution `json:",omitempty"`
}
type Lexer struct {
	position     int
	lastPosition int
	inputLength  int
	lineNo       int
	buffer       bytes.Buffer
	subs         []Substitution
	quoted       bool
	backslash    bool
	backupWidth  int
	input        string
	log          *kisslog.Logger

	IgnoreNewlines bool
	CheckAlias     bool
	CheckKeyword   bool
}

func NewLexer(input string) *Lexer {
	l := &Lexer{
		input:          input,
		inputLength:    len(input),
		subs:           []Substitution{},
		lineNo:         1,
		log:            kisslog.New("Lexer"),
		IgnoreNewlines: false,
		CheckAlias:     true,
		CheckKeyword:   true,
	}
	return l
}
func (l *Lexer) Next() (li LexItem) {
	defer func() {
		l.CheckAlias = false
		l.IgnoreNewlines = false
		l.CheckKeyword = false
	}()

	li = l.nextLexItem()

	for li.Tok == TNewLine && l.IgnoreNewlines {
		li = l.nextLexItem()
	}

	if li.Tok != TWord || li.Quoted {
		return li
	}

	// Check if words are keywords. E.g for
	// CheckKeyword flag disables as `for for in 1 2 3`
	// is valid
	if t, found := KeywordLookup[li.Val]; found && l.CheckKeyword {
		return LexItem{
			Tok:    t,
			Pos:    li.Pos,
			LineNo: li.LineNo,
			Val:    li.Val,
		}
	}

	if l.CheckAlias {

	}

	return li
}

func (l *Lexer) nextLexItem() LexItem {
	t := l.Start()

	li := LexItem{
		Tok:    t,
		Pos:    l.lastPosition,
		LineNo: l.lineNo,
		Val:    l.buffer.String(),
		Quoted: l.quoted,
		Subs:   l.subs,
	}

	l.lastPosition = l.position
	l.quoted = false
	l.subs = []Substitution{}
	l.buffer.Reset()

	return li
}

func (l *Lexer) nextChar() rune {
	if l.position >= l.inputLength {
		l.position++
		return EOFRune
	}
	var (
		r rune
		w int
	)
	for {
		r, w = utf8.DecodeRuneInString(l.input[l.position:])
		l.position += w
		l.backupWidth = w
		if r != '\x01' {
			break
		}
	}
	return r
}

func (l *Lexer) ignore() {
	l.lastPosition = l.position
}

func (l *Lexer) backup() {
	l.position -= l.backupWidth
	l.backupWidth = 0
}

func (l *Lexer) hasNext(r rune) bool {
	if r == l.nextChar() {
		return true
	}
	l.backup()
	return false
}

func (l *Lexer) Start() Token {
	for {
		c := l.nextChar()

		switch c {
		default:
			l.backup()
			return l.Word()
		case EOFRune:
			return TEOF
		case ' ', '\t':
			// Ignore whitespace
			l.ignore()
		case '#':
			// Consume comments upto newline / EOF
			for {
				c := l.nextChar()
				if c == '\n' || c == EOFRune {
					l.backup()
					l.ignore()
					// Break and let the main loop figure out NL vs EOF
					break
				}
			}
		case '\\':
			// Line continuation or escaped character
			if l.hasNext('\n') {
				l.lineNo++
				continue
			}
			l.quoted = true
			l.backslash = true
			return l.Word()
		case '\n':
			l.lineNo++
			return TNewLine
		case '&':
			if l.hasNext('&') {
				return TAnd
			} else {
				return TBackground
			}
		case '|':
			if l.hasNext('|') {
				return TOr
			} else {
				return TPipe
			}
		case ';':
			if l.hasNext(';') {
				return TEndCase
			} else {
				return TSemicolon
			}
		case '(':
			return TLeftParen
		case ')':
			return TRightParen
		}
	}

	return TEOF
}

func (l *Lexer) Word() Token {
OuterLoop:
	for {
		c := l.nextChar()

		if l.backslash {
			if c == EOFRune {
				l.backup()
				l.buffer.WriteRune('\\')
				break
			}
			if c == '\\' {
				l.buffer.WriteRune('\\')
				continue
			}
			l.buffer.WriteRune(SentinalEscape)
			l.buffer.WriteRune(c)
			l.backslash = false
			continue
		}

		switch c {
		case '\n', '\t', ' ', '<', '>', '(', ')', ';', '&', '|', EOFRune:
			// Characters that cause a word break
			l.backup()
			break OuterLoop
		case '\'':
			l.quoted = true
			l.SingleQuote()
		case '"':
			l.quoted = true
			l.DoubleQuote()
		case '`':
			l.BackQuote()
		case '$':
			l.Substitution()
		default:
			l.buffer.WriteRune(c)
		}
	}

	return TWord
}

func (l *Lexer) DoubleQuote() {
	// We have consumed the first quote before entering this state.
	for {
		c := l.nextChar()

		switch c {
		case EOFRune:
			panic(ErrQuotedString) //TODO: Dont make this panic
		case '$':
			l.Substitution()
		case '"':
			return
		case '\\':
			c = l.nextChar()
			switch c {
			case '\n':
				// Ignore an escaped literal newline
			case '\\', '$', '`', '"':
				l.buffer.WriteRune(c)
			default:
				l.backup()
				l.buffer.WriteRune('\\')
			}
		default:
			l.buffer.WriteRune(c)
		}
	}
}

func (l *Lexer) SingleQuote() {
	// We have consumed the first quote before entering this state.
	for {
		c := l.nextChar()

		switch c {
		case EOFRune:
			panic(ErrQuotedString) //TODO: Dont make this panic
		case '\'':
			return
		default:
			l.buffer.WriteRune(c)
		}
	}
}

func (l *Lexer) Substitution() {
	// Upon entering we have only read the '$'
	// Perform the lex of a single complete substitution before returning
	// control to the calling location
	c := l.nextChar()

	switch {
	default:
		l.buffer.WriteRune('$')
		l.backup()
	case c == '(':
		if l.hasNext('(') {
			l.Arith()
		} else {
			l.Subshell()
		}
	case char.IsFirstInVarName(c), char.IsDigit(c), char.IsSpecial(c):
		l.backup()
		l.VariableSimple()
	case c == '{':
		l.VariableComplex()
	}
}

func (l *Lexer) VariableSimple() {
	// When we enter this state we know we have at least one readable
	// char for the varname. That means that any character not valid
	// just terminates the parsing and we dont have to
	// worry about the case of an empty varname
	l.buffer.WriteRune(SentinalSubstitution)
	sv := SubVariable{SubType: VarSubNormal}
	varbuf := bytes.Buffer{}

	c := l.nextChar()
	switch {
	case char.IsSpecial(c):
		varbuf.WriteRune(c)
	case char.IsDigit(c):
		// Positional argv
		for {
			varbuf.WriteRune(c)
			c = l.nextChar()
			if !char.IsDigit(c) {
				l.backup()
				break
			}
		}
	case char.IsFirstInVarName(c):
		for {
			varbuf.WriteRune(c)
			c = l.nextChar()
			if !char.IsInVarName(c) {
				l.backup()
				break
			}
		}
	default:
		l.backup()
	}

	sv.VarName = varbuf.String()
	l.subs = append(l.subs, sv)
	return
}

func (l *Lexer) VariableComplex() {
	// Upon entering we have read the opening '{'
	l.buffer.WriteRune(SentinalSubstitution)
	sv := SubVariable{}
	varbuf := bytes.Buffer{}

	defer func() {
		// We defer this as there are multiple return points
		sv.VarName = varbuf.String()
		l.subs = append(l.subs, sv)
	}()

	if l.hasNext('#') {
		// The NParam Special Var
		if l.hasNext('}') {
			varbuf.WriteRune('#')
			return
		}

		// Length variable operator
		sv.SubType = VarSubLength
	}

	c := l.nextChar()
	switch {
	case char.IsSpecial(c):
		varbuf.WriteRune(c)
	case char.IsDigit(c):
		for {
			varbuf.WriteRune(c)
			c = l.nextChar()
			if !char.IsDigit(c) {
				l.backup()
				break
			}
		}
	case char.IsFirstInVarName(c):
		for {
			varbuf.WriteRune(c)
			c = l.nextChar()
			if !char.IsInVarName(c) {
				l.backup()
				break
			}
		}
	case c == EOFRune:
		l.backup()
		return
	}

	// Either a Enclosed variable '${foo}' or a length operation '${#foo}'
	if l.hasNext('}') {
		return
	}

	// Length operator should have returned since only ${#varname} is valid
	if sv.SubType == VarSubLength {
		l.log.Error("Line %d: Bad substitution (%s)", l.lineNo, l.input[l.lastPosition:l.position])
		os.Exit(1)
	}

	if l.hasNext(':') {
		sv.CheckNull = true
	}

	switch l.nextChar() {
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
		l.log.Error("Line %d: Bad substitution (%s)", l.lineNo, l.input[l.lastPosition:l.position])
		os.Exit(1)
	}

	// Read until '}'
	// In the future to support Nested vars etc create new sublexer from
	// l.input[l.pos:] and take the first lexitem as the sub val then adjust
	// this lexer's position and trash sublexer
	c = l.nextChar()
	subValBuf := bytes.Buffer{}
	for {
		if c == '}' {
			break
		}
		subValBuf.WriteRune(c)
		c = l.nextChar()
	}
	sv.SubVal = subValBuf.String()
}

func (l *Lexer) BackQuote() {
	l.log.Error("lexBackQuote - Not Implemented")
	os.Exit(2)
}

func (l *Lexer) Subshell() {
	l.ignore()

	p := NewParser(l.input[l.position:])
	ss := SubSubshell{}
	ss.N = p.list(AllowEmptyNode)

	// We have to explictitly get the next item to prevent race conditions
	// in accessing the lexers Pos and LineNo fields.
	x := p.lexer.nextLexItem()
	l.position += x.Pos
	l.lineNo += x.LineNo

	l.buffer.WriteRune(SentinalSubstitution)
	l.subs = append(l.subs, ss)

	return
}

func (l *Lexer) Arith() {
	// Upon entering this state we have read the '$(('
	l.buffer.WriteRune(SentinalSubstitution)
	arithBuf := bytes.Buffer{}
	sa := SubArith{}

	parenCount := 0
	for {
		c := l.nextChar()
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
}
