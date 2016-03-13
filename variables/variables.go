package variables

import (
	"fmt"
	"strings"
)

type Variable struct {
	Val      string
	Set      bool
	ReadOnly bool
}

type ScopeOption int

const LocalScope ScopeOption = 1

type VarScope map[string]Variable

type Scope struct {
	scopes       []VarScope
	currentScope int
}

func NewScope() *Scope {
	s := Scope{}
	s.scopes = []VarScope{}
	s.scopes = append(s.scopes, VarScope{})

	return &s
}

// Push adds a VarScope to the scope stack.
func (s *Scope) Push() {
	s.scopes = append(s.scopes, VarScope{})
	s.currentScope++
}

// Pop removes the top VarScope from the scopes stack.
// Uses currentScope to always preseve the root scope.
func (s *Scope) Pop() {
	if s.currentScope > 0 {
		s.scopes = s.scopes[:s.currentScope]
		s.currentScope--
	}
}

// Set walks down the scope stack checking for an existing variable to update.
// If no variable of that name exists it is created in the root scope.
func (s *Scope) Set(name, val string, opts ...ScopeOption) {
	if len(opts) > 0 {
		// We only have Local option ATM forget checking them.
		s.scopes[s.currentScope][name] = Variable{Val: val, Set: true}
	}
	for i := s.currentScope; i >= 0; i-- {
		v, found := s.scopes[i][name]
		if found {
			if !v.ReadOnly {
				v.Val = val
				s.scopes[i][name] = v
				return
			}
			panic(fmt.Sprintf("'%s' is read only"))
		}
	}
	s.scopes[0][name] = Variable{Val: val, Set: true}
}

// SetString Sets a variable that is a single string in the form
// 'A=1'
func (s *Scope) SetString(input string, opts ...ScopeOption) {
	parts := strings.SplitN(input, "=", 2)
	if len(parts) != 2 {
		panic("SetString given a string not containing an assignment")
	}
	s.Set(parts[0], parts[1], opts...)
}

// Get walks down the scope stack and returns the variable if found.
// If it is not set an empty variable is returned.
func (s *Scope) Get(name string) Variable {
	for i := s.currentScope; i >= 0; i-- {
		val, found := s.scopes[i][name]
		if found {
			return val
		}
	}
	return Variable{}
}

// Unset sets the first variable encountered while walking down the
// stack to nil values. We need to do this since unsetting local variables
// still results in them masking set variables in outer scopes.
func (s *Scope) Unset(name string) {
	for i := s.currentScope; i >= 0; i-- {
		v, found := s.scopes[i][name]
		if found {
			v.Set = false
			v.Val = ""
			s.scopes[i][name] = v
			break
		}
	}
}

func (s *Scope) Environ() []string {
	// This cannot be the best way.
	flatMap := map[string]string{}
	for i := 0; i <= s.currentScope; i++ {
		for k, v := range s.scopes[i] {
			if v.Set {
				flatMap[k] = v.Val
			}
		}
	}

	environString := []string{}
	for k, v := range flatMap {
		environString = append(environString, fmt.Sprintf("%s=%s", k, v))
	}
	return environString
}
