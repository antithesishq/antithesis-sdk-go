package antilog

import (
  "encoding/json"
  "path"
  "runtime"
  "strings"
)

// --------------------------------------------------------------------------------
// EmitTracker
// --------------------------------------------------------------------------------
type TrackerInfo struct {
    PassCount int
    FailCount int
}

type EmitTracker map[string]*TrackerInfo

var always_tracker EmitTracker = nil
var sometimes_tracker EmitTracker = nil

func NewTrackerInfo() *TrackerInfo {
    tracker_info := TrackerInfo {
        PassCount: 0,
        FailCount: 0,
    }
    return &tracker_info
}

// --------------------------------------------------------------------------------
// LocationInfo
// --------------------------------------------------------------------------------
type LocationInfo struct {
    Classname string `json:"classname"`
    Funcname string `json:"function"`
    Filename string `json:"filename"`
    Line int `json:"line"`
}

func NewLocationInfo(nframes int) *LocationInfo {
  // Get location info and add to details
  funcname := "*function*"
  classname := "*classname*"
  pc, filename, line, ok := runtime.Caller(nframes)
  if !ok {
    filename = "*filename*"
    line = 0
  } else {
      if this_func := runtime.FuncForPC(pc); this_func != nil {
          fullname := this_func.Name()
          funcname = path.Ext(fullname)
          classname, _ = strings.CutSuffix(fullname, funcname)
          funcname = funcname[1:]
      }
  }
  return &LocationInfo{classname, funcname, filename, line}
}


// --------------------------------------------------------------------------------
// DirectiveInfo
// --------------------------------------------------------------------------------
type DirectiveInfo struct {
    Message string `json:"message"`
    Condition bool `json:"condition"`
    Id string `json:"id"`
    Location *LocationInfo `json:"location"`
    Details map[string]any `json:"details"`
}

type AlwaysInfo struct {
    Directive *DirectiveInfo `json:"ant_always"`
}

type SometimesInfo struct {
    Directive *DirectiveInfo `json:"ant_sometimes"`
}

func NewDirective(message string, condition bool, values any, location_info *LocationInfo) *DirectiveInfo {

  // Validate and format the details
  var data []byte = nil
  var err error
  if values != nil {
      if data, err = json.Marshal(values); err != nil {
          data = nil
      }
  }

  var details_map map[string]any
  
  if data != nil {
      details_map = make(map[string]any)
      if err = json.Unmarshal(data, &details_map); err != nil {
          details_map = nil
      }
  }

  common_info := DirectiveInfo {
      Message: message,
      Condition: condition,
      Id: "*id*",
      Location: location_info,
      Details: details_map,
  }
  return &common_info
}



// --------------------------------------------------------------------------------
// BuildTimeInfo
// --------------------------------------------------------------------------------
type BuildTimeInfo struct {
    Message string `json:"message"`
    Id string `json:"id"`
    Location *LocationInfo `json:"location"`
}

type ExpectInfo struct {
    BuildTime *BuildTimeInfo `json:"ant_expect"`
}

func NewBuildTime(message string, location_info *LocationInfo) *BuildTimeInfo {

  info := BuildTimeInfo {
      Message: message,
      Id: "*id*",
      Location: location_info,
  }
  return &info
}


// --------------------------------------------------------------------------------
// Version
// --------------------------------------------------------------------------------
func Version() string {
  return "0.0.3"
}


// --------------------------------------------------------------------------------
// Version
// --------------------------------------------------------------------------------
func Always(text string, cond bool, details any) {
  if always_tracker == nil {
      always_tracker = make(EmitTracker)
  }
  message_key := text

  var tracker_entry *TrackerInfo
  var ok bool

  if tracker_entry, ok = always_tracker[message_key]; !ok {
      tracker_entry = NewTrackerInfo()
      always_tracker[message_key] = tracker_entry
  }

  // 0 frames back will be NewLocationInfo()
  // 1 frame back will be here 
  // 2 frames back is the caller of this function  
  location_info := NewLocationInfo(2) 

  var err error
  if cond {
      if tracker_entry.PassCount == 0 {
          err = emit_always(text, cond, details, location_info)
      }
      if err == nil {
          tracker_entry.PassCount++
      }
      return
  }
  if tracker_entry.FailCount == 0 {
      err = emit_always(text, cond, details, location_info)
  }
  if err == nil {
      tracker_entry.FailCount++
  }
}

func Sometimes(text string, cond bool, details any) {
  if sometimes_tracker == nil {
      sometimes_tracker = make(EmitTracker)
  }
  message_key := text

  var tracker_entry *TrackerInfo
  var ok bool

  if tracker_entry, ok = sometimes_tracker[message_key]; !ok {
      tracker_entry = NewTrackerInfo()
      sometimes_tracker[message_key] = tracker_entry
  }

  // 0 frames back will be NewLocationInfo()
  // 1 frame back will be here 
  // 2 frames back is the caller of this function  
  location_info := NewLocationInfo(2) 

  var err error
  if cond {
      if tracker_entry.PassCount == 0 {
          err = emit_sometimes(text, cond, details, location_info)
      }
      if err == nil {
          tracker_entry.PassCount++
      }
      return
  }
  if tracker_entry.FailCount == 0 {
      err = emit_sometimes(text, cond, details, location_info)
  }
  if err == nil {
      tracker_entry.FailCount++
  }
}

func Expect(message string, classname string, funcname string, filename string, line int) {
  location_info := &LocationInfo{classname, funcname, filename, line}
  emit_expect(message, location_info)
}


// --------------------------------------------------------------------------------
// Emit JSON structured payloads
// --------------------------------------------------------------------------------
func emit_always(message string, condition bool, values any, location_info *LocationInfo) error {
  var data []byte = nil
  var err error

  common_info := NewDirective(message, condition, values, location_info)
  emit_info := AlwaysInfo{common_info}
  if data, err = json.Marshal(emit_info); err != nil {
      return err
  }
  payload := string(data)
  JSONData(payload)
  return nil
}

func emit_sometimes(message string, condition bool, values any, location_info *LocationInfo) error {
  var data []byte = nil
  var err error

  common_info := NewDirective(message, condition, values, location_info)
  emit_info := SometimesInfo{common_info}
  if data, err = json.Marshal(emit_info); err != nil {
      return err
  }
  payload := string(data)
  JSONData(payload)
  return nil
}


func emit_expect(message string, location_info *LocationInfo) error {
  var data []byte = nil
  var err error

  info := NewBuildTime(message, location_info)
  emit_info := ExpectInfo{info}
  if data, err = json.Marshal(emit_info); err != nil {
      return err
  }
  payload := string(data)
  JSONData(payload)
  return nil
}

// --------------------------------------------------------------------------------
// Text output 
// --------------------------------------------------------------------------------
func OutputText(text string) {
}

func ErrorText(text string) {
}

func SetSource(text string) {
}

