package lifecycle

import (
  "encoding/json"
  "errors"
  "github.com/antithesishq/antithesis-sdk-go/internal"
  "github.com/antithesishq/antithesis-sdk-go/local"
  )

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

func SetupComplete() {
    xLogEvent("sut_setup_status", "complete")
}

