package onnx

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"math"
	"testing"
)

func TestBundledEmbeddingMatchesNativeOutput(t *testing.T) {
	g, err := OpenGather("../../assets/boundmodel/emb.onnx")
	if err != nil {
		t.Fatal(err)
	}
	if g.Rows != 50000 || g.Width != 50 {
		t.Fatalf("embedding dimensions %dx%d, native 50000x50", g.Rows, g.Width)
	}
	ids := make([]int64, g.Rows+3)
	for i := 0; i < g.Rows; i++ {
		ids[i] = int64(i)
	}
	ids[g.Rows], ids[g.Rows+1], ids[g.Rows+2] = -1, -int64(g.Rows), 0
	values, err := g.Lookup(ids)
	if err != nil {
		t.Fatal(err)
	}
	var raw []byte
	for _, x := range values {
		raw = binary.LittleEndian.AppendUint32(raw, math.Float32bits(x))
	}
	// Digest of the original emb model's ONNX Runtime output for these
	// indices, captured by verify_embedding on 2026-10-08. No DLL is loaded.
	digest := sha256.Sum256(raw)
	if got := hex.EncodeToString(digest[:]); got != "a6e41bc08a8cfd387a2c0379b99050fa1a41a90d818745f951da6e0356478c79" {
		t.Fatalf("embedding output digest %s differs from native model", got)
	}
}

func protoBytes(number byte, b []byte) []byte {
	out := []byte{number<<3 | 2}
	out = binary.AppendUvarint(out, uint64(len(b)))
	return append(out, b...)
}

func testGatherModel(op string) []byte {
	node := protoBytes(1, []byte("weight"))
	node = append(node, protoBytes(1, []byte("input"))...)
	node = append(node, protoBytes(2, []byte("output"))...)
	node = append(node, protoBytes(4, []byte(op))...)
	tensor := []byte{8, 2, 8, 2, 16, 1}
	tensor = append(tensor, protoBytes(8, []byte("weight"))...)
	var weights []byte
	for _, bits := range []uint32{0x80000000, 0x7fc00042, 0x3f800000, 0xbf800000} {
		weights = binary.LittleEndian.AppendUint32(weights, bits)
	}
	tensor = append(tensor, protoBytes(9, weights)...)
	graph := protoBytes(1, node)
	graph = append(graph, protoBytes(5, tensor)...)
	graph = append(graph, protoBytes(11, protoBytes(1, []byte("input")))...)
	graph = append(graph, protoBytes(12, protoBytes(1, []byte("output")))...)
	return protoBytes(7, graph)
}

func TestGatherPreservesBitsAndIndexSemantics(t *testing.T) {
	g, err := loadGather(testGatherModel("Gather"))
	if err != nil {
		t.Fatal(err)
	}
	out, err := g.Lookup([]int64{1, -2, -1, 0})
	if err != nil {
		t.Fatal(err)
	}
	want := []uint32{0x3f800000, 0xbf800000, 0x80000000, 0x7fc00042, 0x3f800000, 0xbf800000, 0x80000000, 0x7fc00042}
	for i, bits := range want {
		if math.Float32bits(out[i]) != bits {
			t.Fatalf("value %d: %08x want %08x", i, math.Float32bits(out[i]), bits)
		}
	}
	for _, id := range []int64{-3, 2, math.MaxInt64} {
		if _, err := g.Lookup([]int64{id}); err == nil {
			t.Fatalf("accepted out-of-range index %d", id)
		}
	}
	// Lookups return independent copies, including NaN payloads and signed zero.
	out[0] = 99
	next, err := g.Lookup([]int64{1})
	if err != nil || next[0] != 1 {
		t.Fatalf("weight table was mutated: %v, %v", next, err)
	}
}

func TestGatherRejectsUnsupportedOrMalformedModels(t *testing.T) {
	if _, err := loadGather(testGatherModel("MatMul")); err == nil {
		t.Fatal("accepted an unsupported operator")
	}
	valid := testGatherModel("Gather")
	for i := 0; i < len(valid); i++ {
		if _, err := loadGather(valid[:i]); err == nil {
			t.Fatalf("accepted truncated model at %d", i)
		}
	}
	for _, bad := range [][]byte{{0}, {0x0f}, {0x0a, 0xff}, {0x08, 0x80}, {0x09, 0}} {
		if _, err := fields(bad); err == nil {
			t.Fatalf("accepted malformed protobuf %x", bad)
		}
	}
}
