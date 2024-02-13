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
// Each function also takes a parameter called values. This parameter allows you to optionally provide a key-value map of context information that will be viewable in the [details] tab for any example or counterexample of the associated property.
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

import (
	"fmt"
)

type assertInfo struct {
	Hit        bool           `json:"hit"`
	MustHit    bool           `json:"must_hit"`
	AssertType string         `json:"assert_type"`
	Expecting  bool           `json:"expecting"`
	Category   string         `json:"category"`
	Message    string         `json:"message"`
	Condition  bool           `json:"condition"`
	Id         string         `json:"id"`
	Location   *locationInfo  `json:"location"`
	Details    map[string]any `json:"details"`
}

type wrappedAssertInfo struct {
	A *assertInfo `json:"antithesis_assert"`
}

// --------------------------------------------------------------------------------
// Assertions
// --------------------------------------------------------------------------------
const was_hit = true
const must_be_hit = true
const optionally_hit = false
const expecting_true = true

const universal_test = "every"
const existential_test = "some"
const reachability_test = "none"

// Assert that condition is true every time this function is called, AND that it is called at least once. This test property will be viewable in the "Antithesis SDK: Always" group of your triage report.
func Always(message string, condition bool, values map[string]any) {
	location_info := newLocationInfo(offsetAPICaller)
	assertImpl(message, condition, values, location_info, was_hit, must_be_hit, expecting_true, universal_test)
}

// Assert that condition is true every time this function is called. Unlike the Always function, the test property spawned by AlwaysOrUnreachable will not be marked as failing if the function is never invoked. This test property will be viewable in the "Antithesis SDK: Always" group of your triage report.
func AlwaysOrUnreachable(message string, condition bool, values map[string]any) {
	location_info := newLocationInfo(offsetAPICaller)
	assertImpl(message, condition, values, location_info, was_hit, optionally_hit, expecting_true, universal_test)
}

// Assert that condition is true at least one time that this function was called. The test property spawned by Sometimes will be marked as failing if this function is never called, or if condition is false every time that it is called. This test property will be viewable in the "Antithesis SDK: Sometimes" group.
func Sometimes(message string, condition bool, values map[string]any) {
	location_info := newLocationInfo(offsetAPICaller)
	assertImpl(message, condition, values, location_info, was_hit, must_be_hit, expecting_true, existential_test)
}

// Assert that a line of code is never reached. The test property spawned by Unreachable will be marked as failing if this function is ever called. This test property will be viewable in the "Antithesis SDK: Reachablity assertions" group.
func Unreachable(message string, values map[string]any) {
	location_info := newLocationInfo(offsetAPICaller)
	assertImpl(message, true, values, location_info, was_hit, optionally_hit, expecting_true, reachability_test)
}

// Assert that a line of code is reached at least once. The test property spawned by Reachable will be marked as failing if this function is never called. This test property will be viewable in the "Antithesis SDK: Reachablity assertions" group.
func Reachable(message string, values map[string]any) {
	location_info := newLocationInfo(offsetAPICaller)
	assertImpl(message, true, values, location_info, was_hit, must_be_hit, expecting_true, reachability_test)
}

// This is a low-level method designed to be used by third-party frameworks. Regular users of the assert package should not call it.
func AssertRaw(message string, cond bool, values map[string]any, classname, funcname, filename string, line int, hit bool, must_hit bool, expecting bool, assert_type string) {
	assertImpl(message, cond, values, &locationInfo{classname, funcname, filename, line, columnUnknown}, hit, must_hit, expecting, assert_type)
}

func assertImpl(message string, cond bool, values map[string]any, loc *locationInfo, hit bool, must_hit bool, expecting bool, assert_type string) {
	message_key := makeKey(loc)
	tracker_entry := assert_tracker.get_tracker_entry(message_key)

	aI := &assertInfo{
		Hit:        hit,
		MustHit:    must_hit,
		AssertType: assert_type,
		Expecting:  expecting,
		Category:   "",
		Message:    message,
		Condition:  cond,
		Id:         message_key,
		Location:   loc,
		Details:    values,
	}

	tracker_entry.emit(aI)
}

func makeKey(loc *locationInfo) string {
	return fmt.Sprintf("%s|%d|%d", loc.Filename, loc.Line, loc.Column)
}
