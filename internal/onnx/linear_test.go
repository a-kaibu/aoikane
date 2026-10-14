package onnx

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand/v2"
	"testing"
)

func TestFeatLinearsMatchOriginal(t *testing.T) {
	names := []string{
		"/model/emb_tok/art/block/MatMul", "/model/emb_tok/nima/block/MatMul", "/model/emb_tok/niab/block/MatMul",
		"/model/dec_dur/pro_m/block/MatMul", "/model/dec_dur/pro_y/block/MatMul",
		"/model/dec_f0c/pro_m/block/MatMul", "/model/dec_f0c/pro_s/block/MatMul", "/model/dec_f0c/pro_y2/block/MatMul", "/model/dec_f0c/pro_y1/block/MatMul",
		"/model/dec_pwr/pro_y1/block/MatMul", "/model/dec_pwr/pro_y2/block/MatMul", "/model/dec_bap/pro_y/block/MatMul", "/model/dec_spc/pro_y/block/MatMul",
	}
	hash := sha256.New()
	count := 0
	for _, name := range names {
		l, err := OpenLinear("../../assets/synth/featnet.onnx", name)
		if err != nil {
			t.Fatal(err)
		}
		r := rand.New(rand.NewPCG(9, 2))
		for _, rows := range []int{1, 2, 5, 17, 65, 257} {
			input := make([]float32, rows*l.In)
			for i := range input {
				input[i] = float32(r.Float64()*2 - 1)
			}
			out, err := l.Run(input, rows)
			if err != nil {
				t.Fatal(err)
			}
			if len(out) != rows*l.Out {
				t.Fatal("MatMul output shape differs")
			}
			var raw []byte
			for _, x := range out {
				raw = binary.LittleEndian.AppendUint32(raw, math.Float32bits(x))
			}
			hash.Write(raw)
			count += len(out)
		}
		if _, err := l.Run(nil, 0); err == nil {
			t.Fatal("empty MatMul input accepted")
		}
		if _, err := l.Run(make([]float32, l.In+1), 1); err == nil {
			t.Fatal("incorrect MatMul input shape accepted")
		}
	}
	// Original Runtime output for all thirteen nodes, independently recorded
	// by verify_feat_linears, including scalar/two-channel output projections.
	const want = "cbfbab72f545342c31ff8477ccf65b351173cceffb188865e140dfcb0ddc759e"
	if got := fmt.Sprintf("%x", hash.Sum(nil)); got != want {
		t.Fatalf("%d native MatMul values: SHA256 %s want %s", count, got, want)
	}
}
