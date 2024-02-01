package assert

import "github.com/antithesishq/antithesis-sdk-go/internal"

type trackerInfo struct {
	PassCount int
	FailCount int
}

type emitTracker map[string]*trackerInfo

// assert_tracker (global) keeps track of the unique asserts evaluated
var assert_tracker emitTracker = make(emitTracker)

func (tracker emitTracker) get_tracker_entry(message_key string) *trackerInfo {
	var tracker_entry *trackerInfo
	var ok bool

	if tracker == nil {
		return nil
	}

	if tracker_entry, ok = tracker[message_key]; !ok {
		tracker_entry = newTrackerInfo()
		tracker[message_key] = tracker_entry
	}
	return tracker_entry
}

func newTrackerInfo() *trackerInfo {
	tracker_info := trackerInfo{
		PassCount: 0,
		FailCount: 0,
	}
	return &tracker_info
}

func (ti *trackerInfo) emit(ai *assertInfo) {
	if ti == nil || ai == nil {
		return
	}

	var err error
	cond := ai.Condition

	if cond {
		if ti.PassCount == 0 {
			err = emit_assert(ai)
		}
		if err == nil {
			ti.PassCount++
		}
		return
	}
	if ti.FailCount == 0 {
		err = emit_assert(ai)
	}
	if err == nil {
		ti.FailCount++
	}
}

func emit_assert(assert_info *assertInfo) error {
	return internal.Json_data(wrappedAssertInfo{assert_info})
}
