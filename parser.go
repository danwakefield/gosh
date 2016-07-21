package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/danwakefield/kisslog"

	"github.com/danwakefield/gosh/variables"
)

type Parser struct {
	lexer       *Lexer
	lastLexItem LexItem
	pushBack    bool
	log         kisslog.Logger
}

func NewParser(input string) *Parser {
	p := &Parser{}
	l := NewLexer(input, p)
	p.lexer = l
	p.log = kisslog.New("parser")
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
			p.log.Debug("Expect Successful: %s", expect)
			return
		}
	}
	p.log.Error("Unexpected Token: %s\nWanted one of %s", got, expected)
	os.Exit(1)
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
	p.lexer.CheckAlias = true
	p.lexer.IgnoreNewlines = false
	p.lexer.CheckKeyword = true
	tok := p.next()

	switch tok.Tok {
	case TEOF:
		return NodeEOF{}
	case TNewLine:
		// Looks like this is done in dash to allow for interactive shell use.
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
				p.log.Error("Unexpected Token: %s: %#v", tok.Tok, tok)
				os.Exit(1)
			}
			p.backup()
			return nodes
		}
	}
}

func (p *Parser) andOr() Node {
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
	tok := p.next()
	var returnNode Node

	switch tok.Tok {
	default:
		p.log.Info("command - unexpected token: %s: %#v\n", tok.Tok, tok)
		os.Exit(1)
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

// parseIf creates a single Node (NodeIf) which contains the condition and
// body of an if statement.
// NodeIf also contains an Else field which is an
// optional NodeIf.
// This Else-NodeIf can have an exectuable Condtion that will determine if the
// body will be executed. This is the 'elif' construct.
// It can also use NodeNoop as its condition. NodeNoop
// will always return success and the body subsequently executed. This is the
// 'else' construct.
func parseIf(p *Parser) Node {
	n := NodeIf{}
	// We take the address to simplify handling the 'elif' case
	ifHead := &n

	// We know we should have at least
	//   if <condition>; then
	//       <body>
	//   fi
	// since we have seen the 'if'
	p.lexer.IgnoreNewlines = true
	n.Condition = p.list(IgnoreNewlines)
	p.expect(TThen)
	n.Body = p.list(IgnoreNewlines)

	// Before checking for the 'fi' token we have to check for 'elif'
	for {
		if !p.hasNextToken(TElif) {
			break
		}
		nelif := NodeIf{}

		// 'elif's follow the same construction as else with the only
		// difference being the starting keyword.
		p.lexer.IgnoreNewlines = true
		nelif.Condition = p.list(IgnoreNewlines)
		p.expect(TThen)
		nelif.Body = p.list(IgnoreNewlines)

		// Assign the 'elif' to the last NodeIf. Since we took the address
		// of the first one earlier we can replace it and for subsequent
		// 'elif's.
		n.Else = &nelif
		n = nelif
	}

	if p.hasNextToken(TElse) {
		// When we see an 'else' token we construct the NodeIf with an
		// always true condition.
		nelse := NodeIf{}
		nelse.Condition = NodeNoop{}
		nelse.Body = p.list(IgnoreNewlines)
		n.Else = &nelse
	}

	p.expect(TFi)
	// Return the first NodeIf which references all 'elif's and 'else's
	return *ifHead
}

// parseCase returns a NodeCase that contains the conditions and
// bodies for each part of the 'case' construct. Both the patterns and
// expression to match against have to be expanded before comparison so they
// are stored as Args to contain this.
func parseCase(p *Parser) Node {
	n := NodeCase{Cases: []NodeCaseList{}}

	// Since we have just parsed the 'case' keyword all three lexer
	// flags are false. This means the expression to be matched can be any reserved
	// word or an alias all of which will be returned as TWords. Anything
	// else natively recognized, E.g Metacharacters like '(', are invalid.
	tok := p.next()
	if tok.Tok != TWord {
		p.log.Info("invalid match expression supplied to case on line %d: %s", tok.LineNo, tok.Val)
	}
	n.Expr = Arg{Raw: tok.Val, Subs: tok.Subs, Quoted: tok.Quoted}

	// XXX: Is CheckAlias needed here?
	p.lexer.CheckAlias = true
	p.lexer.IgnoreNewlines = true
	p.lexer.CheckKeyword = true
	p.expect(TIn)

	// We have
	//   case <expr> in
	// we now have to detect cases, their patterns and bodies and the end of
	// the case statement, indicated by the 'esac' token.
	//
	// patterns are in the form
	//   [(][<pattern>[|<pattern>]]) [<body>] ;;|esac
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
			// An empty pattern is possible but if we have a pattern it
			// is possible to have multiple separated by '|'s
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
		// We have this as it is possible for the case to consist of just
		//   <pattern>) ;;
		// In this case a NodeNoop is used as the body to get a success
		ncl.Body = p.list(AllowEmptyNode)

		n.Cases = append(n.Cases, ncl)

		p.lexer.CheckAlias = false
		p.lexer.IgnoreNewlines = true
		p.lexer.CheckKeyword = true
		tok = p.next()

		// a case statement can end with either ';;' or 'esac'.
		// The 'esac' also ends the case construct
		if tok.Tok == TEsac {
			p.lexer.IgnoreNewlines = false
			break
		} else if tok.Tok == TEndCase {
			continue
		} else {
			p.log.Error("Expected ';;' or 'esac' on line %d", tok.LineNo)
			os.Exit(1)
		}
	}

	return n
}

// parseFor return a NodeFor which contains the body of the for loop, the
// variable to assign into and a list of things to assign.
//
// XXX: This does not follow the shell spec, which allows omitting the in
// and then defaults to using all the set positional variables. E.g $1, $2
func parseFor(p *Parser) Node {
	tok := p.next()
	if tok.Tok != TWord || tok.Quoted || !variables.IsGoodName(tok.Val) {
		p.log.Error(fmt.Sprintf("Bad for loop variable name", kisslog.Attrs{
			"name": tok.Val,
			"line": tok.LineNo,
		}))
	}

	n := NodeFor{Args: []Arg{}}
	n.LoopVar = tok.Val

	p.lexer.CheckAlias = true
	p.lexer.IgnoreNewlines = false
	p.lexer.CheckKeyword = true

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
