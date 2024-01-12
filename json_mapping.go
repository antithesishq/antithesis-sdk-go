package antilog

import (
  "encoding/json"
)

// --------------------------------------------------------------------------------
// JSON Helpers
// --------------------------------------------------------------------------------
const TopLevelKey = "."
type AnyContainer struct {
    Any any `json:"."`
}

func to_jsonable_map(name string, item any) map[string]any {
  var item_map map[string]any = make(map[string]any) 
  if item == nil {
      return item_map
  }

  var data []byte = nil
  var err error

  switch val := item.(type) {
  case int, int64:
      item_map[name] = val
  case int8:
      item_map[name] = int(val)
  case int16:
      item_map[name] = int(val)
  case int32:
      item_map[name] = int(val)
  case float64:
      item_map[name] = val
  case float32:
      item_map[name] = float64(val)
  case bool:
      item_map[name] = val
  case uint8:
      item_map[name] = uint(val)
  case uint16:
      item_map[name] = uint(val)
  case uint32:
      item_map[name] = uint(val)
  case uint, uint64:
      item_map[name] = val
  default:
      any_container := AnyContainer{item}
      if data, err = json.Marshal(any_container); err == nil {
        if err = json.Unmarshal(data, &item_map); err == nil {
            item_map[name] = item_map[TopLevelKey] 
            delete(item_map, TopLevelKey)
        }
      }
  }
  return item_map
}

func json_with_renaming(anything any, old_name string, new_name string) string {
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



