// Copyright 2026 The raftseg authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package depcheck holds tests that keep the runtime module free of
// third-party dependencies and of the unsafe parts of Go: go.mod requires
// nothing, no Go file imports anything outside the standard library and
// this module or uses cgo, and the code that products compile in neither
// imports unsafe or reflect nor uses go:linkname.
package depcheck

import (
	"bufio"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const modulePath = "github.com/raftseg/protobuf-go-lite"

// testOnly are the directories (with those below them) whose code is used
// by tests only, and so may use reflect: copies of go-cmp and of testify's
// API, this check, and the generated test messages.
var testOnly = []string{
	"internal/depcheck",
	"internal/gocmp",
	"internal/testutil",
	"testproto",
}

func moduleRoot(t *testing.T) string {
	dir, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil {
		t.Fatal(err)
	}
	return dir
}

// go.mod must declare the module and a Go version, and nothing else: no
// require, replace, exclude, retract, tool or toolchain lines.
func TestGoModHasNoRequirements(t *testing.T) {
	f, err := os.Open(filepath.Join(moduleRoot(t), "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if i := strings.Index(line, "//"); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		if line == "" {
			continue
		}
		switch strings.Fields(line)[0] {
		case "module":
			if line != "module "+modulePath {
				t.Errorf("go.mod:%d: %q", n, line)
			}
		case "go":
		default:
			t.Errorf("go.mod:%d: %q: the module must not depend on anything", n, line)
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(moduleRoot(t), "go.sum")); err == nil {
		t.Error("go.sum exists: the module must not depend on anything")
	}
}

// goFile is a Go file of the module, whatever its build tags.
type goFile struct {
	rel      string // slash-separated, relative to the module root
	test     bool   // a _test.go file
	testOnly bool   // in a testOnly directory
	imports  []string
	linkname bool
}

func goFiles(t *testing.T) []goFile {
	root := moduleRoot(t)
	var files []goFile
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// Go ignores testdata, and directories starting with . or _.
			if path != root && (d.Name() == "testdata" || strings.HasPrefix(d.Name(), ".") || strings.HasPrefix(d.Name(), "_")) {
				return filepath.SkipDir
			}
			// A nested module (gen, the generator) is not part of this one.
			if path != root {
				if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly|parser.ParseComments)
		if err != nil {
			return err
		}
		g := goFile{rel: filepath.ToSlash(rel), test: strings.HasSuffix(path, "_test.go")}
		for _, dir := range testOnly {
			if strings.HasPrefix(g.rel, dir+"/") {
				g.testOnly = true
			}
		}
		for _, imp := range f.Imports {
			p, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				return err
			}
			g.imports = append(g.imports, p)
		}
		// go:linkname can only appear in a comment; ImportsOnly stops
		// before the body, so read the whole file for it.
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		g.linkname = strings.Contains(string(b), "//go:linkname")
		files = append(files, g)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) < 100 {
		t.Fatalf("only %d Go files found under %s", len(files), root)
	}
	return files
}

// Every Go file, tests included and whatever its build tags, imports only
// the standard library and this module, and none uses cgo.
func TestImportsStandardLibraryOnly(t *testing.T) {
	files := goFiles(t)
	for _, f := range files {
		for _, p := range f.imports {
			if p == "C" {
				t.Errorf("%s uses cgo", f.rel)
			} else if !allowed(p) {
				t.Errorf("%s imports %q", f.rel, p)
			}
		}
	}
	t.Logf("%d Go files import only the standard library and %s", len(files), modulePath)
}

// The code products compile in, every non-test file outside the testOnly
// directories, imports neither unsafe nor reflect, uses no go:linkname,
// and does not import the test-only packages.
func TestRuntimeAvoidsUnsafeAndReflect(t *testing.T) {
	n := 0
	for _, f := range goFiles(t) {
		if f.test || f.testOnly {
			continue
		}
		n++
		for _, p := range f.imports {
			switch {
			case p == "unsafe" || p == "reflect":
				t.Errorf("%s imports %s", f.rel, p)
			case isTestOnly(p):
				t.Errorf("%s imports %s, which is for tests only", f.rel, p)
			}
		}
		if f.linkname {
			t.Errorf("%s uses go:linkname", f.rel)
		}
	}
	if n < 50 {
		t.Fatalf("only %d runtime files found", n)
	}
	t.Logf("%d runtime files use neither unsafe nor reflect", n)
}

// allowed reports whether an import path is in the standard library or this
// module. Standard library paths have no dot in their first element.
func allowed(p string) bool {
	if p == modulePath || strings.HasPrefix(p, modulePath+"/") {
		return true
	}
	first, _, _ := strings.Cut(p, "/")
	return !strings.Contains(first, ".")
}

func isTestOnly(p string) bool {
	for _, dir := range testOnly {
		if q := modulePath + "/" + dir; p == q || strings.HasPrefix(p, q+"/") {
			return true
		}
	}
	return false
}
