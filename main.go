package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/austindizzy/securitycenter-cli/auth"
	"github.com/austindizzy/securitycenter-cli/menu"
	"github.com/austindizzy/securitycenter-cli/utils"

	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.EnableBashCompletion = true
	app.Name = "securitycenter-cli"
	app.Usage = "a trusty cli for your trusty nvs"
	app.Version = "0.1a"
	app.Authors = []cli.Author{
		cli.Author{
			Name:  "Austin Siford",
			Email: "Austin.Siford@mail.wvu.edu",
		},
	}
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "host",
			Usage:  "Tenable Nessus SecurityCenter API host",
			EnvVar: "TNS_HOST",
		},
		cli.StringFlag{
			Name:   "token, t",
			Usage:  "Auth token for SecurityCenter.",
			EnvVar: "TNS_TOKEN",
		},
		cli.StringFlag{
			Name:   "session",
			Usage:  "Auth session for SecurityCenter",
			EnvVar: "TNS_SESSION",
		},
		cli.BoolFlag{
			Name:  "debug",
			Usage: "Enable verbose logging.",
		},
		cli.IntFlag{
			Name:  "throttle",
			Usage: "Throttle requests by N milliseconds",
			Value: -1,
		},
	}
	app.Before = func(c *cli.Context) error {
		println()
		var err error
		if !(len(c.GlobalString("host")) > 0) {
			err = errors.New("Error: \"--host\" flag not set.")
		}
		return err
	}

	app.Action = func(c *cli.Context) error {
		cli.ShowAppHelp(c)
		return nil
	}

	app.After = func(c *cli.Context) error {
		println()
		return nil
	}

	app.Commands = []cli.Command{
		{
			Name:      "export",
			Aliases:   []string{"x"},
			Usage:     "export objects from SecurityCenter to a flat file",
			UsageText: fmt.Sprintf("%s export [command options] [data type to export]", app.Name),
			Action: func(c *cli.Context) error {
				return export(c)
			},
			Flags: []cli.Flag{
				cli.StringFlag{Name: "fields", Usage: "fields to export"},
				cli.StringFlag{Name: "filter", Usage: "filter exported records"},
				cli.StringFlag{Name: "output", Usage: "optional file output"},
			},
		},
		{
			Name:    "import",
			Aliases: []string{"i"},
			Usage:   "import objects from a flat file to SecurityCenter",
			Action: func(c *cli.Context) error {
				return doImport(c)
			},
			Flags: []cli.Flag{
				cli.StringFlag{Name: "input", Usage: "file to read/import"},
				cli.BoolFlag{Name: "dryrun", Usage: "no data is sent in dryrun mode"},
			},
		},
		{
			Name:    "test",
			Aliases: []string{"t"},
			Usage:   "test auth token for validity",
			Action: func(c *cli.Context) error {
				auth.Test(c)
				return nil
			},
		}, {
			Name:    "menu",
			Aliases: []string{"m"},
			Usage:   "start interactive menu",
			Action: func(c *cli.Context) error {
				if auth.Test(c) {
					var m = new(menu.Main)
					m.Start(c)
				}
				return nil
			},
		},
		{
			Name:    "auth",
			Aliases: []string{"c"},
			Usage:   "get/set auth tokens",
			Action: func(c *cli.Context) error {
				keys, err := auth.Get(c)
				utils.LogErr(c, err)
				if len(keys) == 0 {
					auth.Do(c)
					keys, err = auth.Get(c)
				}
				utils.LogErr(c, err, keys)

				// i, err := strconv.ParseInt(keys["__timestamp"], 10, 64)
				// utils.LogErr(c, err)
				// fmt.Printf("%s Keys:\ntoken: %s\tsession: %s\n", time.Unix(i, 0), keys["token"], keys["session"])
				return err
			},
			Subcommands: []cli.Command{{
				Name:    "delete",
				Aliases: []string{"d", "del"},
				Usage:   "delete stored authentication",
				Action: func(c *cli.Context) error {
					err := auth.Delete(c)
					if err == nil {
						log.Println("Authentication deleted.")
					}
					return err
				},
			}},
		},
	}

	app.Run(os.Args)
}
