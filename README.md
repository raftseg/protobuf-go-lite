# protobuf-go-lite

A fork of [protobuf-go-lite](https://github.com/aperturerobotics/protobuf-go-lite)
at v0.19.0: Go protocol buffers through generated code, without
reflection, kept free of dependencies and of the unsafe parts of Go, and
checked against Google's implementation.

- **Modules:** `github.com/raftseg/protobuf-go-lite`, the runtime that
  generated code imports, and `github.com/raftseg/protobuf-go-lite/gen`
  (in `gen/`), the code generator, `protoc-gen-go-lite`.
- **No dependencies at run time:** the runtime's `go.mod` requires nothing,
  and every file, tests included, imports only the Go standard library and
  the module. Products compile in no `unsafe`, no `reflect`, no cgo and no
  `go:linkname`. `internal/depcheck` fails the tests otherwise.
- **The generator is a build tool.** It reads descriptors from protoc, so it
  runs on [raftseg/protobuf-go](https://github.com/raftseg/protobuf-go),
  raftseg's fork of Google's implementation; it requires nothing else.
  Products never compile it in.
- **Checked against Google's implementation:** `gen/internal/compat`
  compiles this module's `.proto` files with protoc and decodes them with
  Google's library too. Random valid messages must round-trip; damaged ones
  must be accepted by the runtime exactly when Google's library accepts
  them, and then mean the same message; messages may nest as deep in both.
- **Fixes:** that test, and a test of deeply nested input, found ten ways
  in which upstream's generated code was laxer than Google's or differed
  from it, one of which let 24 MB of input kill the process.
  [FORK.md](FORK.md) explains each; each has a regression test that fails
  on upstream's code. The same test found a crash in Google's library,
  fixed in raftseg/protobuf-go.

[FORK.md](FORK.md) lists every change from upstream and why. Upstream's
README is [UPSTREAM-README.md](UPSTREAM-README.md). Upstream's licenses
apply: BSD 3-Clause ([LICENSE](LICENSE), with Google's patent grant in
[PATENTS](PATENTS)), Apache 2.0 for `json/`; [NOTICE](NOTICE) lists the
code taken from elsewhere and what this fork changed.

## Generating code

Build the generator from a checkout, and run protoc with it:

```
cd gen && go build -o /some/bin/protoc-gen-go-lite ./cmd/protoc-gen-go-lite
protoc --plugin=protoc-gen-go-lite=/some/bin/protoc-gen-go-lite \
  --go-lite_out=. --go-lite_opt=features=all your.proto
```

Generated code imports `github.com/raftseg/protobuf-go-lite`. Upstream's
README describes the features and options.

## Testing

```
go test ./...                           # the runtime; also -race, and GOARCH=386
cd gen && go test ./...                 # the generator and the comparison with Google's library; needs protoc
COMPAT_ITERATIONS=20000 COMPAT_SEED=7 go test ./internal/compat   # a longer comparison
go run ./internal/tools/regenerate      # from gen: writes every generated file again
```

CI uses protoc 34.1, as upstream does, and fails if regenerating changes
any file.
