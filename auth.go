package main

import (
  "bufio"
  "fmt"
  "os"
  "regexp"
  "strings"
  "errors"
  "github.com/boltdb/bolt"
  "github.com/codegangsta/cli"
  "github.com/howeyc/gopass"
)

func getAuthKeys(c *cli.Context) (map[string]string, error) {
  data := map[string]string{}

  if len(c.GlobalString("token")) > 0 && len(c.GlobalString("session")) > 0 {
    data["token"] = c.GlobalString("token")
    data["session"] = c.GlobalString("session")
    return data, nil
  }

  db, err := bolt.Open("auth.db", 0600, nil)
  LogErr(c, err)
  if err != nil {
    return data, err
  }
  defer db.Close()

  db.View(func(tx *bolt.Tx) error {
    b := tx.Bucket([]byte("AuthBucket"))
    b.ForEach(func(k, v []byte) error {
      data[string(k[:])] = string(v[:])
      return nil
    })

    return nil
  })

  return data, nil
}

func setAuthKeys(c *cli.Context, keys map[string]string) {
  db, err := bolt.Open("auth.db", 0600, nil)
  LogErr(c, err)
  defer db.Close()

  for k := range keys {
    db.Update(func(tx *bolt.Tx) error {
      b, _ := tx.CreateBucketIfNotExists([]byte("AuthBucket"))
      b.Put([]byte(k), []byte(keys[k]))
      return nil
    })
  }
}

func doAuth(c *cli.Context) {
  reader := bufio.NewReader(os.Stdin)
  fmt.Printf("Username: ")
  username, _ := reader.ReadString('\n')
  username = strings.TrimSpace(username)
  fmt.Printf("Password: ")
  password := gopass.GetPasswd()

  resp, err := do(c, "GET", "system", nil)
  LogErr(c, err)
  rgx := regexp.MustCompile(`(?:TNS\_SESSIONID=)([a-zA-Z0-9]{32})(?:;)`)
  data := map[string]string{}
  if v := rgx.FindStringSubmatch(resp.Header.Get("Set-Cookie")); len(v) > 1 {
    data["session"] = v[1]
    setAuthKeys(c, data)
  } else {
    LogErr(c, errors.New("Error acquiring session from Nessus."))
    return
  }

  result, err := post(c, "token", map[string]interface{}{
    "password": string(password[:]),
    "username": username,
  })
  LogErr(c, err)
  jsonStr, err := result.data.MarshalJSON()
  LogErr(c, err, string(jsonStr[:]))
  t := result.data.Get("response").Get("token").Interface()
  if len(fmt.Sprint(t)) > 0 {
    data["token"] = fmt.Sprint(t)
    setAuthKeys(c, data)
  }
  LogErr(c, err)

  println(data["token"], data["session"])
}
