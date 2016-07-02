package menu

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gopkg.in/cheggaaa/pb.v1"

	"github.com/austindizzy/securitycenter-cli/api"
	"github.com/austindizzy/securitycenter-cli/auth"
	"github.com/austindizzy/securitycenter-cli/utils"

	"github.com/briandowns/spinner"
	"github.com/urfave/cli"
)

//Export menu
type Export struct {
	menu
}

func (x Export) String() string {
	return `EXPORT Menu
      1.) Active Scans
      2.) Assets (Name, Description, Range)
      3.) Users (Username, Name, Group, Role)
      4.) Groups
      5.) Repositories
			6.) Reports
      ...
      9.) Return to Main Menu`
}

//Start the interactive Export menu
func (x Export) Start(c *cli.Context) {
	fmt.Println(x)
	for s := GetSelection("9"); s != "9"; s = GetSelection("9") {
		x.Process(c, s)
		println()
		fmt.Println(x)
	}
}

//Process the selection chosen from the Export menu
func (x Export) Process(c *cli.Context, selection string) {
	var (
		w *csv.Writer
	)
	if selection != "6" {
		w = GetWriter(c)
	}

	switch selection {
	case "1":
		exportScans(c, w)
	case "2":
		exportAssets(c, w)
	case "3":
		exportUsers(c, w)
	case "4":
		exportGroups(c, w)
	case "5":
		exportRepos(c, w)
	case "6":
		//create a new Report menu and start it
		new(Report).Start(c)
	}
}

func exportScans(c *cli.Context, w *csv.Writer) {
	var (
		records [][]string
		s       = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		query   = map[string]interface{}{
			"filter": "manageable",
			"fields": "owner,groups,ownerGroup,status,name,createdTime,schedule,policy,plugin,type",
		}
		//the header to write to the CSV file
		header    = "id,name,status,type,owner.username,ownerGroup.name,createdTime,plugin.id,plugin.name,policy.id,policy.name,schedule.nextRun,schedule.repeatRule,schedule.start,schedule.type"
		keys, err = auth.Get(c)
		t         = time.Now()
		res       *api.Result
	)

	if err != nil {
		utils.LogErr(c, err)
		return
	}
	s.Prefix = "Exporting..."
	s.Start()

	res, err = api.NewRequest("GET", "scan", query).WithAuth(keys).Do(c)

	var (
		scans = res.Data.GetPath("response", "manageable").MustArray()
		bar   = pb.New(len(scans))
	)

	utils.LogErr(c, err)
	s.Stop()
	bar.Start()

	for i := range scans {
		var (
			row  = make([]string, 15)
			data = scans[i].(map[string]interface{})
		)
		row[0] = fmt.Sprint(data["id"])
		row[1] = fmt.Sprint(data["name"])
		row[2] = fmt.Sprint(data["status"])
		row[3] = fmt.Sprint(data["type"])

		if owner, ok := data["owner"].(map[string]interface{}); ok {
			row[4] = fmt.Sprint(owner["username"])
		}

		if ownergroup, ok := data["ownerGroup"].(map[string]interface{}); ok {
			row[5] = fmt.Sprint(ownergroup["name"])
		}

		row[6] = fmt.Sprint(data["createdTime"])

		if plugin, ok := data["plugin"].(map[string]interface{}); ok {
			row[7] = fmt.Sprint(plugin["id"])
			row[8] = fmt.Sprint(plugin["name"])
		}

		if policy, ok := data["policy"].(map[string]interface{}); ok {
			row[9] = fmt.Sprint(policy["id"])
			row[10] = fmt.Sprint(policy["name"])
		}

		if schedule, ok := data["schedule"].(map[string]interface{}); ok {
			row[11] = fmt.Sprint(schedule["nextRun"])
			row[12] = fmt.Sprint(schedule["repeatRule"])
			row[13] = fmt.Sprint(schedule["start"])
			row[14] = fmt.Sprint(schedule["type"])
		}

		records = append(records, row)
		bar.Increment()
	}

	w.Write(strings.Split(header, ","))
	err = w.WriteAll(records)
	utils.LogErr(c, err)
	w.Flush()

	bar.FinishPrint(fmt.Sprintf("Exported %d scans in %s\n", len(records), time.Since(t)))
}

