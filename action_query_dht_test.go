package holochain

import (
  "fmt"
  "github.com/robertkrimen/otto"
  . "github.com/smartystreets/goconvey/convey"
  "testing"
)

func getLookupHelpers(z *JSRibosome) (
  lookup func(string, string, bool) string,
  lookupRange func(string, interface{}, interface{}, bool) string,
) {

  lookup = func(field string, constraint string, ascending bool) (result string) {
    query := fmt.Sprintf(`
      queryDHT('profile', {
        Field: "%s",
        Constrain: {
          %s
        },
        Ascending: %v
      })`, field, constraint, ascending)
    value, _ := z.Run(query)
    result, _ = value.(*otto.Value).ToString()
    return
  }

  lookupRange = func(field string, from interface{}, to interface{}, ascending bool) string {
    k := fmt.Sprintf("Range: {From: %v, To: %v}", from, to)
    return lookup(field, k, ascending)
  }

  return
}

func hashcat(hashes ...string) string {
  // Just join a bunch of hashes together with commas
  list := fmt.Sprint(hashes[0])
  for _, h := range hashes[1:] {
    list += "," + fmt.Sprint(h)
  }
  return list
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
    So(lookup("address.isUnit", `EQ: true`, true), ShouldContainSubstring, fmt.Sprint(hash))
  })

  Convey("Can return multiple matches using equality", t, func() {
    // add entries onto the chain to get hash values for testing
    profileEntry := `{"firstName":"Willem", "lastName":"de kooningg", "address" : {"isUnit" : true}}`
    hash := commit(h, "profile", profileEntry)
    So(lookup("firstName", `EQ: "Willem"`, true), ShouldContainSubstring, fmt.Sprint(hash))
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
    So(lookup("age", "LT: 100", true), ShouldEqual, hashcat(hash1, hash2, hash3))
    So(lookup("age", "LT: 30", true), ShouldEqual, hashcat(hash1))
    So(lookup("age", "GT: 30", true), ShouldEqual, hashcat(hash2, hash3))
    So(lookup("age", "GTE: 33", true), ShouldEqual, hashcat(hash2, hash3))
    So(lookup("age", "GT: 33", true), ShouldEqual, hashcat(hash3))
  })

  Convey("Can query a range", t, func() {
    So(lookupRange("age", 0, 100, true), ShouldEqual, hashcat(hash1, hash2, hash3))
    So(lookupRange("age", 0, 100, false), ShouldEqual, hashcat(hash3, hash2, hash1))
    So(lookupRange("age", 20, 30, false), ShouldEqual, hashcat(hash1))
    So(lookupRange("age", 30, 40, true), ShouldEqual, hashcat(hash2, hash3))
    So(lookupRange("age", 40, 50, true), ShouldEqual, hashcat(""))
    So(lookupRange("age", 40, 20, true), ShouldEqual, hashcat(""))
  })

  Convey("Can query numeric fields ascending or descending", t, func() {
    forward := hashcat(hash1, hash2, hash3)
    backward := hashcat(hash3, hash2, hash1)
    cases := []string{
      "LT:100", "LTE:100", "GT:0", "GTE:0",
    }
    for _, k := range cases {
      So(lookup("age", k, true), ShouldEqual, hashcat(forward))
      So(lookup("age", k, false), ShouldEqual, hashcat(backward))
    }
    So(lookupRange("age", 30, 40, true), ShouldEqual, hashcat(hash2, hash3))
    So(lookupRange("age", 30, 40, false), ShouldEqual, hashcat(hash3, hash2))
  })
}
