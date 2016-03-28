package main

import (
	"fmt"

	"gopkg.in/logex.v1"

	"github.com/danwakefield/gosh/variables"
)

type Parser struct {
	lexer       *Lexer
	lastLexItem LexItem
	pushBack    bool
}

func NewParser(input string) *Parser {
	return &Parser{
		lexer: NewLexer(input),
	}
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
	logex.Panic("Expected any of: ", expected, ": got:", got)
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
	p.lexer.CheckNewline = false
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
		return p.list(1)
	}
}

func (p *Parser) list(newlineFlag int) Node {
	// TODO: Change newlineFlag to be self documenting.
	// Actually pass in something descriptive
	logex.Debugf("Enter '%d'\n", newlineFlag)
	defer logex.Debug("Exit\n")
	nodes := NodeList{}

	p.lexer.CheckAlias = true
	p.lexer.CheckNewline = true
	p.lexer.CheckKeyword = true
	if newlineFlag == 2 && TokenEndsList[p.peekToken()] {
		return nil
	}
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
			p.lexer.CheckAlias = true
			p.lexer.CheckNewline = true
			p.lexer.CheckKeyword = true
			if TokenEndsList[p.peekToken()] {
				return nodes
			}
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
		p.lexer.CheckAlias = true
		p.lexer.CheckNewline = false
		p.lexer.CheckKeyword = true
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
		logex.Pretty(tok)
		logex.Fatal("Could not understand ^")
	case TIf:
		n := NodeIf{}
		// We need a copy of the orignal pointer to return as the head of the if chain
		ifHead := &n
		p.lexer.CheckNewline = true
		ifCondition := p.list(0)
		n.Condition = &ifCondition
		p.expect(TThen)
		n.Body = p.list(0)

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
	case TFor:
		tok = p.next()
		if tok.Tok != TWord || tok.Quoted || !variables.IsGoodName(tok.Val) {
			logex.Panic(fmt.Sprintf("Bad for loop variable name: '%s'", tok.Val))
		}
		n := NodeFor{Args: []Arg{}}
		n.LoopVar = tok.Val

		p.lexer.CheckAlias = true
		p.lexer.CheckNewline = false
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
			n.Args = append(n.Args, Arg{Raw: tok.Val, Subs: tok.Subs})
		}

		p.lexer.CheckAlias = true
		p.lexer.CheckNewline = true
		p.lexer.CheckKeyword = true

		p.expect(TDo)
		n.Body = p.list(0)
		p.expect(TDone)
		returnNode = n
	case TCase:
		n := NodeCase{Cases: []NodeCaseList{}}
		tok = p.next()
		if tok.Tok != TWord {
			logex.Panic("Expected an expression after case")
		}
		n.Expr = Arg{Raw: tok.Val, Subs: tok.Subs}

		p.lexer.CheckAlias = true
		p.lexer.CheckNewline = true
		p.lexer.CheckKeyword = true
		p.expect(TIn)

		for {
			p.lexer.CheckAlias = false
			p.lexer.CheckNewline = true
			p.lexer.CheckKeyword = true
			tok = p.next()
			if tok.Tok == TEsac {
				break
			}
			if tok.Tok == TLeftParen {
				p.lexer.CheckAlias = false
				p.lexer.CheckNewline = true
				p.lexer.CheckKeyword = true
				tok = p.next()
				// Consume LeftParen if it exists
				logex.Debug("Consume left Paren")
			}
			ncl := NodeCaseList{Patterns: []Arg{}}

			for {
				ncl.Patterns = append(ncl.Patterns, Arg{Raw: tok.Val, Subs: tok.Subs})
				if p.hasNextToken(TPipe) {
					tok = p.next()
				} else {
					break
				}
			}
			logex.Pretty(ncl)
			p.expect(TRightParen)
			ncl.Body = p.list(2)

			n.Cases = append(n.Cases, ncl)

			p.lexer.CheckAlias = false
			p.lexer.CheckNewline = true
			p.lexer.CheckKeyword = true
			tok = p.next()
			if tok.Tok == TEsac {
				break
			} else if tok.Tok == TEndCase {
				continue
			} else {
				ExitShellWithMessage(
					ExitFailure,
					fmt.Sprintf("Expected ';;' or 'esac' on line %d", tok.LineNo),
				)
			}
		}
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

// simpleCommand
func (p *Parser) simpleCommand() NodeCommand {
	logex.Debugf("Enter\n")
	tok := p.next()
	assignments := []string{}
	args := []Arg{}
	startLine := tok.LineNo
	assignmentAllowed := true

	p.lexer.CheckAlias = true
	p.lexer.CheckNewline = false
	p.lexer.CheckKeyword = false

OuterLoop:
	for {
		switch tok.Tok {
		case TWord:
			if assignmentAllowed && variables.IsAssignment(tok.Val) {
				assignments = append(assignments, tok.Val)
				p.lexer.CheckAlias = false
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
