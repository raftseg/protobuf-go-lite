// Copyright 2026 The raftseg authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command regenerate writes every generated .pb.go file of the runtime
// module again, with this module's protoc-gen-go-lite: the well-known
// types, descriptorpb, pluginpb and the test messages. It replaces
// upstream's Makefile target (gengo), which needed protowrap and
// goimports. Run it from the gen directory, with protoc on PATH or named
// by -protoc:
//
//	go run ./internal/tools/regenerate
//
// CI runs it and fails if any file changes.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const runtimeModule = "github.com/raftseg/protobuf-go-lite"

// A group is one protoc run: the .proto files directly in dir (or, with
// recursive, anywhere below it), generated with opt.
type group struct {
	dir       string
	recursive bool
	opt       string
}

// wktFeatures are the well-known types' features: upstream's, without
// unmarshal_unsafe, which this fork does not have.
const wktFeatures = "features=marshal+marshal_strict+unmarshal+size+equal+clone+text"

func groups(root string) ([]group, error) {
	var gs []group
	known, err := os.ReadDir(filepath.Join(root, "types", "known"))
	if err != nil {
		return nil, err
	}
	for _, e := range known {
		if e.IsDir() {
			gs = append(gs, group{dir: "types/known/" + e.Name(), opt: wktFeatures})
		}
	}
	return append(gs,
		group{dir: "types/descriptorpb"},
		group{dir: "types/pluginpb"},
		group{dir: "testproto", recursive: true},
	), nil
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("regenerate: ")
	protoc := flag.String("protoc", "protoc", "the protoc binary")
	flag.Parse()
	if err := run(*protoc); err != nil {
		log.Fatal(err)
	}
}

func run(protoc string) error {
	root, err := filepath.Abs("..")
	if err != nil {
		return err
	}
	if b, err := os.ReadFile(filepath.Join(root, "go.mod")); err != nil || !bytes.HasPrefix(b, []byte("module "+runtimeModule+"\n")) {
		return fmt.Errorf("run from the gen directory of %s", runtimeModule)
	}
	tmp, err := os.MkdirTemp("", "regenerate")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	plugin := filepath.Join(tmp, "protoc-gen-go-lite")
	if err := command(".", "go", "build", "-o", plugin, "./cmd/protoc-gen-go-lite"); err != nil {
		return err
	}
	// protoc sees the runtime module at its import path, so that generated
	// files land where their go_package says and name their source by it.
	include := filepath.Join(tmp, "include")
	link := filepath.Join(include, filepath.FromSlash(runtimeModule))
	if err := os.MkdirAll(filepath.Dir(link), 0o777); err != nil {
		return err
	}
	if err := os.Symlink(root, link); err != nil {
		return err
	}

	gs, err := groups(root)
	if err != nil {
		return err
	}
	for _, g := range gs {
		files, err := protoFiles(root, g)
		if err != nil {
			return err
		}
		if len(files) == 0 {
			continue
		}
		args := []string{"-I", include, "--plugin=protoc-gen-go-lite=" + plugin, "--go-lite_out=" + include}
		if g.opt != "" {
			args = append(args, "--go-lite_opt="+g.opt)
		}
		for _, f := range files {
			args = append(args, runtimeModule+"/"+f)
		}
		if err := command(root, protoc, args...); err != nil {
			return err
		}
	}
	return nil
}

// protoFiles lists a group's .proto files, relative to root, sorted.
func protoFiles(root string, g group) ([]string, error) {
	var files []string
	dir := filepath.Join(root, filepath.FromSlash(g.dir))
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && p != dir && !g.recursive {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(p, ".proto") {
			rel, err := filepath.Rel(root, p)
			if err != nil {
				return err
			}
			files = append(files, filepath.ToSlash(rel))
		}
		return nil
	})
	sort.Strings(files)
	return files, err
}

func command(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %v", name, strings.Join(args, " "), err)
	}
	return nil
}
