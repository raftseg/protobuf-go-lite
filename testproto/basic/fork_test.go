// Copyright 2026 The raftseg authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package basic

import (
	"bytes"
	"errors"
	"io"
	"math"
	"testing"
)

// Regression tests for the decoding and encoding differences from
// google.golang.org/protobuf that raftseg's fork fixed (FORK.md). The
// generator module's differential test (gen/internal/compat) found them;
// each case here failed on upstream's generated code.

// A field number of 2^32+1 wrapped around to 1 in int32(wire >> 3), and
// was decoded as int32_field.
func TestFieldNumberOutOfRange(t *testing.T) {
	// tag (2^32+1)<<3 | varint, then the value 5.
	in := append(appendVarint(nil, (1<<32+1)<<3), 5)
	var m BasicMsg
	if err := m.UnmarshalVT(in); err == nil {
		t.Fatalf("field number 2^32+1 accepted: %+v", m.Int32Field)
	}
	// The largest valid field number is kept as an unknown field.
	in = append(appendVarint(nil, (1<<29-1)<<3), 5)
	m = BasicMsg{}
	if err := m.UnmarshalVT(in); err != nil {
		t.Fatalf("field number 2^29-1: %v", err)
	}
	if out, _ := m.MarshalVT(); !bytes.Equal(out, in) {
		t.Fatalf("unknown field re-encoded as %x, want %x", out, in)
	}
}

// An unknown group must end with its own field number, unknown fields
// must have valid field numbers, and their varints must not overflow.
func TestUnknownFieldsValidated(t *testing.T) {
	for _, in := range [][]byte{
		{0xab, 0x01, 0xb4, 0x01},       // group 21... ended by field 22
		{0xab, 0x02, 0xac, 0x01},       // group 37 ended by field 21
		{0xfb, 0x01, 0x03, 0xfc, 0x01}, // field 0 inside an unknown group
		// field 37, a varint whose 10th byte overflows 64 bits
		{0xa8, 0x02, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x02},
	} {
		var m BasicMsg
		if err := m.UnmarshalVT(in); err == nil {
			t.Errorf("%x accepted", in)
		}
	}
	// A well-formed unknown group is kept.
	in := []byte{0xab, 0x02, 0x08, 0x01, 0xac, 0x02} // group 37 {1: 1}
	var m BasicMsg
	if err := m.UnmarshalVT(in); err != nil {
		t.Fatal(err)
	}
	if out, _ := m.MarshalVT(); !bytes.Equal(out, in) {
		t.Fatalf("re-encoded as %x, want %x", out, in)
	}
}

// A packed field whose last element ran past the field's end was decoded,
// and decoding went on in the middle of the next field.
func TestPackedElementPastFieldEnd(t *testing.T) {
	// repeated_int32_field (16), packed, 1 byte: 0x80 starts a varint that
	// the next field's bytes would complete.
	in := []byte{0x82, 0x01, 0x01, 0x80, 0x08, 0x01}
	var m BasicMsg
	if err := m.UnmarshalVT(in); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("got %v, want io.ErrUnexpectedEOF", err)
	}
}

// Proto3 strings must be valid UTF-8, as in google.golang.org/protobuf;
// upstream checked editions' strings only.
func TestProto3StringsAreValidUTF8(t *testing.T) {
	for _, in := range [][]byte{
		{0x72, 0x01, 0xff},                               // string_field
		{0x92, 0x01, 0x01, 0xff},                         // oneof_string
		{0xaa, 0x01, 0x03, 0x12, 0x01, 0xff},             // nested_message.nested_string
		{0x8a, 0x01, 0x05, 0x0a, 0x01, 0xff, 0x10, 0x01}, // map key
	} {
		var m BasicMsg
		if err := m.UnmarshalVT(in); err == nil {
			t.Errorf("%x accepted", in)
		}
	}
}

// A known field of another wire type is kept as an unknown field, as in
// google.golang.org/protobuf; upstream failed the message.
func TestKnownFieldOfAnotherWireType(t *testing.T) {
	// int32_field (1) as bytes, then string_field (14) "a".
	in := []byte{0x0a, 0x01, 0x07, 0x72, 0x01, 'a'}
	var m BasicMsg
	if err := m.UnmarshalVT(in); err != nil {
		t.Fatal(err)
	}
	if m.Int32Field != 0 || m.StringField != "a" {
		t.Fatalf("decoded %+v", &m)
	}
	out, err := m.MarshalVT()
	if err != nil {
		t.Fatal(err)
	}
	// Known fields first, then the unknown one.
	if want := []byte{0x72, 0x01, 'a', 0x0a, 0x01, 0x07}; !bytes.Equal(out, want) {
		t.Fatalf("re-encoded as %x, want %x", out, want)
	}
}

// Map entries: a key or value of another wire type is skipped, not decoded
// as if it had the right one, and the entry's key and value must end where
// the entry does.
func TestMapEntries(t *testing.T) {
	// key "a", then key as a varint (skipped), value 7.
	in := []byte{0x8a, 0x01, 0x07, 0x0a, 0x01, 'a', 0x08, 0x00, 0x10, 0x07}
	var m BasicMsg
	if err := m.UnmarshalVT(in); err != nil {
		t.Fatal(err)
	}
	if len(m.MapStringInt32Field) != 1 || m.MapStringInt32Field["a"] != 7 {
		t.Fatalf("map %v, want {a: 7}", m.MapStringInt32Field)
	}
	// value 0x80... runs past the entry's end into the next field.
	in = []byte{0x8a, 0x01, 0x04, 0x0a, 0x01, 'a', 0x10, 0x80, 0x08, 0x01}
	m = BasicMsg{}
	if err := m.UnmarshalVT(in); err == nil {
		t.Fatalf("entry overrun accepted: %v", m.MapStringInt32Field)
	}
}

// An implicit-presence float of -0 is encoded, as in
// google.golang.org/protobuf; upstream dropped it.
func TestNegativeZeroEncoded(t *testing.T) {
	m := BasicMsg{FloatField: float32(math.Copysign(0, -1)), DoubleField: math.Copysign(0, -1)}
	out, err := m.MarshalVT()
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != m.SizeVT() || len(out) != 14 {
		t.Fatalf("encoded %d bytes (%x), SizeVT %d, want 14", len(out), out, m.SizeVT())
	}
	var back BasicMsg
	if err := back.UnmarshalVT(out); err != nil {
		t.Fatal(err)
	}
	if !math.Signbit(float64(back.FloatField)) || !math.Signbit(back.DoubleField) {
		t.Fatalf("decoded %v, %v", back.FloatField, back.DoubleField)
	}
	// +0 is the zero value, and not encoded.
	if out, _ := (&BasicMsg{}).MarshalVT(); len(out) != 0 {
		t.Fatalf("zero message encoded as %x", out)
	}
}

func appendVarint(b []byte, v uint64) []byte {
	for v >= 0x80 {
		b = append(b, byte(v)|0x80)
		v >>= 7
	}
	return append(b, byte(v))
}
