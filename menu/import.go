package menu

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/cheggaaa/pb.v1"

	"github.com/austindizzy/securitycenter-cli/api"
	"github.com/austindizzy/securitycenter-cli/auth"
	"github.com/austindizzy/securitycenter-cli/utils"
	"github.com/urfave/cli"
)

//Import menu
type Import struct {
	menu
}

func (i Import) String() string {
	return `IMPORT Menu
      1.) External Scan Results
      2.) Assets (Name, Description, Range)
      3.) Users (Username, Name, Group, Role)
      4.) Groups
      5.)
      ...
      9.) Return to Main Menu`
}

//Start the interactive Import menu
func (i *Import) Start(c *cli.Context) {
	fmt.Println(i)
	for s := GetSelection("9"); s != "9"; s = GetSelection("9") {
		i.Process(c, s)
		println()
		fmt.Println(i)
	}
}

//Process the selection made from the Import menu
func (i *Import) Process(c *cli.Context, selection string) {
	var (
		filePath  = GetInput("Please enter location of file to import (.csv)")
		file, err = os.Open(filePath) //open the file located at the user-supplied file path
		r         *csv.Reader
	)

	utils.LogErr(c, err)
	defer file.Close() //close the file when we're done with it

	r = csv.NewReader(file) //open a new CSV reader with the opened file
	r.LazyQuotes = true

	switch selection {
	case "2":
		do(c, importAssets, r)
	case "3":
		do(c, importUsers, r)
	case "4":
		do(c, importGroups, r)
	}
}

const (
	importAssets = iota
	importUsers
	importGroups
)

