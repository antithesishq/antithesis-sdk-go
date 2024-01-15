package assert

// --------------------------------------------------------------------------------
// EmitTracker
// --------------------------------------------------------------------------------
type TrackerInfo struct {
    PassCount int
    FailCount int
}

type EmitTracker map[string]*TrackerInfo

// assert_tracker (global) keeps track of the unique asserts evaluated
var assert_tracker EmitTracker = make(EmitTracker)

func (tracker EmitTracker) get_tracker_entry(message_key string) *TrackerInfo {
  var tracker_entry *TrackerInfo
  var ok bool

  if (tracker == nil) {
      return nil
  }

  if tracker_entry, ok = tracker[message_key]; !ok {
      tracker_entry = NewTrackerInfo()
      tracker[message_key] = tracker_entry
  }
  return tracker_entry
}


func NewTrackerInfo() *TrackerInfo {
    tracker_info := TrackerInfo {
        PassCount: 0,
        FailCount: 0,
    }
    return &tracker_info
}

func (ti *TrackerInfo) emit(ai *AssertInfo) {
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
