package main

import (
  "fmt"
  "github.com/codegangsta/cli"
)

func export(c *cli.Context) {
  fmt.Println("exporting", c.Args().First())
}

func testToken(c *cli.Context) {
  fields := map[string]interface{}{"fields":"firstname,lastname,username"}
  res, err := get(c, "currentUser", fields)
  LogErr(c, err, "testing token")
  if res.data != nil {
    data := res.data.Get("response")
    fmt.Printf("Hello %s %s (%s).", data.Get("firstname"), data.Get("lastname"), data.Get("username"))
  } else {
    println("Uh oh. Something's not right...")
  }
}
