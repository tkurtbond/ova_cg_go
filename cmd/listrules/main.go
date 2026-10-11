// Command listrules prints every item of package rules as
// "PAGE<TAB>NAME", in the book's order, for make check-pages, which
// compares it with the rulebook PDF's headings
// (tools/rulebook-headings.py).
package main

import (
	"fmt"

	"github.com/tkurtbond/ova_cg_go/internal/rules"
)

func main() {
	for k := rules.Ability; k <= rules.Flaw; k++ {
		for i := 0; i < rules.Count(k); i++ {
			it := rules.Get(k, i)
			fmt.Printf("%d\t%s\n", it.Page, it.Name)
		}
	}
}
