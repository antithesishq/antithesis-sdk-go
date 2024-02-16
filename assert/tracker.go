package assert

import "github.com/antithesishq/antithesis-sdk-go/internal"

type trackerInfo struct {
	PassCount int
	FailCount int
}

type emitTracker map[string]*trackerInfo

// assert_tracker (global) keeps track of the unique asserts evaluated
var assertTracker emitTracker = make(emitTracker)

func (tracker emitTracker) getTrackerEntry(messageKey string) *trackerInfo {
	var trackerEntry *trackerInfo
	var ok bool

	if tracker == nil {
		return nil
	}

	if trackerEntry, ok = tracker[messageKey]; !ok {
		trackerEntry = newTrackerInfo()
		tracker[messageKey] = trackerEntry
	}
	return trackerEntry
}

func newTrackerInfo() *trackerInfo {
	trackerInfo := trackerInfo{
		PassCount: 0,
		FailCount: 0,
	}
	return &trackerInfo
}

func (ti *trackerInfo) emit(ai *assertInfo) {
	if ti == nil || ai == nil {
		return
	}

	var err error
	cond := ai.Condition

	if cond {
		if ti.PassCount == 0 {
			err = emitAssert(ai)
		}
		if err == nil {
			ti.PassCount++
		}
		return
	}
	if ti.FailCount == 0 {
		err = emitAssert(ai)
	}
	if err == nil {
		ti.FailCount++
	}
}

func emitAssert(ai *assertInfo) error {
	return internal.Json_data(wrappedAssertInfo{ai})
}
