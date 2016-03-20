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

	SkipNewlines bool
}

func NewParser(input string) *Parser {
	return &Parser{
		y: NewLexer(input),
	}
}

func (p *Parser) next() LexItem {
	if p.pushBack {
		p.pushBack = false
		logex.Debugf("Token [Re]read:")
		// logex.Struct(p.lastLexItem)
		return p.lastLexItem
	}
	li := p.y.NextLexItem()
	p.lastLexItem = li
	logex.Debugf("Token read:")
	// logex.Struct(p.lastLexItem)
	if p.SkipNewlines && li.Tok == TNewLine {
		logex.Debug("Abandon Newline")
		return p.next()
	}
	return li
}

func (p *Parser) expect(expected ...Token) {
	got := p.next()
	for _, expect := range expected {
		if expect == got.Tok {
			logex.Debugf("Success %s", expect)
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
	logex.Debug("Enter\n")
	defer logex.Debug("Exit\n")
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

	return r
}

func (p *Parser) list() Node {
	return p.andOr()
}

func (p *Parser) andOr() Node {
	return p.pipeline()
}

func (p *Parser) pipeline() Node {
	logex.Debug("Enter\n")
	defer logex.Debug("Exit\n")
	negate := false

	if p.hasNextToken(TNot) {
		negate = !negate
	}

	returnNode := p.command()

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
		logex.Fatal(fmt.Sprintf("Command Doesnt understand\n %#v\n Token: %s", tok, tok.Tok))
	case TIf:
		n := NodeIf{}
		// We need a copy of the orignal pointer to return as the head of the if chain
		ifHead := &n
		p.SkipNewlines = true
		ifCondition := p.simpleCommand() //TODO: list(0)
		n.Condition = &ifCondition
		p.expect(TThen)
		n.Body = p.simpleCommand() //TODO: list(0)

		// Elif's
		for {
			if p.hasNextToken(TElif) {
				nelif := NodeIf{}
				ifCondition = p.simpleCommand() //TODO: list(0)
				nelif.Condition = &ifCondition
				p.expect(TThen)
				nelif.Body = p.simpleCommand() //TODO: list(0)
				n.Else = &nelif
				n = nelif
			} else {
				break
			}
			/*
			* list(0) means that TSEMI/TNL is advanced and we dont have to
			* expect it in if, while, etc.
			 */
		}

		if p.hasNextToken(TElse) {
			nelse := NodeIf{}
			nelse.Body = p.simpleCommand()
			n.Else = &nelse
		}

		p.expect(TFi)
		p.SkipNewlines = false
		returnNode = *ifHead
	case TWhile, TUntil:
		n := NodeLoop{}
		n.Type = NWhile // While is more common
		if tok.Tok == TUntil {
			n.Type = NUntil
		}

		p.SkipNewlines = true
		n.Condition = p.simpleCommand() //TODO: list(0)
		p.expect(TDo)
		n.Body = p.simpleCommand() //TODO: list(0)
		p.expect(TDone)

		p.SkipNewlines = false
		returnNode = n
	case TBegin:
		returnNode = p.simpleCommand() //TODO: list(0)
		p.expect(TEnd)
	case TWord:
		p.backup()
		returnNode = p.simpleCommand()
	}

	return returnNode
}

func (p *Parser) simpleCommand() NodeCommand {
	logex.Debugf("Enter\n")
	tok := p.next()
	assignments := []string{}
	args := []Arg{}
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
				args = append(args, Arg{Raw: tok.Val})
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
