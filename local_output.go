package antilog

import (
  "encoding/json"
  "errors"
  "fmt"
  "os"
  "time"
)

// --------------------------------------------------------------------------------
// Local Output
// --------------------------------------------------------------------------------
type LocalOutput struct {
   out_f *os.File
   can_be_opened bool
   start_time time.Time
   source_name string
}

var local_output = &LocalOutput {
    out_f: nil,
    can_be_opened: true,
    start_time: time.Now(),
    source_name: "",
}

func (pout *LocalOutput) get_ticks() int64 {
    duration := time.Since(pout.start_time)
    return duration.Nanoseconds()
}

func (pout *LocalOutput) get_time() string {
    utc := time.Now().UTC()
    return utc.Format(time.RFC3339Nano)
}

func (pout *LocalOutput) get_source_name() string {
    return pout.source_name
}

func (pout *LocalOutput) set_source_name(name string) {
     local_output.source_name = name
}

func (pout *LocalOutput) emit(payload string) {
  var err error
  if !pout.can_be_opened {
      return
  }
  if pout.out_f == nil {
    if err = pout.open_output_file(); err != nil {
      pout.can_be_opened = false
      return
    }
  }
  pout.out_f.WriteString(payload + "\n")
  return 
}

func (pout *LocalOutput) open_output_file() error {
  var file *os.File
  var err error

  // Make sure we have user intent to open a file
  out_path := os.Getenv("ANTILOG_OUTPUT") // Write output to this file
  if (len(out_path) == 0) {
    return errors.New("No local output")
  }

  // Open the file R/W (create if needed and possible)
  if file, err = os.OpenFile(out_path, os.O_RDWR | os.O_CREATE, 0644); err != nil {
    file = nil
    return err
  }

  // Truncate the file if possible (if not, consider the file unusable)
  if file != nil {
    if err = file.Truncate(0); err != nil {
      file = nil
      return err
    }
  }


  if file != nil {
    pout.out_f = file
  }

  return err
}


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

type LocalLogAssertInfo struct {
    LocalLogInfo
    WrappedAssertInfo
}

type LocalLogJSONDataInfo struct {
    LocalLogInfo
    JSONDataInfo
}

func NewLocalLogInfo(stream string, text string) *LocalLogInfo {
    log_info := LocalLogInfo{
        Ticks: local_output.get_ticks(),
        TimeUTC: local_output.get_time(),
        Source: local_output.get_source_name(),
        Stream: stream,
        OutputText: text,
    }
    return &log_info
  }

func log_text(text string, stream string) {
  var err error
  var data []byte = nil

  output_info := NewLocalLogInfo(stream, text)
  if data, err = json.Marshal(output_info); err != nil {
      fmt.Fprintf(os.Stderr, "%s\n", err.Error())
      return
  }
  payload := string(data)
  local_output.emit(payload)
  err = nil
}

