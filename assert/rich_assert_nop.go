//go:build no_antithesis_sdk

package assert

func AlwaysGreaterThan[T Number](left, right T, message, details map[string]any)             {}
func AlwaysGreaterThanOrEqualTo[T Number](left, right T, message, details map[string]any)    {}
func SometimesGreaterThan[T Number](left, right T, message, details map[string]any)          {}
func SometimesGreaterThanOrEqualTo[T Number](left, right T, message, details map[string]any) {}
func AlwaysLessThan[T Number](left, right T, message, details map[string]any)                {}
func AlwaysLessThanOrEqualTo[T Number](left, right T, message, details map[string]any)       {}
func SometimesLessThan[T Number](left, right T, message, details map[string]any)             {}
func SometimesLessThanOrEqualTo[T Number](left, right T, message, details map[string]any)    {}

func AlwaysSome(pairs []Pairs, message, details map[string]any)   {}
func SometimesAll(pairs []Pairs, message, details map[string]any) {}
