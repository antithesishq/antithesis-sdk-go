//go:build !no_antithesis_sdk

package assert

import (
	"math"
	"sync"

	"github.com/antithesishq/antithesis-sdk-go/internal"
)

type NumericTrackerType int

const (
	TrackInteger NumericTrackerType = iota
	TrackFloat
)

func TrackerTypeForNumber[T Number](num T) NumericTrackerType {
	trackerType := TrackInteger
	switch any(num).(type) {
	case int, int8, int16, int32, int64:
		trackerType = TrackInteger
	case uint, uint8, uint16, uint32:
		trackerType = TrackInteger
	case float32, float64:
		trackerType = TrackFloat
	}
	return trackerType
}

// Go does not permit using an interface constraint as a type
// If it did, we could declare ExtremeValue as 'Number' (assert_types.go)
// See no-constraint-type-as-type rationale here:
// https://go.googlesource.com/proposal/+/refs/heads/master/design/43651-type-parameters.md#permitting-constraints-as-ordinary-interface-types
//
// Instead, we use int64 as the largest signed value available
// and use it for any of the signed or unsigned types
//
// For GuidepostMaximize extremeValue is the largest value sent so far
// For GuidepostMinimize extremeValue is the smallest value sent so far
type numericGPInfo struct {
	descriminator       NumericTrackerType
	extremeIntegerValue int64
	extremeFloatValue   float64
	maximize            bool
}

type numericGPTracker map[string]*numericGPInfo

var (
	numeric_gp_tracker       numericGPTracker = make(numericGPTracker)
	numeric_gp_tracker_mutex sync.Mutex
	numeric_gp_info_mutex    sync.Mutex
)

func (tracker numericGPTracker) getTrackerEntry(messageKey string, trackerType NumericTrackerType, maximize bool) *numericGPInfo {
	var trackerEntry *numericGPInfo
	var ok bool

	if tracker == nil {
		return nil
	}

	numeric_gp_tracker_mutex.Lock()
	defer numeric_gp_tracker_mutex.Unlock()
	if trackerEntry, ok = numeric_gp_tracker[messageKey]; !ok {
		trackerEntry = newNumericGPInfo(trackerType, maximize)
		tracker[messageKey] = trackerEntry
	}

	return trackerEntry
}

// Create an numeric guidance entry
func newNumericGPInfo(trackerType NumericTrackerType, maximize bool) *numericGPInfo {
	var extreme_integer_value int64 = math.MaxInt64
	var extreme_float_value = math.MaxFloat64
	if maximize {
		extreme_integer_value = math.MinInt64
		extreme_float_value = 0.0 - math.MaxFloat64
	}
	descriminator := TrackInteger
	if trackerType == TrackFloat {
		descriminator = TrackFloat
	}
	trackerInfo := numericGPInfo{
		descriminator:       descriminator,
		extremeIntegerValue: extreme_integer_value,
		extremeFloatValue:   extreme_float_value,
		maximize:            maximize,
	}
	return &trackerInfo
}

func (tI *numericGPInfo) should_maximize() bool {
	return tI.maximize
}

func (tI *numericGPInfo) is_integer() bool {
	return tI.descriminator == TrackInteger
}

// --------------------------------------------------------------------------------
// Not used
//
// func (tI *numericGPInfo) is_float() bool {
//   return tI.descriminator == TrackFloat
// }
// --------------------------------------------------------------------------------

func (tI *numericGPInfo) send_value_if_needed(gI *guidanceInfo) {
	if tI == nil {
		return
	}

	numeric_gp_info_mutex.Lock()
	defer numeric_gp_info_mutex.Unlock()

	var int_value int64
	var float_value float64
	if tI.should_maximize() {
		int_value = math.MinInt64
		float_value = 0.0 - math.MaxFloat64
	} else {
		int_value = math.MaxInt64
		float_value = math.MaxFloat64
	}

	// var value = gI.Data

	switch value := gI.Data.(type) {
	case int64:
		int_value = value
	case float64:
		float_value = value
	}

	should_send := false

	if tI.should_maximize() {
		if tI.is_integer() {
			should_send = int_value > tI.extremeIntegerValue
		} else {
			should_send = float_value > tI.extremeFloatValue
		}
	} else {
		if tI.is_integer() {
			should_send = int_value < tI.extremeIntegerValue
		} else {
			should_send = float_value < tI.extremeFloatValue
		}
	}

	if should_send {
		if tI.is_integer() {
			tI.extremeIntegerValue = int_value
		} else {
			tI.extremeFloatValue = float_value
		}
		emitGuidance(gI)
	}
}

func emitGuidance(gI *guidanceInfo) error {
	return internal.Json_data(map[string]any{"antithesis_guidance": gI})
}
