package local

import (
  "encoding/json"
  "fmt"
  "os"
)

// --------------------------------------------------------------------------------
// Local Logging
// --------------------------------------------------------------------------------
type LocalLogInfo struct {
    Ticks int64 `json:"ticks"`
    TimeUTC string `json:"time"`
    Source string `json:"source"`
    Stream string `json:"stream"`
    OutputText string `json:"output_text"`
}
type LocalJSONDataInfo struct {
    Any any `json:"."`
}

type LogJSONDataInfo struct {
    LocalLogInfo
    LocalJSONDataInfo
}

func NewLogInfo(stream string, text string) *LocalLogInfo {
    log_info := LocalLogInfo{
        Ticks: Get_ticks(),
        TimeUTC: Get_time(),
        Source: Get_source_name(),
        Stream: stream,
        OutputText: text,
    }
    return &log_info
  }

func format_log_text(text string, stream string) string {
  var err error
  var data []byte = nil

  output_info := NewLogInfo(stream, text)
  if data, err = json.Marshal(output_info); err != nil {
      fmt.Fprintf(os.Stderr, "%s\n", err.Error())
      return ""
  }
  return string(data)
}
