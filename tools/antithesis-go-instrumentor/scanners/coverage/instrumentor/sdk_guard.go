package instrumentor

import (
	"go/ast"
	"strconv"

	"github.com/antithesishq/antithesis-sdk-go/tools/antithesis-go-instrumentor/common"
)

// sdkEnabledConst is the name of the SDK constant that reports whether the SDK
// was compiled in. It is true in every build that does not set the
// no_antithesis_sdk build tag, so an "if assert.Enabled" guard has a condition
// the Go compiler folds away.
const sdkEnabledConst = "Enabled"

// assertPackageAliases returns every identifier a file binds the SDK's assert
// package to, or nil if it does not import it in a form we can resolve.
//
// A file may import one package under several names, so every import is
// considered. A blank import cannot qualify a selector, and a dot import leaves
// a bare "Enabled" identifier that this AST-only pass has no way to attribute
// to the SDK; both are skipped without giving up on the rest of the imports.
func assertPackageAliases(f *ast.File) map[string]bool {
	if f == nil {
		return nil
	}
	var aliases map[string]bool
	for _, spec := range f.Imports {
		if spec.Path == nil {
			continue
		}
		// Import paths may be raw-string quoted, so compare the decoded value
		// rather than the literal spelling.
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil || path != common.AssertPackageName() {
			continue
		}
		alias := common.ASSERT_PACKAGE
		if spec.Name != nil {
			switch spec.Name.Name {
			case "_", ".":
				continue
			default:
				alias = spec.Name.Name
			}
		}
		if aliases == nil {
			aliases = map[string]bool{}
		}
		aliases[alias] = true
	}
	return aliases
}

// isSDKEnabledGuard reports whether cond is exactly "<alias>.Enabled", naming
// the SDK constant rather than something that merely spells the same way.
//
// Instrumenting the "not taken" side of such a guard would record a coverage
// edge the compiler is guaranteed to discard, leaving a row in the symbol table
// that can never be notified. Matching has to stay exact for the same reason it
// is worth doing at all: a false positive deletes a real edge.
func isSDKEnabledGuard(cond ast.Expr, aliases map[string]bool) bool {
	if len(aliases) == 0 || cond == nil {
		return false
	}
	sel, ok := cond.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil || sel.Sel.Name != sdkEnabledConst {
		return false
	}
	qualifier, ok := sel.X.(*ast.Ident)
	if !ok || !aliases[qualifier.Name] {
		return false
	}
	// Both back ends parse without parser.SkipObjectResolution, so go/parser
	// populates Obj for identifiers declared in the file. A package qualifier
	// resolves to nothing; a non-nil Obj means a local variable, parameter or
	// field shadows the import, and the condition is a genuine runtime value
	// whose not-taken edge must be recorded.
	return qualifier.Obj == nil
}
