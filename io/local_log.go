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

type LocalLogJSONDataInfo struct {
    LocalLogInfo
    JSONDataInfo
}

func NewLocalLogInfo(stream string, text string) *LocalLogInfo {
    log_info := LocalLogInfo{
        Ticks: local_handler.get_ticks(),
        TimeUTC: local_handler.get_time(),
        Source: local_handler.get_source_name(),
        Stream: stream,
        OutputText: text,
    }
    return &log_info
  }

func local_log_text(text string, stream string) {
  var err error
  var data []byte = nil

  output_info := NewLocalLogInfo(stream, text)
  if data, err = json.Marshal(output_info); err != nil {
      fmt.Fprintf(os.Stderr, "%s\n", err.Error())
      return
  }
  payload := string(data)
  local_handler.emit(payload)
  err = nil
}
