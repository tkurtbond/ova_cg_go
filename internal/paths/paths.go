// Package paths does file paths: made absolute, a file's directory, and
// one path relative to a directory.
//
// ova_cg_po's PLAN.md, "The second tab": a character's portrait is saved
// as a path relative to its YAML file (Changes to ova_fmt, portrait), so
// the window finds the image from the file's directory, and Save As to
// another directory rebases the path. Paths here are text alone: no
// file is looked at, and symbolic links aren't followed, so "a/../b" is
// "b" even where a is a link. A path is "/"-separated; "" is none.
//
// The port of ova_cg_po's OvaPaths.
package paths

import (
	"os"
	"strings"
)

// split is path's parts between "/"s, without empty ones and ".", each
// ".." taking out the part before it (one at the start of an absolute
// path is dropped; of a relative one, kept).
func split(path string) []string {
	abs := IsAbsolute(path)
	var parts []string
	for _, part := range strings.Split(path, "/") {
		n := len(parts)
		switch {
		case part == "" || part == ".":
		case part == ".." && n > 0 && parts[n-1] != "..":
			parts = parts[:n-1]
		case part == ".." && abs:
		default:
			parts = append(parts, part)
		}
	}
	return parts
}

func join(parts []string, abs bool) string {
	r := strings.Join(parts, "/")
	if abs {
		r = "/" + r
	}
	return r
}

// IsAbsolute says whether path starts at the root.
func IsAbsolute(path string) bool { return strings.HasPrefix(path, "/") }

// Normalize is path without "." parts, doubled "/"s, a "/" at the end,
// or ".." after a name: "a/./b//../c/" is "a/c". "." for a relative path
// that comes to nothing.
func Normalize(path string) string {
	r := join(split(path), IsAbsolute(path))
	if r == "" {
		r = "."
	}
	return r
}

func cwd() string {
	d, err := os.Getwd()
	if err != nil {
		return "/"
	}
	return d
}

// Absolute is path made absolute against directory base (itself made
// absolute against the working directory), and normalized; "" for "".
func Absolute(path, base string) string {
	if path == "" {
		return ""
	}
	r := path
	if !IsAbsolute(path) {
		if !IsAbsolute(base) {
			b := cwd()
			if base != "" {
				b += "/" + base
			}
			base = b
		}
		r = base + "/" + path
	}
	return Normalize(r)
}

// Dir is the absolute directory file is in.
func Dir(file string) string {
	abs := Absolute(file, "")
	slash := strings.LastIndexByte(abs, '/')
	if slash <= 0 {
		return "/"
	}
	return abs[:slash]
}

// Relative is path (made absolute as Absolute does, against the working
// directory) relative to directory dir: "../images/a.png". "." for dir
// itself.
func Relative(path, dir string) string {
	p, d := split(Absolute(path, "")), split(Absolute(dir, ""))
	k := 0
	for k < len(p) && k < len(d) && p[k] == d[k] {
		k++
	}
	var r []string
	for range d[k:] {
		r = append(r, "..")
	}
	r = append(r, p[k:]...)
	if len(r) == 0 {
		return "."
	}
	return strings.Join(r, "/")
}

// Rebase is path, given relative to directory from, rewritten relative
// to directory to: a portrait's path when its YAML file moves. An
// absolute path is made relative to to: a character never saved (from
// "") holds the absolute path it was given until its first save. A
// relative path with no from to resolve it against stays as it is.
func Rebase(path, from, to string) string {
	if path == "" {
		return ""
	}
	if from == "" && !IsAbsolute(path) {
		return path
	}
	return Relative(Absolute(path, from), to)
}
