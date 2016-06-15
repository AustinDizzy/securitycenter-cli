package main

import "github.com/urfave/cli"

func doImport(c *cli.Context) error {
	//because 'import' is a protected name
	// if c.IsSet("input") {
	//   fmt.Println("importing", c.String("input"), "to", c.Args().First())
	//   file, err := os.Open(c.String("input"))
	//   LogErr(c, err, "opening " + c.String("input"))
	//   defer file.Close()
	//   var (
	//     r = csv.NewReader(file)
	//     data = simplejson.New()
	//     headers []string
	//   )
	//
	//   records, err := r.ReadAll()
	//   LogErr(c, err)
	//
	//   for i, d := range records {
	//     if i == 0 {
	//       headers = records[i]
	//     }
	//
	//   }
	//
	// } else {
	//   LogErr(c, nil, "no input file(s)")
	// }
	return nil
}
