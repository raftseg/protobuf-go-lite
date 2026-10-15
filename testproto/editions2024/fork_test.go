// Copyright 2026 The raftseg authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package editions2024

import "testing"

// A delimited (group) field must end with its own field number, as in
// google.golang.org/protobuf; upstream accepted any end-group tag.
func TestGroupEndsWithItsFieldNumber(t *testing.T) {
	// required_int32 (3) = 1, then delimited_group (11) {label: "a"}.
	start := []byte{0x18, 0x01, 0x5b, 0x0a, 0x01, 'a'}
	var m Edition2024Fixture
	if err := m.UnmarshalVT(append(start, 0x5c)); err != nil { // end group 11
		t.Fatal(err)
	}
	if m.GetDelimitedGroup().GetLabel() != "a" {
		t.Fatalf("group %v", m.GetDelimitedGroup())
	}
	m = Edition2024Fixture{}
	if err := m.UnmarshalVT(append(start, 0x64)); err == nil { // end group 12
		t.Fatalf("group 11 ended by field 12 accepted: %v", m.GetDelimitedGroup())
	}
}
