package instrumentor

import (
	"fmt"
	"go/ast"
)

// funcScope names the function whose body is being instrumented the way Go's
// own tooling does. A method's receiver type is its class. A function literal
// is numbered within its enclosing function -- Outer.func1, and Outer.func1.1
// when nested in another literal -- which is how the compiler names it in
// stack traces; a literal at package level is glob..func1, likewise.
type funcScope struct {
	class    string
	name     string
	literal  bool // a function literal: nested literals append ".N" rather than ".funcN"
	literals int  // function literals seen directly inside this scope so far
}

// funcScopes is the stack of enclosing functions. Its bottom frame stands for
// the package level, which the compiler calls "glob.".
type funcScopes []funcScope

func newFuncScopes() funcScopes { return funcScopes{{name: "glob."}} }

func (s *funcScopes) enterDecl(decl *ast.FuncDecl) {
	*s = append(*s, funcScope{class: receiverType(decl), name: decl.Name.Name})
}

func (s *funcScopes) enterLit() {
	parent := &(*s)[len(*s)-1]
	parent.literals++
	suffix := ".func"
	if parent.literal {
		suffix = "."
	}
	*s = append(*s, funcScope{
		class:   parent.class,
		name:    fmt.Sprintf("%s%s%d", parent.name, suffix, parent.literals),
		literal: true,
	})
}

func (s *funcScopes) leave() { *s = (*s)[:len(*s)-1] }

// current is the class and function an edge recorded now belongs to.
func (s funcScopes) current() (class, name string) {
	if len(s) == 1 {
		return "", "" // package level: no edge is ever recorded here
	}
	top := s[len(s)-1]
	return top.class, top.name
}

// receiverType is the base type name of a method's receiver: Reader for
// (r *Reader), (r Reader), or (r *Reader[T]); "" for a plain function.
func receiverType(decl *ast.FuncDecl) string {
	if decl.Recv == nil || len(decl.Recv.List) == 0 {
		return ""
	}
	t := decl.Recv.List[0].Type
	for {
		switch x := t.(type) {
		case *ast.StarExpr:
			t = x.X
		case *ast.ParenExpr:
			t = x.X
		case *ast.IndexExpr:
			t = x.X
		case *ast.IndexListExpr:
			t = x.X
		case *ast.Ident:
			return x.Name
		default:
			return ""
		}
	}
}

func isFuncLit(n ast.Node) bool {
	_, ok := n.(*ast.FuncLit)
	return ok
}
