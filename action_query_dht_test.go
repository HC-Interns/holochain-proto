package holochain

import (
  "fmt"
  "github.com/robertkrimen/otto"
  . "github.com/smartystreets/goconvey/convey"
  "testing"
  "encoding/json"
)

func getLookupHelpers(z *JSRibosome) (
  lookup func(string, string, bool, int, int, bool) string,
  lookupRange func(string, interface{}, interface{}, bool, int, int, bool) string,
) {

  lookup = func(field string, constraint string, ascending bool, count int, page int, load bool) (result string) {
    query := fmt.Sprintf(`
      JSON.stringify(queryDHT('profile', {
        Field: "%s",
        Constrain: {
          %s
        },
        Ascending: %v,
        Count: %v,
        Page: %v,
        Load: %v
      }))`, field, constraint, ascending, count, page, load)
    value, _ := z.Run(query)
    result, _ = value.(*otto.Value).ToString()
    return
  }

  lookupRange = func(field string, from interface{}, to interface{}, ascending bool, count int, page int, load bool) string {
    k := fmt.Sprintf("Range: {From: %v, To: %v}", from, to)
    return lookup(field, k, ascending, count, page, load)
  }

  return
}

func hashcat(hashes ...string) string {
  // Just join a bunch of hashes together with commas
  j, _ := json.Marshal(hashes)
  // list := fmt.Sprint(hashes[0])
  // for _, h := range hashes[1:] {
  //   list += "," + fmt.Sprint(h)
  // }
  return string(j)
}


func TestJSQueryDHT(t *testing.T) {
  d, _, h := PrepareTestChain("test")
  defer CleanupTestChain(h, d)
  zome, _ := h.GetZome("jsSampleZome")
  v, err := NewJSRibosome(h, zome)
  if err != nil {
    panic(err)
  }
  z := v.(*JSRibosome)

  lookup, _ := getLookupHelpers(z)

  Convey("Can query a string field using equality", t, func() {
    // add entries onto the chain to get hash values for testing
    profileEntry := `{"firstName":"Willem", "lastName":"Dafoe"}`
    hash := commit(h, "profile", profileEntry)
    results, _ := z.Run(`
      queryDHT('profile', {
        Field: "firstName",
        Constrain: {
          EQ: "Willem"
        },
        Ascending: true
      })`)

    res, _ := results.(*otto.Value).ToString()
    So(res, ShouldContainSubstring, fmt.Sprint(hash))
  })

  Convey("Can query a numeric field using equality", t, func() {
    // add entries onto the chain to get hash values for testing
    profileEntry := `{"firstName":"Willem", "lastName":"Dafoe", "age" : 62}`
    hash := commit(h, "profile", profileEntry)
    results, _ := z.Run(`
      queryDHT('profile', {
        Field: "age",
        Constrain: {
          EQ: 62
        },
        Ascending: true
      })`)

    res, _ := results.(*otto.Value).ToString()
    So(res, ShouldContainSubstring, fmt.Sprint(hash))
  })

  Convey("Can query a nested field using equality", t, func() {
    // add entries onto the chain to get hash values for testing
    profileEntry := `{"firstName":"Willem", "lastName":"Dafoe", "address" : {"isUnit" : true}}`
    hash := commit(h, "profile", profileEntry)
    So(lookup("address.isUnit", `EQ: true`, true, 10, 0, false), ShouldContainSubstring, fmt.Sprint(hash))
  })

  Convey("Can return multiple matches using equality", t, func() {
    // add entries onto the chain to get hash values for testing
    profileEntry := `{"firstName":"Willem", "lastName":"de kooningg", "address" : {"isUnit" : true}}`
    hash := commit(h, "profile", profileEntry)
    So(lookup("firstName", `EQ: "Willem"`, true, 10, 0, false), ShouldContainSubstring, fmt.Sprint(hash))
  })

  Convey("Can index strings with spaces", t, func() {
    // add entries onto the chain to get hash values for testing
    profileEntry := `{"firstName":"Willem", "lastName":"a last name", "address" : {"isUnit" : true}}`
    hash := commit(h, "profile", profileEntry)

    results, _ := z.Run(`
      queryDHT('profile', {
        Field: "firstName",
        Constrain: {
          EQ: "Willem"
        },
        Ascending: true
      })`)

    res, _ := results.(*otto.Value).ToString()
    So(res, ShouldContainSubstring, fmt.Sprint(hash))
  })
}

