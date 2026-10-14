package phrase

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/a-kaibu/aoikane/internal/onnx"
)

// The standard voice's boundary predictor (fe_bound::OnnxRuntimeBound). Three
// ONNX models run in sequence: emb embeds the base-form token ids, bou
// classifies every morph (no boundary / accent phrase / secondary boundary) and
// pau estimates the pause that follows. The decoded models and the vocabulary
// are private data the caller supplies.

const (
	embedDims   = 50
	featureDims = 233
	classCount  = 3
	// predictorMask holds the kinds the predictor must not overwrite
	// (bits of 0x43006000).
	predictorMask = 0x43006000
)

// Encoder converts UTF-8 to the engine's CP932 byte strings.
type Encoder interface {
	Encode(string) ([]byte, error)
}

// BoundModel is the loaded predictor.
type BoundModel struct {
	emb          *onnx.Gather
	bou          *onnx.BoundLayers
	pau          *onnx.PauseModel
	vocab        map[string]int64 // CP932 base form -> first index
	average, std float32
	nan          float32
	defaultPause [3]float32
	enc          Encoder
}

// boundMeta mirrors the observed model configuration (meta.json).
type boundMeta struct {
	Average      float32
	Std          float32
	NaN          float32
	DefaultPause [3]float32
}

// OpenBoundModel loads emb.onnx, bou.onnx, pau.onnx and vocab.json (a JSON list
// of hex-encoded CP932 words) from dir. The former runtime-library argument
// is retained for call compatibility and is no longer used.
func OpenBoundModel(_ string, dir string, enc Encoder) (*BoundModel, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "vocab.json"))
	if err != nil {
		return nil, err
	}
	var words []string
	if err := json.Unmarshal(raw, &words); err != nil {
		return nil, err
	}
	m := &BoundModel{vocab: make(map[string]int64, len(words)), enc: enc,
		average: math.Float32frombits(0x40015b60), std: math.Float32frombits(0x3fdc0f8a), nan: -2}
	for i, w := range words {
		b, err := hex.DecodeString(w)
		if err != nil {
			return nil, err
		}
		if _, ok := m.vocab[string(b)]; !ok {
			m.vocab[string(b)] = int64(i)
		}
	}
	if m.emb, err = onnx.OpenGather(filepath.Join(dir, "emb.onnx")); err != nil {
		return nil, err
	}
	if m.emb.Width != embedDims || m.emb.Rows != len(words) {
		return nil, fmt.Errorf("embedding dimensions %dx%d do not match vocabulary %dx%d", m.emb.Rows, m.emb.Width, len(words), embedDims)
	}
	if m.bou, err = onnx.OpenBoundLayers(filepath.Join(dir, "bou.onnx")); err != nil {
		m.Close()
		return nil, err
	}
	if m.pau, err = onnx.OpenPauseModel(filepath.Join(dir, "pau.onnx")); err != nil {
		m.Close()
		return nil, err
	}
	return m, nil
}

// Close is retained for callers; immutable Go models own no external handles.
func (m *BoundModel) Close() {}

func idIndex(list []uint32, id uint32) int {
	for i, v := range list {
		if v == id {
			return i
		}
	}
	return len(list) - 1
}

// tokenID is the vocabulary index of the morph's base form (1 when unknown).
func (m *BoundModel) tokenID(mo *Morph) (int64, error) {
	if mo.Base == "" {
		return 1, nil
	}
	b, err := m.enc.Encode(mo.Base)
	if err != nil {
		return 1, nil
	}
	if id, ok := m.vocab[string(b)]; ok {
		return id, nil
	}
	return 1, nil
}

// moraFeature is 18fa60. The native loop visits the first expansion morph,
// then advances only while the next morph itself has a successor. Thus a
// multi-node detached chain excludes its tail; a single-node chain is counted.
// Chain morphs hold no cached count yet, so count their pronunciation.
func moraFeature(mo *Morph) int32 {
	n := mo.Moras
	for x := mo.ExpFirst; x != nil; {
		if c := x.SelectedCandidate(); c != nil {
			for _, r := range c.Pronunciation {
				if startsMora(r) {
					n++
				}
			}
		}
		if x == mo.ExpLast {
			break
		}
		x = x.Next
		if x != nil && x.Next == nil {
			break
		}
	}
	return n
}

