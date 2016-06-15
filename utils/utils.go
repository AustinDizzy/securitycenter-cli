package utils

import (
	"log"
	"runtime"

	"github.com/urfave/cli"
)

//LogErr logs an error or message to stdout taking debug flags and other
//formatting issues into account.
func LogErr(c *cli.Context, err error, data ...interface{}) {
	pc := make([]uintptr, 10)
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])

	if len(data) > 0 && c.GlobalBool("debug") {
		for i := range data {
			log.Printf("[%s] - %#v", f.Name(), data[i])
		}
	}

	if err != nil {
		log.Println("["+f.Name()+"]", err)
	}
}

//RemoveDupes removes the duplicate entries in a string slice which
//is initially passed as a pointer
func RemoveDupes(xs *[]string) {
	found := make(map[string]bool)
	j := 0
	for i, x := range *xs {
		if !found[x] {
			found[x] = true
			(*xs)[j] = (*xs)[i]
			j++
		}
	}
	*xs = (*xs)[:j]
}
