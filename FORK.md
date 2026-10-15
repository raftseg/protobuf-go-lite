# How this fork differs from protobuf-go-lite v0.19.0

This module is a fork of
[protobuf-go-lite](https://github.com/aperturerobotics/protobuf-go-lite)
at tag v0.19.0 (commit 561b6c1), with upstream's full history below it.
It exists so that raftseg can use protocol buffers under its dependency
policy (first-party modules only, each requiring nothing) without
reflection or `unsafe` at run time. This file lists every change and the
reason for it. Every modified file says so in its header.

## Layout: two modules

Upstream is one module whose `go.mod` requires Google's protobuf module,
golang/protobuf, json-iterator-lite, go-cmp and testify. The code
generator needs Google's module to read descriptors; the code it generates
does not. The fork splits them:

- **`github.com/raftseg/protobuf-go-lite`** (the repository's root): the
  runtime that generated code imports (the root package, `json`,
  `registry`, `encoding/protowire`, `types/...`, `compiler/protogenlite`)
  and the generated test messages (`testproto/...`). `go.mod` requires
  nothing.
- **`github.com/raftseg/protobuf-go-lite/gen`** (`gen/`): the generator,
  moved there with the packages only it uses: `cmd/protoc-gen-go-lite`,
  `compiler/protogen`, `features/...`, `generator/...`, and
  `internal/{editionssupport,encoding/defval,encoding/tag,encoding/text,genid,order,protobuild,strs,version}`.
  Import paths are those of upstream with `gen/` inserted. `go.mod`
  requires the runtime (from the checkout, `replace => ../`) and
  [raftseg/protobuf-go](https://github.com/raftseg/protobuf-go) in place of
  `google.golang.org/protobuf`. `gen/internal/depcheck` keeps it to those.

## No dependencies

- `github.com/aperturerobotics/json-iterator-lite` v1.1.0 (MIT), which
  `json` and `structpb` use, is copied into `internal/jsoniter`, with its
  tests.
- `github.com/google/go-cmp` v0.7.0 (BSD 3-Clause), which tests use, is
  copied into `internal/gocmp`, without its tests.
- testify is replaced by `internal/testutil`, raftseg's implementation of
  the part of its API the tests use (as in raftseg/raft), extended with
  `Condition`, `Contains`, `ErrorContains`, `Truef` and `New`.
  `internal/testutil/testdata/testifycompat` compares it with testify
  check by check (`sh run.sh`; outside the module).
- `internal/weakdeps`, which made `go.mod` require golang/protobuf, is
  removed.
- `types/known/structpb`'s test used `protoreflect.Name` as an example of a
  named string type; a local type replaces it.
- The generator's tests read `google/protobuf/go_features.proto` from
  Google's module's `src` directory, which raftseg/protobuf-go does not
  keep; `gen/cmd/protoc-gen-go-lite/testdata/src` has a copy of v1.36.11's.

## No unsafe, no reflection at run time

`internal/depcheck` fails if any non-test file outside `internal/gocmp`,
`internal/testutil`, `internal/depcheck` and `testproto` imports `unsafe`,
`reflect` or a test-only package, or uses `go:linkname`, and if any file
uses cgo. To get there:

- The `unmarshal_unsafe` feature, which aliased decoded strings and bytes
  with the input, is not registered, and the generated `UnmarshalVTUnsafe`
  methods are gone; so is `DecodeStringUnsafe`, which they called. The
  generator's code for them is kept, unreachable, so that upstream changes
  still apply. Upstream's tests of those methods are removed.
- `internal/jsoniter` copies a number's text into a string instead of
  aliasing it with `unsafe`.
- `internal/errors/is_go112.go`, built only by Go versions before 1.13,
  which used `reflect`, is removed.

The generator itself uses reflection, through Google's descriptors; it is
a build tool.

## Fixes

`gen/internal/compat` decodes every message of this module's
`descriptor.proto` and test schemas with both the generated code and
Google's library (raftseg/protobuf-go, through `dynamicpb`), on random
valid messages and on damaged ones. It found the differences below, and a
test of deeply nested input found the first. Each is fixed in the
generator (and the generated files regenerated) or the runtime, and has a
regression test that fails on upstream's code:
`types/descriptorpb/recursion_test.go` and the `fork_test.go` files of
`testproto/basic`, `testproto/editions2024` and `testproto/maps`.

1. **Nesting was unbounded: 24 MB of input killed the process.** Decoding
   recursed once per nested message, so a crafted message of 5 million
   levels overflowed the goroutine's stack, a fatal error no `recover`
   catches. Generated decoders now stop at `RecursionLimit` (10,000)
   levels, the outermost included, and return `ErrRecursionLimit`, as
   Google's library does. `UnmarshalVT` calls the new
   `UnmarshalVTDepth(dAtA, depth)` with the limit.
2. **Field numbers wrapped around.** Decoders took `int32(wire >> 3)`, so a
   field number of 2^32+1 was decoded as field 1. Field numbers outside 1
   to 2^29-1 are now an error.
3. **Unknown fields were checked loosely.** `Skip` accepted a field number
   of 0, a group ending with another field number, and a 10-byte varint
   overflowing 64 bits, and kept them in the unknown fields, which then
   went out again when the message was re-encoded, to decoders that reject
   them. `Skip` now uses `protowire.ConsumeFieldValue`, as Google's library
   does.
4. **A known group could end with another field number.** It is now an
   error.
5. **A packed field's last element could run past the field.** Decoding
   then went on from the middle of the next field. The elements must now
   end where the field does.
6. **Proto3 strings were not checked for valid UTF-8.** Upstream checked
   editions' strings only; Google's library rejects invalid UTF-8 in
   proto3 strings too, as the language requires.
7. **A known field of another wire type failed the message.** Google's
   library keeps such a field as an unknown field, so a message one peer
   accepted another rejected. The generated code now does the same.
8. **Map entries were decoded loosely.** A key or value of another wire
   type was decoded as if it had the right one, field numbers were not
   checked, and a key or value could run past the end of its entry. They
   are now handled as Google's library handles them.
9. **A map entry's message value that appeared twice was replaced.** It is
   now merged, as for any message field.
10. **A float of -0 was dropped.** Implicit-presence `float` and `double`
    fields equal to -0 were not encoded (`-0 == 0`); Google's library
    encodes them, since only +0 is the zero value. Encoding and the size
    helpers now test the bits.

The same comparison found a crash in Google's library: a map entry with a
key followed by a key of another wire type panicked when decoded into a
`dynamicpb` message. raftseg/protobuf-go fixes it (its FORK.md, Fixes).

What the comparison does not cover yet: JSON (`json`, `MarshalJSON`,
`UnmarshalJSON`) against Google's `protojson`, and the text format.

## The generator

- **Deterministic output.** Generated files' headers name the generator's
  version as a constant, `v0.19.0-raftseg` (`gen/internal/version`), where
  upstream wrote the version of the module the plugin was built from,
  which depends on how it was built. The generator now groups the standard
  library's imports apart from the others itself, as upstream's Makefile
  had `goimports` do, so its output needs no further formatting.
- **`gen/internal/tools/regenerate`** writes every generated file of the
  runtime module again with protoc and this generator, as upstream's
  Makefile did with protowrap and goimports. CI runs it and fails on any
  change. Before the fixes above, the regenerated files differed from
  upstream's only in the version, the import paths and the removed
  `UnmarshalVTUnsafe` methods; they now differ by the fixes too.
- **`compiler/protogen`'s `trimTypesPrefix`** stays
  `google.golang.org/protobuf/types/`: it is the `go_package` of protoc's
  well-known types, not an import path. Likewise the `go_package` options in
  `types/pluginpb/plugin.proto` and the other `.proto` files that name
  Google's packages.
- The generator's tests build the plugin and fixture modules from the new
  layout.

## Removed

Upstream's plumbing: CI configuration, Makefile, `hack/` (a module of
build tools), `.golangci.yml`, `package.json`, `bun.lock`, `doc/` and
`go.sum`; the TypeScript bindings of the well-known types (`*.pb.ts`); and
`testproto/unsafe`, which tested `UnmarshalVTUnsafe`. README.md is renamed
UPSTREAM-README.md.

## Known upstream issues left as they are

- `internal/jsoniter/iter_skip_sloppy_test.go` (build tag
  `jsoniter_sloppy`) does not compile: it calls APIs json-iterator-lite no
  longer has. The tag's code builds; CI does not test the tag, as upstream
  did not.

## Updating

Upstream's history is below this fork's, so upstream commits come in by
`git cherry-pick -x`, with paths under the moved packages mapped to
`gen/`, and are listed in `upstream.json`. Then regenerate (`go run
./internal/tools/regenerate` in `gen`) and run all tests, the comparison
with Google's library included.
