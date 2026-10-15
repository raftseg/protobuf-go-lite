// Copyright 2026 The raftseg authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package compat checks the runtime's generated code against
// google.golang.org/protobuf (raftseg's fork of it) on the same schemas:
// the runtime's own descriptor.proto and its test messages, compiled by
// protoc and decoded by Google's library through dynamicpb.
//
//   - Random valid messages, encoded by Google's library, must decode,
//     re-encode to SizeVT bytes, and decode back in Google's library to an
//     equal message.
//   - Damaged encodings must be accepted by the runtime exactly when Google's
//     library accepts them, and then mean the same message.
//   - Messages may nest exactly as deep in both.
//
// The tests need protoc on PATH. COMPAT_ITERATIONS sets how many messages
// each test makes per message type (default 200; 20 with -short), and
// COMPAT_SEED offsets the random seeds, for soaks.
package compat

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	protobuf_go_lite "github.com/raftseg/protobuf-go-lite"
	"github.com/raftseg/protobuf-go-lite/testproto/basic"
	"github.com/raftseg/protobuf-go-lite/testproto/editions2024"
	maps "github.com/raftseg/protobuf-go-lite/testproto/maps"
	"github.com/raftseg/protobuf-go-lite/testproto/proto2"
	"github.com/raftseg/protobuf-go-lite/testproto/proto3opt"
	"github.com/raftseg/protobuf-go-lite/testproto/sizebaseline"
	"github.com/raftseg/protobuf-go-lite/testproto/wkt"
	litedpb "github.com/raftseg/protobuf-go-lite/types/descriptorpb"
	"github.com/raftseg/protobuf-go/proto"
	"github.com/raftseg/protobuf-go/reflect/protodesc"
	"github.com/raftseg/protobuf-go/reflect/protoreflect"
	"github.com/raftseg/protobuf-go/reflect/protoregistry"
	"github.com/raftseg/protobuf-go/types/descriptorpb"
	"github.com/raftseg/protobuf-go/types/dynamicpb"
)

const runtimeModule = "github.com/raftseg/protobuf-go-lite"

// liteMessage is what the runtime generates for every message.
type liteMessage interface {
	UnmarshalVT([]byte) error
	MarshalVT() ([]byte, error)
	SizeVT() int
}

// A schema is one protoc run over the runtime's .proto files, and the
// generated types of the messages it checks.
type schema struct {
	files    []string // relative to the runtime module's root
	messages map[protoreflect.FullName]func() liteMessage
}

var schemas = []schema{{
	files: []string{"types/descriptorpb/descriptor.proto"},
	messages: map[protoreflect.FullName]func() liteMessage{
		"google.protobuf.FileDescriptorProto": func() liteMessage { return new(litedpb.FileDescriptorProto) },
	},
}, {
	files: []string{
		"testproto/basic/basic.proto",
		"testproto/editions2024/editions2024.proto",
		"testproto/maps/maps.proto",
		"testproto/proto2/scalars.proto",
		"testproto/proto3opt/opt.proto",
		"testproto/sizebaseline/sizebaseline.proto",
		"testproto/wkt/wkt.proto",
	},
	messages: map[protoreflect.FullName]func() liteMessage{
		"basic.BasicMsg":                  func() liteMessage { return new(basic.BasicMsg) },
		"editions2024.Edition2024Fixture": func() liteMessage { return new(editions2024.Edition2024Fixture) },
		"testproto.maps.MsgWithMaps":      func() liteMessage { return new(maps.MsgWithMaps) },
		"DoubleMessage":                   func() liteMessage { return new(proto2.DoubleMessage) },
		"FloatMessage":                    func() liteMessage { return new(proto2.FloatMessage) },
		"Int32Message":                    func() liteMessage { return new(proto2.Int32Message) },
		"Int64Message":                    func() liteMessage { return new(proto2.Int64Message) },
		"Uint32Message":                   func() liteMessage { return new(proto2.Uint32Message) },
		"Uint64Message":                   func() liteMessage { return new(proto2.Uint64Message) },
		"Sint32Message":                   func() liteMessage { return new(proto2.Sint32Message) },
		"Sint64Message":                   func() liteMessage { return new(proto2.Sint64Message) },
		"Fixed32Message":                  func() liteMessage { return new(proto2.Fixed32Message) },
		"Fixed64Message":                  func() liteMessage { return new(proto2.Fixed64Message) },
		"Sfixed32Message":                 func() liteMessage { return new(proto2.Sfixed32Message) },
		"Sfixed64Message":                 func() liteMessage { return new(proto2.Sfixed64Message) },
		"BoolMessage":                     func() liteMessage { return new(proto2.BoolMessage) },
		"StringMessage":                   func() liteMessage { return new(proto2.StringMessage) },
		"BytesMessage":                    func() liteMessage { return new(proto2.BytesMessage) },
		"EnumMessage":                     func() liteMessage { return new(proto2.EnumMessage) },
		"OptionalFieldInProto3":           func() liteMessage { return new(proto3opt.OptionalFieldInProto3) },
		"sizebaseline.SizeBaseline":       func() liteMessage { return new(sizebaseline.SizeBaseline) },
		"MessageWithWKT":                  func() liteMessage { return new(wkt.MessageWithWKT) },
	},
}}

