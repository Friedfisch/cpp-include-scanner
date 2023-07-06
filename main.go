package main

import (
	"os"
	"path/filepath"
	"strings"
)

type IncludeScanner struct {
	newLineSeperator  string
	rootDirectory     string
	includeExtensions []string
}

var includeScanner = IncludeScanner{
	newLineSeperator:  "\n",                                                // TODO .. hardcoded, no nice trim available... TT,
	rootDirectory:     "C:/Users/Oliver/source/repos/hikogui/src/hikogui/", // TODO .. obviously hardcoded
	includeExtensions: []string{".cpp", "hpp"},
}

func (is *IncludeScanner) rel(fqfn string) string {
	return strings.TrimPrefix(strings.ReplaceAll(fqfn, "\\", "/"), is.rootDirectory)
}

func (is *IncludeScanner) searchIncludes(fqfn string) []string {
	// TODO use filepath.Base maybe properly
	relFn := is.rel(fqfn)
	p, _ := filepath.Split(relFn)
	isRoot := len(p) == 0
	pathTokens := strings.Split(filepath.Dir(fqfn), string(filepath.Separator)) // No function in filepath available to tokenize a path

	fcb, err := os.ReadFile(fqfn)
	if err != nil {
		panic(err)
	}
	lines := strings.Split(string(fcb), is.newLineSeperator)
	var r = make([]string, 0)
	var pragmeOnceFound = false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) <= 9 { // instant skip
			continue
		}
		tokens := strings.SplitN(trimmed, " ", 2)
		if len(tokens) < 2 { // we want "#include X", will break on comments, who adds a comment to an include... will not break if fn contains a space
			continue
		}
		a, b := tokens[0], tokens[1]
		if a == "#pragma" {
			if b == "once" {
				pragmeOnceFound = true
			}
		} else if a == "#include" {
			if strings.HasPrefix(b, "<") {
				continue
			}
			b = strings.Trim(b, "\"")
			// TODO only one level of ".." supported
			if strings.HasPrefix(b, "../") {
				b = strings.TrimPrefix(b, "../")
			} else {
				dir := pathTokens[len(pathTokens)-1]
				if !isRoot {
					b = is.rel(filepath.Join(dir, b))
				}
			}
			r = append(r, b)
		}
	}

	if strings.HasSuffix(fqfn, ".hpp") && !pragmeOnceFound {
		println("hpp without pragma once", relFn)
	} else if strings.HasSuffix(fqfn, ".cpp") && pragmeOnceFound {
		println("cpp with pragma once", relFn)
	}

	return r
}

func (is *IncludeScanner) validExt(filename string) bool {
	for _, ext := range is.includeExtensions {
		if strings.HasSuffix(filename, ext) {
			return true
		}
	}
	return false
}

func (is *IncludeScanner) rec(path string) map[string][]string {
	d, err := os.ReadDir(path)
	if err != nil {
		panic(err)
	}

	var r = make(map[string][]string, 50)
	for _, s := range d {
		var fullFilename = filepath.Join(path, s.Name())
		if s.IsDir() { // No need to filter "."" and ".." - nice job GO.. :D
			childs := is.rec(fullFilename)
			for k, v := range childs {
				r[k] = v
			}
		} else if is.validExt(s.Name()) {
			r[is.rel(fullFilename)] = is.searchIncludes(fullFilename)
		}
		// else println("ignore", fullFilename)
	}
	return r
}

func imp(d []string) string {
	var r string = ""
	for _, v := range d {
		r += v + " "
	}
	return r
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}

func scan(allIncludes map[string][]string, root string, key string, carry []string, d int) []string {
	// TODO this is wrong, need better idea how to detect circular dependencies

	if contains(carry, key) { // We just assume that #pragma once is used everywhere
		return carry
	}
	if d > 0 && key == root {
		panic(root + " " + key)
	}
	println("Checking:  ", d, key)
	if len(carry) > 1000000 { // This is far beyond reasonable for the project i am scanning
		panic(key)
	}
	carry = append(carry, key)
	check := allIncludes[key]
	if len(check) == 0 {
		return carry
	}
	for _, i := range check {
		// FIXME This most likely is totally wrong
		carry = append(carry, scan(allIncludes, root, i, carry, d+1)...)
	}
	return carry
}

func main() {
	allIncludes := includeScanner.rec(includeScanner.rootDirectory)

	var c int
	for _, i := range allIncludes {
		c = c + len(i)
	}
	println("total amount of files:", len(allIncludes))
	println("total amount of includes:", c)

	for k, _ := range allIncludes {
		r := scan(allIncludes, k, k, []string{}, 0)
		println(k, len(r))
	}
}
