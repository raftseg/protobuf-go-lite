// Modified by the raftseg authors: the checks of UnmarshalVTUnsafe, which
// this fork does not generate, are removed.

package proto2

import (
	"errors"
	"io"
	"testing"
)

func TestPackedFixed32MalformedLength(t *testing.T) {
	wire := []byte{0x22, 0x03, 0x01, 0x02, 0x03}

	var safeOut Fixed32Message
	if err := safeOut.UnmarshalVT(wire); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("UnmarshalVT malformed packed fixed32 = %v, want ErrUnexpectedEOF", err)
	}
}
