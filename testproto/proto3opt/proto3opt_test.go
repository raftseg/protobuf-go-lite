// Modified by the raftseg authors: import paths rewritten for this
// fork's modules.

package proto3opt

import (
	"testing"

	"github.com/raftseg/protobuf-go-lite/internal/testutil/require"
)

func TestEmptyBytesMarshalling(t *testing.T) {
	a := &OptionalFieldInProto3{
		OptionalBytes: nil,
	}
	b := &OptionalFieldInProto3{
		OptionalBytes: []byte{},
	}

	type Message interface {
		MarshalVT() ([]byte, error)
	}

	for i, msg := range []Message{a, b} {
		vt, err := msg.MarshalVT()
		require.NoError(t, err)
		if i == 0 {
			require.Empty(t, vt)
		} else {
			require.NotEmpty(t, vt)
		}
		// goog, err := proto.Marshal(msg)
		// require.NoError(t, err)
		// require.Equal(t, goog, vt)
	}
}
