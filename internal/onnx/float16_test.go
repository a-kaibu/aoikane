package onnx

import (
	"encoding/binary"
	"math"
	"testing"
)

func TestFloat16Weights(t *testing.T) {
	cases := []struct {
		half uint16
		want uint32
	}{
		{0x0000, 0x00000000}, {0x8000, 0x80000000},
		{0x0001, 0x33800000}, {0x03ff, 0x387fc000},
		{0x0400, 0x38800000}, {0x3c00, 0x3f800000}, {0xc000, 0xc0000000},
		{0x7bff, 0x477fe000}, {0x7c00, 0x7f800000}, {0xfe00, 0xffc00000},
	}
	var raw []byte
	for _, c := range cases {
		raw = binary.LittleEndian.AppendUint16(raw, c.half)
	}
	fields := []field{{number: 1, wire: 0, value: uint64(len(cases))}, {number: 2, wire: 0, value: 10}, {number: 9, wire: 2, data: raw}}
	value, err := tensorValue(fields)
	if err != nil || value.Kind != 1 || len(value.F) != len(cases) {
		t.Fatalf("incompatible float16 value: %+v, %v", value, err)
	}
	g := Gather{Rows: 1, Width: len(cases), weights: raw, half: true}
	row, err := g.Lookup([]int64{-1})
	if err != nil {
		t.Fatal(err)
	}
	for i, c := range cases {
		if graph, gather := math.Float32bits(value.F[i]), math.Float32bits(row[i]); graph != c.want || gather != c.want {
			t.Errorf("%04x: graph=%08x gather=%08x want=%08x", c.half, graph, gather, c.want)
		}
	}
	fields[2].data = raw[:len(raw)-1]
	if _, err := tensorValue(fields); err == nil {
		t.Fatal("truncated FLOAT16 initializer accepted")
	}
}
