package variables

import "testing"

func TestNewScope(t *testing.T) {
	s := NewScope()

	if s.currentScope != 0 {
		t.Errorf("currentScope should equal 0 when newly created")
	}

	if len(s.scopes) != 1 {
		t.Errorf("should be 1 root VariableScope inside scope")
	}
}

func TestScopeStack(t *testing.T) {
	s := NewScope()

	T := func(a, b int) {
		if s.currentScope != a {
			t.Errorf("Scope.currentScope should be '%d' not '%d'", a, b)
		}
		if len(s.scopes) != b {
			t.Errorf("len(Scope.scopes) should be '%d' not '%d'", a, b)
		}
	}

	s.Push()
	s.Push()
	T(2, 3)

	s.Pop()
	T(1, 2)

	s.Pop()
	T(0, 1)
	s.Pop()
	T(0, 1)
}

func TestScopeVariables(t *testing.T) {
	s := NewScope()

	s.Set("foo", "bar")
	v := s.Get("foo")
	if v.Val != "bar" {
		t.Errorf("Retrieved variable is not the same as stored")
	}

	s.Push()
	v = s.Get("foo")
	if v.Val != "bar" {
		t.Errorf("Retrieved variable is not the same as stored")
	}

	s.Unset("foo")
	v = s.Get("foo")
	if v.Val != "" {
		t.Errorf("unsetting variable did not work")
	}
}

func TestSetString(t *testing.T) {
	s := NewScope()

	s.SetString("foo=bar")
	v := s.Get("foo")
	if v.Val != "bar" {
		t.Errorf("SetString did not work")
	}

	// Check variable is split at first =
	s.SetString("bar=foo=baz")
	v = s.Get("bar")
	if v.Val != "foo=baz" {
		t.Errorf("SetString did not split variable string correctl")
	}
}
