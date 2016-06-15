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
		file, err = os.Open(filePath)
		r         *csv.Reader
	)

	utils.LogErr(c, err)
	defer file.Close()

	r = csv.NewReader(file)
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

func do(c *cli.Context, typeSwitch int, r *csv.Reader) {
	r.FieldsPerRecord = 0
	records, err := r.ReadAll()
	utils.LogErr(c, err)

	var (
		headers      = records[0]
		bar          = pb.New(len(records))
		success      = 0
		t            = time.Now()
		endpoint     string
		loadedAssets *api.Result
		loadedRepos  *api.Result
		loadedRoles  *api.Result
		loadedGroups *api.Result
		finishMsg    = "Successfully imported %d/%d %s(s) in %s"
	)

	keys, err := auth.Get(c)
	if err != nil {
		utils.LogErr(c, err)
		return
	}

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

	bar.Start()
	for i, row := range records[1:] {
		var (
			data = make(map[string]interface{})
		)
		for j, v := range row {
			data[headers[j]] = v
		}

		switch typeSwitch {
		case importUsers:
			data["preferences"] = []map[string]string{
				map[string]string{
					"name":  "timezone",
					"value": "America/New_York",
				},
			}
			data["responsibleAssetID"] = -1
			if group, ok := data["group"]; ok {
				for _, g := range loadedGroups.Data.Get("response").MustArray() {
					if g.(map[string]interface{})["name"].(string) == fmt.Sprint(group) {
						data["groupID"] = g.(map[string]interface{})["id"]
					}
				}
				delete(data, "group")
			}
			if role, ok := data["role"]; ok {
				for _, r := range loadedRoles.Data.Get("response").MustArray() {
					if r.(map[string]interface{})["name"].(string) == fmt.Sprint(role) {
						data["roleID"] = r.(map[string]interface{})["id"]
						fmt.Println("roleID is " + data["roleID"].(string))
					}
				}
				if _, ok = data["roleID"]; !ok {
					utils.LogErr(c, fmt.Errorf("Role '%s' is not a valid role in the SecurityCenter appliance you are importing to. Please create this role manually before continuing.", role))
				}
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
			if repos, ok := data["repositories"]; ok {
				var (
					repoData []map[string]interface{}
				)
				for _, r := range strings.Split(fmt.Sprint(repos), "|") {
					for _, p := range loadedRepos.Data.Get("response").MustArray() {
						if p.(map[string]interface{})["name"].(string) == r {
							repoData = append(repoData, map[string]interface{}{
								"id": p.(map[string]interface{})["id"],
							})
						}
					}
				}
				data["repositories"] = repoData
			}
			if assets, ok := data["assets"]; ok {
				var (
					assetData []map[string]interface{}
				)
				for _, a := range strings.Split(fmt.Sprint(assets), "|") {
					for _, f := range loadedAssets.Data.GetPath("response", "manageable").MustArray() {
						if f.(map[string]interface{})["name"].(string) == a {
							assetData = append(assetData, f.(map[string]interface{}))
						}
					}
				}
				data["assets"] = assetData
			}
		}

		dataString, _ := json.Marshal(data)
		utils.LogErr(c, nil, string(dataString[:]))

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
	bar.FinishPrint(fmt.Sprintf(finishMsg, success, len(records)-1, endpoint, time.Since(t)))
}
