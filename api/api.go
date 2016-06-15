package api

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"time"

	"github.com/bitly/go-simplejson"
	"github.com/urfave/cli"
)

//Request to be sent to SecurityCenter API
type Request struct {
	Keys         map[string]string
	method, path string
	Data         map[string]interface{}
}

//Result from SecurityCenter API
type Result struct {
	//Status code of the HTTP request made
	Status int
	//URL of the HTTP request made
	URL string
	//Data response from the API in simplejson/json format
	Data *simplejson.Json
	//HTTPRes is the raw net/http request for easy request customizations
	HTTPRes *http.Response
}

//NewRequest forms a new request to send to the SecurityCenter API
func NewRequest(method, path string, data ...map[string]interface{}) *Request {
	var (
		r = &Request{
			method: method,
			path:   path,
		}
	)

	if len(data) > 0 {
		r.Data = data[0]
	}

	return r
}

//WithAuth loads the authentication keys given into the request for the
//request to be authenticated as a user
func (r *Request) WithAuth(keys map[string]string) *Request {
	r.Keys = keys
	return r
}

//Do performs the request and returns the result and any errors.
func (r *Request) Do(c *cli.Context) (*Result, error) {
	var (
		err      error
		reqData  []byte
		req      *http.Request
		uri      *url.URL
		client   *http.Client
		jsonResp *simplejson.Json
		res      *Result
	)
	if !(len(c.GlobalString("host")) > 0) {
		err = errors.New("Error: \"--host\" flag not set.")
		return nil, err
	}

	uri, err = url.Parse(c.GlobalString("host"))
	if err != nil {
		return nil, err
	}
	uri.Path = "/rest/" + r.path

	if r.method == "GET" {
		params := url.Values{}
		for k := range r.Data {
			params.Add(k, fmt.Sprint(r.Data[k]))
		}
		uri.RawQuery = params.Encode()
	} else if r.method == "POST" {
		reqData, err = json.Marshal(r.Data)
		if err != nil {
			return nil, err
		}
	}

	client = &http.Client{
		Timeout: time.Duration(90 * time.Second),
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	req, err = http.NewRequest(r.method, uri.String(), bytes.NewBuffer(reqData))
	if err != nil {
		return nil, err
	}

	pc := make([]uintptr, 10)
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])

	if !(strings.HasSuffix(f.Name(), "auth.Do") && r.path == "system") {
		for key := range r.Keys {
			switch key {
			case "session":
				req.AddCookie(&http.Cookie{
					Name:  "TNS_SESSIONID",
					Value: r.Keys[key],
				})
			case "token":
				if len(r.Keys[key]) > 0 && r.path != "token" {
					req.Header.Add("X-SecurityCenter", r.Keys[key])
				}
			}
		}
	}

	if r.method == "POST" || r.method == "PATCH" {
		req.Header.Add("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	jsonResp, err = simplejson.NewFromReader(resp.Body)
	res = &Result{
		Status:  resp.StatusCode,
		URL:     resp.Request.URL.String(),
		Data:    jsonResp,
		HTTPRes: resp,
	}

	return res, err
}
