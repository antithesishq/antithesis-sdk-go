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
//   - gap is the largest value sent so far
//
// For GuidepostMinimize:
//   - gap is the most negative value sent so far
//
// --------------------------------------------------------------------------------
type numericGPInfo struct {
	gap           any
	descriminator NumericGapType
	maximize      bool
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

	var gap any
	if trackerType == IntegerGap {
		gap = newGapValue(uint64(math.MaxUint64), maximize)
	} else {
		gap = newGapValue(float64(math.MaxFloat64), maximize)
	}
	trackerInfo := numericGPInfo{
		maximize:      maximize,
		descriminator: trackerType,
		gap:           gap,
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
type GapValue[T NumType] struct {
	gap_size        T
	gap_is_negative bool
}

func newGapValue[T NumType](sz T, is_neg bool) any {
	switch any(sz).(type) {
	case uint64:
		return GapValue[uint64]{gap_size: uint64(sz), gap_is_negative: is_neg}

	case float64:
		return GapValue[float64]{gap_size: float64(sz), gap_is_negative: is_neg}
	}
	return nil
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

func is_greater_than[T NumType](left GapValue[T], right GapValue[T]) bool {
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

func is_less_than[T NumType](left GapValue[T], right GapValue[T]) bool {
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

func send_value_if_needed(tI *numericGPInfo, gI *guidanceInfo) {
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

	should_send := false
	maximize := tI.should_maximize()

	var gap GapValue[uint64]
	var float_gap GapValue[float64]

	// Needs to have individual case statements to assist
	// the compiler to infer the actual type of the var named 'operands'
	switch operands := (gI.Data).(type) {
	case numericOperands[int32]:
		gap = makeGap(operands)
	case numericOperands[int64]:
		gap = makeGap(operands)
	case numericOperands[uint64]:
		gap = makeGap(operands)
	case numericOperands[float64]:
		float_gap = makeFloatGap(operands)
	}

	var prev_gap GapValue[uint64]
	var prev_float_gap GapValue[float64]
	has_prev_gap := false
	has_prev_float_gap := false

	prev_gap, has_prev_gap = tI.gap.(GapValue[uint64])
	if !has_prev_gap {
		prev_float_gap, has_prev_float_gap = tI.gap.(GapValue[float64])
	}

	if has_prev_gap {
		if maximize {
			should_send = is_greater_than(gap, prev_gap)
		} else {
			should_send = is_less_than(gap, prev_gap)
		}
	}

	if has_prev_float_gap {
		if maximize {
			should_send = is_greater_than(float_gap, prev_float_gap)
		} else {
			should_send = is_less_than(float_gap, prev_float_gap)
		}
	}

	if should_send {
		if tI.is_integer_gap() {
			tI.gap = gap
		} else {
			tI.gap = float_gap
		}
		emitGuidance(gI)
	}
}

func emitGuidance(gI *guidanceInfo) error {
	return internal.Json_data(map[string]any{"antithesis_guidance": gI})
}

// When left and right are the same sign (both negative, or both non-negative)
// Calculate: <result> = (left - right).  The gap_size is abs(<result>) and
// gap_is_negative is (right > left)
func makeGap[Op Operand](operand numericOperands[Op]) GapValue[uint64] {

	var gap_size uint64
	var gap_is_negative bool

	switch this_op := any(operand).(type) {
	case numericOperands[int32]:
		result := int64(this_op.Left) - int64(this_op.Right)
		gap_size = abs_int64(result)
		gap_is_negative = result < 0

	case numericOperands[int64]:
		if is_same_sign(this_op.Left, this_op.Right) {
			result := int64(this_op.Left) - int64(this_op.Right)
			gap_size = abs_int64(result)
			gap_is_negative = result < 0
			break
		}

		// Otherwise left and right are opposite signs
		// gap = abs(left) + abs(right)
		// gap_is_negative = left < right
		left_gap_size := abs_int64(this_op.Left)
		right_gap_size := abs_int64(this_op.Right)
		gap_size = left_gap_size + right_gap_size
		gap_is_negative = this_op.Left < this_op.Right

	case numericOperands[uint64]:
		left_val := this_op.Left
		right_val := this_op.Right
		gap_is_negative = false
		if left_val < right_val {
			gap_is_negative = true
			gap_size = right_val - left_val
		} else {
			gap_size = left_val - right_val
		}

	default:
		zero_gap, _ := newGapValue(uint64(0), false).(GapValue[uint64])
		return zero_gap
	}

	this_gap, _ := newGapValue(gap_size, gap_is_negative).(GapValue[uint64])
	return this_gap
} // MakeGap

func makeFloatGap[Op Operand](operand numericOperands[Op]) GapValue[float64] {
	switch this_op := any(operand).(type) {
	case numericOperands[float64]:
		left_val := this_op.Left
		right_val := this_op.Right
		gap_is_negative := false
		var gap_size float64
		if left_val < right_val {
			gap_is_negative = true
			gap_size = right_val - left_val
		} else {
			gap_size = left_val - right_val
		}

		this_gap, _ := newGapValue(gap_size, gap_is_negative).(GapValue[float64])
		return this_gap

	default:
		zero_gap, _ := newGapValue(float64(0.0), false).(GapValue[float64])
		return zero_gap
	}
} // MakeFloatGap
