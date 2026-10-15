// Copyright © 2021 The Things Industries B.V.
// SPDX-License-Identifier: Apache-2.0
//
// Modified by the raftseg authors: import paths rewritten for this
// fork's modules.

package json

import jsoniter "github.com/raftseg/protobuf-go-lite/internal/jsoniter"

func valueTypeString(v jsoniter.ValueType) string {
	switch v {
	case jsoniter.StringValue:
		return "String"
	case jsoniter.NumberValue:
		return "Number"
	case jsoniter.NilValue:
		return "Null"
	case jsoniter.BoolValue:
		return "Bool"
	case jsoniter.ArrayValue:
		return "Array"
	case jsoniter.ObjectValue:
		return "Object"
	default:
		return "unknown"
	}
}
