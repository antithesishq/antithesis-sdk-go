package local

import (
  "errors"
  "os"
  "time"
)

type LocalHandling struct {
   out_f *os.File
   can_be_opened bool
   start_time time.Time
   source_name string
}

var local_handler = &LocalHandling {
    out_f: nil,
    can_be_opened: true,
    start_time: time.Now(),
    source_name: "",
}

func Get_ticks() int64 {
    return local_handler.get_ticks()
}

func Get_time() string {
    return local_handler.get_time()
}

func Get_source_name() string {
    return local_handler.get_source_name()
}

func Set_source_name(name string) {
     local_handler.set_source_name(name)
}

func Log_text(text string, stream string) {
    if payload := format_log_text(text, stream); payload != "" {
        Emit(payload)
    }
}

func Emit(payload string) {
    local_handler.emit(payload)
}


// --------------------------------------------------------------------------------
// Local Handler (carries state to support local output)
// --------------------------------------------------------------------------------

func (pout *LocalHandling) get_ticks() int64 {
    duration := time.Since(pout.start_time)
    return duration.Nanoseconds()
}

func (pout *LocalHandling) get_time() string {
    utc := time.Now().UTC()
    return utc.Format(time.RFC3339Nano)
}

func (pout *LocalHandling) get_source_name() string {
    return pout.source_name
}

func (pout *LocalHandling) set_source_name(name string) {
     pout.source_name = name
}

// func (pout *LocalHandling) log_text(text string, stream string) {
//     emit_log_text(text, stream) 
// }

func (pout *LocalHandling) emit(payload string) {
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

func (pout *LocalHandling) open_output_file() error {
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
