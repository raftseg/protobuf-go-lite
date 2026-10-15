// Copyright 2026 The raftseg authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package testproto_maps

import "testing"

// A map entry whose message value appears twice merges the two, as
// google.golang.org/protobuf does; upstream kept the second only.
func TestMapMessageValueMerged(t *testing.T) {
	// int_keys (2): {key: 1, value: {nanos: 5}, value: {seconds: 7}}
	in := []byte{0x12, 0x0a, 0x08, 0x01, 0x12, 0x02, 0x10, 0x05, 0x12, 0x02, 0x08, 0x07}
	var m MsgWithMaps
	if err := m.UnmarshalVT(in); err != nil {
		t.Fatal(err)
	}
	v := m.IntKeys[1]
	if v.GetSeconds() != 7 || v.GetNanos() != 5 {
		t.Fatalf("value %v, want seconds 7 and nanos 5", v)
	}
}