func exportAssets(c *cli.Context, w *csv.Writer) {
	var (
		records [][]string
		s       = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		query   = map[string]interface{}{
			"filter": "manageable",
			"fields": "id,type,name,description,typeFields,groups,owner,ownerGroup",
		}
		keys, err = auth.Get(c)
		t         = time.Now()
		res       *api.Result
	)

	if err != nil {
		utils.LogErr(c, err)
		return
	}
	s.Prefix = "Exporting..."
	s.Start()

	res, err = api.NewRequest("GET", "asset", query).WithAuth(keys).Do(c)

	var (
		assets = res.Data.GetPath("response", "manageable").MustArray()
		bar    = pb.New(len(assets))
	)

	utils.LogErr(c, err)
	s.Stop()
	bar.Start()

	for _, d := range assets {
		var (
			row  = make([]string, 9)
			data = d.(map[string]interface{})
		)

		row[0] = fmt.Sprint(data["id"])
		row[1] = fmt.Sprint(data["type"])
		row[2] = fmt.Sprint(data["name"])
		row[3] = fmt.Sprint(data["description"])

		if ips, ok := data["typeFields"].(map[string]interface{})["definedIPs"]; ok {
			row[4] = fmt.Sprint(ips)
		}

		if owner, ok := data["owner"].(map[string]interface{})["username"]; ok {
			row[5] = fmt.Sprint(owner)
		}

		if ownerGroup, ok := data["ownerGroup"].(map[string]interface{})["name"]; ok {
			row[6] = fmt.Sprint(ownerGroup)
		}

		var (
			groups     []map[string]interface{}
			groupNames []string
			groupData  = data["groups"].([]interface{})
		)

		for _, g := range groupData {
			switch g.(type) {
			case map[string]interface{}:
				groups = append(groups, g.(map[string]interface{}))
			}
		}

		for _, group := range groups {
			groupNames = append(groupNames, fmt.Sprint(group["name"]))
		}

		row[7] = strings.Join(groupNames, "|")

		if data["type"] == "dynamic" {
			if rules, ok := data["typeFields"].(map[string]interface{})["rules"]; ok {
				rulesStr, _ := json.Marshal(rules.(map[string]interface{}))
				row[8] = string(rulesStr)
			}
		}

		records = append(records, row)
		bar.Increment()
	}

	w.Write([]string{"id", "type", "name", "description", "definedIPs", "owner", "ownerGroup", "groups", "rules"})
	err = w.WriteAll(records)
	utils.LogErr(c, err)
	w.Flush()

	bar.FinishPrint(fmt.Sprintf("Exported %d assets in %s\n", len(records), time.Since(t)))
}

func exportUsers(c *cli.Context, w *csv.Writer) {
	var (
		records [][]string
		s       = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		fields  = "username,email,firstname,lastname,group,role,authType"
		query   = map[string]interface{}{
			"fields": fields,
		}
		keys, err = auth.Get(c)
		t         = time.Now()
		res       *api.Result
	)

	if err != nil {
		utils.LogErr(c, err)
		return
	}
	s.Prefix = "Exporting..."
	s.Start()

	res, err = api.NewRequest("GET", "user", query).WithAuth(keys).Do(c)

	utils.LogErr(c, err)
	s.Stop()

	for _, d := range res.Data.Get("response").MustArray() {
		var (
			row  = make([]string, 7)
			data = d.(map[string]interface{})
		)

		row[0] = fmt.Sprint(data["username"])
		row[1] = fmt.Sprint(data["email"])
		row[2] = fmt.Sprint(data["firstname"])
		row[3] = fmt.Sprint(data["lastname"])
		row[4] = fmt.Sprint(data["group"].(map[string]interface{})["name"])
		row[5] = fmt.Sprint(data["role"].(map[string]interface{})["name"])
		row[6] = fmt.Sprint(data["authType"])

		records = append(records, row)
	}

	w.Write(strings.Split(fields, ","))
	err = w.WriteAll(records)
	utils.LogErr(c, err)
	w.Flush()

	fmt.Printf("Exported %d users in %s\n", len(records), time.Since(t))
}

