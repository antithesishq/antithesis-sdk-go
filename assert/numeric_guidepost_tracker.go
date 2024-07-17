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
	TrackUnsigned
	TrackFloat
)

type IntegralType int

const (
	Unsupported IntegralType = iota
	SmallInt64
	FullInt64
	UnsignedInt64
	Float64
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
	descriminator NumericTrackerType // TrackInteger, TrackUnsigned, TrackFloat

	// used for TrackInteger, TrackUnsigned extreme values
	gap GapValue

	// used for TrackFloat extreme values
	float_gap FloatGapValue
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

	trackerInfo := numericGPInfo{
		maximize:      maximize,
		descriminator: trackerType,

		gap:       newGapValue(uint64(math.MaxUint64), maximize),
		float_gap: newFloatGapValue(float64(math.MaxFloat64), maximize),
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

// --------------------------------------------------------------------------------
// Represents left and right operand values
// --------------------------------------------------------------------------------
type Int64Value struct {
	int_value  int64
	is_invalid bool
	is_small   bool
}

type UInt64Value struct {
	uint_value uint64
	is_invalid bool
}

type Float64Value struct {
	float_value float64
	is_invalid  bool
}

// --------------------------------------------------------------------------------
// TODO: Constraint for how gaps are measured
// --------------------------------------------------------------------------------
// type GapMeasure interface {
//    uint64 | float64
// }

// --------------------------------------------------------------------------------
// Represents integral and floating point extremes
// --------------------------------------------------------------------------------
type GapValue struct {
	gap_size        uint64
	gap_is_negative bool
}

type FloatGapValue struct {
	gap_size        float64
	gap_is_negative bool
}

func newGapValue(sz uint64, is_neg bool) GapValue {
	return GapValue{
		gap_size:        sz,
		gap_is_negative: is_neg}
}

func newFloatGapValue(sz float64, is_neg bool) FloatGapValue {
	return FloatGapValue{
		gap_size:        sz,
		gap_is_negative: is_neg}
}

// --------------------------------------------------------------------------------
// Distinguish numeric types to ensure gap size calculations are done accurately
// --------------------------------------------------------------------------------
func get_integral_type(v any) IntegralType {
	itype := Unsupported

	switch v.(type) {
	case uint, uint64, uintptr:
		itype = UnsignedInt64
	case int, int64:
		itype = FullInt64
	case int8, uint8, int16, uint16, int32, uint32:
		itype = SmallInt64
	case float32, float64:
		itype = Float64
	}
	return itype
}

// --------------------------------------------------------------------------------
// Represent int8, int16 and int32 values as int64 so they can be safely subtracted
// Represent uint8, uint16 and uint32 values as int64 so they can be safely subtracted
// Represent int and int64, setting {is_small: false} to prevent performing direct subtraction
// --------------------------------------------------------------------------------
func as_int64(maybe_int any) Int64Value {
	var int_value int64
	is_valid := true
	is_small := true // most of the types accepted are small
	switch converted_val := maybe_int.(type) {
	case int:
		int_value = int64(converted_val)
		is_small = false
	case int64:
		int_value = converted_val
		is_small = false
	case int8:
		int_value = int64(converted_val)
	case int16:
		int_value = int64(converted_val)
	case int32:
		int_value = int64(converted_val)
	case uint8:
		int_value = int64(converted_val)
	case uint16:
		int_value = int64(converted_val)
	case uint32:
		int_value = int64(converted_val)
	default:
		is_valid = false
	}
	return Int64Value{int_value, is_valid, is_small}
}

// --------------------------------------------------------------------------------
// Represent uint, uint64, uintptr values so the gap size can be calculated
// --------------------------------------------------------------------------------
func as_uint64(maybe_uint any) UInt64Value {
	var uint_value uint64
	is_valid := true
	switch converted_val := maybe_uint.(type) {
	case uint:
		uint_value = uint64(converted_val)
	case uint64:
		uint_value = converted_val
	case uintptr:
		uint_value = uint64(converted_val)
	default:
		is_valid = false
	}
	return UInt64Value{uint_value, is_valid}
}

// --------------------------------------------------------------------------------
// Represent float32, float64 values so the gap size can be calculated
// --------------------------------------------------------------------------------
func as_float64(maybe_float any) Float64Value {
	var float_value float64
	is_valid := true
	switch converted_val := maybe_float.(type) {
	case float32:
		float_value = float64(converted_val)
	case float64:
		float_value = converted_val
	default:
		is_valid = false
	}
	return Float64Value{float_value, is_valid}
}

func is_same_sign(left_val int64, right_val int64) bool {
	same_sign := false
	if left_val < 0 {
		// left is negative
		if right_val < 0 {
			same_sign = true
		}
	} else {
		// left is non-negative
		if right_val >= 0 {
			same_sign = true
		}
	}
	return same_sign
}

func abs_int64(val int64) uint64 {
	if val >= 0 {
		return uint64(val)
	}
	return uint64(0 - val)
}

// When left and right are the same sign (both negative, or both non-negative)
// Calculate: <result> = (left - right).  The gap_size is abs(<result>) and
// gap_is_negative is (right > left)

func gap_from_int64(left Int64Value, right Int64Value) GapValue {
	same_sign := is_same_sign(left.int_value, right.int_value)
	if same_sign || left.is_small {
		result := left.int_value - right.int_value
		gap_size := abs_int64(result)
		return GapValue{
			gap_size:        gap_size,
			gap_is_negative: result < 0,
		}
	}

	// Otherwise left and right are opposite signs
	// gap = abs(left) + abs(right)
	// gap_is_negative = abs(right) > abs(left)
	left_gap_size := abs_int64(left.int_value)
	right_gap_size := abs_int64(right.int_value)
	gap_size := left_gap_size + right_gap_size
	gap_is_negative := right_gap_size > left_gap_size
	return GapValue{
		gap_size,
		gap_is_negative,
	}
}

func gap_from_uint64(left UInt64Value, right UInt64Value) GapValue {
	var gap_size uint64
	var gap_is_negative = false
	if left.uint_value < right.uint_value {
		gap_is_negative = true
		gap_size = right.uint_value - left.uint_value
	} else {
		gap_size = left.uint_value - right.uint_value
	}
	return GapValue{
		gap_size,
		gap_is_negative,
	}
}

func gap_from_float64(left Float64Value, right Float64Value) FloatGapValue {
	var gap_size float64
	var gap_is_negative = false
	if left.float_value < right.float_value {
		gap_is_negative = true
		gap_size = right.float_value - left.float_value
	} else {
		gap_size = left.float_value - right.float_value
	}
	return FloatGapValue{
		gap_size,
		gap_is_negative,
	}
}

func is_greater_than(left GapValue, right GapValue) bool {
	if !left.gap_is_negative && !right.gap_is_negative {
		return left.gap_size > right.gap_size
	}
	if !left.gap_is_negative && right.gap_is_negative {
		return true // any positive is greater than a negative
	}
	if left.gap_is_negative && right.gap_is_negative {
		return right.gap_size > left.gap_size
	}
	if left.gap_is_negative && !right.gap_is_negative {
		return false // any negative is less than a positive
	}
	return false
}

func floating_is_greater_than(left FloatGapValue, right FloatGapValue) bool {
	if !left.gap_is_negative && !right.gap_is_negative {
		return left.gap_size > right.gap_size
	}
	if !left.gap_is_negative && right.gap_is_negative {
		return true // any positive is greater than a negative
	}
	if left.gap_is_negative && right.gap_is_negative {
		return right.gap_size > left.gap_size
	}
	if left.gap_is_negative && !right.gap_is_negative {
		return false // any negative is less than a positive
	}
	return false
}

func is_less_than(left GapValue, right GapValue) bool {
	if !left.gap_is_negative && !right.gap_is_negative {
		return left.gap_size < right.gap_size
	}
	if !left.gap_is_negative && right.gap_is_negative {
		return false // any positive is greater than a negative
	}
	if left.gap_is_negative && right.gap_is_negative {
		return right.gap_size < left.gap_size
	}
	if left.gap_is_negative && !right.gap_is_negative {
		return true // any negative is less than a positive
	}
	return true
}

func floating_is_less_than(left FloatGapValue, right FloatGapValue) bool {
	if !left.gap_is_negative && !right.gap_is_negative {
		return left.gap_size < right.gap_size
	}
	if !left.gap_is_negative && right.gap_is_negative {
		return false // any positive is greater than a negative
	}
	if left.gap_is_negative && right.gap_is_negative {
		return right.gap_size < left.gap_size
	}
	if left.gap_is_negative && !right.gap_is_negative {
		return true // any negative is less than a positive
	}
	return true
}

func (tI *numericGPInfo) send_value_if_needed(gI *guidanceInfo) {
	if tI == nil {
		return
	}

	numeric_gp_info_mutex.Lock()
	defer numeric_gp_info_mutex.Unlock()

	// if this is a catalog entry (gI.hit is false)
	// do not update the reference gap in the tracker (tI *numericGPInfo)
	if !gI.Hit {
		emitGuidance(gI)
		return
	}

	// left and right are type any, in practice, they
	// will be type Number (technically Number is a type constraint, not a type)
	operands := gI.Data.(numericOperands)
	itype := get_integral_type(operands.Left)
	if itype != get_integral_type(operands.Right) {
		itype = Unsupported
	}

	// when maximizing, the starting Integral gap size is the most negative integer representable
	// it is appropriate for numeric guidance that strives to maximize its values
	// when minimizing, indicate that the gap is negative
	gap := newGapValue(math.MaxUint64, tI.maximize)

	// when maximizing, thge starting Float gap size is the most negative float representable
	// it is appropriate for numeric guidance that strives to maximize its values
	// when minimizing, indicate that the gap is negative
	float_gap := newFloatGapValue(math.MaxFloat64, tI.maximize)

	switch itype {
	case Unsupported:
		// TODO: Implement
	case SmallInt64:
		left_int64 := as_int64(operands.Left)
		right_int64 := as_int64(operands.Right)
		gap = gap_from_int64(left_int64, right_int64)
	case FullInt64:
		left_int64 := as_int64(operands.Left)
		right_int64 := as_int64(operands.Right)
		gap = gap_from_int64(left_int64, right_int64)
	case UnsignedInt64:
		left_uint64 := as_uint64(operands.Left)
		right_uint64 := as_uint64(operands.Right)
		gap = gap_from_uint64(left_uint64, right_uint64)
	case Float64:
		left_float64 := as_float64(operands.Left)
		right_float64 := as_float64(operands.Right)
		float_gap = gap_from_float64(left_float64, right_float64)
	}

	should_send := false

	if tI.should_maximize() {
		if tI.is_integer() || tI.is_unsigned() {
			should_send = is_greater_than(gap, tI.gap)
		} else {
			should_send = floating_is_greater_than(float_gap, tI.float_gap)
		}
	} else {
		if tI.is_integer() || tI.is_unsigned() {
			should_send = is_less_than(gap, tI.gap)
		} else {
			should_send = floating_is_less_than(float_gap, tI.float_gap)
		}
	}

	if should_send {
		if tI.is_integer() || tI.is_unsigned() {
			tI.gap = gap
		} else {
			tI.float_gap = float_gap
		}
		emitGuidance(gI)
	}
}

func emitGuidance(gI *guidanceInfo) error {
	return internal.Json_data(map[string]any{"antithesis_guidance": gI})
}
