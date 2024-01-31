// Package assert allows callers to configure test oracles for the [Antithesis testing platform].
//
// For full functionality, code should be indexed by the exigen command
// so that Antithesis can know what invocations to expect. This is needed
// for Always, Sometime, and Reachable. It will make reporting about
// Unreachable and AlwaysOrUnreachable more understandable.
//
// [Antithesis testing platform]: https://antithesis.com
package assert

import (
	"fmt"
	"os"
	"strings"

	"github.com/antithesishq/antithesis-sdk-go/internal"
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

// Assert that condition is true one or more times during a test. Callers of
// `Always` can see failures in two cases:
// 1. If this function is ever invoked with a `false` for the conditional or
// 2. If an "indexed" invocation of Always is not covered at least once.
// message will be used as a display name in reporting and should therefore be
// useful to a broad audience. The map of values is used to supply context useful
// for understanding the reason that condition had the value it did. For instance,
// in an asertion that x > 5, it could be helpful to send the value of x so failing
// cases can be better understood.
func Always(message string, condition bool, values map[string]any, options ...string) {
	location_info := newLocationInfo(offsetAPICaller)
	assertImpl(message, condition, values, location_info, was_hit, must_be_hit, expecting_true, universal_test, options...)
}

// Assert that condition is true if it is ever evaluated. Callers will not
// see a failure in their test if the condition is never evaluated.
// message will be used as a display name in reporting and should therefore be
// useful to a broad audience. The map of values is used to supply context useful
// for understanding the reason that condition had the value it did. For instance,
// in an asertion that x > 5, it could be helpful to send the value of x so failing
// cases can be better understood.
func AlwaysOrUnreachable(message string, condition bool, values map[string]any, options ...string) {
	location_info := newLocationInfo(offsetAPICaller)
	assertImpl(message, condition, values, location_info, was_hit, optionally_hit, expecting_true, universal_test, options...)
}

// Assert that condition is true at least once in a test. Callers that invoke Sometimes will
// only see an error if that particualr invocation is neven called with condtition true.
// message will be used as a display name in reporting and should therefore be
// useful to a broad audience. The map of values is used to supply context useful
// for understanding the reason that condition had the value it did. For instance,
// in an asertion that x > 5, it could be helpful to send the value of x so failing
// cases can be better understood.
func Sometimes(message string, condition bool, values map[string]any, options ...string) {
	location_info := newLocationInfo(offsetAPICaller)
	assertImpl(message, condition, values, location_info, was_hit, must_be_hit, expecting_true, existential_test, options...)
}

// Assert that some path of code is not taken. A failure will be raised if this
// function is ever called.
// message will be used as a display name in reporting and should therefore be
// useful to a broad audience. The map of values is used to supply context useful
// for understanding the reason that this code path was taken.
func Unreachable(message string, values map[string]any, options ...string) {
	location_info := newLocationInfo(offsetAPICaller)
	assertImpl(message, true, values, location_info, was_hit, optionally_hit, expecting_true, reachability_test, options...)
}

// Assert that some path of code is tested. If any call to Reachable is not
// invoked during the course of a test a failure will be noted.
// message will be used as a display name in reporting and should therefore be
// useful to a broad audience. The map of values is used to supply context useful
// for understanding the reason that this code path was taken.
func Reachable(message string, values map[string]any, options ...string) {
	location_info := newLocationInfo(offsetAPICaller)
	assertImpl(message, true, values, location_info, was_hit, must_be_hit, expecting_true, reachability_test, options...)
}

func assertImpl(message string, cond bool, values map[string]any, loc *locationInfo, hit bool, must_hit bool, expecting bool, assert_type string, options ...string) {
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

	var before, after, opt_name, opt_value string
	var found, did_apply bool

	for _, option := range options {
		// option should be key:value
		if before, after, found = strings.Cut(option, ":"); found {
			opt_name = strings.ToLower(strings.TrimSpace(before))
			opt_value = strings.TrimSpace(after)
			if did_apply = aI.applyOption(opt_name, opt_value); !did_apply {
				fmt.Fprintf(os.Stderr, "Unable to apply option %s(%q)\n", opt_name, opt_value)
			}
		}
		if !found {
			fmt.Fprintf(os.Stderr, "Unable to parse %q\n", option)
		}
	}

	tracker_entry.emit(aI)
}

func (aI *assertInfo) applyOption(opt_name string, opt_value string) bool {
	if opt_name == "id" {
		aI.Id = fmt.Sprintf("%s (%s)", aI.Message, opt_value)
		return true
	}
	return false
}

func makeKey(loc *locationInfo) string {
	return fmt.Sprintf("%s|%d|%d", loc.Filename, loc.Line, loc.Column)
}

func emit_assert(assert_info *assertInfo) error {
	return internal.Json_data(wrappedAssertInfo{assert_info})
}
