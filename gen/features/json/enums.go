// Copyright © 2024 Aperture Robotics, LLC.
// Copyright © 2021 The Things Industries B.V.
// SPDX-License-Identifier: Apache-2.0
//
// Modified by the raftseg authors: import paths rewritten for this
// fork's modules.

package json

import (
	"github.com/raftseg/protobuf-go-lite/gen/compiler/protogen"
)

func (g *jsonGenerator) genEnum(enum *protogen.Enum) {
	if !hasDisableJsonComment(enum.Comments.Leading) {
		g.genEnumMarshaler(enum)
		g.genStdEnumMarshaler(enum)

		g.genEnumUnmarshaler(enum)
		g.genStdEnumUnmarshaler(enum)
	}
}
