//go:build !no_antithesis_sdk

package assert

import (
	"math"
	"sync"

	"github.com/antithesishq/antithesis-sdk-go/internal"
)

// --------------------------------------------------------------------------------
// IntegerGap is used for:
// - int, int8, int16, int32, int64:
// - uint, uint8, uint16, uint32, uint64, uintptr:
//
// FloatGap is used for:
// - float32, float64
// --------------------------------------------------------------------------------
type NumericGapType int

const (
	IntegerGap NumericGapType = iota
	FloatGap
)

func GapTypeForOperand[T Number](num T) NumericGapType {
	gapType := IntegerGap

	switch any(num).(type) {
	case float32, float64:
		gapType = FloatGap
	}
	return gapType
}


// --------------------------------------------------------------------------------
// numericGPTracker - Tracking Info for Numeric Guideposts
//
// For GuidepostMaximize:
//   - gap is the largest integer value sent so far
//   - float_gap is the largest floating point value sent so far
//
// For GuidepostMinimize:
//   - gap is the most negative integer value sent so far
//   - float_gap is the most negative floating point value sent so far
//
// --------------------------------------------------------------------------------
type numericGPInfo struct {
	maximize      bool
	descriminator NumericGapType // IntegerGap, FloatGap

	// used for IntegerGap extreme values
	gap GapValue

	// used for FloatGap extreme values
	float_gap FloatGapValue
}

type numericGPTracker map[string]*numericGPInfo

var (
	numeric_gp_tracker       numericGPTracker = make(numericGPTracker)
	numeric_gp_tracker_mutex sync.Mutex
	numeric_gp_info_mutex    sync.Mutex
)

func (tracker numericGPTracker) getTrackerEntry(messageKey string, trackerType NumericGapType, maximize bool) *numericGPInfo {
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
func newNumericGPInfo(trackerType NumericGapType, maximize bool) *numericGPInfo {

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

func (tI *numericGPInfo) is_integer_gap() bool {
	return tI.descriminator == IntegerGap
}


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
// Distinguish numeric operand types to ensure gap size calculations are accurate
// --------------------------------------------------------------------------------
type OperandType int

const (
	Unsupported OperandType = iota
	SmallInt64
	FullInt64
	UnsignedInt64
	Float64
)

func get_operand_type(v any) OperandType {
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
// Represents left and right operand values
// --------------------------------------------------------------------------------
type Int64Operand struct {
	int_value  int64
	is_invalid bool
	is_small   bool
}

type UInt64Operand struct {
	uint_value uint64
	is_invalid bool
}

type Float64Operand struct {
	float_value float64
	is_invalid  bool
}

// --------------------------------------------------------------------------------
// Represent int8, int16 and int32 values as int64 so they can be safely subtracted
// Represent uint8, uint16 and uint32 values as int64 so they can be safely subtracted
// Represent int and int64, setting {is_small: false} to prevent performing direct subtraction
// --------------------------------------------------------------------------------
func newInt64Operand(maybe_int any) Int64Operand {
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
	return Int64Operand{
		int_value:  int_value,
		is_invalid: !is_valid,
		is_small:   is_small,
	}
}

// --------------------------------------------------------------------------------
// Represent uint, uint64, uintptr values so the gap size can be calculated
// --------------------------------------------------------------------------------
func newUInt64Operand(maybe_uint any) UInt64Operand {
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
	return UInt64Operand{
		uint_value: uint_value,
		is_invalid: !is_valid,
	}
}

// --------------------------------------------------------------------------------
// Represent float32, float64 values so the gap size can be calculated
// --------------------------------------------------------------------------------
func newFloat64Operand(maybe_float any) Float64Operand {
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
	return Float64Operand{
		float_value: float_value,
		is_invalid:  !is_valid,
	}
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
func gap_from_int64(left Int64Operand, right Int64Operand) GapValue {
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
		gap_size:        gap_size,
		gap_is_negative: gap_is_negative,
	}
}

func gap_from_uint64(left UInt64Operand, right UInt64Operand) GapValue {
	var gap_size uint64
	var gap_is_negative = false
	if left.uint_value < right.uint_value {
		gap_is_negative = true
		gap_size = right.uint_value - left.uint_value
	} else {
		gap_size = left.uint_value - right.uint_value
	}
	return GapValue{
		gap_size:        gap_size,
		gap_is_negative: gap_is_negative,
	}
}

func gap_from_float64(left Float64Operand, right Float64Operand) FloatGapValue {
	var gap_size float64
	var gap_is_negative = false
	if left.float_value < right.float_value {
		gap_is_negative = true
		gap_size = right.float_value - left.float_value
	} else {
		gap_size = left.float_value - right.float_value
	}
	return FloatGapValue{
		gap_size:        gap_size,
		gap_is_negative: gap_is_negative,
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
	// will be a type that satisifes the Number type constraint
	// int, uint, ... int64, uint64, float32, float64
	operands := gI.Data.(numericOperands)
	should_send := false
	maximize := tI.should_maximize()

	var gap GapValue
	var float_gap FloatGapValue

	if tI.is_integer_gap() {
		gap = calculateGap(operands, maximize)
		if maximize {
			should_send = is_greater_than(gap, tI.gap)
		} else {
			should_send = is_less_than(gap, tI.gap)
		}
	} else {
		float_gap = calculateFloatGap(operands, maximize)
		if maximize {
			should_send = floating_is_greater_than(float_gap, tI.float_gap)
		} else {
			should_send = floating_is_less_than(float_gap, tI.float_gap)
		}
	}

	if should_send {
		if tI.is_integer_gap() {
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

// when maximizing, the default Integer gap size is the most negative integer
// it is appropriate for numeric guidance that strives to maximize its values
// when minimizing, indicate that the default Integer gap is positive
func calculateGap(operands numericOperands, maximize bool) GapValue {
	operand_type := get_operand_type(operands.Left)
	if operand_type != get_operand_type(operands.Right) || operand_type == Float64 {
		operand_type = Unsupported
	}

	switch operand_type {
	case Unsupported, Float64:
		return newGapValue(math.MaxUint64, maximize)
	case SmallInt64, FullInt64:
		left_int64 := newInt64Operand(operands.Left)
		right_int64 := newInt64Operand(operands.Right)
		return gap_from_int64(left_int64, right_int64)
	case UnsignedInt64:
		left_uint64 := newUInt64Operand(operands.Left)
		right_uint64 := newUInt64Operand(operands.Right)
		return gap_from_uint64(left_uint64, right_uint64)
	default:
		return newGapValue(math.MaxUint64, maximize)
	}
}

// when maximizing, the default Floating Point gap size is the most negative Float64
// it is appropriate for numeric guidance that strives to maximize its values
// when minimizing, indicate that the default Floating Point gap size is the most positive Float64
func calculateFloatGap(operands numericOperands, maximize bool) FloatGapValue {
	operand_type := get_operand_type(operands.Left)
	if operand_type != get_operand_type(operands.Right) || operand_type != Float64 {
		operand_type = Unsupported
	}

	switch operand_type {
	case Unsupported, SmallInt64, FullInt64, UnsignedInt64:
		return newFloatGapValue(math.MaxFloat64, maximize)
	case Float64:
		left_float64 := newFloat64Operand(operands.Left)
		right_float64 := newFloat64Operand(operands.Right)
		return gap_from_float64(left_float64, right_float64)
	default:
		return newFloatGapValue(math.MaxFloat64, maximize)
	}
}
