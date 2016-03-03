package main

type Variable struct {
	Val string
	Set bool
}

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
func (s *Scope) Set(name, val string) {
	for i := s.currentScope; i >= 0; i-- {
		_, found := s.scopes[i][name]
		if found {
			v := s.scopes[i][name]
			v.Val = val
			s.scopes[i][name] = v
			return
		}
	}
	s.scopes[0][name] = Variable{Val: val, Set: true}
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
		_, found := s.scopes[i][name]
		if found {
			v := s.scopes[i][name]
			v.Set = false
			v.Val = ""
			s.scopes[i][name] = v
			break
		}
	}
}
