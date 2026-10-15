// Copyright 2026 The raftseg authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package descriptorpb_test

import (
	"encoding/binary"
	"errors"
	"testing"

	protobuf_go_lite "github.com/raftseg/protobuf-go-lite"
	"github.com/raftseg/protobuf-go-lite/types/descriptorpb"
)

// nestedDescriptor returns a DescriptorProto with levels-1 levels of
// nested_type (field 3) below it: levels messages in all.
func nestedDescriptor(levels int) []byte {
	sizes := make([]int, levels) // sizes[k]: bytes of the message k levels below the outermost
	for k := levels - 2; k >= 0; k-- {
		sizes[k] = 1 + len(binary.AppendUvarint(nil, uint64(sizes[k+1]))) + sizes[k+1]
	}
	b := make([]byte, 0, sizes[0])
	for k := 1; k < levels; k++ {
		b = append(b, 0x1a)
		b = binary.AppendUvarint(b, uint64(sizes[k]))
	}
	return b
}

func depthOf(m *descriptorpb.DescriptorProto) int {
	n := 1
	for len(m.GetNestedType()) > 0 {
		m = m.GetNestedType()[0]
		n++
	}
	return n
}

// Messages may nest RecursionLimit deep, the outermost included, and no
// deeper: as in google.golang.org/protobuf.
func TestRecursionLimit(t *testing.T) {
	var m descriptorpb.DescriptorProto
	if err := m.UnmarshalVT(nestedDescriptor(protobuf_go_lite.RecursionLimit)); err != nil {
		t.Fatalf("%d levels: %v", protobuf_go_lite.RecursionLimit, err)
	}
	if got := depthOf(&m); got != protobuf_go_lite.RecursionLimit {
		t.Fatalf("decoded %d levels, want %d", got, protobuf_go_lite.RecursionLimit)
	}
	m = descriptorpb.DescriptorProto{}
	if err := m.UnmarshalVT(nestedDescriptor(protobuf_go_lite.RecursionLimit + 1)); !errors.Is(err, protobuf_go_lite.ErrRecursionLimit) {
		t.Fatalf("%d levels: got %v, want ErrRecursionLimit", protobuf_go_lite.RecursionLimit+1, err)
	}
	if err := m.UnmarshalVTDepth(nestedDescriptor(3), 2); !errors.Is(err, protobuf_go_lite.ErrRecursionLimit) {
		t.Fatalf("3 levels with depth 2: got %v, want ErrRecursionLimit", err)
	}
}

// Before the limit, about 24 MB of nested messages overflowed the
// goroutine's stack, a fatal error that killed the process.
func TestNestingCannotExhaustStack(t *testing.T) {
	if testing.Short() {
		t.Skip("builds 24 MB of input")
	}
	in := nestedDescriptor(5_000_000)
	var m descriptorpb.DescriptorProto
	if err := m.UnmarshalVT(in); !errors.Is(err, protobuf_go_lite.ErrRecursionLimit) {
		t.Fatalf("%d bytes of nesting: got %v, want ErrRecursionLimit", len(in), err)
	}
}