// A target is one message type: its descriptor, for Google's library, and
// its generated type.
type target struct {
	desc protoreflect.MessageDescriptor
	lite func() liteMessage
}

func targets(t *testing.T) []target {
	t.Helper()
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Fatal("protoc is not on PATH")
	}
	// protoc sees the runtime module at its import path, as when the code
	// was generated.
	include := t.TempDir()
	link := filepath.Join(include, filepath.FromSlash(runtimeModule))
	if err := os.MkdirAll(filepath.Dir(link), 0o777); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(root, link); err != nil {
		t.Fatal(err)
	}
	var out []target
	for i, s := range schemas {
		set := filepath.Join(t.TempDir(), fmt.Sprintf("set%d.binpb", i))
		args := []string{"-I", include, "--include_imports", "--descriptor_set_out=" + set}
		for _, f := range s.files {
			args = append(args, runtimeModule+"/"+f)
		}
		if b, err := exec.Command("protoc", args...).CombinedOutput(); err != nil {
			t.Fatalf("protoc: %v\n%s", err, b)
		}
		b, err := os.ReadFile(set)
		if err != nil {
			t.Fatal(err)
		}
		var fds descriptorpb.FileDescriptorSet
		if err := proto.Unmarshal(b, &fds); err != nil {
			t.Fatal(err)
		}
		files, err := protodesc.NewFiles(&fds)
		if err != nil {
			t.Fatal(err)
		}
		for name, lite := range s.messages {
			d, err := files.FindDescriptorByName(name)
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			out = append(out, target{d.(protoreflect.MessageDescriptor), lite})
		}
	}
	return out
}

// random fills m with random valid values: every required field, and each
// other field with probability one half, nesting at most depth deep.
func random(r *rand.Rand, m protoreflect.Message, depth int) {
	fields := m.Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		if fd.Cardinality() != protoreflect.Required && r.Intn(2) == 0 {
			continue
		}
		if fd.Message() != nil && depth <= 0 && fd.Cardinality() != protoreflect.Required {
			continue
		}
		switch {
		case fd.IsMap():
			mp := m.Mutable(fd).Map()
			for n := r.Intn(4); n > 0; n-- {
				k := randomValue(r, fd.MapKey(), nil, depth)
				mp.Set(k.MapKey(), randomValue(r, fd.MapValue(), mp.NewValue, depth))
			}
		case fd.IsList():
			l := m.Mutable(fd).List()
			for n := r.Intn(4); n > 0; n-- {
				l.Append(randomValue(r, fd, l.NewElement, depth))
			}
		default:
			m.Set(fd, randomValue(r, fd, func() protoreflect.Value { return m.NewField(fd) }, depth))
		}
	}
}

var strs = []string{"", "a", "raft", "héllo", "日本語", "\U0001F600", strings.Repeat("x", 200)}

