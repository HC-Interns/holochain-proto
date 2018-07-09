package holochain

import (
  "fmt"
  "github.com/robertkrimen/otto"
  . "github.com/smartystreets/goconvey/convey"
  "testing"
)

func TestJSQueryDHT(t *testing.T) {
  d, _, h := PrepareTestChain("test")
  defer CleanupTestChain(h, d)
  zome, _ := h.GetZome("jsSampleZome")
  v, err := NewJSRibosome(h, zome)
  if err != nil {
    panic(err)
  }
  z := v.(*JSRibosome)

  Convey("Can query a string field using equality", t, func() {
    // add entries onto the chain to get hash values for testing
    profileEntry := `{"firstName":"Willem", "lastName":"Dafoe"}`
    hash := commit(h, "profile", profileEntry)
    fmt.Println(hash)

    results, _ := z.Run(`
      queryDHT('profile', {
        Field: "firstName",
        Constrain: {
          EQ: "Willem"
        },
        Ascending: true
      })`)

    res, _ := results.(*otto.Value).ToString()
    fmt.Println(res)
    So(res, ShouldContainSubstring, fmt.Sprint(hash))
  })

  Convey("Can query a numeric field using equality", t, func() {
    // add entries onto the chain to get hash values for testing
    profileEntry := `{"firstName":"Willem", "lastName":"Dafoe", "age" : 62}`
    hash := commit(h, "profile", profileEntry)
    fmt.Println(hash)

    results, _ := z.Run(`
      queryDHT('profile', {
        Field: "age",
        Constrain: {
          EQ: 62
        },
        Ascending: true
      })`)

    res, _ := results.(*otto.Value).ToString()
    fmt.Println(res)
    So(res, ShouldContainSubstring, fmt.Sprint(hash))
  })

  Convey("Can query a nested field using equality", t, func() {
    // add entries onto the chain to get hash values for testing
    profileEntry := `{"firstName":"Willem", "lastName":"Dafoe", "address" : {"isUnit" : true}}`
    hash := commit(h, "profile", profileEntry)
    fmt.Println(hash)

    results, _ := z.Run(`
      queryDHT('profile', {
        Field: "address.isUnit",
        Constrain: {
          EQ: true
        },
        Ascending: true
      })`)

    res, _ := results.(*otto.Value).ToString()
    fmt.Println(res)
    So(res, ShouldContainSubstring, fmt.Sprint(hash))
  })

  Convey("Can return multiple matches using equality", t, func() {
    // add entries onto the chain to get hash values for testing
    profileEntry := `{"firstName":"Willem", "lastName":"de kooningg", "address" : {"isUnit" : true}}`
    hash := commit(h, "profile", profileEntry)
    fmt.Println(hash)

    results, _ := z.Run(`
      queryDHT('profile', {
        Field: "firstName",
        Constrain: {
          EQ: "Willem"
        },
        Ascending: true
      })`)

    res, _ := results.(*otto.Value).ToString()
    fmt.Println(res)
    So(res, ShouldContainSubstring, fmt.Sprint(hash))
  })

  Convey("Can index strings with spaces", t, func() {
    // add entries onto the chain to get hash values for testing
    profileEntry := `{"firstName":"Willem", "lastName":"a last name", "address" : {"isUnit" : true}}`
    hash := commit(h, "profile", profileEntry)
    fmt.Println(hash)

    results, _ := z.Run(`
      queryDHT('profile', {
        Field: "firstName",
        Constrain: {
          EQ: "Willem"
        },
        Ascending: true
      })`)

    res, _ := results.(*otto.Value).ToString()
    fmt.Println(res)
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

  // add entries onto the chain to get hash values for testing
  profileEntry1 := `{"firstName":"Willem", "lastName":"a last name", "age" : 26}`
  profileEntry2 := `{"firstName":"Maackle", "lastName":"Diggity", "age" : 33}`
  profileEntry3 := `{"firstName":"Polly", "lastName":"Person", "age" : 37}`
  hash1 := fmt.Sprint(commit(h, "profile", profileEntry1))
  hash2 := fmt.Sprint(commit(h, "profile", profileEntry2))
  hash3 := fmt.Sprint(commit(h, "profile", profileEntry3))

  lookup := func(constraint string, ascending bool) (result string) {
    query := fmt.Sprintf(`
      queryDHT('profile', {
        Field: "age",
        Constrain: {
          %s
        },
        Ascending: %v
      })`, constraint, ascending)
    value, _ := z.Run(query)
    result, _ = value.(*otto.Value).ToString()
    return
  }

  lookupRange := func(from interface{}, to interface{}, ascending bool) string {
    k := fmt.Sprintf("Range: {From: %v, To: %v}", from, to)
    return lookup(k, ascending)
  }

  hashcat := func(hashes ...string) string {
    // Just join a bunch of hashes together with commas
    list := hashes[0]
    for _, h := range hashes[1:] {
      list += "," + h
    }
    return list
  }

  Convey("Can query numeric fields using ordinal lookup", t, func() {
    So(lookup("LT: 100", true), ShouldEqual, hashcat(hash1, hash2, hash3))
    So(lookup("LT: 30", true), ShouldEqual, hashcat(hash1))
    So(lookup("GT: 30", true), ShouldEqual, hashcat(hash2, hash3))
    So(lookup("GTE: 33", true), ShouldEqual, hashcat(hash2, hash3))
    So(lookup("GT: 33", true), ShouldEqual, hashcat(hash3))
  })

  Convey("Can query a range", t, func() {
    So(lookupRange(0, 100, true), ShouldEqual, hashcat(hash1, hash2, hash3))
    So(lookupRange(0, 100, false), ShouldEqual, hashcat(hash3, hash2, hash1))
    So(lookupRange(20, 30, false), ShouldEqual, hashcat(hash1))
    So(lookupRange(30, 40, true), ShouldEqual, hashcat(hash2, hash3))
    So(lookupRange(40, 50, true), ShouldEqual, hashcat(""))
    So(lookupRange(40, 20, true), ShouldEqual, hashcat(""))
  })

  Convey("Can query numeric fields ascending or descending", t, func() {
    forward := hashcat(hash1, hash2, hash3)
    backward := hashcat(hash3, hash2, hash1)
    cases := []string{
      "LT:100", "LTE:100", "GT:0", "GTE:0",
    }
    for _, k := range cases {
      So(lookup(k, true), ShouldEqual, hashcat(forward))
      So(lookup(k, false), ShouldEqual, hashcat(backward))
    }
    So(lookupRange(30, 40, true), ShouldEqual, hashcat(hash2, hash3))
    So(lookupRange(30, 40, false), ShouldEqual, hashcat(hash3, hash2))
  })
}
