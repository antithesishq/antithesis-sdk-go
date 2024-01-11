package antilog

import (
  "encoding/json"
  "errors"
)


// --------------------------------------------------------------------------------
// Text output 
// --------------------------------------------------------------------------------
const TopLevelKey = "."
type AnyContainer struct {
    Any any `json:"."`
}


func LogEvent(name string, event any) {
  if event == nil {
      return
  }

  var event_map map[string]any = make(map[string]any,10) 
  var data []byte = nil
  var err error

  switch val := event.(type) {
  case int, int64:
      event_map[name] = val
  case int8:
      event_map[name] = int(val)
  case int16:
      event_map[name] = int(val)
  case int32:
      event_map[name] = int(val)
  case float64:
      event_map[name] = val
  case float32:
      event_map[name] = float64(val)
  case bool:
      event_map[name] = val
  case uint8:
      event_map[name] = uint(val)
  case uint16:
      event_map[name] = uint(val)
  case uint32:
      event_map[name] = uint(val)
  case uint, uint64:
      event_map[name] = val
  default:
      any_container := AnyContainer{event}
      if data, err = json.Marshal(any_container); err == nil {
        if err = json.Unmarshal(data, &event_map); err == nil {
            event_map[name] = event_map[TopLevelKey] 
            delete(event_map, TopLevelKey)
        }
      }
  }

  // Check for an empty map
  if len(event_map) == 0 {
      return
  }

  // Encode the map 
  if data, err = json.Marshal(event_map); err != nil {
      return 
  }
  text := string(data)

  // Try the DSO first
  if err := json_data(text); errors.Is(err, DSOError) {
      local_info := LocalLogJSONDataInfo{
        LocalLogInfo: *NewLocalLogInfo("", ""),
        JSONDataInfo: JSONDataInfo{event_map},
      }
      payload := JSONWithRenaming(local_info, ".", name)
     local_output.emit(payload)
  }
}

func JSONWithRenaming(anything any, old_name string, new_name string) string {
  var temp_map map[string]any = make(map[string]any,10) 
  var data []byte = nil
  var err error

      if data, err = json.Marshal(anything); err == nil {
        if err = json.Unmarshal(data, &temp_map); err == nil {
            temp_map[new_name] = temp_map[old_name] 
            delete(temp_map, old_name)
        }
      }

  if data, err = json.Marshal(temp_map); err != nil {
      return ""
  }
  return string(data)
}

func OutputText(text string) {
  // Try the DSO first
  if err := info_message(text); err != nil {
     log_text(text, "info")
  }
}

func ErrorText(text string) {
  // Try the DSO first
  if err := error_message(text); err != nil {
     log_text(text, "err")
  }
}

// --------------------------------------------------------------------------------
// Setting the source name
// --------------------------------------------------------------------------------
func SetSourceName(name string) {
  var err error

  // Try the DSO first
  if err = set_source_name(name); err != nil {
     local_output.set_source_name(name)
  }
  return
}

