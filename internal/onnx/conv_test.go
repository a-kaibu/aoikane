package onnx

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"testing"
)

func TestFeatConvsMatchOriginal(t *testing.T) {
	for _, tc := range []struct{ name, want string }{
		{"/model/dec_dur/pro_z2/Conv", "c549990e0cfe4bd7834427fff2114fbb33a79be47b27b69514beb9948ab36c6c"},
		{"/model/dec_dur/pro_z1/Conv", "f9f4e6b5df0a6e812d862f61adbdee07cd657d165675408b630fc729d28dae13"},
		{"/model/dec_dur/dec_y/0/blocks/1/Conv", "8b24559f01493f132cce7ac136c42cde28023bddaf7ba5c5242e5a33b840581b"},
		{"/model/dec_f0c/dec_y1/2/blocks/1/Conv", "fecefbb71513d0731336cd46e1b2b831acad145e5c6f271c37d7a9e48d1498c3"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, err := OpenConv1D("../../assets/synth/featnet.onnx", tc.name)
			if err != nil {
				t.Fatal(err)
			}
			r := rand.New(rand.NewPCG(9, 2))
			hash := sha256.New()
			for _, length := range []int{1, 2, 5, 17, 65, 257} {
				input := make([]float32, c.In*length)
				for i := range input {
					input[i] = float32(r.Float64()*2 - 1)
				}
				out, err := c.Run(input, length)
				if err != nil {
					t.Fatal(err)
				}
				var raw []byte
				for _, x := range out {
					raw = binary.LittleEndian.AppendUint32(raw, math.Float32bits(x))
				}
				hash.Write(raw)
			}
			if got := fmt.Sprintf("%x", hash.Sum(nil)); got != tc.want {
				t.Fatalf("original Conv SHA256 %s want %s", got, tc.want)
			}
			if _, err := c.Run(nil, 0); err == nil {
				t.Fatal("empty Conv input accepted")
			}
			if _, err := c.Run(make([]float32, c.In+1), 1); err == nil {
				t.Fatal("invalid Conv input shape accepted")
			}
		})
	}
}

func TestConvRejectsUnsupportedAttributes(t *testing.T) {
	model := func(attrs ...[]byte) []byte {
		n := protoBytes(1, []byte("input"))
		n = append(n, protoBytes(1, []byte("weight"))...)
		n = append(n, protoBytes(3, []byte("conv"))...)
		n = append(n, protoBytes(4, []byte("Conv"))...)
		for _, a := range attrs {
			n = append(n, protoBytes(5, a)...)
		}
		tensor := []byte{8, 4, 8, 1, 8, 1, 16, 1}
		tensor = append(tensor, protoBytes(8, []byte("weight"))...)
		var raw []byte
		for i := 0; i < 4; i++ {
			raw = binary.LittleEndian.AppendUint32(raw, math.Float32bits(1))
		}
		tensor = append(tensor, protoBytes(9, raw)...)
		graph := protoBytes(1, n)
		graph = append(graph, protoBytes(5, tensor)...)
		return protoBytes(7, graph)
	}
	attr := func(name string, value []byte) []byte { return append(protoBytes(1, []byte(name)), value...) }
	path := filepath.Join(t.TempDir(), "conv.onnx")
	valid := model()
	if err := os.WriteFile(path, valid, 0600); err != nil {
		t.Fatal(err)
	}
	c, err := OpenConv1D(path, "conv")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Run([]float32{2, 3}, 2); err != nil {
		t.Fatal(err)
	}
	// These attributes now load for original-kernel execution, but the MLAS
	// path must continue to reject them rather than silently ignore them.
	for _, name := range []string{"strides", "dilations"} {
		if err := os.WriteFile(path, model(attr(name, protoBytes(8, []byte{2}))), 0600); err != nil {
			t.Fatal(err)
		}
		c, err := OpenConv1D(path, "conv")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := c.Run([]float32{2, 3}, 2); err == nil {
			t.Fatal("MLAS accepted non-unit Conv stride/dilation")
		}
	}
	for _, bad := range [][]byte{
		model(attr("group", []byte{24, 2})),
		model(attr("strides", protoBytes(8, []byte{0}))),
		model(attr("dilations", protoBytes(8, []byte{0}))),
		model(attr("strides", protoBytes(8, []byte{1, 1}))),
		model(attr("dilations", protoBytes(8, []byte{1, 1}))),
		model(attr("auto_pad", protoBytes(4, []byte("SAME_LOWER")))),
		model(attr("pads", protoBytes(8, []byte{0, 0})), attr("auto_pad", protoBytes(4, []byte("VALID")))),
		model(attr("group", []byte{24, 1}), attr("group", []byte{24, 1})),
		valid[:len(valid)-1],
	} {
		if err := os.WriteFile(path, bad, 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := OpenConv1D(path, "conv"); err == nil {
			t.Fatal("unsupported or malformed Conv accepted")
		}
	}
}
