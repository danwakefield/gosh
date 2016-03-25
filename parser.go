package main

import (
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
		return p.lastLexItem
	}
	p.lastLexItem = p.y.NextLexItem()

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
	logex.Panic("Expected any of: ", expected, "\n got:", got)
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

	switch tok.Tok {
	case TEOF:
		return NodeEOF{}
	case TNewLine:
		// Looks like this is done in dash to allow for interactive shell
		// use.
		return nil
	default:
		p.backup()
		return p.list(1)
	}
}

func (p *Parser) list(newlineFlag int) Node {
	logex.Debug("Enter\n")
	defer logex.Debug("Exit\n")
	nodes := NodeList{}

	p.y.CheckAlias = true
	p.y.CheckNewline = true
	p.y.CheckKeyword = true
	for {
		n := p.andOr()
		tok := p.next()

		if tok.Tok == TRedirection {
			//
		}
		nodes = append(nodes, n)

		switch tok.Tok {
		case TNewLine:
			if newlineFlag == 1 {
				return nodes
			}
			fallthrough
		case TBackground, TSemicolon:
			p.y.CheckAlias = true
			p.y.CheckNewline = true
			p.y.CheckKeyword = true
			tok = p.next()
			if TokenEndsList[tok.Tok] {
				p.backup()
				return nodes
			}
			p.backup()
		case TEOF:
			p.backup()
			return nodes
		default:
			if newlineFlag == 1 {
				logex.Panic("Unexpected Token:\n", tok)
			}
			p.backup()
			return nodes
		}
	}
}

func (p *Parser) andOr() Node {
	logex.Debug("Enter\n")
	defer logex.Debug("Exit\n")
	return p.pipeline()
}

func (p *Parser) pipeline() Node {
	logex.Debug("Enter\n")
	defer logex.Debug("Exit\n")
	negate := false

	if p.hasNextToken(TNot) {
		negate = !negate
		p.y.CheckAlias = true
		p.y.CheckNewline = false
		p.y.CheckKeyword = true
	}

	returnNode := p.command()
	// if p.hasNextToken(TPipe)
	// add commands to the pipeline.
	// Maybe change Eval signature so we can pass IO redirs through to
	// NodeCommand.

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
		logex.Struct(tok)
		logex.Fatal("Could not understand ^")
	case TIf:
		n := NodeIf{}
		// We need a copy of the orignal pointer to return as the head of the if chain
		ifHead := &n
		p.y.CheckNewline = true
		ifCondition := p.list(0)
		n.Condition = &ifCondition
		p.expect(TThen)
		n.Body = p.list(0)

		// Elif's
		for {
			if p.hasNextToken(TElif) {
				nelif := NodeIf{}
				ifCondition = p.list(0)
				nelif.Condition = &ifCondition
				p.expect(TThen)
				nelif.Body = p.list(0)
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
			nelse.Body = p.list(0)
			n.Else = &nelse
		}

		p.expect(TFi)
		returnNode = *ifHead
	case TWhile, TUntil:
		n := NodeLoop{}
		if tok.Tok == TWhile {
			n.IsWhile = true
		}

		n.Condition = p.list(0)
		p.expect(TDo)
		n.Body = p.list(0)
		p.expect(TDone)

		returnNode = n
	case TBegin:
		returnNode = p.list(0)
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

	p.y.CheckAlias = true
	p.y.CheckNewline = false
	p.y.CheckKeyword = false

OuterLoop:
	for {
		switch tok.Tok {
		case TWord:
			if assignmentAllowed && variables.IsAssignment(tok.Val) {
				assignments = append(assignments, tok.Val)
				p.y.CheckAlias = false
			} else {
				assignmentAllowed = false
				args = append(args, Arg{Raw: tok.Val, Subs: tok.Subs})
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
