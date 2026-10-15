// Modified by the raftseg authors: import paths rewritten for this
// fork's modules.

package generator_test

import (
	"testing"

	"github.com/raftseg/protobuf-go-lite/gen/compiler/protogen"
	_ "github.com/raftseg/protobuf-go-lite/gen/features/marshal"
	_ "github.com/raftseg/protobuf-go-lite/gen/features/size"
	_ "github.com/raftseg/protobuf-go-lite/gen/features/unmarshal"
	"github.com/raftseg/protobuf-go-lite/gen/generator"
	"github.com/raftseg/protobuf-go/types/pluginpb"
)

func TestNewGeneratorValidatesRegistryFeatures(t *testing.T) {
	plugin, err := (protogen.Options{}).New(&pluginpb.CodeGeneratorRequest{})
	if err != nil {
		t.Fatal(err)
	}

	for _, features := range [][]string{
		{"size", "marshal"},
		{"size", "unmarshal"},
		{"marshal", "unmarshal"},
	} {
		_, err := generator.NewGenerator(plugin, features, &generator.Config{Registry: true})
		if err == nil {
			t.Fatalf("NewGenerator(%q) accepted registry without all message features", features)
		}
	}

	if _, err := generator.NewGenerator(plugin, []string{"size", "marshal", "unmarshal"}, &generator.Config{Registry: true}); err != nil {
		t.Fatalf("NewGenerator rejected complete registry features: %v", err)
	}
}
