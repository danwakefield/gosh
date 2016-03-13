package main

import (
	"fmt"

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
		fmt.Printf("Parser.next: Reread '%s'\n", p.lastLexItem)
		return p.lastLexItem
	}
	li := p.y.NextLexItem()
	p.lastLexItem = li
	fmt.Printf("Parser.next: '%s'\n", p.lastLexItem)
	return li
}

func (p *Parser) hasNextToken(want Token) bool {
	tok := p.next()
	if tok.Tok == want {
		return true
	}
	p.pushBack = true
	return false
}

func (p *Parser) Parse() Node {
	fmt.Printf("Parser.Parse: Enter\n")
	tok := p.next()
	var r Node // Return Node

	switch tok.Tok {
	case TEOF:
		r = NodeEOF{}
	case TNewLine:
		r = nil
	default:
		p.pushBack = true
		r = p.list()
	}

	fmt.Printf("Parser.Parse: Exit\n")
	return r
}

func (p *Parser) list() Node {
	return p.andOr()
}

func (p *Parser) andOr() Node {
	return p.pipeline()
}

func (p *Parser) pipeline() Node {
	fmt.Printf("Parser.pipeline: Enter\n")
	negate := false

	if p.hasNextToken(TNot) {
		negate = !negate
	}

	n1 := p.command()

	if negate {
		return NodeNegate{N: n1}
	}
	fmt.Printf("Parser.pipeline: Exit\n")
	return n1
}

func (p *Parser) command() Node {
	fmt.Printf("Parser.command: Enter\n")
	tok := p.next()
	var r Node

	switch tok.Tok {
	default:
		panic(fmt.Sprintf("Command Doesnt understand\n %#v\n Token: %s", tok, tok.Tok))
	case TWord:
		p.pushBack = true
		r = p.simpleCommand()
	}

	fmt.Printf("Parser.command: Exit\n")
	return r
}

func (p *Parser) simpleCommand() Node {
	fmt.Printf("Parser.simpleCommand: Enter\n")
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
			p.pushBack = true
			break OuterLoop
		}
		tok = p.next()
	}

	n := NodeCommand{}
	n.Assign = assignments
	n.Args = args
	n.LineNo = startLine
	fmt.Printf("Parser.simpleCommand: Exit\n")
	return n
}