func randomValue(r *rand.Rand, fd protoreflect.FieldDescriptor, newMsg func() protoreflect.Value, depth int) protoreflect.Value {
	ints := []int64{0, 1, -1, 127, 128, math.MaxInt32, math.MinInt32, math.MaxInt64, math.MinInt64, r.Int63(), -r.Int63()}
	i := ints[r.Intn(len(ints))]
	switch fd.Kind() {
	case protoreflect.BoolKind:
		return protoreflect.ValueOfBool(r.Intn(2) == 0)
	case protoreflect.EnumKind:
		vals := fd.Enum().Values()
		return protoreflect.ValueOfEnum(vals.Get(r.Intn(vals.Len())).Number())
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return protoreflect.ValueOfInt32(int32(i))
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return protoreflect.ValueOfInt64(i)
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return protoreflect.ValueOfUint32(uint32(i))
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return protoreflect.ValueOfUint64(uint64(i))
	case protoreflect.FloatKind:
		fs := []float32{0, 1, -1.5, math.MaxFloat32, math.SmallestNonzeroFloat32, float32(math.Inf(1)), float32(math.Inf(-1)), r.Float32()}
		return protoreflect.ValueOfFloat32(fs[r.Intn(len(fs))])
	case protoreflect.DoubleKind:
		fs := []float64{0, 1, -1.5, math.MaxFloat64, math.SmallestNonzeroFloat64, math.Inf(1), math.Inf(-1), r.Float64()}
		return protoreflect.ValueOfFloat64(fs[r.Intn(len(fs))])
	case protoreflect.StringKind:
		return protoreflect.ValueOfString(strs[r.Intn(len(strs))])
	case protoreflect.BytesKind:
		b := make([]byte, r.Intn(20))
		r.Read(b)
		return protoreflect.ValueOfBytes(b)
	case protoreflect.MessageKind, protoreflect.GroupKind:
		v := newMsg()
		random(r, v.Message(), depth-1)
		return v
	}
	panic(fmt.Sprintf("kind %v", fd.Kind()))
}

// mutate damages a valid encoding: bytes replaced, a byte inserted or
// removed, or the end cut off.
func mutate(r *rand.Rand, b []byte) []byte {
	b = append([]byte(nil), b...)
	for n := r.Intn(3) + 1; n > 0; n-- {
		switch r.Intn(4) {
		case 0, 1:
			if len(b) > 0 {
				b[r.Intn(len(b))] = byte(r.Intn(256))
			}
		case 2:
			i := r.Intn(len(b) + 1)
			b = append(b[:i], append([]byte{byte(r.Intn(256))}, b[i:]...)...)
		case 3:
			if len(b) > 0 {
				i := r.Intn(len(b))
				b = append(b[:i], b[i+1:]...)
			}
		}
	}
	if r.Intn(8) == 0 && len(b) > 0 {
		b = b[:r.Intn(len(b))]
	}
	return b
}

func googleUnmarshal(d protoreflect.MessageDescriptor, b []byte) (proto.Message, error) {
	m := dynamicpb.NewMessage(d)
	err := proto.UnmarshalOptions{Resolver: (*protoregistry.Types)(nil)}.Unmarshal(b, m)
	return m, err
}

func iterations(t *testing.T) int {
	if s := os.Getenv("COMPAT_ITERATIONS"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil {
			t.Fatalf("COMPAT_ITERATIONS: %v", err)
		}
		return n
	}
	if testing.Short() {
		return 20
	}
	return 200
}

// seed returns the test's seed, offset by COMPAT_SEED.
func seed(t *testing.T, base int64) int64 {
	if s := os.Getenv("COMPAT_SEED"); s != "" {
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			t.Fatalf("COMPAT_SEED: %v", err)
		}
		return base + n*1000
	}
	return base
}

func TestRandomMessagesRoundTrip(t *testing.T) {
	r := rand.New(rand.NewSource(seed(t, 1)))
	n := iterations(t)
	for _, tg := range targets(t) {
		for i := 0; i < n; i++ {
			m := dynamicpb.NewMessage(tg.desc)
			random(r, m, 3)
			g, err := proto.MarshalOptions{Deterministic: true}.Marshal(m)
			if err != nil {
				t.Fatalf("%s: google marshal: %v", tg.desc.FullName(), err)
			}
			l := tg.lite()
			if err := l.UnmarshalVT(g); err != nil {
				t.Fatalf("%s: runtime rejects a valid message: %v\n%x", tg.desc.FullName(), err, g)
			}
			lb, err := l.MarshalVT()
			if err != nil {
				t.Fatalf("%s: runtime marshal: %v", tg.desc.FullName(), err)
			}
			if len(lb) != l.SizeVT() {
				t.Fatalf("%s: SizeVT %d, encoding %d bytes", tg.desc.FullName(), l.SizeVT(), len(lb))
			}
			back, err := googleUnmarshal(tg.desc, lb)
			if err != nil {
				t.Fatalf("%s: google rejects the runtime's encoding: %v\n%x", tg.desc.FullName(), err, lb)
			}
			if !proto.Equal(back, m) {
				t.Fatalf("%s: round trip changed the message\nin:  %x\nout: %x", tg.desc.FullName(), g, lb)
			}
		}
	}
}

