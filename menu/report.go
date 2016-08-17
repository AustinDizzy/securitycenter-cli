package menu

import (
	"encoding/csv"
	"fmt"
	"io"
	"mime"
	"os"
	"path"
	"strings"
	"time"

	"gopkg.in/cheggaaa/pb.v1"

	"github.com/austindizzy/securitycenter-cli/api"
	"github.com/austindizzy/securitycenter-cli/auth"
	"github.com/austindizzy/securitycenter-cli/utils"
	"github.com/briandowns/spinner"
	"github.com/urfave/cli"
)

//Report menu
type Report struct {
	menu
}

func (r Report) String() string {
	return `REPORT Menu
  1.) Report Listing (metadata about reports)
  2.) Report Definitions (using SC's export for later native import)
  3.) Report Result Listing (metadata about results)
  4.) Report Results (batch data with actual report results)
  ...
  9.) Return to EXPORT Menu`
}

//Start the interactive Report menu
func (r Report) Start(c *cli.Context) {
	fmt.Println(r)
	for s := GetSelection("9"); s != "9"; s = GetSelection("9") {
		r.Process(c, s)
		println()
		fmt.Println(r)
	}
}

//Process the selection chosen from the Report menu
func (r Report) Process(c *cli.Context, selection string) {
	switch selection {
	case "1":
		exportReportList(c, GetWriter(c))
	case "2":
		exportReports(c, GetFolder())
	case "3":
		exportResultsList(c, GetWriter(c))
	case "4":
		exportResults(c, GetFolder())
	}
}

func exportReportList(c *cli.Context, w *csv.Writer) {
	var (
		records [][]string
		s       = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		query   = map[string]interface{}{
			"filter": "manageable",
			"fields": "id,name,type,ownerGroup,owner,schedule,status,createdTime,modifiedTime",
		}
		keys, err = auth.Get(c)
		t         = time.Now()
		res       *api.Result
	)

	if err != nil {
		utils.LogErr(c, err)
		return
	}
	s.Prefix = exportPfx
	s.Start()

	res, err = api.NewRequest("GET", "reportDefinition", query).WithAuth(keys).Do(c)

	var (
		reports = res.Data.GetPath("response", "manageable").MustArray()
		bar     = pb.New(len(reports))
	)

	utils.LogErr(c, err)
	s.Stop()
	bar.Start()

	for _, r := range reports {
		var (
			row = make([]string, 12)
		)
		for k, v := range r.(map[string]interface{}) {
			switch k {
			case "id":
				row[0] = fmt.Sprint(v)
			case "name":
				row[1] = fmt.Sprint(v)
			case "status":
				row[2] = fmt.Sprint(v)
			case "type":
				row[3] = fmt.Sprint(v)
			case "createdTime":
				row[4] = fmt.Sprint(v)
			case "modifiedTime":
				row[5] = fmt.Sprint(v)
			case "owner":
				row[6] = fmt.Sprint(v.(map[string]interface{})["username"])
			case "ownerGroup":
				row[7] = fmt.Sprint(v.(map[string]interface{})["name"])
			case "schedule":
				data := v.(map[string]interface{})
				row[8] = fmt.Sprint(data["nextRun"])
				if fmt.Sprint(data["nextRun"]) != "0" {
					row[9] = fmt.Sprint(data["repeatRule"])
					row[10] = fmt.Sprint(data["start"])
					row[11] = fmt.Sprint(data["type"])
				}
			}
		}

		records = append(records, row)
		bar.Increment()
	}

	w.Write(strings.Split("id,name,status,type,createdTime,modifiedTime,owner,ownerGroup,schedule.nextRun,schedule.repeatRule,schedule.start,schedule.type", ","))
	err = w.WriteAll(records)
	utils.LogErr(c, err)
	w.Flush()

	bar.FinishPrint(fmt.Sprintf("Exported %d reports in %s\n", len(records), time.Since(t)))
}

func exportReports(c *cli.Context, folderpath string) {
	var (
		s         = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		keys, err = auth.Get(c)
		t         = time.Now()
		successes = 0
		res       *api.Result
	)

	if err != nil {
		utils.LogErr(c, err)
		return
	}

	s.Prefix = exportPfx
	s.Start()

	res, err = api.NewRequest("GET", "reportDefinition", map[string]interface{}{
		"filter": "manageable",
		"fields": "id,name",
	}).WithAuth(keys).Do(c)

	var (
		reportList = res.Data.GetPath("response", "manageable").MustArray()
		bar        = pb.New(len(reportList))
	)

	utils.LogErr(c, err)
	s.Stop()
	bar.Start()

	for _, r := range reportList {
		var (
			report         = r.(map[string]interface{})
			endpoint       = fmt.Sprintf("reportDefinition/%s/export", report["id"])
			reportRes, err = api.NewRequest("POST", endpoint, map[string]interface{}{
				"exportType": "placeholders",
			}).WithAuth(keys).Do(c)
		)

		utils.LogErr(c, err, "exporting", endpoint, report, reportRes.HTTPRes.Status, reportRes.HTTPRes.Header)

		if reportRes.Status == 200 && reportRes.HTTPRes.Header.Get("Content-Disposition") != "" {
			_, fileParams, err := mime.ParseMediaType(reportRes.HTTPRes.Header.Get("Content-Disposition"))
			utils.LogErr(c, err)

			out, err := os.Create(path.Join(folderpath, fileParams["filename"]))
			utils.LogErr(c, err, fmt.Sprintf("creating report file %s", fileParams["filename"]))
			defer out.Close()

			if err != nil {
				_, err = io.Copy(out, reportRes.HTTPRes.Body)
				if err == nil {
					successes++
				}
				utils.LogErr(c, err, "saving %s (%d) to %s", report["name"], report["id"], fileParams["filename"])
			}
		}
		bar.Increment()
	}

	bar.FinishPrint(fmt.Sprintf("Exported %d/%d report tasks in %s\n", successes, len(reportList), time.Since(t)))
}

