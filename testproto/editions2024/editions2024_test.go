// Modified by the raftseg authors: the checks of UnmarshalVTUnsafe, which
// this fork does not generate, are removed.

package editions2024

import (
	stdjson "encoding/json"
	"strings"
	"testing"
)

func TestEdition2024GeneratedBehavior(t *testing.T) {
	required := int32(7)
	name := "nested"
	label := "group"
	explicit := "value"
	state := Edition2024Fixture_STATE_READY
	msg := &Edition2024Fixture{
		RequiredInt32:  &required,
		ExplicitString: &explicit,
		ExplicitState:  &state,
		NestedMessage:  &Edition2024Fixture_Nested{Name: &name, Value: 12},
		DelimitedGroup: &Edition2024Fixture_DelimitedGroup{Label: &label},
		NestedMap: map[string]*Edition2024Fixture_Nested{
			"key": &Edition2024Fixture_Nested{Name: &name, Value: 13},
		},
	}

	wire, err := msg.MarshalVT()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Edition2024Fixture
	if err := got.UnmarshalVT(wire); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.DelimitedGroup == nil || got.DelimitedGroup.GetLabel() != "group" {
		t.Fatalf("delimited group round trip failed: %#v", got.DelimitedGroup)
	}
	if !msg.EqualVT(msg.CloneVT()) {
		t.Fatal("clone/equal mismatch")
	}
	text := msg.String()
	if !strings.Contains(text, "delimited_group: DelimitedGroup") || !strings.Contains(text, `label: "group"`) {
		t.Fatalf("text output omitted delimited group: %s", text)
	}

	j, err := stdjson.Marshal(msg)
	if err != nil {
		t.Fatalf("json marshal: %v", err)
	}
	var fromJSON Edition2024Fixture
	if err := stdjson.Unmarshal(j, &fromJSON); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if fromJSON.GetExplicitString() != "value" || fromJSON.GetExplicitState() != Edition2024Fixture_STATE_READY {
		t.Fatalf("json round trip mismatch: %#v", fromJSON)
	}

	invalidUTF8 := []byte{0x18, 0x01, 0x22, 0x01, 0xff}
	var invalid Edition2024Fixture
	if err := invalid.UnmarshalVT(invalidUTF8); err == nil || !strings.Contains(err.Error(), "invalid UTF-8") {
		t.Fatalf("safe unmarshal invalid UTF-8 error = %v", err)
	}
}
