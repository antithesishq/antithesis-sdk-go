package instrumentor

import (
	"go/ast"
)

type Stack []ast.Node

// IsEmpty: check if stack is empty
func (s *Stack) IsEmpty() bool {
	return len(*s) == 0
}

// Push a new integer onto the stack
func (s *Stack) Push(x ast.Node) {
	*s = append(*s, x)
}

// Pop: remove and return top element of stack, return false if stack is empty
func (s *Stack) Pop() (ast.Node, bool) {
	if s.IsEmpty() {
		return nil, false
	}

	i := len(*s) - 1
	x := (*s)[i]
	*s = (*s)[:i]

	return x, true
}

// Peek: return top element of stack, return false if stack is empty
func (s *Stack) Peek() (ast.Node, bool) {
	if s.IsEmpty() {
		return nil, false
	}

	i := len(*s) - 1
	x := (*s)[i]

	return x, true
}