func exportResultsList(c *cli.Context, w *csv.Writer) {
	var (
		records [][]string
		s       = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		query   = map[string]interface{}{
			"filter":    "manageable",
			"startTime": "1",
			"fields":    "id,name,type,ownerGroup,owner,status,startTime,finishTime,totalSteps,running",
		}
		keys, err = auth.Get(c)
		t         = time.Now()
		res       *api.Result
	)

	if err != nil {
		utils.LogErr(c, err)
		return
	}
	s.Prefix = exportPfx
	s.Start()

	res, err = api.NewRequest("GET", "report", query).WithAuth(keys).Do(c)

	var (
		reports = res.Data.GetPath("response", "manageable").MustArray()
		bar     = pb.New(len(reports))
	)

	utils.LogErr(c, err)
	s.Stop()
	bar.Start()

	for _, r := range reports {
		var (
			row = make([]string, 8)
		)
		for k, v := range r.(map[string]interface{}) {
			switch k {
			case "id":
				row[0] = fmt.Sprint(v)
			case "name":
				row[1] = fmt.Sprint(v)
			case "status":
				row[2] = fmt.Sprint(v)
			case "type":
				row[3] = fmt.Sprint(v)
			case "owner":
				row[4] = fmt.Sprint(v.(map[string]interface{})["username"])
			case "ownerGroup":
				row[5] = fmt.Sprint(v.(map[string]interface{})["name"])
			case "startTime":
				row[6] = fmt.Sprint(v)
			case "finishTime":
				row[7] = fmt.Sprint(v)
			}
		}

		records = append(records, row)
		bar.Increment()
	}

	w.Write(strings.Split("id,name,status,type,owner,ownerGroup,startTime,finishTime", ","))
	err = w.WriteAll(records)
	utils.LogErr(c, err)
	w.Flush()

	bar.FinishPrint(fmt.Sprintf("Exported %d reports in %s\n", len(records), time.Since(t)))
}

func exportResults(c *cli.Context, folderpath string) {
	var (
		s         = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		keys, err = auth.Get(c)
		t         = time.Now()
		successes = 0
		res       *api.Result
	)

	if err != nil {
		utils.LogErr(c, err)
		return
	}

	s.Prefix = exportPfx
	s.Start()

	res, err = api.NewRequest("GET", "report", map[string]interface{}{
		"filter":    "manageable",
		"startTime": "1",
		"fields":    "id,name",
	}).WithAuth(keys).Do(c)

	var (
		reportList = res.Data.GetPath("response", "manageable").MustArray()
		bar        = pb.New(len(reportList))
	)

	utils.LogErr(c, err)
	s.Stop()
	bar.Start()

	for _, r := range reportList {
		var (
			report         = r.(map[string]interface{})
			endpoint       = fmt.Sprintf("report/%s/download", report["id"])
			reportRes, err = api.NewRequest("POST", endpoint, map[string]interface{}{
				"exportType": "placeholders",
			}).WithAuth(keys).Do(c)
		)

		utils.LogErr(c, err, "exporting", endpoint, report, reportRes.HTTPRes.Status, reportRes.HTTPRes.Header)

		if reportRes.Status == 200 && reportRes.HTTPRes.Header.Get("Content-Disposition") != "" {
			_, fileParams, err := mime.ParseMediaType(reportRes.HTTPRes.Header.Get("Content-Disposition"))
			utils.LogErr(c, err)

			out, err := os.Create(path.Join(folderpath, fileParams["filename"]))
			utils.LogErr(c, err, fmt.Sprintf("creating report file %s", fileParams["filename"]))
			defer out.Close()

			if err != nil {
				_, err = io.Copy(out, reportRes.HTTPRes.Body)
				if err == nil {
					successes++
				}
				utils.LogErr(c, err, "saving %s (%d) to %s", report["name"], report["id"], fileParams["filename"])
			}
		}
		bar.Increment()
	}

	bar.FinishPrint(fmt.Sprintf("Exported %d/%d reports in %s\n", successes, len(reportList), time.Since(t)))
}
