package main

import (
	"fmt"

	"gopkg.in/logex.v1"

	"github.com/danwakefield/gosh/variables"
)

type Parser struct {
	y           *Lexer
	lastLexItem LexItem
	pushBack    bool
}

func NewParser(input string) *Parser {
	return &Parser{
		y: NewLexer(input),
	}
}

func (p *Parser) next() LexItem {
	if p.pushBack {
		p.pushBack = false
		logex.Debugf("Token [Re]read: '%s'\n", p.lastLexItem)
		return p.lastLexItem
	}
	li := p.y.NextLexItem()
	p.lastLexItem = li
	logex.Debugf("Token read: %s'\n", p.lastLexItem)
	return li
}

func (p *Parser) expect(expected ...Token) {
	got := p.next()
	for _, expect := range expected {
		if expect == got {
			return
		}
	}
	logex.Fatal("Expected :", expected)
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

func (p *Parser) Parse() Node {
	logex.Debugf("Enter\n")
	tok := p.next()
	var r Node // Return Node

	switch tok.Tok {
	case TEOF:
		r = NodeEOF{}
	case TNewLine:
		r = nil
	default:
		p.backup()
		r = p.list()
	}

	logex.Debugf("Exit\n")
	return r
}

func (p *Parser) list() Node {
	return p.andOr()
}

func (p *Parser) andOr() Node {
	return p.pipeline()
}

func (p *Parser) pipeline() Node {
	logex.Debugf("Enter\n")
	negate := false

	if p.hasNextToken(TNot) {
		negate = !negate
	}

	n1 := p.command()

	if negate {
		return NodeNegate{N: n1}
	}
	logex.Debugf("Exit\n")
	return n1
}

func (p *Parser) command() Node {
	logex.Debugf("Enter\n")
	tok := p.next()
	var r Node

	switch tok.Tok {
	default:
		logex.Fatal(fmt.Sprintf("Command Doesnt understand\n %#v\n Token: %s", tok, tok.Tok))
	case TIf:
		n := NodeIf{}
		n.Condition = p.simpleCommand()
		p.expect(TThen)
		n.Body = p.simpleCommand()

		// Elif's
		for {
			if p.hasNextToken(TElif) {
			}
		}

	case TWord:
		p.backup()
		r = p.simpleCommand()
	}

	logex.Debugf("Exit\n")
	return r
}

func (p *Parser) simpleCommand() Node {
	logex.Debugf("Enter\n")
	tok := p.next()
	assignments := []string{}
	args := []CommandArg{}
	startLine := tok.LineNo
	assignmentAllowed := true

OuterLoop:
	for {
		switch tok.Tok {
		case TWord:
			if assignmentAllowed && variables.IsAssignment(tok.Val) {
				assignments = append(assignments, tok.Val)
			} else {
				assignmentAllowed = false
				args = append(args, CommandArg{Raw: tok.Val})
			}
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
	logex.Debugf("Exit\n")
	return n
}
