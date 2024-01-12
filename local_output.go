package antilog

import (
  "bufio"
  "crypto/rand"
  "encoding/json"
  "errors"
  "fmt"
  "golang.org/x/term"
  "math"
  "math/big"
  "os"
  "time"
  "unicode/utf8"
)

// --------------------------------------------------------------------------------
// Local Handling
// --------------------------------------------------------------------------------
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
     local_handler.source_name = name
}

func (pout *LocalHandling) getchar() (r rune, err error) {
    return local_fuzz_getchar() 
}

func (pout *LocalHandling) putchar(r rune) rune {
    return local_fuzz_putchar(r) 
}

func (pout *LocalHandling) flush() {
    local_fuzz_flush() 
}

func (pout *LocalHandling) get_random() uint64 {
    return local_fuzz_get_random() 
}

func (pout *LocalHandling) coin_flip() bool {
    return local_fuzz_coin_flip() 
}

func (pout *LocalHandling) log_text(text string, stream string) {
    local_log_text(text, stream) 
}

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


// --------------------------------------------------------------------------------
// Local Handling for SDK functions - not assertion related
// --------------------------------------------------------------------------------

// sbuf is a static buffer for getchar/putchar
var s_data [4096]byte
var sbuf []byte = s_data[:0]

func local_fuzz_getchar() (r rune, err error) {
    local_fuzz_flush()
    var state *term.State
    var stdin = int(os.Stdin.Fd())
    if state, err = term.MakeRaw(stdin); err != nil {
      return 0, err 
    }
    defer func() {
        if err = term.Restore(stdin, state); err != nil {
            fmt.Fprintln(os.Stderr, "warning, failed to restore terminal:", err)
        }
    }()

    in := bufio.NewReader(os.Stdin)
    r, _, err = in.ReadRune()
    return r, err
}

func local_fuzz_putchar(r rune) rune {
    len_buf := len(sbuf)
    cap_buf := cap(sbuf)
    if len_buf == cap_buf {
        local_fuzz_flush()
        len_buf = 0
    }
    rune_len := 0
    if rune_len = utf8.RuneLen(r); rune_len == -1 {
        return 0
    }

    if room_avail := cap_buf - len_buf; room_avail < rune_len {
        local_fuzz_flush()
    }
    utf8.AppendRune(sbuf, r)
    return r
}

func local_fuzz_flush() {
    s := string(sbuf)
    OutputText(s)
    sbuf = s_data[:0]
}

func local_fuzz_get_random() uint64 {
    var err error
    var randInt *big.Int
    max := big.NewInt(math.MaxInt64)
    if randInt, err = rand.Int(rand.Reader, max); err != nil {
        return 0
    }
    return randInt.Uint64()
}

func local_fuzz_coin_flip() bool {
    n := local_fuzz_get_random()
    return ((n % 2) == 0)
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