// features is 12a350 without the embedding columns.
func (m *BoundModel) features(words []*Morph) ([]int64, []float32, error) {
	ids := make([]int64, len(words))
	feat := make([]float32, len(words)*featureDims)
	for i, mo := range words {
		id, err := m.tokenID(mo)
		if err != nil {
			return nil, nil, err
		}
		ids[i] = id
		row := feat[i*featureDims : (i+1)*featureDims]
		code := idIndex(boundPOSIDs[:], mo.POS)
		for j, off := range [4]int{0, 0xf, 0x35, 0x44} {
			row[embedDims+int(boundPOSFeature[code][j])+off] = 1
		}
		row[embedDims+0x49+int(boundTypeFeature[idIndex(boundTypeIDs[:], mo.POS3)])] = 1
		row[embedDims+0x96+int(boundFormFeature[idIndex(boundFormIDs[:], mo.POS2)])] = 1
		v := float32(moraFeature(mo)) - m.average
		if v != v {
			v = m.nan
		} else {
			v /= m.std
		}
		row[featureDims-1] = v
	}
	return ids, feat, nil
}

func i64bytes(v []int64) []byte {
	b := make([]byte, 8*len(v))
	for i, x := range v {
		binary.LittleEndian.PutUint64(b[8*i:], uint64(x))
	}
	return b
}

// ruleClass is 12aad0: the model class a rule-based boundary kind forces.
func ruleClass(kind int32) (int, bool) {
	switch kind {
	case 2, 7, 0xd, 0xe, 0xf, 0x11, 0x12, 0x16, 0x1b, 0x1e:
		return 2, true
	case 0xc:
		return 0, true
	case 0x10:
		return 1, true
	}
	return 0, false
}

func argmax(v []float32) int {
	best := 0
	for i := 1; i < len(v); i++ {
		if v[i] > v[best] {
			best = i
		}
	}
	return best
}

// Predict runs the models over the morphs (everything before the end-of-text
// morph), whose Kind already holds the rule-based boundary, and stores the
// predicted kinds (12d970, 12aad0, 12bd40).
func (m *BoundModel) Predict(doc *Document) error {
	var words []*Morph
	for _, mo := range doc.Morphs {
		if mo == doc.EOS {
			break
		}
		words = append(words, mo)
	}
	if len(words) == 0 {
		return nil
	}
	ids, feat, err := m.features(words)
	if err != nil {
		return err
	}
	embeddings, err := m.emb.Lookup(ids)
	if err != nil {
		return err
	}
	if len(embeddings) != len(words)*embedDims {
		return fmt.Errorf("embedding model returned %d values for %d morphs", len(embeddings), len(words))
	}
	for i := range words {
		copy(feat[i*featureDims:], embeddings[i*embedDims:(i+1)*embedDims])
	}
	prob, err := m.bou.Predict(feat, len(words))
	if err != nil {
		return err
	}
	if len(prob) != len(words)*classCount {
		return fmt.Errorf("boundary model returned %d values for %d morphs", len(prob), len(words))
	}
	for i, mo := range words {
		if c, ok := ruleClass(mo.Kind); ok {
			for j := 0; j < classCount; j++ {
				prob[i*classCount+j] = 0
			}
			prob[i*classCount+c] = 1
		}
	}
	pause, err := m.pau.Predict(feat, prob, len(words))
	if err != nil {
		return err
	}
	if len(pause) != len(words)*classCount {
		return fmt.Errorf("pause model returned %d values for %d morphs", len(pause), len(words))
	}
	for i, mo := range words {
		var kind int32
		switch argmax(prob[i*classCount : (i+1)*classCount]) {
		case 1:
			kind = 0x10
		case 2:
			kind = 0x11
		}
		if kind != 0 && uint32(mo.Kind)-0x1a > 1 {
			var p [3]float32
			for j := range p {
				p[j] = pause[i*classCount+j] - m.defaultPause[j]
			}
			switch argmax(p[:]) {
			case 1:
				kind = 0x1c
			case 2:
				kind = 0x1d
			}
		}
		if uint32(mo.Kind) > 0x1e || predictorMask>>(uint32(mo.Kind)&0x1f)&1 == 0 {
			mo.Kind = kind
		}
	}
	return nil
}
