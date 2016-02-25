package main

type Variable struct {
	Val string
	Set bool // Used to distinguish unset variables from variables with val=""
}

type VariableScope map[string]Variable

type Scope struct {
	scopes       []VariableScope
	currentScope int
}

func NewScope() *Scope {
	s := Scope{}
	s.scopes = []VariableScope{}
	s.scopes = append(s.scopes, VariableScope{})

	return &s
}

func (s *Scope) Push() {
	s.scopes = append(s.scopes, VariableScope{})
	s.currentScope++
}

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

// Unset deletes
func (s *Scope) Unset(name string) {
	for i := s.currentScope; i >= 0; i-- {
		_, found := s.scopes[i][name]
		if found {
			delete(s.scopes[i], name)
		}
	}
}
