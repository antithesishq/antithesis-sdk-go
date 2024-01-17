package io

import (
  "encoding/json"
  "errors"
  "github.com/antithesishq/antithesis-sdk-go/internal"
  "github.com/antithesishq/antithesis-sdk-go/local"
  )

type JSONDataInfo struct {
    Any any `json:"."`
}

func xLogEvent(name string, event any) {
  var data []byte = nil
  var err error

  event_map := internal.ToJsonableMap(name, event)

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
  if err := internal.Json_data(text); errors.Is(err, internal.DSOError) {
      local_info := local.LogJSONDataInfo{
        LocalLogInfo: *local.NewLogInfo("", ""),
        LocalJSONDataInfo: local.LocalJSONDataInfo{event_map},
      }
      payload := internal.JsonWithRenaming(local_info, ".", name)
     local.Emit(payload)
  }
}

func xOutputText(text string) {
  // Try the DSO first, then revert to local_output (if it was enabled)
  if err := internal.Info_message(text); err != nil {
     local.Log_text(text, "info")
  }
}

func xErrorText(text string) {
  // Try the DSO first, then revert to local_output (if it was enabled)
  if err := internal.Error_message(text); err != nil {
     local.Log_text(text, "err")
  }
}

// --------------------------------------------------------------------------------
// Setting the source name
// --------------------------------------------------------------------------------
func xSetSourceName(name string) {
  var err error

  // Try the DSO first, then update the source name for local output.
  if err = internal.Set_source_name(name); err != nil {
     local.Set_source_name(name)
  }
  return
}


// --------------------------------------------------------------------------------
// Console I/O
// --------------------------------------------------------------------------------
func xGetchar() (r rune, err error) {

  // Try the DSO first, then use the local getchar
  if r, err = internal.Getchar(); err != nil {
     r, err = local.Fuzz_getchar()
  }
  return r, err
}

// [PH] func Putchar(r rune) rune {
// [PH]   var err error
// [PH]   var r2 rune
// [PH] 
// [PH]   // Try the DSO first, then use the local putchar
// [PH]   if r2, err = internal.Putchar(r); err != nil {
// [PH]      r2 = local.Fuzz_putchar(r)
// [PH]   }
// [PH]   return r2
// [PH] }

func xFlush() {
  var err error

  // Try the DSO first, then use the local flush
  if err = internal.Flush(); err != nil {
     local.Fuzz_flush()
  }
  return
}

func xCoinFlip() bool {
    var err error
    var b bool

  // Try the DSO first, then use the local coin_flip
  if b, err = internal.Coin_flip(); err != nil {
     b = local.Fuzz_coin_flip()
  }
  return b
}

func GetRandom() uint64 {
    var err error
    var v uint64

  // Try the DSO first, then use the local get_random
  if v, err = internal.Get_random(); err != nil {
     v = local.Fuzz_get_random()
  }
  return v
}

func SetupComplete() {
    xLogEvent("sut_setup_status", "complete")
}

func RandomChoice(things []any) any {
    if len(things) < 1 {
        return nil
    }
    return things[0]
}
