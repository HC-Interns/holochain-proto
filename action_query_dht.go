package holochain

import (
  "fmt"
  "reflect"
  "strings"
  "encoding/json"
  "strconv"
  "github.com/tidwall/buntdb"
  . "github.com/HC-Interns/holochain-proto/hash"
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
  Load bool
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

type QueryDHTResponse struct {
  Hash string
  Entry string
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

const DEFAULT_COUNT = 20

type IterFn func (key, val string) bool

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}
func max(a, b int) int {
    if a > b {
        return a
    }
    return b
}

func (a *APIFnQueryDHT) Call(h *Holochain) (response interface{}, err error) {

  entryType := a.entryType
  fieldPath := a.options.Field
  constrain := a.options.Constrain
  ascending := a.options.Ascending
  load := a.options.Load
  db := h.dht.ht.(*BuntHT).db
  err = nil
  // https://golang.org/pkg/encoding/json/#Unmarshal

  indexName := buildIndexName(&IndexDef{ZomeName: a.zome.Name, FieldPath: fieldPath, EntryType: entryType})
  var hashList []string

  // TODO: stop iteration after count entries when possible
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
    if ascending {
      pivot1 := buildPivot(fieldPath, constrain.Range.From)
      pivot2 := buildPivot(fieldPath, constrain.Range.To)
      hashList = collectHashes(db, false, func (tx *buntdb.Tx, f IterFn) error {
        return tx.AscendRange(indexName, pivot1, pivot2, f)
      })
    } else {
      pivot1 := buildPivot(fieldPath, constrain.Range.From)
      pivot2 := buildPivot(fieldPath, constrain.Range.To)
      hashList = collectHashes(db, false, func (tx *buntdb.Tx, f IterFn) error {
        return tx.DescendRange(indexName, pivot1, pivot2, f)
      })
    }
  } else {
    panic(fmt.Sprintf("Invalid constraints: %v", constrain))
  }

  count := a.options.Count
  if count == 0 {
    count = DEFAULT_COUNT
  }
  offset := max(0, min(count * a.options.Page, len(hashList)))
  end := max(0, min(offset + count, len(hashList)))

  limitedHashlist := hashList[offset:end]

  if load && err==nil {
    return loadEntries(h, limitedHashlist)
  } else {
    return limitedHashlist, err
  }
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

func loadEntries(h *Holochain, hashList []string) (response []QueryDHTResponse, err error) {
  var r []QueryDHTResponse
  for _, hashString := range hashList {
    hash, _ := NewHash(hashString)
    result := QueryDHTResponse{Hash: hashString}
    opts := GetOptions{GetMask: GetMaskEntry, StatusMask: StatusDefault} // can add params to query DHT to change these later
    
    req := GetReq{H: hash, StatusMask: StatusDefault, GetMask: opts.GetMask}
    var rsp interface{}
    rsp, err = callGet(h, req, &opts)
    if err == nil {
      // code borrowed from action_getlinks. Good thing this is the proto version ;)
      entry := rsp.(GetResp).Entry
      switch content := entry.Content().(type) {
      case string:
        result.Entry = content
      case []byte:
        var j []byte
        j, err = json.Marshal(content)
        if err != nil {
          return
        }
        result.Entry = string(j)
      default:
        err = fmt.Errorf("bad type in entry content: %T:%v", content, content)
      }
    }
    r = append(r, result)
  }
  response = r
  return
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
