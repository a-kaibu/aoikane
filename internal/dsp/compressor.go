// Package dsp implements the model-independent waveform processing stages.
package dsp

import (
	"errors"
	"math"

	"github.com/a-kaibu/aoikane/internal/onnx"
)

// Compressor contains the analysis and overlap windows of the original model.
// The window coefficients are supplied by the private model conversion step.
type Compressor struct {
	FrameSamples     int       `json:"frame_samples"`
	PowerScale       float32   `json:"power_scale"`
	AmplitudeDivisor float32   `json:"amplitude_divisor"`
	WindowPower      []float32 `json:"window_power"`
	WindowOverlap    []float32 `json:"window_overlap"`
}

// Process matches the native compressor's centered power measurement,
// per-frame gain, overlap-add, and final soft limiting. Input is float32 PCM.
func (c Compressor) Process(wave, power []float32) ([]float32, error) {
	return c.process(wave, power, false)
}

// ProcessWithAVX512Unary uses the reconstructed Intel VML Sqrt and Tanh orders.
// These scalar Go kernels run without AVX512 hardware or a shared library.
func (c Compressor) ProcessWithAVX512Unary(wave, power []float32) ([]float32, error) {
	return c.process(wave, power, true)
}

func (c Compressor) process(wave, power []float32, avx512Unary bool) ([]float32, error) {
	if c.FrameSamples != 240 || len(c.WindowPower) != 2048 || len(c.WindowOverlap) != 480 {
		return nil, errors.New("unsupported compressor window configuration")
	}
	if len(power) == 0 || len(wave)%c.FrameSamples != 0 || len(wave)/c.FrameSamples != len(power) {
		return nil, errors.New("waveform and power frame counts differ")
	}
	if c.AmplitudeDivisor <= 0 || !finite(c.AmplitudeDivisor) || !finite(c.PowerScale) {
		return nil, errors.New("invalid compressor magnitude configuration")
	}
	for _, values := range [][]float32{wave, power, c.WindowPower, c.WindowOverlap} {
		for _, v := range values {
			if !finite(v) {
				return nil, errors.New("compressor input contains a non-finite value")
			}
		}
	}
	sqrt, tanh := onnx.MKLSqrt, onnx.MKLTanh
	if avx512Unary {
		sqrt, tanh = onnx.MKLSqrtAVX512, onnx.MKLTanhAVX512
	}
	out := make([]float32, len(wave))
	const threshold float32 = 1e-5
	for frame, p := range power {
		// TensorIterator's outer reduction processes complete 32-frame
		// vector groups directly. The remaining frames reduce four
		// interleaved rows, changing their float32 addition order.
		sum := c.originalPowerSum(wave, frame, frame >= len(power)/32*32)
		rms := sqrt(sum)
		ratio := float32(1)
		if rms > threshold {
			p = min(max(p, 0), 1)
			exponent := float32(float32(p*c.PowerScale)-c.PowerScale) / c.AmplitudeDivisor
			target := originalMagnitudePow(exponent)
			ratio = target / rms
		}
		for j, w := range c.WindowOverlap {
			i := frame*c.FrameSamples + j - 120
			if i >= 0 && i < len(out) {
				out[i] = float32(out[i] + float32(float32(wave[i]*w)*ratio))
			}
		}
	}
	for i, v := range out {
		out[i] = tanh(v)
	}
	return out, nil
}

func finite(v float32) bool { return !math.IsNaN(float64(v)) && !math.IsInf(float64(v), 0) }

// Adapted from PyTorch 2.0.1 SumKernel.cpp multi_row_sum/row_sum
// (Copyright the PyTorch contributors; BSD-3-Clause, see PYTORCH-LICENSE).
// The original 2048-row outer reduction uses four levels with step 16.
func (c Compressor) originalPowerSum(wave []float32, frame int, interleaved bool) float32 {
	streams := 1
	if interleaved {
		streams = 4
	}
	var acc [4][4]float32
	for row := 0; row < 2048/streams; row++ {
		for lane := 0; lane < streams; lane++ {
			j := row*streams + lane
			pos := frame*c.FrameSamples + j - 904
			square := float32(0)
			if pos >= 0 && pos < len(wave) {
				value := float32(wave[pos] * c.WindowPower[j])
				square = float32(value * value)
			}
			acc[0][lane] = float32(acc[0][lane] + square)
		}
		if (row+1)%16 == 0 {
			for level := 1; level < 4; level++ {
				for lane := 0; lane < streams; lane++ {
					acc[level][lane] = float32(acc[level][lane] + acc[level-1][lane])
					acc[level-1][lane] = 0
				}
				if (row+1)&(15<<(level*4)) != 0 {
					break
				}
			}
		}
	}
	for level := 1; level < 4; level++ {
		for lane := 0; lane < streams; lane++ {
			acc[0][lane] = float32(acc[0][lane] + acc[level][lane])
		}
	}
	sum := acc[0][0]
	for lane := 1; lane < streams; lane++ {
		sum = float32(sum + acc[0][lane])
	}
	return sum
}
