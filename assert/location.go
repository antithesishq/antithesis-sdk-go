package assert

import (
	"path"
	"runtime"
	"strings"
)

// stackFrameOffset indicates how many frames to go up in the
// call stack to find the filename/location/line info.  As
// this work is always done in NewLocationInfo(), the offset is
// specified from the perspective of NewLocationInfo
type stackFrameOffset int

// Order is important here since iota is being used
const (
	offsetNewLocationInfo stackFrameOffset = iota
	offsetHere
	offsetAPICaller
	offsetAPICallersCaller
)

// locationInfo represents the attributes known at instrumentation time
// for each Antithesis assertion discovered
type locationInfo struct {
	Classname string `json:"classname"`
	Funcname  string `json:"function"`
	Filename  string `json:"filename"`
	Line      int    `json:"line"`
	Column    int    `json:"column"`
}

// columnUnknown is used when the column associated with
// a locationInfo is not available
const columnUnknown = 0

// NewLocationInfo creates a locationInfo directly from
// the current execution context
func newLocationInfo(nframes stackFrameOffset) *locationInfo {
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
	return &locationInfo{classname, funcname, filename, line, columnUnknown}
}
