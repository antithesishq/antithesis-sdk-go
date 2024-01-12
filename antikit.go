package antilog

import (
  "encoding/json"
  "errors"
  )

// --------------------------------------------------------------------------------
// JSON and general text output 
// --------------------------------------------------------------------------------
func LogEvent(name string, event any) {
  var data []byte = nil
  var err error

  event_map := to_jsonable_map(name, event)

  // Make sure JSON Marshaling delivered something useful
  if len(event_map) == 0 {
      return
  }

  // Encode the map 
  if data, err = json.Marshal(event_map); err != nil {
      return 
  }
  text := string(data)

  // Try the DSO first, then revert to local_output (if it was enabled)
  if err := json_data(text); errors.Is(err, DSOError) {
      local_info := LocalLogJSONDataInfo{
        LocalLogInfo: *NewLocalLogInfo("", ""),
        JSONDataInfo: JSONDataInfo{event_map},
      }
      payload := json_with_renaming(local_info, ".", name)
     local_handler.emit(payload)
  }
}

func OutputText(text string) {
  // Try the DSO first, then revert to local_output (if it was enabled)
  if err := info_message(text); err != nil {
     local_handler.log_text(text, "info")
  }
}

func ErrorText(text string) {
  // Try the DSO first, then revert to local_output (if it was enabled)
  if err := error_message(text); err != nil {
     local_handler.log_text(text, "err")
  }
}

// --------------------------------------------------------------------------------
// Setting the source name
// --------------------------------------------------------------------------------
func SetSourceName(name string) {
  var err error

  // Try the DSO first, then update the source name for local output.
  if err = set_source_name(name); err != nil {
     local_handler.set_source_name(name)
  }
  return
}


// --------------------------------------------------------------------------------
// Console I/O
// --------------------------------------------------------------------------------
func Getchar() (r rune, err error) {

  // Try the DSO first, then use the local getchar
  if r, err = getchar(); err != nil {
     r, err = local_handler.getchar()
  }
  return r, err
}

func Putchar(r rune) rune {
  var err error
  var r2 rune

  // Try the DSO first, then use the local putchar
  if r2, err = putchar(r); err != nil {
     r2 = local_handler.putchar(r)
  }
  return r2
}

func Flush() {
  var err error

  // Try the DSO first, then use the local flush
  if err = flush(); err != nil {
     local_handler.flush()
  }
  return
}

func GetRandom() uint64 {
    var err error
    var v uint64

  // Try the DSO first, then use the local get_random
  if v, err = get_random(); err != nil {
     v = local_handler.get_random()
  }
  return v
}

func CoinFlip() bool {
    var err error
    var b bool

  // Try the DSO first, then use the local coin_flip
  if b, err = coin_flip(); err != nil {
     b = local_handler.coin_flip()
  }
  return b
}

