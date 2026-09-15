//go:build enable_antithesis_sdk

package assert

// Enabled reports whether the Antithesis SDK was compiled into this program,
// that is, whether the program was built with the enable_antithesis_sdk build
// tag.
//
// It says nothing about where the program is running. An enabled build outside
// the Antithesis environment still reports true; the SDK simply falls back to a
// local or silent output handler. Use it only to decide whether SDK calls will
// do anything at all.
//
// # Why it exists
//
// Building without -tags=enable_antithesis_sdk leaves every assertion as an
// empty function, but those functions are still ordinary Go calls, so their
// arguments are evaluated at the call site. Work done solely to produce a
// condition or a details map is therefore still paid for in a build that has no
// SDK compiled in:
//
//	// snapshots twice and allocates the map, even without the SDK
//	assert.Always(svc.SnapshotState().Valid(), "state is valid",
//		map[string]any{"state": svc.SnapshotState()})
//
// Because Enabled is a constant, guarding that work makes the compiler discard
// it entirely:
//
//	if assert.Enabled {
//		st := svc.SnapshotState()
//		assert.Always(st.Valid(), "state is valid", map[string]any{"state": st})
//	}
//
// Assertions inside the guard are still discovered by the instrumentor and
// still registered in the generated assertion catalog, so no test property is
// lost by using it.
//
// # Write the guard in the positive form
//
// Prefer a plain "if assert.Enabled { ... }" with no else branch. The coverage
// instrumentor recognizes exactly that shape and skips the "not taken" edge it
// would otherwise synthesize for it — an edge that could never be reached,
// since the condition is a constant. Negated guards such as
// "if !assert.Enabled { return }", guards with an explicit else, and guards
// written as comparisons are all correct Go, but the instrumentor does not
// special-case them and they leave an unreachable edge behind.
const Enabled = true
