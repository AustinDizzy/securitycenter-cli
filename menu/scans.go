package menu

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"gopkg.in/cheggaaa/pb.v1"

	"github.com/austindizzy/securitycenter-cli/api"
	"github.com/austindizzy/securitycenter-cli/auth"
	"github.com/austindizzy/securitycenter-cli/utils"
	"github.com/briandowns/spinner"
	"github.com/urfave/cli"
)

//Scans menu
type Scans struct {
	menu
}

func (r Scans) String() string {
	return `SCANS Menu
	1.) Import Active Scans
  2.) Import Scan Results (.nessus File)
  3.) Batch Import Results (.nessus Files)
  4.) Return to IMPORT Menu`
}

//Start the interactive Results menu
func (r Scans) Start(c *cli.Context) {
	fmt.Println(r)
	for s := GetSelection("4"); s != "4"; s = GetSelection("4") {
		r.Process(c, s)
		println()
		fmt.Println(r)
	}
}

//Process the selection chosen from the Report menu
func (r Scans) Process(c *cli.Context, selection string) {
	var (
		err error
	)
	switch selection {
	case "1":
		err = importScans(c)
	case "2":

	case "3":

	}

	if err != nil {
		utils.LogErr(c, err)
	}
}

func importScans(c *cli.Context) error {
	var (
		filePath  = GetInput("Please enter location of file to import (.csv)")
		file, err = os.Open(filePath) //open the file located at the user-supplied file path
		r         *csv.Reader
		records   [][]string
	)

	if err != nil {
		return err
	}
	defer file.Close() //close the file when we're done with it

	r = csv.NewReader(file) //open a new CSV reader with the opened file
	r.LazyQuotes = true
	r.FieldsPerRecord = 0
	records, err = r.ReadAll()
	if err != nil {
		utils.LogErr(c, nil, records)
		return err
	}

	var (
		headers        = records[0]
		bar            = pb.New(len(records[1:]))
		s              = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		success        = 0
		t              = time.Now()
		loadedAssets   *api.Result
		loadedRepos    *api.Result
		loadedPolicies *api.Result
		finishMsg      = "Successfully imported %d/%d scan(s) in %s"
		keys           map[string]string
		req            *api.Request
		res            *api.Result
	)

	s.Prefix = importPfx
	s.Start()

	//load auth keys (session, token) to complete request(s)
	keys, err = auth.Get(c)
	if err != nil {
		return err
	}

	//preload assets, repos, and policies
	loadedAssets, err = api.NewRequest("GET", "asset", map[string]interface{}{
		"fields": "id,name",
		"filter": "usable",
	}).WithAuth(keys).Do(c)
	if err != nil {
		return err
	}

	loadedRepos, err = api.NewRequest("GET", "repository", map[string]interface{}{
		"fields": "id,name",
	}).WithAuth(keys).Do(c)
	if err != nil {
		return err
	}

	loadedPolicies, err = api.NewRequest("GET", "policy", map[string]interface{}{
		"filter": "usable",
	}).WithAuth(keys).Do(c)
	if err != nil {
		return err
	}

	s.Stop()
	bar.Start() //start the progress bar

	var ( //instantiate variables required for rate limiting / throttling
		rate     time.Duration
		throttle <-chan time.Time
	)

	if c.GlobalInt("throttle") > 0 {
		rate, _ = time.ParseDuration(fmt.Sprintf("%dms", c.GlobalInt("throttle")))
		throttle = time.Tick(rate)
	}

	//for all the records, minus the first line - which are the headers
	for i, row := range records[1:] {
		var (
			// data is the key-value object which will be converted to a
			// JSON object to be posted to its respective endpoint
			data = make(map[string]interface{})
		)
		for pos, value := range row {
			key := headers[pos]
			if !strings.HasPrefix(key, "credentials.") && !strings.HasPrefix(key, "owner") {
				data[key] = value
			}
		}

		if _, ok := data["id"]; !ok {
			data["createdTime"] = 0
			data["modifiedTime"] = 0
			data["status"] = -1
		}

		//if the row has a value for the "repository" field
		if repo, ok := data["repository.name"]; ok {
			//then range over the loaded repositories from the system
			for _, p := range loadedRepos.Data.Get("response").MustArray() {
				//to find one with the same name as the value from the field
				if p.(map[string]interface{})["name"].(string) == repo {
					//then assign the JSON object containing the ID as the proper "repository" value
					data["repository"] = map[string]interface{}{
						"id": p.(map[string]interface{})["id"],
					}
				}
			}
			delete(data, "repository.name")
		}

		//if the row has a value for the "assets" field
		if assets, ok := data["assets"]; ok {
			var assetData []map[string]interface{}
			//split the field on the pipe character and range over its values
			for _, a := range strings.Split(fmt.Sprint(assets), "|") {
				//also range over the loaded assets from the system
				for _, f := range loadedAssets.Data.GetPath("response", "usable").MustArray() {
					asset := f.(map[string]interface{})
					//to find an asset with the same name as the value from the split field
					if fmt.Sprint(asset["name"]) == a {
						//then add the asset to the scan
						assetData = append(assetData, map[string]interface{}{
							"id": asset["id"],
						})
					}
				}
			}
			//and assign the JSON objects as the proper values for the respective fields
			data["assets"] = assetData
		}

		//if the row has a value for the "policy.name" field
		if policyName, ok := data["policy.name"]; ok {
			for _, p := range loadedPolicies.Data.GetPath("response", "usable").MustArray() {
				policy := p.(map[string]interface{})
				utils.LogErr(c, nil, fmt.Sprintf("%s === %s ? = %t", policy["name"], policyName, fmt.Sprint(policy["name"]) == fmt.Sprint(policyName)))
				if fmt.Sprint(policy["name"]) == fmt.Sprint(policyName) {
					data["policy"] = map[string]interface{}{
						"id": policy["id"],
					}
				}
			}
			delete(data, "policy.name")
			delete(data, "policy.id")
		}

		//if the row has a value for the "schedule.type" field
		if _, ok := data["schedule.type"]; ok {
			data["schedule"] = map[string]interface{}{
				"type":       data["schedule.type"],
				"repeatRule": data["schedule.repeatRule"],
				"start":      data["schedule.start"],
			}
			delete(data, "schedule.type")
			delete(data, "schedule.repeatRule")
			delete(data, "schedule.start")
			delete(data, "schedule.nextRun")
		}

		if _, hasAssets := data["assets"]; !hasAssets {
			data["assets"] = make([]map[string]interface{}, 0)
		}

		if _, hasIP := data["ipList"]; !hasIP {
			data["ipList"] = ""
		}

		//if the user specifies a valid throttle length
		if c.GlobalInt("throttle") > 0 {
			//waits for a tick from the "throttle" channel, effectively throttling (by
			//blocking) the process for as long as the rate time.Duration specifies
			<-throttle
		}

		utils.LogErr(c, nil, data)
		if id, ok := data["id"]; ok {
			delete(data, "id")
			req = api.NewRequest("PATCH", fmt.Sprint("scan/", id), data)
			fmt.Printf("PATCHing %s: %#v\n", id, data)
		} else {
			req = api.NewRequest("POST", "scan", data)
		}
		res, err = req.WithAuth(keys).Do(c)
		bar.Increment()

		utils.LogErr(c, err, res.Data.Interface())

		//if there is no error, response status is 200 OK, and "error_code" = 0, we have a success
		if err == nil && res.Status == 200 && res.Data.Get("error_code").MustInt() == 0 {
			success++ //so then increment the numer of success
		} else { //else, log the error and break the loop to prevent further errors
			errData, err := res.Data.Encode()
			utils.LogErr(c, err)
			finishMsg = fmt.Sprintf("Error adding scan %d: %s\n", i, string(errData[:])) + finishMsg
			break
		}
	}
	bar.FinishPrint(fmt.Sprintf(finishMsg, success, len(records[1:]), time.Since(t)))
	return err
}

func importResult(c *cli.Context, res io.Reader, repo string) error {
	//TODO: add WithBody(io.Reader) to api to make this work using the current api
	return nil
}