func TestJSQueryDHTOrdinal(t *testing.T) {
  d, _, h := PrepareTestChain("test")
  defer CleanupTestChain(h, d)
  zome, _ := h.GetZome("jsSampleZome")
  v, err := NewJSRibosome(h, zome)
  if err != nil {
    panic(err)
  }
  z := v.(*JSRibosome)

  lookup, lookupRange := getLookupHelpers(z)

  // add entries onto the chain to get hash values for testing
  profileEntry1 := `{"firstName":"Willem", "lastName":"a last name", "age" : 26}`
  profileEntry2 := `{"firstName":"Maackle", "lastName":"Diggity", "age" : 33}`
  profileEntry3 := `{"firstName":"Polly", "lastName":"Person", "age" : 37}`
  hash1 := fmt.Sprint(commit(h, "profile", profileEntry1))
  hash2 := fmt.Sprint(commit(h, "profile", profileEntry2))
  hash3 := fmt.Sprint(commit(h, "profile", profileEntry3))

  Convey("Can query numeric fields using ordinal lookup", t, func() {
    So(lookup("age", "LT: 100", true, 10, 0, false), ShouldEqual, hashcat(hash1, hash2, hash3))
    So(lookup("age", "LT: 30", true, 10, 0, false), ShouldEqual, hashcat(hash1))
    So(lookup("age", "GT: 30", true, 10, 0, false), ShouldEqual, hashcat(hash2, hash3))
    So(lookup("age", "GTE: 33", true, 10, 0, false), ShouldEqual, hashcat(hash2, hash3))
    So(lookup("age", "GT: 33", true, 10, 0, false), ShouldEqual, hashcat(hash3))
  })

  Convey("Can query a range", t, func() {
    So(lookupRange("age", 0, 100, true, 10, 0, false), ShouldEqual, hashcat(hash1, hash2, hash3))
    So(lookupRange("age", 100, 0, false, 10, 0, false), ShouldEqual, hashcat(hash3, hash2, hash1))
    So(lookupRange("age", 30, 20, false, 10, 0, false), ShouldEqual, hashcat(hash1))
    So(lookupRange("age", 30, 40, true, 10, 0, false), ShouldEqual, hashcat(hash2, hash3))
    So(lookupRange("age", 40, 50, true, 10, 0, false), ShouldEqual, hashcat(""))
    So(lookupRange("age", 40, 20, true, 10, 0, false), ShouldEqual, hashcat(""))
  })

  Convey("Can query numeric fields ascending or descending", t, func() {
    forward := hashcat(hash1, hash2, hash3)
    backward := hashcat(hash3, hash2, hash1)
    cases := []string{
      "LT:100", "LTE:100", "GT:0", "GTE:0",
    }
    for _, k := range cases {
      So(lookup("age", k, true, 10, 0, false), ShouldEqual, forward)
      So(lookup("age", k, false, 10, 0, false), ShouldEqual, backward)
    }
    So(lookupRange("age", 30, 40, true, 10, 0, false), ShouldEqual, hashcat(hash2, hash3))
    So(lookupRange("age", 40, 30, false, 10, 0, false), ShouldEqual, hashcat(hash3, hash2))
  })

  Convey("Can load values in ordinal query", t, func() {
    So(lookupRange("age", 30, 40, true, 10, 0, true), ShouldEqual, `[{"Entry":"{\"firstName\":\"Maackle\", \"lastName\":\"Diggity\", \"age\" : 33}","Hash":"QmUUSqWxPi88CVVj6VGgsKXghWYS997VmabLDq9DnJokjT"},{"Entry":"{\"firstName\":\"Polly\", \"lastName\":\"Person\", \"age\" : 37}","Hash":"QmbeiEd1mSwjWbXd7fe355TJpPj5cF5fhkjwxxHqFcjVNK"}]`)
  })
}

