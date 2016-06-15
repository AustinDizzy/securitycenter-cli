package menu

import (
	"bufio"
	"fmt"
	"os"
	"strings"
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