// do does the import by reading over the provided csv.Reader, switching
// logic for which object type is being imported based on the provided typeSwitch
func do(c *cli.Context, typeSwitch int, r *csv.Reader) {
	r.FieldsPerRecord = 0
	records, err := r.ReadAll()
	utils.LogErr(c, err, records)

	var (
		headers      = records[0]
		bar          = pb.New(len(records[1:]))
		success      = 0
		t            = time.Now()
		endpoint     string //the endpoint which to POST the object data to
		loadedAssets *api.Result
		loadedRepos  *api.Result
		loadedRoles  *api.Result
		loadedGroups *api.Result
		finishMsg    = "Successfully imported %d/%d %s(s) in %s"
		keys         map[string]string
	)

	//load auth keys (session, token) to complete request(s)
	keys, err = auth.Get(c)
	if err != nil {
		utils.LogErr(c, err)
		return
	}

	//preload data from the system if it is required for properly getting object ID(s)
	switch typeSwitch {
	case importAssets:
		endpoint = "asset"
	case importUsers:
		endpoint = "user"
		loadedRoles, err = api.NewRequest("GET", "role", map[string]interface{}{
			"fields": "id,name",
		}).WithAuth(keys).Do(c)
		utils.LogErr(c, err)
		loadedGroups, err = api.NewRequest("GET", "group").WithAuth(keys).Do(c)
		utils.LogErr(c, err)
	case importGroups:
		endpoint = "group"
		loadedAssets, err = api.NewRequest("GET", "asset", map[string]interface{}{
			"fields": "id,name,description",
			"filter": "manageable",
		}).WithAuth(keys).Do(c)
		utils.LogErr(c, err)
		loadedRepos, err = api.NewRequest("GET", "repository", map[string]interface{}{
			"fields": "id,name",
		}).WithAuth(keys).Do(c)
		utils.LogErr(c, err)
	}

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
			data[headers[pos]] = value
		}

		//switch manipulating the data object to be POSTed based on the typeSwitch
		switch typeSwitch {
		case importUsers:
			//default user preferences
			data["preferences"] = []map[string]string{
				map[string]string{
					"name":  "timezone",
					"value": "America/New_York",
				},
			}
			data["responsibleAssetID"] = -1
			//if the row has a value set for the "group" field
			if group, ok := data["group"]; ok {
				//then range over all the present groups loaded from the system
				for _, g := range loadedGroups.Data.Get("response").MustArray() {
					//and find the one with the same name as the group found in the row
					if g.(map[string]interface{})["name"].(string) == fmt.Sprint(group) {
						//then assign the "groupID" field to the group's ID
						data["groupID"] = g.(map[string]interface{})["id"]
					}
				}
				//then delete the value from the "group" field, which is an invalid field
				delete(data, "group")
			}
			//if the row has a value set for the "role" field
			if role, ok := data["role"]; ok {
				//then range over all the roles available in the system
				for _, r := range loadedRoles.Data.Get("response").MustArray() {
					//and find one with the same name as found in the row
					if r.(map[string]interface{})["name"].(string) == fmt.Sprint(role) {
						//then set the "roleID" field to the role's ID in the system
						data["roleID"] = r.(map[string]interface{})["id"]
					}
				}
				//if after range over all the available roles, this row's role was not found
				if _, ok = data["roleID"]; !ok {
					//log this error to the user
					utils.LogErr(c, fmt.Errorf("Role '%s' is not a valid role in the SecurityCenter appliance you are importing to. Please create this role manually before continuing.", role))
				}
				//delete and disregard the "role" field
				delete(data, "role")
			} else {
				utils.LogErr(c, errors.New("Missing required field 'role'."))
			}
		case importGroups:
			data["createdTime"] = 0
			data["context"] = ""
			data["status"] = -1
			data["group"] = map[string]interface{}{
				"id": 0,
			}
			//if the row has a value for the "users" field
			if _, ok := data["users"]; ok {
				//delete and disregard it, group membership is done via user import
				delete(data, "users")
			}
			//if the row has a value for the "repositories" field
			if repos, ok := data["repositories"]; ok {
				var (
					repoData []map[string]interface{}
				)
				//then split the field by the pipe ('|') character and range over its values
				for _, r := range strings.Split(fmt.Sprint(repos), "|") {
					//and range over the loaded repositories from the system
					for _, p := range loadedRepos.Data.Get("response").MustArray() {
						//to find one with the same name as the value from the split field
						if p.(map[string]interface{})["name"].(string) == r {
							//then assign its ID to a properly formatted JSON object
							repoData = append(repoData, map[string]interface{}{
								"id": p.(map[string]interface{})["id"],
							})
						}
					}
				}
				//and assign the JSON object as the proper value for the "repositories" field
				data["repositories"] = repoData
			}
			//if the row has a value for the "assets" field
			if assets, ok := data["assets"]; ok {
				var (
					assetData   []map[string]interface{}
					viewableIPs []map[string]interface{}
				)
				//split the field on the pipe character and range over its values
				for _, a := range strings.Split(fmt.Sprint(assets), "|") {
					//also range over the loaded assets from the system
					for _, f := range loadedAssets.Data.GetPath("response", "manageable").MustArray() {
						asset := f.(map[string]interface{})
						//to find an asset with the same name as the value from the split field
						if fmt.Sprint(asset["name"]) == a {
							//then "Share" the asset with the group
							assetData = append(assetData, asset)
							//and add the asset's ID to the group's Viewable IPs
							viewableIPs = append(viewableIPs, map[string]interface{}{
								"id": asset["id"],
							})
						}
					}
				}
				//and assign the JSON objects as the proper values for the respective fields
				data["assets"] = assetData
				data["definingAssets"] = viewableIPs
			}
		}

		//get a string representation of the JSON object to post
		dataString, _ := json.Marshal(data)
		utils.LogErr(c, nil, string(dataString[:]))

		//if the user specifies a valid throttle length
		if c.GlobalInt("throttle") > 0 {
			//waits for a tick from the "throttle" channel, effectively throttling (by
			// blocking) the process for as long as the rate time.Duration specifies
			<-throttle
		}

		res, err := api.NewRequest("POST", endpoint, data).WithAuth(keys).Do(c)
		bar.Increment()

		if err == nil && res.Status == 200 && res.Data.Get("error_code").MustInt() == 0 {
			success++
		} else {
			errData, err := res.Data.Encode()
			utils.LogErr(c, err)
			finishMsg = fmt.Sprintf("Error adding %s %d: %s\n", endpoint, i, string(errData[:])) + finishMsg
			break
		}
	}
	bar.FinishPrint(fmt.Sprintf(finishMsg, success, len(records[1:]), endpoint, time.Since(t)))
}