func exportGroups(c *cli.Context, w *csv.Writer) {
	var (
		records [][]string
		s       = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		query   = map[string]interface{}{
			"fields": "id,name,description,assets,definingAssets,users,repositories",
		}
		keys, err = auth.Get(c)
		t         = time.Now()
		res       *api.Result
		repos     *api.Result
	)

	if err != nil {
		utils.LogErr(c, err)
		return
	}
	s.Prefix = "Exporting..."
	s.Start()

	repos, err = api.NewRequest("GET", "repository", map[string]interface{}{
		"fields": "id,name",
	}).WithAuth(keys).Do(c)
	utils.LogErr(c, err)

	res, err = api.NewRequest("GET", "group", query).WithAuth(keys).Do(c)
	utils.LogErr(c, err)

	s.Stop()

	for _, g := range res.Data.Get("response").MustArray() {
		var (
			row        = make([]string, 6)
			data       = g.(map[string]interface{})
			assetNames []string
			usernames  []string
			repoNames  []string
		)

		row[0] = fmt.Sprint(data["id"])
		row[1] = fmt.Sprint(data["name"])
		if data["description"] != nil {
			row[2] = fmt.Sprint(data["description"])
		} else {
			row[2] = ""
		}

		var (
			assetData    = append(data["definingAssets"].([]interface{}), data["assets"].([]interface{})...)
			users        []map[string]interface{}
			userData     = data["users"].([]interface{})
			repositories []map[string]interface{}
			repoData     = data["repositories"].([]interface{})
		)

		for _, d := range assetData {
			switch d.(type) {
			case map[string]interface{}:
				name := fmt.Sprint(d.(map[string]interface{})["name"])
				assetNames = append(assetNames, name)
			}
		}

		utils.RemoveDupes(&assetNames)

		for _, u := range userData {
			switch u.(type) {
			case map[string]interface{}:
				users = append(users, u.(map[string]interface{}))
			}
		}

		for _, r := range repoData {
			switch r.(type) {
			case map[string]interface{}:
				repositories = append(repositories, r.(map[string]interface{}))
			}
		}

		for _, user := range users {
			usernames = append(usernames, fmt.Sprint(user["username"]))
		}

		for _, groupRepo := range repositories {
			for _, repo := range repos.Data.Get("response").MustArray() {
				if repo.(map[string]interface{})["id"] == groupRepo["id"] {
					repoNames = append(repoNames, fmt.Sprint(repo.(map[string]interface{})["name"]))
				}
			}
		}

		row[3] = strings.Join(assetNames, "|")
		row[4] = strings.Join(usernames, "|")
		row[5] = strings.Join(repoNames, "|")

		records = append(records, row)
	}

	w.Write(strings.Split("id,name,description,assets,users,repositories", ","))
	err = w.WriteAll(records)
	utils.LogErr(c, err)
	w.Flush()

	fmt.Printf("Exported %d groups in %s\n", len(records), time.Since(t))
}

func exportRepos(c *cli.Context, w *csv.Writer) {
	var (
		records [][]string
		s       = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		query   = map[string]interface{}{
			"fields": "id,name,description,typeFields",
		}
		keys, err = auth.Get(c)
		t         = time.Now()
		res       *api.Result
	)

	if err != nil {
		utils.LogErr(c, err)
		return
	}
	s.Prefix = "Exporting..."
	s.Start()

	res, err = api.NewRequest("GET", "repository", query).WithAuth(keys).Do(c)

	var (
		repos = res.Data.Get("response").MustArray()
		bar   = pb.New(len(repos))
	)

	utils.LogErr(c, err)
	s.Stop()
	bar.Start()

	for _, r := range repos {
		var (
			row  = make([]string, 4)
			data = r.(map[string]interface{})
		)

		row[0] = fmt.Sprint(data["id"])
		row[1] = fmt.Sprint(data["name"])
		if data["description"] != nil {
			row[2] = fmt.Sprint(data["description"])
		}
		if data["typeFields"] != nil {
			fields := data["typeFields"].(map[string]interface{})
			if _, ok := fields["ipRange"]; ok {
				row[3] = fmt.Sprint(fields["ipRange"])
			}
		}

		records = append(records, row)
		bar.Increment()
	}

	w.Write([]string{"id", "name", "description", "ipRange"})
	err = w.WriteAll(records)
	utils.LogErr(c, err)
	w.Flush()

	bar.FinishPrint(fmt.Sprintf("Exported %d repositories in %s\n", len(records), time.Since(t)))
}
