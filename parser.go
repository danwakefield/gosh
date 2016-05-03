package main

import (
	"fmt"
	"strings"

	"gopkg.in/logex.v1"

	"github.com/danwakefield/gosh/T"
	"github.com/danwakefield/gosh/variables"
)

type Parser struct {
	lexer       *Lexer
	lastLexItem LexItem
	pushBack    bool
}

func NewParser(input string) *Parser {
	p := &Parser{}
	l := NewLexer(input, p)
	p.lexer = l
	return p
}

func (p *Parser) next() LexItem {
	if p.pushBack {
		p.pushBack = false
		return p.lastLexItem
	}
	p.lastLexItem = p.lexer.NextLexItem()

	return p.lastLexItem
}

func (p *Parser) expect(expected ...Token) {
	got := p.next()
	for _, expect := range expected {
		if expect == got.Tok {
			logex.Debugf("Expect Successful: %s", expect)
			return
		}
	}
	logex.Fatal(fmt.Sprintf("Expected any of: %s\nGot: %s\n", expected, got))
}

func (p *Parser) backup() {
	p.pushBack = true
}

func (p *Parser) hasNextToken(want Token) bool {
	tok := p.next()
	if tok.Tok == want {
		return true
	}
	p.backup()
	return false
}

func (p *Parser) peekToken() Token {
	t := p.next()
	p.backup()
	return t.Tok
}

func (p *Parser) Parse() Node {
	logex.Debug("Enter\n")
	defer logex.Debug("Exit\n")
	p.lexer.CheckAlias = true
	p.lexer.IgnoreNewlines = false
	p.lexer.CheckKeyword = true
	tok := p.next()

	switch tok.Tok {
	case TEOF:
		return NodeEOF{}
	case TNewLine:
		// Looks like this is done in dash to allow for interactive shell
		// use.
		return nil
	default:
		p.backup()
		return p.list(ObserveNewlines)
	}
}

type NewlineFlag int

const (
	IgnoreNewlines  NewlineFlag = 0
	ObserveNewlines NewlineFlag = 1
	AllowEmptyNode  NewlineFlag = 2
)

func (p *Parser) list(nlf NewlineFlag) Node {
	logex.Debugf("Enter '%d'\n", nlf)
	defer logex.Debug("Exit\n")
	nodes := NodeList{}

	p.lexer.CheckAlias = true
	p.lexer.IgnoreNewlines = true
	p.lexer.CheckKeyword = true
	if nlf == AllowEmptyNode && TokenEndsList[p.peekToken()] {
		return NodeNoop{}
	}
	for {
		n := p.andOr()
		tok := p.next()

		nodes = append(nodes, n)

		switch tok.Tok {
		case TNewLine:
			if nlf == ObserveNewlines {
				return nodes
			}
			fallthrough
		case TBackground, TSemicolon:
			p.lexer.CheckAlias = true
			p.lexer.IgnoreNewlines = true
			p.lexer.CheckKeyword = true
			if TokenEndsList[p.peekToken()] {
				return nodes
			}
		case TEOF:
			p.backup()
			return nodes
		default:
			if nlf == ObserveNewlines {
				logex.Fatal(fmt.Sprintf("Unexpected Token: %s: %#v\n", tok.Tok, tok))
			}
			p.backup()
			return nodes
		}
	}
}

func (p *Parser) andOr() Node {
	logex.Debug("Enter\n")
	defer logex.Debug("Exit\n")
	var returnNode Node

	returnNode = p.pipeline()
	for {
		tok := p.next()
		if tok.Tok == TAnd || tok.Tok == TOr {
			n := NodeBinary{IsAnd: tok.Tok == TAnd}

			n.Left = returnNode

			p.lexer.CheckAlias = true
			p.lexer.IgnoreNewlines = true
			p.lexer.CheckKeyword = true
			n.Right = p.pipeline()

			returnNode = n
		} else {
			p.backup()
			break
		}
	}
	return returnNode
}

func (p *Parser) pipeline() Node {
	logex.Debug("Enter\n")
	defer logex.Debug("Exit\n")
	negate := false

	if p.hasNextToken(TNot) {
		negate = true
		p.lexer.CheckAlias = true
		p.lexer.IgnoreNewlines = false
		p.lexer.CheckKeyword = true
	}

	returnNode := p.command()

	if p.hasNextToken(TPipe) {
		n := NodePipe{Commands: NodeList{returnNode}}

		for {
			p.lexer.CheckAlias = true
			p.lexer.IgnoreNewlines = true
			p.lexer.CheckKeyword = true
			n.Commands = append(n.Commands, p.command())

			if !p.hasNextToken(TPipe) {
				break
			}
		}
		returnNode = n
	}

	if negate {
		return NodeNegate{N: returnNode}
	}
	return returnNode
}

func (p *Parser) command() Node {
	logex.Debug("Enter\n")
	defer logex.Debug("Exit\n")
	tok := p.next()
	var returnNode Node

	switch tok.Tok {
	default:
		logex.Fatal(fmt.Sprintf("command - unexpected token: %s\n%#v\n"), tok.Tok, tok)
	case TIf:
		returnNode = parseIf(p)
	case TWhile, TUntil:
		n := NodeLoop{}
		if tok.Tok == TWhile {
			n.IsWhile = true
		}

		n.Condition = p.list(IgnoreNewlines)
		p.expect(TDo)
		n.Body = p.list(IgnoreNewlines)
		p.expect(TDone)

		returnNode = n
	case TFor:
		returnNode = parseFor(p)
	case TCase:
		returnNode = parseCase(p)
	case TBegin:
		returnNode = p.list(IgnoreNewlines)
		p.expect(TEnd)
	case TWord:
		p.backup()
		returnNode = p.simpleCommand()
	}

	return returnNode
}

