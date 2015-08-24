package main

import (
  "github.com/bitly/go-simplejson"
  "net/http"
  "net/url"
  "errors"
  "io/ioutil"
  "github.com/codegangsta/cli"
)

type result struct {
  url string
  data *simplejson.Json
}

func get(c *cli.Context, path string, data map[string]string) (result, error) {
  if !(len(c.GlobalString("host")) > 0) {
    LogErr(c, errors.New("Error: \"--host\" flag not set."))
  }
  u, err := url.Parse(c.GlobalString("host"))
  LogErr(c, err)
  u.Path = "/rest/" + path

  params := url.Values{}
  for k := range data {
    params.Add(k, data[k])
  }

  u.RawQuery = params.Encode()

  client := &http.Client{}
  req, err := http.NewRequest("GET", u.String(), nil)
  if !(len(c.GlobalString("token")) > 0) {
    LogErr(c, errors.New("Error: No token set."))
  }
  req.Header.Add("X-SecurityCenter", c.GlobalString("token"))
  resp, err := client.Do(req)
  defer resp.Body.Close()

  LogErr(c, err, "API get:", u.String())

  json, err := simplejson.NewFromReader(resp.Body)
  res := result{u.String(), json}
  b, _ := ioutil.ReadAll(resp.Body)
  println(string(b))

  return res, err
}
