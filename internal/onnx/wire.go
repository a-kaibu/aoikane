// Package onnx implements the model operations migrated to Go, without a
// shared inference library. Unsupported model graphs are rejected explicitly.
package onnx

import (
	"encoding/binary"
	"fmt"
)

// Field numbers follow the installed ONNX onnx.proto. Keeping the decoder
// small avoids a generated protobuf runtime for the supported model subset.
type field struct {
	number uint64
	wire   uint64
	value  uint64
	data   []byte
}

func fields(data []byte) ([]field, error) {
	var out []field
	for len(data) > 0 {
		tag, n := binary.Uvarint(data)
		if n <= 0 || tag>>3 == 0 {
			return nil, fmt.Errorf("invalid protobuf field tag")
		}
		data = data[n:]
		f := field{number: tag >> 3, wire: tag & 7}
		switch f.wire {
		case 0:
			f.value, n = binary.Uvarint(data)
			if n <= 0 {
				return nil, fmt.Errorf("invalid protobuf varint")
			}
			data = data[n:]
		case 1, 5:
			width := 8
			if f.wire == 5 {
				width = 4
			}
			if len(data) < width {
				return nil, fmt.Errorf("truncated protobuf fixed field")
			}
			f.data, data = data[:width], data[width:]
		case 2:
			length, n := binary.Uvarint(data)
			if n <= 0 || length > uint64(len(data)-n) {
				return nil, fmt.Errorf("invalid protobuf bytes length")
			}
			data = data[n:]
			f.data, data = data[:int(length)], data[int(length):]
		default:
			return nil, fmt.Errorf("unsupported protobuf wire type %d", f.wire)
		}
		out = append(out, f)
	}
	return out, nil
}

func bytesFields(fs []field, number uint64) ([][]byte, error) {
	var values [][]byte
	for _, f := range fs {
		if f.number == number {
			if f.wire != 2 {
				return nil, fmt.Errorf("field %d must be bytes", number)
			}
			values = append(values, f.data)
		}
	}
	return values, nil
}

func oneBytes(fs []field, number uint64) ([]byte, error) {
	v, err := bytesFields(fs, number)
	if err != nil {
		return nil, err
	}
	if len(v) != 1 {
		return nil, fmt.Errorf("expected one field %d, got %d", number, len(v))
	}
	return v[0], nil
}
