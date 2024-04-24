//go:build !no_antithesis_sdk

// This package is part of the [Antithesis Go SDK], which enables Go applications to integrate with the [Antithesis platform].
//
// The assert package allows you to define new [test properties] for your program or [workload].
//
// Code that uses this package should be instrumented with the [antithesis-go-generator] utility. This step is required for the Always, Sometime, and Reachable methods. It is not required for the Unreachable and AlwaysOrUnreachable methods, but it will improve the experience of using them.
//
// These functions are no-ops with minimal performance overhead when called outside of the Antithesis environment. However, if the environment variable ANTITHESIS_SDK_LOCAL_OUTPUT is set, these functions will log to the file pointed to by that variable using a structured JSON format defined [here]. This allows you to make use of the Antithesis assertions package in your regular testing, or even in production. In particular, very few assertions frameworks offer a convenient way to define [Sometimes assertions], but they can be quite useful even outside Antithesis.
//
// Each function in this package takes a parameter called message. This value of this parameter will become part of the name of the test property defined by the function, and will be viewable in your [triage report], so it should be human interpretable. Assertions in different parts of your code with the same message value will be grouped into the same test property, but if one of them fails you will be able to see which file and line number are associated with each failure.
//
// Each function also takes a parameter called details. This parameter allows you to optionally provide a key-value map of context information that will be viewable in the [details] tab for any example or counterexample of the associated property.
//
// [Antithesis Go SDK]: https://antithesis.com/docs/using_antithesis/sdk/go_sdk.html
// [Antithesis platform]: https://antithesis.com
// [test properties]: https://antithesis.com/docs/using_antithesis/properties.html
// [workload]: https://antithesis.com/docs/getting_started/workload.html
// [antithesis-go-generator]: https://antithesis.com/docs/using_antithesis/sdk/go_sdk.html#assertion-indexer
// [triage report]: https://antithesis.com/docs/reports/triage.html
// [details]: https://antithesis.com/docs/reports/triage.html#details
// [here]: https://antithesis.com/docs/using_antithesis/sdk/fallback_sdk.html
// [Sometimes assertions]: https://antithesis.com/docs/best_practices/sometimes_assertions.html

package assert

type assertInfo struct {
	Location    *locationInfo  `json:"location"`
	Details     map[string]any `json:"details"`
	AssertType  string         `json:"assert_type"`
	DisplayType string         `json:"display_type"`
	Message     string         `json:"message"`
	Id          string         `json:"id"`
	Hit         bool           `json:"hit"`
	MustHit     bool           `json:"must_hit"`
	Condition   bool           `json:"condition"`
}

type wrappedAssertInfo struct {
	A *assertInfo `json:"antithesis_assert"`
}

// --------------------------------------------------------------------------------
// Assertions
// --------------------------------------------------------------------------------
const (
	wasHit        = true
	mustBeHit     = true
	optionallyHit = false
	expectingTrue = true
)

const (
	universalTest    = "always"
	existentialTest  = "sometimes"
	reachabilityTest = "reachability"
)

const (
	alwaysDisplay              = "Always"
	alwaysOrUnreachableDisplay = "AlwaysOrUnreachable"
	sometimesDisplay           = "Sometimes"
	reachableDisplay           = "Reachable"
	unreachableDisplay         = "Unreachable"
)

// Assert that condition is true every time this function is called, AND that it is called at least once. This test property will be viewable in the "Antithesis SDK: Always" group of your triage report.
func Always(condition bool, message string, details map[string]any) bool {
	locationInfo := newLocationInfo(offsetAPICaller)
	id := makeKey(message, locationInfo)
	assertImpl(condition, message, details, locationInfo, wasHit, mustBeHit, universalTest, alwaysDisplay, id)
	return condition
}

// Assert that condition is true every time this function is called. Unlike the Always function, the test property spawned by AlwaysOrUnreachable will not be marked as failing if the function is never invoked. This test property will be viewable in the "Antithesis SDK: Always" group of your triage report.
func AlwaysOrUnreachable(condition bool, message string, details map[string]any) bool {
	locationInfo := newLocationInfo(offsetAPICaller)
	id := makeKey(message, locationInfo)
	assertImpl(condition, message, details, locationInfo, wasHit, optionallyHit, universalTest, alwaysOrUnreachableDisplay, id)
	return condition
}

// Assert that condition is true at least one time that this function was called. The test property spawned by Sometimes will be marked as failing if this function is never called, or if condition is false every time that it is called. This test property will be viewable in the "Antithesis SDK: Sometimes" group.
func Sometimes(condition bool, message string, details map[string]any) bool {
	locationInfo := newLocationInfo(offsetAPICaller)
	id := makeKey(message, locationInfo)
	assertImpl(condition, message, details, locationInfo, wasHit, mustBeHit, existentialTest, sometimesDisplay, id)
	return condition
}

// Assert that a line of code is never reached. The test property spawned by Unreachable will be marked as failing if this function is ever called. This test property will be viewable in the "Antithesis SDK: Reachablity assertions" group.
func Unreachable(message string, details map[string]any) {
	locationInfo := newLocationInfo(offsetAPICaller)
	id := makeKey(message, locationInfo)
	assertImpl(false, message, details, locationInfo, wasHit, optionallyHit, reachabilityTest, unreachableDisplay, id)
}

// Assert that a line of code is reached at least once. The test property spawned by Reachable will be marked as failing if this function is never called. This test property will be viewable in the "Antithesis SDK: Reachablity assertions" group.
func Reachable(message string, details map[string]any) {
	locationInfo := newLocationInfo(offsetAPICaller)
	id := makeKey(message, locationInfo)
	assertImpl(true, message, details, locationInfo, wasHit, mustBeHit, reachabilityTest, reachableDisplay, id)
}

// This is a low-level method designed to be used by third-party frameworks. Regular users of the assert package should not call it.
func AssertRaw(cond bool, message string, details map[string]any,
	classname, funcname, filename string, line int,
	hit bool, mustHit bool,
	assertType string, displayType string,
	id string,
) bool {
	assertImpl(cond, message, details,
		&locationInfo{classname, funcname, filename, line, columnUnknown},
		hit, mustHit,
		assertType, displayType,
		id)
	return cond
}

func assertImpl(cond bool, message string, details map[string]any,
	loc *locationInfo,
	hit bool, mustHit bool,
	assertType string, displayType string,
	id string,
) {
	trackerEntry := assertTracker.getTrackerEntry(id)

	aI := &assertInfo{
		Hit:         hit,
		MustHit:     mustHit,
		AssertType:  assertType,
		DisplayType: displayType,
		Message:     message,
		Condition:   cond,
		Id:          id,
		Location:    loc,
		Details:     details,
	}

	trackerEntry.emit(aI)
}

func makeKey(message string, _ *locationInfo) string {
	return message
}
