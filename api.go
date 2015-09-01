package main

import (
  "encoding/json"
  "bytes"
  "fmt"
  "github.com/bitly/go-simplejson"
  "net/http"
  "net/url"
  "errors"
  "io/ioutil"
  "runtime"
  "github.com/codegangsta/cli"
)

type result struct {
  url string
  data *simplejson.Json
}

func do(c *cli.Context, method, path string, data map[string]interface{}) (*http.Response, error) {
  var (
    err error
    postData []byte
  )
  if !(len(c.GlobalString("host")) > 0) {
    err = errors.New("Error: \"--host\" flag not set.")
    LogErr(c, err)
    return nil, err
  }

  u, err := url.Parse(c.GlobalString("host"))
  LogErr(c, err)
  u.Path = "/rest/" + path

  if method == "GET" {
    params := url.Values{}
    for k := range data {
      params.Add(k, fmt.Sprint(data[k]))
    }

    u.RawQuery = params.Encode()
  } else if method == "POST" {
    postData, err = json.Marshal(data)
    LogErr(c, err)
    if err != nil {
      return nil, err
    }
  }

  client := &http.Client{}
  req, err := http.NewRequest(method, u.String(), bytes.NewBuffer(postData))
  LogErr(c, err, method + " request to " + u.String())
  if err != nil {
    return nil, err
  }

  pc := make([]uintptr, 10)
  runtime.Callers(2, pc)
  f := runtime.FuncForPC(pc[0])

  if f.Name() != "main.doAuth" && path != "system" {
    keys, err := getAuthKeys(c)
    LogErr(c, err, keys)
    if err != nil {
      return nil, err
    }

    for key, _ := range keys {
      switch(key) {
      case "session":
        req.AddCookie(&http.Cookie{
          Name: "TNS_SESSIONID",
          Value: keys[key],
        })
        LogErr(c, nil, "adding session " + keys[key])
      case "token":
        if len(keys[key]) > 0 && path != "token" {
          req.Header.Add("X-SecurityCenter", keys[key])
          LogErr(c, nil, "adding token " + keys[key])
        }
      }
    }
  }

  if method == "POST" {
    req.Header.Add("Content-Type", "application/json")
  }

  return client.Do(req)
}

func post(c *cli.Context, path string, data map[string]interface{}) (*result, error) {
  resp, err := do(c, "POST", path, data)
  LogErr(c, err)
  if err != nil {
    return nil, err
  }

  defer resp.Body.Close()

  LogErr(c, err, "API post:", path, data)

  json, err := simplejson.NewFromReader(resp.Body)
  res := &result{resp.Request.URL.String(), json}
  b, _ := ioutil.ReadAll(resp.Body)
  println(string(b))

  return res, err
}

func get(c *cli.Context, path string, data map[string]interface{}) (*result, error) {
  resp, err := do(c, "GET", path, data)
  LogErr(c, err)
  if err != nil {
    return nil, err
  }

  defer resp.Body.Close()

  LogErr(c, err, "API get:", path, data)

  json, err := simplejson.NewFromReader(resp.Body)
  res := &result{resp.Request.URL.String(), json}
  b, _ := ioutil.ReadAll(resp.Body)
  println(string(b))

  return res, err
}