// The runtime accepts a damaged encoding exactly when Google's library
// does, and then decodes the same message.
func TestDamagedMessages(t *testing.T) {
	r := rand.New(rand.NewSource(seed(t, 2)))
	n := 2 * iterations(t)
	for _, tg := range targets(t) {
		var valid [][]byte
		for i := 0; i < 20; i++ {
			m := dynamicpb.NewMessage(tg.desc)
			random(r, m, 3)
			b, err := proto.MarshalOptions{Deterministic: true}.Marshal(m)
			if err != nil {
				t.Fatal(err)
			}
			valid = append(valid, b)
		}
		for i := 0; i < n; i++ {
			b := mutate(r, valid[r.Intn(len(valid))])
			gm, gerr := googleUnmarshal(tg.desc, b)
			l := tg.lite()
			lerr := l.UnmarshalVT(b)
			switch {
			case lerr == nil && gerr != nil:
				t.Errorf("%s: runtime accepts what google rejects (%v):\n%x", tg.desc.FullName(), gerr, b)
			case lerr == nil:
				lb, err := l.MarshalVT()
				if err != nil {
					t.Fatalf("%s: runtime marshal: %v", tg.desc.FullName(), err)
				}
				back, err := googleUnmarshal(tg.desc, lb)
				if err != nil {
					t.Errorf("%s: google rejects the runtime's re-encoding: %v\nin:  %x\nout: %x", tg.desc.FullName(), err, b, lb)
				} else if !proto.Equal(back, gm) {
					t.Errorf("%s: runtime and google decode differently\nin:  %x\nout: %x", tg.desc.FullName(), b, lb)
				}
			case gerr == nil:
				t.Errorf("%s: runtime rejects what google accepts: %v\n%x", tg.desc.FullName(), lerr, b)
			}
		}
	}
}

// nestedDescriptor is a FileDescriptorProto whose first message has
// levels-1 levels of nested_type (DescriptorProto field 3) below it.
func nestedDescriptor(levels int) []byte {
	var b []byte
	for i := 0; i < levels-2; i++ {
		b = append(append([]byte{0x1a}, appendVarint(nil, uint64(len(b)))...), b...)
	}
	// FileDescriptorProto.message_type (field 4)
	return append(append([]byte{0x22}, appendVarint(nil, uint64(len(b)))...), b...)
}

func appendVarint(b []byte, v uint64) []byte {
	for v >= 0x80 {
		b = append(b, byte(v)|0x80)
		v >>= 7
	}
	return append(b, byte(v))
}

// Messages nest as deep in both, the limit included.
func TestRecursionLimitMatchesGoogle(t *testing.T) {
	var fdp target
	for _, tg := range targets(t) {
		if tg.desc.FullName() == "google.protobuf.FileDescriptorProto" {
			fdp = tg
		}
	}
	for _, levels := range []int{protobuf_go_lite.RecursionLimit - 1, protobuf_go_lite.RecursionLimit, protobuf_go_lite.RecursionLimit + 1} {
		b := nestedDescriptor(levels)
		_, gerr := googleUnmarshal(fdp.desc, b)
		lerr := fdp.lite().UnmarshalVT(b)
		if (gerr == nil) != (lerr == nil) {
			t.Errorf("%d levels: google %v, runtime %v", levels, gerr, lerr)
		}
		if levels <= protobuf_go_lite.RecursionLimit && lerr != nil {
			t.Errorf("%d levels: runtime %v", levels, lerr)
		}
	}
	if !bytes.Equal(nestedDescriptor(3), []byte{0x22, 0x02, 0x1a, 0x00}) {
		t.Fatalf("nestedDescriptor(3) = %x", nestedDescriptor(3))
	}
}
