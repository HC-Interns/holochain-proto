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
  Range QueryDHTRange
}

type QueryDHTRange struct {
  From interface{}
  To interface{}
}

func (a *APIFnQueryDHT) Name() string {
	return "queryDHT"
}

func (a *APIFnQueryDHT) Args() []Arg {
	return []Arg{
    {Name: "entryType", Type: StringArg},
    {Name: "queryOptions", Type: MapArg, MapType: reflect.TypeOf(QueryDHTOptions{}), Optional: false},
  }
}

type IterFn func (key, val string) bool

func (a *APIFnQueryDHT) Call(h *Holochain) (response interface{}, err error) {
  entryType := a.entryType
  fieldPath := a.options.Field
  constrain := a.options.Constrain
  ascending := a.options.Ascending
  // ascending := a.options.Ascending
  db := h.dht.ht.(*BuntHT).db
  err = nil
  // https://golang.org/pkg/encoding/json/#Unmarshal

  indexName := buildIndexName(&IndexDef{ZomeName: a.zome.Name, FieldPath: fieldPath, EntryType: entryType})
  var hashList []string

  if constrain.EQ != nil {
    hashList = collectHashes(db, !ascending, func (tx *buntdb.Tx, f IterFn) error {
      return tx.AscendEqual(indexName, buildPivot(fieldPath, constrain.EQ), f)
    })
  } else if constrain.LT != nil {
    hashList = collectHashes(db, !ascending, func (tx *buntdb.Tx, f IterFn) error {
      return tx.AscendLessThan(indexName, buildPivot(fieldPath, constrain.LT), f)
    })
  } else if constrain.GT != nil {
    hashList = collectHashes(db, ascending, func (tx *buntdb.Tx, f IterFn) error {
      return tx.DescendGreaterThan(indexName, buildPivot(fieldPath, constrain.GT), f)
    })
  } else if constrain.LTE != nil {
    hashList = collectHashes(db, ascending, func (tx *buntdb.Tx, f IterFn) error {
      return tx.DescendLessOrEqual(indexName, buildPivot(fieldPath, constrain.LTE), f)
    })
  } else if constrain.GTE != nil {
    hashList = collectHashes(db, !ascending, func (tx *buntdb.Tx, f IterFn) error {
      return tx.AscendGreaterOrEqual(indexName, buildPivot(fieldPath, constrain.GTE), f)
    })
  } else if constrain.Range.From != nil && constrain.Range.To != nil {
    pivot1 := buildPivot(fieldPath, constrain.Range.From)
    pivot2 := buildPivot(fieldPath, constrain.Range.To)
    hashList = collectHashes(db, !ascending, func (tx *buntdb.Tx, f IterFn) error {
      return tx.AscendRange(indexName, pivot1, pivot2, f)
    })
  } else {
    panic(fmt.Sprintf("Invalid constraints: %v", constrain))
  }
  // TODO: page, count
  return hashList, err
}

func collectHashes (db *buntdb.DB, reverse bool, iterateFn func (*buntdb.Tx, IterFn) error) []string {
  // combinator to abstract away some of the common logic between different constraints
  var hashList []string
  innerFunc := func (key, val string) bool {
    hashList = append(hashList, getHash(key))
    return true
  }
  db.View(func (tx *buntdb.Tx) (err error) {
    err = iterateFn(tx, innerFunc)
    return
  })
  if reverse {
    hashList = reverseArray(hashList)
  }
  return hashList
}

func reverseArray(vals []string) []string {
	for i := 0; i < len(vals)/2; i++ {
		j := len(vals) - i - 1
		vals[i], vals[j] = vals[j], vals[i]
	}
	return vals
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
    panic(fmt.Sprintf("could not convert value to string, float or bool: %v", v))
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
