package main

import (
  "log"
  "github.com/codegangsta/cli"
)

func main() {
	app := cli.NewApp()
	app.EnableBashCompletion = true
	app.Name = "nessus"
	app.Usage = "a trusty cli for your trusty pvs"
  app.Authors = []cli.Author{
    cli.Author{
      Name: "Austin Siford",
      Email: "Austin.Siford@mail.wvu.edu",
    },
  }
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "host",
			Usage:  "Nessus Vulnerability Scanner API host",
			EnvVar: "NVS_HOST",
		},
    cli.StringFlag{
      Name: "token, t",
      Usage: "Auth token for Nessus SecurityCenter.",
      EnvVar: "NVS_TOKEN",
    },
    cli.BoolFlag{
      Name: "debug",
      Usage: "Enable verbose logging.",
    },
	}
	app.Action = func(c *cli.Context) {
		cli.ShowAppHelp(c)
	}

	app.Commands = []cli.Command{
		{
			Name:    "export",
			Aliases: []string{"x"},
			Usage:   "export objects from Nessus to a flat file",
			Action: func(c *cli.Context) {
				export(c)
			},
		},
		{
			Name:    "test",
			Aliases: []string{"c"},
			Usage:   "test auth token for validity",
			Action: func(c *cli.Context) {
				testToken(c)
			},
		},
	}

	app.RunAndExitOnError()
}

func LogErr(c *cli.Context, err error, data ...interface{}) {
  if err != nil {
    log.Println(err)
  }
  if len(data) > 0 && c.GlobalBool("debug") {
    for i := range data {
      log.Println(data[i])
    }
  }
}
