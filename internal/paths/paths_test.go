package paths

// po_ova_cg's TestPaths. Paths are text alone here, so these need no
// files.

import (
	"os"
	"testing"
)

func TestPaths(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range []struct{ got, want, label string }{
		{Normalize("a/./b//../c/"), "a/c", "Normalize: . and doubled / dropped, .. takes out a name"},
		{Normalize("/../a/.."), "/", "Normalize: .. at the root stays at the root"},
		{Normalize("../../a"), "../../a", "Normalize: a relative path's leading .. kept"},
		{Normalize("a/.."), ".", "Normalize: nothing left is ."},

		{Absolute("images/a.png", "/home/u/chars"), "/home/u/chars/images/a.png", "Absolute: against the base"},
		{Absolute("../a.png", "/home/u/chars"), "/home/u/a.png", "Absolute: .. out of the base"},
		{Absolute("/x/y.png", "/home/u"), "/x/y.png", "Absolute: an absolute path is itself"},
		{Absolute("a.png", ""), cwd + "/a.png", "Absolute: no base is the working directory"},
		{Absolute("", "/a"), "", "Absolute: none is none"},

		{Dir("/home/u/chars/hana.yaml"), "/home/u/chars", "Dir: the file's directory"},
		{Dir("/hana.yaml"), "/", "Dir: at the root"},
		{Dir("hana.yaml"), cwd, "Dir: of a relative name, the working directory"},

		{Relative("/home/u/chars/images/a.png", "/home/u/chars"), "images/a.png", "Relative: below"},
		{Relative("/home/u/pics/a.png", "/home/u/chars"), "../pics/a.png", "Relative: beside"},
		{Relative("/x/a.png", "/home/u/chars"), "../../../x/a.png", "Relative: all the way up"},
		{Relative("/home/u/chars", "/home/u/chars"), ".", "Relative: the directory itself"},
		{Relative("/home/u/charsx/a.png", "/home/u/chars"), "../charsx/a.png",
			"Relative: a name's prefix isn't a directory"},

		{Rebase("images/a.png", "/home/u/chars", "/home/u/backup"), "../chars/images/a.png",
			"Rebase: the file moved, the image didn't"},
		{Rebase("/home/u/pics/a.png", "", "/home/u/chars"), "../pics/a.png",
			"Rebase: an absolute path, on the first save"},
		{Rebase("a.png", "", "/home/u"), "a.png", "Rebase: a relative path with no from stays"},
		{Rebase("", "/a", "/b"), "", "Rebase: no portrait, none"},
	} {
		if x.got != x.want {
			t.Errorf("%s\n    got: %s", x.label, x.got)
		}
	}
}