// simpleCommand
func (p *Parser) simpleCommand() Node {
	logex.Debugf("Enter\n")
	defer logex.Debugf("Exit\n")
	tok := p.next()
	assignments := map[string]Arg{}
	args := []Arg{}
	startLine := tok.LineNo
	assignmentAllowed := true

	p.lexer.CheckAlias = true
	p.lexer.IgnoreNewlines = false
	p.lexer.CheckKeyword = false

OuterLoop:
	for {
		switch tok.Tok {
		case TWord:
			if assignmentAllowed && variables.IsAssignment(tok.Val) {
				parts := strings.SplitN(tok.Val, "=", 2)
				assignments[parts[0]] = Arg{Raw: parts[1], Subs: tok.Subs, Quoted: tok.Quoted}
				p.lexer.CheckAlias = false
			} else {
				assignmentAllowed = false
				args = append(args, Arg{Raw: tok.Val, Subs: tok.Subs, Quoted: tok.Quoted})
			}
		case TLeftParen:
			if len(args) == 1 && len(assignments) == 0 {
				p.expect(TRightParen)
				name := args[0]
				if !variables.IsGoodName(name.Raw) {
					panic("Bad function name: " + name.Raw)
				}
				p.lexer.CheckAlias = true
				p.lexer.IgnoreNewlines = true
				p.lexer.CheckKeyword = true
				n := NodeFunction{}
				n.Body = p.command()
				n.Name = name.Raw
				return n
			}
			fallthrough
		default:
			p.backup()
			break OuterLoop
		}
		tok = p.next()
	}

	n := NodeCommand{}
	n.Assign = assignments
	n.Args = args
	n.LineNo = startLine
	return n
}

func parseIf(p *Parser) Node {
	n := NodeIf{}
	ifHead := &n

	p.lexer.IgnoreNewlines = true
	n.Condition = p.list(IgnoreNewlines)
	p.expect(TThen)
	n.Body = p.list(IgnoreNewlines)

	for {
		if !p.hasNextToken(TElif) {
			break
		}
		nelif := NodeIf{}

		p.lexer.IgnoreNewlines = true
		nelif.Condition = p.list(IgnoreNewlines)
		p.expect(TThen)
		nelif.Body = p.list(IgnoreNewlines)

		n.Else = &nelif
		n = nelif
	}

	if p.hasNextToken(TElse) {
		nelse := NodeIf{}
		nelse.Condition = NodeNoop{}
		nelse.Body = p.list(IgnoreNewlines)
		n.Else = &nelse
	}

	p.expect(TFi)
	return *ifHead
}

func parseCase(p *Parser) Node {
	n := NodeCase{Cases: []NodeCaseList{}}

	tok := p.next()
	if tok.Tok != TWord {
		logex.Fatal("Expected an expression after case")
	}
	n.Expr = Arg{Raw: tok.Val, Subs: tok.Subs, Quoted: tok.Quoted}

	p.lexer.CheckAlias = true
	p.lexer.IgnoreNewlines = true
	p.lexer.CheckKeyword = true
	p.expect(TIn)

	for {
		p.lexer.CheckAlias = false
		p.lexer.IgnoreNewlines = true
		p.lexer.CheckKeyword = true
		tok = p.next()

		if tok.Tok == TEsac {
			break
		} else if tok.Tok == TLeftParen {
			// Optional left bracket before patterns
			p.lexer.CheckAlias = false
			p.lexer.IgnoreNewlines = true
			p.lexer.CheckKeyword = true
			tok = p.next()
		}

		ncl := NodeCaseList{Patterns: []Arg{}}
		for {
			// We should always have one pattern so keep appending while the
			// next character is the pattern seperator TPipe
			if tok.Tok != TWord {
				p.backup()
				break
			}
			ncl.Patterns = append(ncl.Patterns, Arg{Raw: tok.Val, Subs: tok.Subs})
			if !p.hasNextToken(TPipe) {
				break
			}
			tok = p.next()
		}
		p.expect(TRightParen)
		ncl.Body = p.list(AllowEmptyNode)

		n.Cases = append(n.Cases, ncl)

		p.lexer.CheckAlias = false
		p.lexer.IgnoreNewlines = true
		p.lexer.CheckKeyword = true
		tok = p.next()

		if tok.Tok == TEsac {
			p.lexer.IgnoreNewlines = false
			break
		} else if tok.Tok == TEndCase {
			continue
		} else {
			ExitShellWithMessage(
				T.ExitFailure,
				fmt.Sprintf("Expected ';;' or 'esac' on line %d", tok.LineNo),
			)
		}
	}

	return n
}

func parseFor(p *Parser) Node {
	tok := p.next()
	if tok.Tok != TWord || tok.Quoted || !variables.IsGoodName(tok.Val) {
		logex.Fatal(fmt.Sprintf("Bad for loop variable name: '%s'", tok.Val))
	}

	n := NodeFor{Args: []Arg{}}
	n.LoopVar = tok.Val

	p.lexer.CheckAlias = true
	p.lexer.IgnoreNewlines = false
	p.lexer.CheckKeyword = true

	// Only deal with in blah for now.
	p.expect(TIn)
	for {
		tok = p.next()
		if tok.Tok != TWord {
			p.backup()
			p.expect(TNewLine, TSemicolon)
			break
		}
		n.Args = append(n.Args, Arg{Raw: tok.Val, Subs: tok.Subs, Quoted: tok.Quoted})
	}

	p.lexer.CheckAlias = true
	p.lexer.IgnoreNewlines = true
	p.lexer.CheckKeyword = true

	p.expect(TDo)
	n.Body = p.list(IgnoreNewlines)
	p.expect(TDone)

	return n
}
