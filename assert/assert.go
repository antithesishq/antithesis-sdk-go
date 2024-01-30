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
const expecting_false = false

const universal_test = "every"
const existential_test = "some"
const reachability_test = "none"

// Always asserts that when this is evaluated
// the condition will always be true, and that this is evaluated at least once.
func Always(text string, cond bool, values map[string]any, options ...string) {
	location_info := newLocationInfo(offsetAPICaller)
	assertImpl(text, cond, values, location_info, was_hit, must_be_hit, expecting_true, universal_test, options...)
}

// AlwaysOrUnreachable asserts that when this is evaluated
// the condition will always be true, or that this is never reaached and evaluated.
func AlwaysOrUnreachable(text string, cond bool, values map[string]any, options ...string) {
	location_info := newLocationInfo(offsetAPICaller)
	assertImpl(text, cond, values, location_info, was_hit, optionally_hit, expecting_true, universal_test, options...)
}

// Sometimes asserts that when this is evaluated
// the condition will sometimes be true, and that this is evaluated at least once.
func Sometimes(text string, cond bool, values map[string]any, options ...string) {
	location_info := newLocationInfo(offsetAPICaller)
	assertImpl(text, cond, values, location_info, was_hit, must_be_hit, expecting_true, existential_test, options...)
}

// Unreachable asserts that this is never evaluated.
// This assertion will fail if it is evaluated.
func Unreachable(text string, values map[string]any, options ...string) {
	location_info := newLocationInfo(offsetAPICaller)
	assertImpl(text, true, values, location_info, was_hit, optionally_hit, expecting_true, reachability_test, options...)
}

// Reachable asserts that this is evaluated at least once.
// This assertion will fail if it is not evaluated, and otherwise will pass.
func Reachable(text string, values map[string]any, options ...string) {
	location_info := newLocationInfo(offsetAPICaller)
	assertImpl(text, true, values, location_info, was_hit, must_be_hit, expecting_true, reachability_test, options...)
}

func assertImpl(text string, cond bool, values map[string]any, loc *locationInfo, hit bool, must_hit bool, expecting bool, assert_type string, options ...string) {
	message_key := makeKey(loc)
	tracker_entry := assert_tracker.get_tracker_entry(message_key)

	aI := &assertInfo{
		Hit:        hit,
		MustHit:    must_hit,
		AssertType: assert_type,
		Expecting:  expecting,
		Category:   "",
		Message:    text,
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
