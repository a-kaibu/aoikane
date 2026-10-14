package onnx

import "fmt"

// PauseModel executes the audited pau graph without a shared library.
type PauseModel struct{ network *BoundLayers }

func OpenPauseModel(path string) (*PauseModel, error) {
	n, err := loadLayers(path, "587a99c21dd0f9b6b940a39186d73f04b8a853ecfd4e04010da5cc76bc0c61fc", []lstmSpec{
		{236, "onnx::LSTM_422", "onnx::LSTM_423", "onnx::LSTM_421"},
		{128, "onnx::LSTM_465", "onnx::LSTM_466", "onnx::LSTM_464"},
	}, "onnx::MatMul_467")
	if err != nil {
		return nil, err
	}
	return &PauseModel{network: n}, nil
}

// pauseFeatures implements ArgMax(select_last_index=0), EyeLike/Gather,
// Pad on the sequence axis, and concatenation after the 233 input features.
// Short option sequences have zero padding; long ones are cropped at the end.
func pauseFeatures(features, opt []float32, steps int) ([]float32, error) {
	if steps <= 0 || len(features)%233 != 0 || len(features)/233 != steps || len(opt) == 0 || len(opt)%3 != 0 {
		return nil, fmt.Errorf("pause model requires steps*233 features and triples of class probabilities")
	}
	input := make([]float32, steps*236)
	for s := 0; s < steps; s++ {
		copy(input[s*236:], features[s*233:(s+1)*233])
		if s >= len(opt)/3 {
			continue
		}
		best := 0
		for j := 1; j < 3; j++ {
			if opt[s*3+j] > opt[s*3+best] {
				best = j
			}
		}
		input[s*236+233+best] = 1
	}
	return input, nil
}

// Predict accepts one sequence and boundary probabilities (usually the bou
// output after rule overrides). It returns three pause probabilities per step.
func (p *PauseModel) Predict(features, opt []float32, steps int) ([]float32, error) {
	input, err := pauseFeatures(features, opt, steps)
	if err != nil {
		return nil, err
	}
	return p.network.Predict(input, steps)
}

// RunLayers exposes both intermediate hidden sequences for original-model QA.
func (p *PauseModel) RunLayers(features, opt []float32, steps int) ([]float32, []float32, error) {
	input, err := pauseFeatures(features, opt, steps)
	if err != nil {
		return nil, nil, err
	}
	return p.network.Run(input, steps)
}
