package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"github.com/austindizzy/securitycenter-cli/api"
	"github.com/austindizzy/securitycenter-cli/auth"
	"github.com/austindizzy/securitycenter-cli/utils"
	"github.com/urfave/cli"
)

func export(c *cli.Context) error {
	fmt.Println("exporting", c.Args().First())
	var (
		w         *csv.Writer
		headers   []string
		query     = map[string]interface{}{}
		keys, err = auth.Get(c)
		file      *os.File
		res       *api.Result
	)

	if err != nil {
		utils.LogErr(c, err)
		return err
	}

	if c.IsSet("fields") {
		for _, field := range strings.Split(c.String("fields"), ",") {
			headers = append(headers, strings.Split(field, ".")[0])
		}
		query["fields"] = strings.Join(headers, ",")
	}

	if c.IsSet("output") {
		file, err = os.Create(c.String("output"))
		if err != nil {
			return err
		}
		utils.LogErr(c, err)
		defer file.Close()
		w = csv.NewWriter(file)
	} else {
		w = csv.NewWriter(os.Stdout)
	}

	println("getting " + c.Args().First())
	res, err = api.NewRequest("GET", c.Args().First(), query).WithAuth(keys).Do(c)
	utils.LogErr(c, err)
	if err == nil {
		var (
			records [][]string
		)
		if c.IsSet("fields") {
			records = append(records, strings.Split(c.String("fields"), ","))
		}

		var dataArr []interface{}

		if dataArr, err = res.Data.Get("response").Array(); err != nil {
			dataArr = res.Data.Get("response").Get("manageable").MustArray()
		}

		for _, d := range dataArr {
			var (
				row  []string
				data = d.(map[string]interface{})
			)
			if c.IsSet("fields") {
				for _, v := range strings.Split(c.String("fields"), ",") {
					if strings.Contains(v, ".") {
						var (
							tmp = strings.Split(v, ".")
							obj = data[tmp[0]].(map[string]interface{})
						)
						row = append(row, fmt.Sprint(obj[tmp[1]]))
					} else {
						row = append(row, fmt.Sprint(data[v]))
					}
				}
			} else {
				for _, v := range data {
					row = append(row, fmt.Sprint(v))
				}
			}
			records = append(records, row)
		}
		err = w.WriteAll(records)
		utils.LogErr(c, err)

		w.Flush()
	}
	return err
}
