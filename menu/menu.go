package menu

import (
	"fmt"

	"github.com/urfave/cli"
)

type menu interface {
	String() string
	Start(c *cli.Context)
	Process(c *cli.Context, input string)
}

//Main menu
type Main struct {
	menu
}

func (m Main) String() string {
	return `Main Menu
  1.) Export  2.) Import  3.) Exit 4.) Unicorn`
}

//Start the interactive Main menu
func (m Main) Start(c *cli.Context) {
	fmt.Println(m)
	for s := GetSelection("4"); s != "3"; s = GetSelection("4") {
		m.Process(c, s)
		println()
		fmt.Println(m)
	}
	fmt.Println("\nTerminating interactive session")
	fmt.Printf("[NOTE]: In order to deauthenticate from SecurityCenter, either let the token expire (60m default) or run `%s auth delete`.\n", c.App.Name)
}

//Process the selection made from the Main menu
func (m Main) Process(c *cli.Context, selection string) {
	var (
		i = new(Import)
		x = new(Export)
	)
	switch selection {
	case "1":
		x.Start(c)
	case "2":
		i.Start(c)
	case "4":
		printUnicorn()
	default:
		sayUnicorn("YOUR INVALID SELECTION HAS ANGERED THE PROGRAM. QUIT MESSING UP.")
	}
}