func TestJSQueryDHTPaging(t *testing.T) {
  d, _, h := PrepareTestChain("test")
  defer CleanupTestChain(h, d)
  zome, _ := h.GetZome("jsSampleZome")
  v, err := NewJSRibosome(h, zome)
  if err != nil {
    panic(err)
  }
  z := v.(*JSRibosome)

  lookup, lookupRange := getLookupHelpers(z)

  var hashes []string

  for i := 1; i <= 20; i++ {
    entry := fmt.Sprintf(
      `{"firstName":"Hiro", "lastName":"Protagonist", "age": %v}`, i*5,
    )
    hashes = append(hashes, fmt.Sprint(commit(h, "profile", entry)))
  }

  sehsah := make([]string, 20)
  copy(sehsah, hashes)
  reverseArray(sehsah)

  Convey("Can get multiple pages ascending", t, func() {
    So(lookup("age", "LT:  50", true, 5, 0, false), ShouldEqual, hashcat(hashes[0:5]...))
    So(lookup("age", "LT:  50", true, 5, 1, false), ShouldEqual, hashcat(hashes[5:9]...))
    So(lookup("age", "LTE: 50", true, 5, 1, false), ShouldEqual, hashcat(hashes[5:10]...))
    So(lookup("age", "LTE: 50", true, 5, 2, false), ShouldEqual, hashcat(""))
    So(lookup("age", "LTE: 60", true, 5, 2, false), ShouldEqual, hashcat(hashes[10:12]...))

    So(lookup("age", "GT: 50", true, 5, 0, false), ShouldEqual, hashcat(hashes[10:15]...))

    So(lookupRange("age", 15, 100, true, 5, 0, false), ShouldEqual, hashcat(hashes[2:7]...))
    So(lookupRange("age", 15, 100, true, 5, 1, false), ShouldEqual, hashcat(hashes[7:12]...))
    So(lookupRange("age", 15, 100, true, 50, 0, false), ShouldEqual, hashcat(hashes[2:19]...))
    So(lookupRange("age", 15, 101, true, 50, 0, false), ShouldEqual, hashcat(hashes[2:]...))
  })

  Convey("Can get multiple pages descending", t, func() {
    So(lookup("age", "LT:  50", false, 5, 0, false), ShouldEqual, hashcat(sehsah[11:16]...))
    So(lookup("age", "LT:  50", false, 5, 1, false), ShouldEqual, hashcat(sehsah[16:20]...))
    So(lookup("age", "LTE: 50", false, 5, 1, false), ShouldEqual, hashcat(sehsah[15:20]...))
    So(lookup("age", "LTE: 50", false, 5, 2, false), ShouldEqual, hashcat(""))
    So(lookup("age", "LTE: 60", false, 5, 2, false), ShouldEqual, hashcat(sehsah[18:20]...))

    So(lookup("age", "GT: 50", false, 5, 0, false), ShouldEqual, hashcat(sehsah[0:5]...))

    So(lookupRange("age", 85, 15, false, 5, 0, false), ShouldEqual, hashcat(sehsah[3:8]...))
    So(lookupRange("age", 85, 15, false, 5, 1, false), ShouldEqual, hashcat(sehsah[8:13]...))
    So(lookupRange("age", 85, 15, false, 50, 0, false), ShouldEqual, hashcat(sehsah[3:17]...))
    So(lookupRange("age", 101, 15, false, 50, 0, false), ShouldEqual, hashcat(sehsah[:17]...))
  })
}
