// Copyright 2026 The raftseg authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package depcheck holds a test that keeps the generator module's
// dependencies first-party: go.mod requires only the runtime module and
// raftseg's fork of Go protocol buffers, and no Go file imports anything
// else outside the standard library, or uses cgo. The generator is a build
// tool; products never compile it in.
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

const modulePath = "github.com/raftseg/protobuf-go-lite/gen"

// requires are the modules go.mod may require: the runtime (replaced by
// the checkout above, ..) and the fork of Go protocol buffers.
var requires = []string{
	"github.com/raftseg/protobuf-go-lite",
	"github.com/raftseg/protobuf-go",
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

func TestGoModRequiresFirstPartyOnly(t *testing.T) {
	f, err := os.Open(filepath.Join(moduleRoot(t), "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	inBlock := ""
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if i := strings.Index(line, "//"); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		word := fields[0]
		if inBlock != "" {
			if word == ")" {
				inBlock = ""
				continue
			}
			fields = append([]string{inBlock}, fields...)
			word = inBlock
		} else if len(fields) == 2 && fields[1] == "(" {
			inBlock = word
			continue
		}
		switch word {
		case "module":
			if line != "module "+modulePath {
				t.Errorf("go.mod:%d: %q", n, line)
			}
		case "go":
		case "require":
			if !isRequired(fields[1]) {
				t.Errorf("go.mod:%d: requires %s", n, fields[1])
			}
		case "replace":
			if line != "replace github.com/raftseg/protobuf-go-lite => ../" {
				t.Errorf("go.mod:%d: %q", n, line)
			}
		default:
			t.Errorf("go.mod:%d: %q", n, line)
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
}

// Every Go file, tests included and whatever its build tags, imports only
// the standard library, this module and the modules it requires, and none
// uses cgo.
func TestImportsFirstPartyOnly(t *testing.T) {
	root := moduleRoot(t)
	files := 0
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root && (d.Name() == "testdata" || strings.HasPrefix(d.Name(), ".") || strings.HasPrefix(d.Name(), "_")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		files++
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		for _, imp := range f.Imports {
			p, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				return err
			}
			if p == "C" {
				t.Errorf("%s uses cgo", rel)
			} else if !allowed(p) {
				t.Errorf("%s imports %q", rel, p)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if files < 30 {
		t.Fatalf("only %d Go files found under %s", files, root)
	}
	t.Logf("%d Go files import only the standard library and first-party modules", files)
}

func isRequired(p string) bool {
	for _, r := range requires {
		if p == r {
			return true
		}
	}
	return false
}

func allowed(p string) bool {
	for _, m := range append([]string{modulePath}, requires...) {
		if p == m || strings.HasPrefix(p, m+"/") {
			return true
		}
	}
	first, _, _ := strings.Cut(p, "/")
	return !strings.Contains(first, ".")
}
