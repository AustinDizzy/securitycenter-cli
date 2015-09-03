package main

import (
  "encoding/csv"
  "fmt"
  "github.com/codegangsta/cli"
  "os"
  "strings"
)

func export(c *cli.Context) {
  fmt.Println("exporting", c.Args().First())
  query := map[string]interface{}{}
  var w *csv.Writer

  if c.IsSet("fields") {
    query["fields"] = c.String("fields")
  }

  if c.IsSet("output") {
    file, err := os.Create(c.String("output"))
    LogErr(c, err)
    defer file.Close()
    w = csv.NewWriter(file)
  } else {
    w = csv.NewWriter(os.Stdout)
  }

  resp, err := get(c, c.Args().First(), query)
  LogErr(c, err)
  if err == nil {
    var (
      records [][]string
    )
    if c.IsSet("fields") {
      records = append(records, strings.Split(c.String("fields"), ","))
    }

    for _, s := range []string{"usable","manageable"} {
      for _, d := range resp.data.Get("response").Get(s).MustArray() {
        var (
          row []string
          data = d.(map[string]interface{})
        )
        if c.IsSet("fields") {
          for _, v := range strings.Split(c.String("fields"), ",") {
            row = append(row, data[v].(string))
          }
        } else {
          for _, v := range data {
            row = append(row, v.(string))
          }
        }
        records = append(records, row)
      }
    }
    err = w.WriteAll(records)
    LogErr(c, err)
    w.Flush()
  }
}

func testToken(c *cli.Context) {
  fields := map[string]interface{}{"fields":"firstname,lastname,username"}
  res, err := get(c, "currentUser", fields)
  LogErr(c, err, "testing token")
  if res.data != nil {
    data := res.data.Get("response")
    fmt.Printf("Hello %s %s (%s).", data.Get("firstname").MustString(), data.Get("lastname").MustString(), data.Get("username").MustString())
  } else {
    println("Uh oh. Something's not right...")
  }
}
