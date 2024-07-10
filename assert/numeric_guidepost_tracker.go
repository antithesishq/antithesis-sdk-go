//go:build !no_antithesis_sdk

package assert

import (
	// [DEL] "math"
	"sync"

	"github.com/antithesishq/antithesis-sdk-go/internal"
)

type NumericTrackerType int

const (
	TrackInteger NumericTrackerType = iota
	TrackUnsigned
	TrackFloat
)

func TrackerTypeForNumber[T Number](num T) NumericTrackerType {
	trackerType := TrackInteger
	switch any(num).(type) {
	case int, int8, int16, int32, int64:
		trackerType = TrackInteger
	case uint, uint8, uint16, uint32, uint64, uintptr:
		trackerType = TrackUnsigned
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
// For GuidepostMaximize extremeValue is the largest value sent so far
// For GuidepostMinimize extremeValue is the smallest value sent so far
type numericGPInfo struct {
	maximize      bool
	descriminator NumericTrackerType
	// [DEL] extremeIntegerValue int64
	// [DEL] extremeUnsignedValue uint64
	// [DEL] extremeFloatValue   float64
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
	// [DEL] var extreme_integer_value int64 = math.MaxInt64
	// [DEL] var extreme_unsigned_value uint64 = math.MaxUint64
	// [DEL] var extreme_float_value = math.MaxFloat64
	// [DEL] if maximize {
	// [DEL] 	extreme_integer_value = math.MinInt64
	// [DEL]   extreme_unsigned_value = 0
	// [DEL] 	extreme_float_value = 0.0 - math.MaxFloat64
	// [DEL] }

	trackerInfo := numericGPInfo{
		maximize:      maximize,
		descriminator: trackerType,
		// [DEL] extremeIntegerValue: extreme_integer_value,
		// [DEL] extremeUnsignedValue: extreme_unsigned_value,
		// [DEL] extremeFloatValue:   extreme_float_value,
	}
	return &trackerInfo
}

func (tI *numericGPInfo) should_maximize() bool {
	return tI.maximize
}

func (tI *numericGPInfo) is_integer() bool {
	return tI.descriminator == TrackInteger
}

func (tI *numericGPInfo) is_unsigned() bool {
	return tI.descriminator == TrackUnsigned
}

// --------------------------------------------------------------------------------
// func (tI *numericGPInfo) is_float() bool {
//   return tI.descriminator == TrackFloat
// }
// --------------------------------------------------------------------------------

func (tI *numericGPInfo) send_value(gI *guidanceInfo) {
	if tI == nil {
		return
	}

	numeric_gp_info_mutex.Lock()
	defer numeric_gp_info_mutex.Unlock()

	emitGuidance(gI)
}

// [DEL] func (tI *numericGPInfo) send_value_if_needed(gI *guidanceInfo) {
// [DEL] 	if tI == nil {
// [DEL] 		return
// [DEL] 	}
// [DEL]
// [DEL] 	numeric_gp_info_mutex.Lock()
// [DEL] 	defer numeric_gp_info_mutex.Unlock()
// [DEL]
// [DEL]   // Values derived from gI (Data.Left - Data.Right)
// [DEL] 	var int_value int64
// [DEL]   var uint_value uint64
// [DEL] 	var float_value float64
// [DEL]
// [DEL]   // Assign fallback values to not trigger a send
// [DEL] 	if tI.should_maximize() {
// [DEL] 		int_value = math.MinInt64
// [DEL]     uint_value = 0
// [DEL] 		float_value = 0.0 - math.MaxFloat64
// [DEL] 	} else {
// [DEL] 		int_value = math.MaxInt64
// [DEL]     uint_value = math.MaxUint64
// [DEL] 		float_value = math.MaxFloat64
// [DEL] 	}
// [DEL]
// [DEL]
// [DEL]   // left and right are type any, in practice, they
// [DEL]   // will be type Number (technically Number is a type constraint, not a type)
// [DEL]   operands := gI.Data.(numericOperands)
// [DEL]   left := operands.Left
// [DEL]   right := operands.Right
// [DEL]
// [DEL]   switch left_value := left.(type) {
// [DEL]   case int, int8, int16, int32, int64:
// [DEL]     int_value = int64(left_value)
// [DEL]
// [DEL]   case uint, uint8, uint16, uint32, uint64, uintptr:
// [DEL]     uint_value = uint64(left_value)
// [DEL]   }
// [DEL]
// [DEL]   case float32, float64:
// [DEL]     float_value = float64(left_value)
// [DEL]
// [DEL] 	switch value := gI.Data.(type) {
// [DEL] 	case int64:
// [DEL] 		int_value = value
// [DEL] 	case uint64:
// [DEL] 		uint_value = value
// [DEL] 	case float64:
// [DEL] 		float_value = value
// [DEL] 	}
// [DEL]
// [DEL] 	should_send := false
// [DEL]
// [DEL] 	if tI.should_maximize() {
// [DEL] 		if tI.is_integer() {
// [DEL] 			should_send = int_value > tI.extremeIntegerValue
// [DEL] 		} else if tI.is_unsigned() {
// [DEL] 			should_send = uint_value > tI.extremeUnsignedValue
// [DEL] 		} else {
// [DEL] 			should_send = float_value > tI.extremeFloatValue
// [DEL] 		}
// [DEL] 	} else {
// [DEL] 		if tI.is_integer() {
// [DEL] 			should_send = int_value < tI.extremeIntegerValue
// [DEL] 		} else if tI.is_unsigned() {
// [DEL] 			should_send = uint_value < tI.extremeUnsignedValue
// [DEL] 		} else {
// [DEL] 			should_send = float_value < tI.extremeFloatValue
// [DEL] 		}
// [DEL] 	}
// [DEL]
// [DEL] 	if should_send {
// [DEL] 		if tI.is_integer() {
// [DEL] 			tI.extremeIntegerValue = int_value
// [DEL] 		} else if tI.is_unsigned() {
// [DEL] 			tI.extremeUnsignedValue = uint_value
// [DEL] 		} else {
// [DEL] 			tI.extremeFloatValue = float_value
// [DEL] 		}
// [DEL] 		emitGuidance(gI)
// [DEL] 	}
// [DEL] }

func emitGuidance(gI *guidanceInfo) error {
	return internal.Json_data(map[string]any{"antithesis_guidance": gI})
}
