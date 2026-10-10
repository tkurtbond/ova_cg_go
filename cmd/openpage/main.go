// Command openpage is test/viewer.sh's driver: viewer.Open with a real
// viewer.
//
//	openpage VIEWER PDF PAGE...
//
// Opens each printed PAGE of PDF in VIEWER (as viewer.Parse takes it),
// one after another in this one process, three seconds apart, as the
// program's ? buttons would: so a mupdf started for one page is ended
// before the next. Prints each Open's message, and exits 1 if one fails.
//
// The port of po_ova_cg's test/OpenPage.Mod.
package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/tkurtbond/go_ova_cg/internal/viewer"
)

const pause = 3 * time.Second

func main() {
	var v viewer.Viewer
	ok := len(os.Args) >= 4
	if ok {
		v, ok = viewer.Parse(os.Args[1])
	}
	if !ok {
		fmt.Println("usage: openpage VIEWER PDF PAGE...")
		os.Exit(2)
	}
	pdf := os.Args[2]
	for i, arg := range os.Args[3:] {
		if i > 0 {
			time.Sleep(pause)
		}
		page, _ := strconv.Atoi(arg)
		msg, err := v.Open(pdf, page)
		if err != nil {
			fmt.Println(err)
			ok = false
		} else if msg != "" {
			fmt.Println(msg)
		}
	}
	if !ok {
		os.Exit(1)
	}
}
