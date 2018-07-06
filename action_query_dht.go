package holochain

import (
  "fmt"
  "reflect"
  "strings"
  "strconv"
  "github.com/tidwall/buntdb"
)

//------------------------------------------------------------
// Query

type APIFnQueryDHT struct {
  entryType string
	options *QueryDHTOptions
  zome *Zome
}

type QueryDHTOptions struct {
  Field string
  Constrain QueryDHTConstraint
  Ascending bool
  Page int
  Count int
}

type QueryDHTConstraint struct {
  EQ interface{}
  LT interface{}
  LTE interface{}
  GT interface{}
  GTE interface{}
  // Range QueryDHTRange
}

// type QueryDHTRange struct {
//   Lower string
//   Upper string
// }

func (a *APIFnQueryDHT) Name() string {
	return "queryDHT"
}

func (a *APIFnQueryDHT) Args() []Arg {
	return []Arg{
    {Name: "entryType", Type: StringArg},
    {Name: "queryOptions", Type: MapArg, MapType: reflect.TypeOf(QueryDHTOptions{}), Optional: false},
  }
}

func (a *APIFnQueryDHT) Call(h *Holochain) (response interface{}, err error) {
  entryType := a.entryType
  fieldPath := a.options.Field
  constrain := a.options.Constrain
  // ascending := a.options.Ascending
  db := h.dht.ht.(*BuntHT).db
  err = nil
  fmt.Println(constrain)
  // https://golang.org/pkg/encoding/json/#Unmarshal
  var hashList []string

  if constrain.EQ != nil {
    pivot := buildPivot(fieldPath, constrain.EQ)
    indexName := buildIndexName(&IndexDef{ZomeName: a.zome.Name, FieldPath: fieldPath, EntryType: entryType})
    fmt.Println(indexName)
    fmt.Println(pivot)
    db.View(func (tx *buntdb.Tx) (err error) {
      err = tx.AscendEqual(indexName, pivot, func (key, val string) bool {
        hashList = append(hashList, getHash(key))
        return true
      })
      return
    })
  } else {
    fmt.Println("sorry SOL")
  }
  // TODO: page, count
  return hashList, err
}

func getHash (key string) string {
  // "entry:[entryType]:[hash]" -> hash
  return strings.Split(key, ":")[2]
}

func unmarshalledValueToString (value interface{}) string {
  switch v := value.(type) {
  case string:
    return `"` + v + `"`
  case bool:
    if v {
      return "true"
    } else {
      return "false"
    }
  case float64:
    return strconv.FormatFloat(v, 'f', -1, 64)
  default:
    panic("could not convert value to string, float or bool")
  }
}

func buildPivot (fieldPath string, value interface{}) (result string) {
  // i.e. "address.city" => `{"address": {"city": value}}`



  fields := strings.Split(fieldPath, ".")
  result = unmarshalledValueToString(value)
  for i := len(fields) - 1; i >= 0; i-- {
    key := fields[i]
    result = fmt.Sprintf(`{"%s": %s}`, key, result)
  }
  return
}
//
// func getBuntIterator(constrain *QueryDHTConstraint, ascending bool) (func (a, b string) bool) {
//
// }
