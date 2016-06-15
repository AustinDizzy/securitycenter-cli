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
	query := map[string]interface{}{}
	var w *csv.Writer

	if c.IsSet("fields") {
		query["fields"] = c.String("fields")
	}

	if c.IsSet("output") {
		file, err := os.Create(c.String("output"))
		if err != nil {
			return err
		}
		utils.LogErr(c, err)
		defer file.Close()
		w = csv.NewWriter(file)
	} else {
		w = csv.NewWriter(os.Stdout)
	}

	keys, err := auth.Get(c)
	if err != nil {
		utils.LogErr(c, err)
		return err
	}

	println("getting " + c.Args().First())
	res, err := api.NewRequest("GET", c.Args().First(), query).WithAuth(keys).Do(c)
	utils.LogErr(c, err)
	if err == nil {
		var (
			records [][]string
		)
		if c.IsSet("fields") {
			records = append(records, strings.Split(c.String("fields"), ","))
		}

		for _, d := range res.Data.Get("response").Get("manageable").MustArray() {
			var (
				row  []string
				data = d.(map[string]interface{})
			)
			if c.IsSet("fields") {
				for _, v := range strings.Split(c.String("fields"), ",") {
					row = append(row, fmt.Sprint(data[v]))
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
