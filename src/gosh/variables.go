package main

type Variable struct {
	Val string
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
	s.scopes[0][name] = Variable{Val: val}
}

func (s *Scope) Get(name string) Variable {
	for i := s.currentScope; i >= 0; i-- {
		val, found := s.scopes[i][name]
		if found {
			return val
		}
	}
	return Variable{}
}

func (s *Scope) Unset(name string) {
	for i := s.currentScope; i >= 0; i-- {
		_, found := s.scopes[i][name]
		if found {
			delete(s.scopes[i], name)
		}
	}
}
