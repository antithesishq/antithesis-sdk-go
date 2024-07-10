//go:build !no_antithesis_sdk

package assert

import (
	"sync"

	"github.com/antithesishq/antithesis-sdk-go/internal"
)

// TODO: Tracker is intended to prevent sending the same guidance
// more than once.  In this case, we always send, so the tracker
// is not presently used.
type booleanGPInfo struct {
}

type booleanGPTracker map[string]*booleanGPInfo

var (
	boolean_gp_tracker       booleanGPTracker = make(booleanGPTracker)
	boolean_gp_tracker_mutex sync.Mutex
	boolean_gp_info_mutex    sync.Mutex
)

func (tracker booleanGPTracker) getTrackerEntry(messageKey string) *booleanGPInfo {
	var trackerEntry *booleanGPInfo
	var ok bool

	if tracker == nil {
		return nil
	}

	boolean_gp_tracker_mutex.Lock()
	defer boolean_gp_tracker_mutex.Unlock()
	if trackerEntry, ok = boolean_gp_tracker[messageKey]; !ok {
		trackerEntry = newBooleanGPInfo()
		tracker[messageKey] = trackerEntry
	}

	return trackerEntry
}

// Create a boolean guidance tracker
func newBooleanGPInfo() *booleanGPInfo {
	trackerInfo := booleanGPInfo{}
	return &trackerInfo
}

func (tI *booleanGPInfo) send_value(bgI *booleanGuidanceInfo) {
	if tI == nil {
		return
	}

	boolean_gp_info_mutex.Lock()
	defer boolean_gp_info_mutex.Unlock()

	// The tracker entry should be consulted to determine
	// if this Guidance info has already been sent, or not.

	emitBooleanGuidance(bgI)
}

func emitBooleanGuidance(bgI *booleanGuidanceInfo) error {
	return internal.Json_data(map[string]any{"antithesis_guidance": bgI})
}
