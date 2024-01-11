package antilog

import (
  "path"
  "runtime"
  "strings"
)

// --------------------------------------------------------------------------------
// LocationInfo
// --------------------------------------------------------------------------------

// StackFrameOffset indicates how many frames to go up in the 
// call stack to find the filename/location/line info.  As 
// this work is always done in NewLocationInfo(), the offset is 
// specified from the perspective of NewLocationInfo
type StackFrameOffset int

// Order is important here since iota is being used
const (
    OffsetNewLocationInfo StackFrameOffset = iota
    OffsetHere
    OffsetAPICaller
    OffsetAPICallersCaller
)

type LocationInfo struct {
    Classname string `json:"classname"`
    Funcname string `json:"function"`
    Filename string `json:"filename"`
    Line int `json:"line"`
    Column int `json:"column"`
}

const ColumnUnknown = 0 

func NewLocInfo(classname, funcname, filename string, line int) *LocationInfo {
  return &LocationInfo{classname, funcname, filename, line, ColumnUnknown}
}

func NewLocationInfo(nframes StackFrameOffset) *LocationInfo {
  // Get location info and add to details
  funcname := "*function*"
  classname := "*classname*"
  pc, filename, line, ok := runtime.Caller(int(nframes))
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
  return &LocationInfo{classname, funcname, filename, line, ColumnUnknown}
}
