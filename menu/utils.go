package menu

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/austindizzy/securitycenter-cli/api"
	"github.com/austindizzy/securitycenter-cli/auth"
	"github.com/austindizzy/securitycenter-cli/utils"
	"github.com/urfave/cli"
)

// GetInput gets a user-supplied string using the supplied
// parameter msg as a prompting message.
func GetInput(msg string) string {
	var (
		reader = bufio.NewReader(os.Stdin)
		input  string
	)

	fmt.Printf("%s: ", msg)
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(input)
	fmt.Println()

	return input
}

// GetWriter gets a csv.Writer for a CSV file specified by the user, used
// for determining where to write exported records to.
func GetWriter(c *cli.Context) *csv.Writer {
	var (
		filePath  = GetInput("Please enter location to save export (.csv)")
		file, err = os.Create(filePath)
		w         *csv.Writer
	)

	utils.LogErr(c, err)

	w = csv.NewWriter(file)

	return w
}

// GetSelection gets a user-supplied selection used for traversing menus
// with the supplied default parameter used if no input is given.
func GetSelection(def ...string) string {
	var (
		str       = "\nPlease enter your selection %s: "
		reader    = bufio.NewReader(os.Stdin)
		selection string
	)
	if len(def) > 0 {
		str = fmt.Sprintf(str, "("+def[0]+")")
	} else {
		str = fmt.Sprintf(str, "")
	}
	fmt.Printf(str)
	selection, _ = reader.ReadString('\n')
	selection = strings.TrimSpace(selection)

	fmt.Println()

	if len(selection) == 0 && len(def) > 0 {
		return def[0]
	}

	return selection
}

// GetFolder gets the path to a user supplied folder, used for exporting
// a number of files to (for instance, report .xml files, .nessus scan results, etc).
func GetFolder() string {
	var (
		t    = false
		path string
		err  error
	)

	for t == false {
		path = GetInput("Enter a directory for exported file(s)")
		_, err = os.Stat(path)
		if err == nil {
			t = true
		}
		if os.IsNotExist(err) {
			fmt.Printf("The directory \"%s\" does not exist. Please try again.\n", path)
		}
	}

	return path
}

// GetRepo fetches a list of all the repositories in the system and displays
// them as a choose list for batch importing various objects which require a user-selected repository.
func GetRepo(c *cli.Context) (string, error) {
	var (
		keys, err  = auth.Get(c)
		w          = new(tabwriter.Writer)
		repoFilter = map[string]interface{}{
			"fields": "id,name",
		}
		repoRes *api.Result
	)

	if err != nil {
		return "", err
	}

	w.Init(os.Stdout, 0, 8, 0, '\t', 0)
	repoRes, err = api.NewRequest("GET", "repository", repoFilter).WithAuth(keys).Do(c)
	if err != nil {
		return "", err
	}

	fmt.Println("Repositories available in SecurityCenter:")
	for i, r := range repoRes.Data.Get("response").MustArray() {
		str := "[%d] %s\t"
		if i%3 == 0 {
			str += "\n"
		}
		fmt.Fprintf(w, str, i, r.(map[string]interface{})["name"])
	}

	return GetSelection(), nil
}
