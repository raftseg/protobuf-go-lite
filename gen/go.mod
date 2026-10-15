module github.com/raftseg/protobuf-go-lite/gen

go 1.23

require (
	github.com/raftseg/protobuf-go v0.0.0-20261015185112-8848c3b6bdfa
	github.com/raftseg/protobuf-go-lite v0.0.0
)

// The generator is built from a checkout, with the runtime it generates
// code for.
replace github.com/raftseg/protobuf-go-lite => ../
