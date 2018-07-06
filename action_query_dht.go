package holochain

import (
  "fmt"
  "reflect"
  "strings"

  "github.com/tidwall/buntdb"
)

//------------------------------------------------------------
// Query

type APIFnQueryDHT struct {
  entryType string
	options *QueryDHTOptions
}

type QueryDHTOptions struct {
  Field string
  Constrain QueryDHTConstraint
  Ascending bool
  Page int
  Count int
}

type QueryDHTConstraint struct {
  EQ string
  LT string
  LTE string
  GT string
  GTE string
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
  if constrain.GTE != "" {
    pivot := buildPivot(fieldPath, constrain.GTE)
    response := make([]string, 0)
    db.View(func (tx *buntdb.Tx) (err error) {
      indexName := buildIndexName(entryType, fieldPath)
      err = tx.AscendGreaterOrEqual(indexName, pivot, func (key, val string) bool {
        response = append(response, getHash(key))
        return true
      })
      return
    })
  } else {
    fmt.Println("sorry SOL")
  }
  // TODO: page, count
  return
}

func getHash (key string) string {
  // "entry:[entryType]:[hash]" -> hash
  return strings.Split(key, ":")[2]
}

func buildPivot (fieldPath string, value string) (result string) {
  // i.e. "address.city" => `{"address": {"city": value}}`
  fields := strings.Split(fieldPath, ".")
  result = `"` + value + `"`
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
