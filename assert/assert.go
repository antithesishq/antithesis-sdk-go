package assert

import (
  "encoding/json"
  "errors"
  "fmt"
  "github.com/antithesishq/antithesis-sdk-go/internal"
  "github.com/antithesishq/antithesis-sdk-go/local"
  "os"
  "strings"
)

type AssertInfo struct {
    Hit bool `json:"hit"`
    MustHit bool `json:"must_hit"`
    ExpectType string `json:"expect_type"`
    Expecting bool `json:"expecting"`
    Category string `json:"category"`
    Message string `json:"message"`
    Condition bool `json:"condition"`
    Id string `json:"id"`
    Location *LocationInfo `json:"location"`
    Details map[string]any `json:"details"`
}

type WrappedAssertInfo struct {
    A *AssertInfo `json:"ant_assert"`
}

type LocalLogAssertInfo struct {
    local.LocalLogInfo
    WrappedAssertInfo
}

// --------------------------------------------------------------------------------
// Version
// --------------------------------------------------------------------------------
func Version() string {
  return "0.1.0"
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
// const reachability_check "none"

// AlwaysTrue asserts that when this is evaluated
// the condition will always be true, and that this is evaluated at least once.
// Alternative name is Always()
func AlwaysTrue(text string, cond bool, values any, options ...string) {
  location_info := NewLocationInfo(OffsetAPICaller) 
  AssertImpl(text, cond, values, location_info, was_hit, must_be_hit, expecting_true, universal_test, options...)
}

// AlwaysTrueIfOccurs asserts that when this is evaluated
// the condition will always be true, or that this is never evaluated.
// Alternative name is UnreachableOrAlways()
func AlwaysTrueIfOccurs(text string, cond bool, values any, options ...string) {
  location_info := NewLocationInfo(OffsetAPICaller) 
  AssertImpl(text, cond, values, location_info, was_hit, optionally_hit, expecting_true, universal_test, options...)
}

// SometimesTrue asserts that when this is evaluated
// the condition will sometimes be true, and that this is evaluated at least once.
// Alternative name is Sometimes()
func SometimesTrue(text string, cond bool, values any, options ...string) {
  location_info := NewLocationInfo(OffsetAPICaller) 
  AssertImpl(text, cond, values, location_info, was_hit, must_be_hit, expecting_true, existential_test, options...)
}

// NeverOccurs asserts that this is never evaluated.
// This assertion will fail if it is evaluated.
// Alternative name is Unreachable()
func NeverOccurs(text string, values any, options ...string) {
  location_info := NewLocationInfo(OffsetAPICaller) 
  AssertImpl(text, false, values, location_info, was_hit, optionally_hit, expecting_true, universal_test, options...)
}

// SometimesOccurs asserts that this is evaluated at least once.
// This assertion will fail if it is not evaluated, and otherwise will pass.
// Alternative name is Reachable()
func SometimesOccurs(text string, values any, options ...string) {
  location_info := NewLocationInfo(OffsetAPICaller) 
  AssertImpl(text, true, values, location_info, was_hit, must_be_hit, expecting_true, existential_test, options...)
}

func AssertImpl(text string, cond bool, values any, loc *LocationInfo, hit bool, must_hit bool, expecting bool, expect_type string, options ...string) {
  message_key := makeKey(loc)
  tracker_entry := assert_tracker.get_tracker_entry(message_key)
  details_map := struct_to_map(values)

  aI := &AssertInfo{
      Hit: hit,
      MustHit: must_hit,
      ExpectType: expect_type,
      Expecting: expecting,
      Category: "",
      Message: text,
      Condition: cond,
      Id: message_key,
      Location: loc,
      Details: details_map,
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

func (aI *AssertInfo) applyOption(opt_name string, opt_value string) bool {
    // fmt.Printf("Applying %s(%q)\n", opt_name, opt_value)
    if (opt_name == "id") {
        aI.Id = fmt.Sprintf("%s (%s)", aI.Message, opt_value)
        return true
    }
    return false
}


func makeKey(loc *LocationInfo) string {
    return fmt.Sprintf("%s|%d|%d", loc.Filename, loc.Line, loc.Column)
}

func struct_to_map(values any) map[string]any {

  var details_map map[string]any

  // Validate and format the details
  var data []byte = nil
  var err error
  if values != nil {
      if data, err = json.Marshal(values); err != nil {
          return details_map
      }
  }

  details_map = make(map[string]any)
  if err = json.Unmarshal(data, &details_map); err != nil {
      details_map = nil
  }
  return details_map
}


// --------------------------------------------------------------------------------
// Emit JSON structured payloads
// --------------------------------------------------------------------------------
func emit_assert(assert_info *AssertInfo) error {
  var data []byte = nil
  var err error

  wrapped_assert := WrappedAssertInfo{assert_info}
  if data, err = json.Marshal(wrapped_assert); err != nil {
      return err
  }
  payload := string(data)
  if err = internal.Json_data(payload); errors.Is(err, internal.DSOError) {
      local_info := LocalLogAssertInfo{
        LocalLogInfo: *local.NewLogInfo("", ""),
        WrappedAssertInfo: wrapped_assert,
      }
      if data, err = json.Marshal(local_info); err != nil {
          return err
      }
      payload = string(data)
      local.Emit(payload)
      err = nil
  }
  return err
}
