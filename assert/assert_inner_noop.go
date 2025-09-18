//go:build no_antithesis_sdk

package assert

// Use this function instead of the standard assertion function when building a wrapper function.
// Always asserts that condition is true every time this function is called, and that it is called at least once. The corresponding test property will be viewable in the Antithesis SDK: Always group of your triage report.
func AlwaysInner(condition bool, message string, details map[string]any) {}

// Use this function instead of the standard assertion function when building a wrapper function.
// AlwaysOrUnreachable asserts that condition is true every time this function is called. The corresponding test property will pass if the assertion is never encountered (unlike Always assertion types). This test property will be viewable in the “Antithesis SDK: Always” group of your triage report.
func AlwaysOrUnreachableInner(condition bool, message string, details map[string]any) {}

// Use this function instead of the standard assertion function when building a wrapper function.
// Sometimes asserts that condition is true at least one time that this function was called. (If the assertion is never encountered, the test property will therefore fail.) This test property will be viewable in the “Antithesis SDK: Sometimes” group.
func SometimesInner(condition bool, message string, details map[string]any) {}

// Use this function instead of the standard assertion function when building a wrapper function.
// Unreachable asserts that a line of code is never reached. The corresponding test property will fail if this function is ever called. (If it is never called the test property will therefore pass.) This test property will be viewable in the “Antithesis SDK: Reachablity assertions” group.
func UnreachableInner(message string, details map[string]any) {}

// Use this function instead of the standard assertion function when building a wrapper function.
// Reachable asserts that a line of code is reached at least once. The corresponding test property will pass if this function is ever called. (If it is never called the test property will therefore fail.) This test property will be viewable in the “Antithesis SDK: Reachablity assertions” group.
func ReachableInner(message string, details map[string]any) {}

/* Rich Assertions */

// Use this function instead of the standard assertion function when building a wrapper function.
// Equivalent to asserting Always(left > right, message, details). Information about left and right will automatically be added to the details parameter, with keys left and right. If you use this function for assertions that compare numeric quantities, you may help Antithesis find more bugs.
func AlwaysGreaterThanInner[T Number](left, right T, message string, details map[string]any) {}

// Use this function instead of the standard assertion function when building a wrapper function.
// Equivalent to asserting Always(left >= right, message, details). Information about left and right will automatically be added to the details parameter, with keys left and right. If you use this function for assertions that compare numeric quantities, you may help Antithesis find more bugs.
func AlwaysGreaterThanOrEqualToInner[T Number](left, right T, message string, details map[string]any) {
}

// Use this function instead of the standard assertion function when building a wrapper function.
// Equivalent to asserting Sometimes(T left > T right, message, details). Information about left and right will automatically be added to the details parameter, with keys left and right. If you use this function for assertions that compare numeric quantities, you may help Antithesis find more bugs.
func SometimesGreaterThanInner[T Number](left, right T, message string, details map[string]any) {}

// Use this function instead of the standard assertion function when building a wrapper function.
// Equivalent to asserting Sometimes(T left >= T right, message, details). Information about left and right will automatically be added to the details parameter, with keys left and right. If you use this function for assertions that compare numeric quantities, you may help Antithesis find more bugs.
func SometimesGreaterThanOrEqualToInner[T Number](left, right T, message string, details map[string]any) {
}

// Use this function instead of the standard assertion function when building a wrapper function.
// Equivalent to asserting Always(left < right, message, details). Information about left and right will automatically be added to the details parameter, with keys left and right. If you use this function for assertions that compare numeric quantities, you may help Antithesis find more bugs.
func AlwaysLessThanInner[T Number](left, right T, message string, details map[string]any) {}

// Use this function instead of the standard assertion function when building a wrapper function.
// Equivalent to asserting Always(left <= right, message, details). Information about left and right will automatically be added to the details parameter, with keys left and right. If you use this function for assertions that compare numeric quantities, you may help Antithesis find more bugs.
func AlwaysLessThanOrEqualToInner[T Number](left, right T, message string, details map[string]any) {
}

// Use this function instead of the standard assertion function when building a wrapper function.
// Equivalent to asserting Sometimes(T left < T right, message, details). Information about left and right will automatically be added to the details parameter, with keys left and right. If you use this function for assertions that compare numeric quantities, you may help Antithesis find more bugs.
func SometimesLessThanInner[T Number](left, right T, message string, details map[string]any) {}

// Use this function instead of the standard assertion function when building a wrapper function.
// Equivalent to asserting Sometimes(T left <= T right, message, details). Information about left and right will automatically be added to the details parameter, with keys left and right. If you use this function for assertions that compare numeric quantities, you may help Antithesis find more bugs.
func SometimesLessThanOrEqualToInner[T Number](left, right T, message string, details map[string]any) {
}

// Use this function instead of the standard assertion function when building a wrapper function.
// Asserts that every time this is called, at least one bool in named_bools is true. Equivalent to Always(named_bools[0].second || named_bools[1].second || ..., message, details). If you use this for assertions about the behavior of booleans, you may help Antithesis find more bugs. Information about named_bools will automatically be added to the details parameter, and the keys will be the names of the bools.
func AlwaysSomeInner(named_bools []NamedBool, message string, details map[string]any) {}

// Use this function instead of the standard assertion function when building a wrapper function.
// Asserts that at least one time this is called, every bool in named_bools is true. Equivalent to Sometimes(named_bools[0].second && named_bools[1].second && ..., message, details). If you use this for assertions about the behavior of booleans, you may help Antithesis find more bugs. Information about named_bools will automatically be added to the details parameter, and the keys will be the names of the bools.
func SometimesAllInner(named_bools []NamedBool, message string, details map[string]any) {}
