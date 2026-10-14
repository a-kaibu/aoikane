package dsp

import (
	"fmt"
	"math"
	"math/cmplx"
)

// CrossCorrelation returns the full linear cross-correlation. Index len(b)-1
// means zero lag, and positive lag advances a relative to b. The native chunk
// alignment uses this operation on a window centered in the waveform overlap.
func CrossCorrelation(a, b []float32) ([]float32, error) {
	if len(a) == 0 || len(b) == 0 || len(a) > 65536 || len(b) > 65536 {
		return nil, fmt.Errorf("correlation requires nonempty vectors of at most 65536 samples")
	}
	for _, values := range [][]float32{a, b} {
		for _, v := range values {
			if !finite(v) {
				return nil, fmt.Errorf("correlation input contains a non-finite value")
			}
		}
	}
	length := len(a) + len(b) - 1
	n := 1
	for n < length {
		n <<= 1
	}
	x, y := make([]complex128, n), make([]complex128, n)
	for i, v := range a {
		x[i] = complex(float64(v), 0)
	}
	for i, v := range b {
		y[len(b)-1-i] = complex(float64(v), 0)
	}
	fft(x, false)
	fft(y, false)
	for i := range x {
		x[i] *= y[i]
	}
	fft(x, true)
	out := make([]float32, length)
	for i := range out {
		out[i] = float32(real(x[i]))
	}
	return out, nil
}

// Iterative radix-2 transform. Double precision matches the native alignment
// transform's arithmetic; its correlation output is stored as float32.
func fft(x []complex128, inverse bool) {
	for i, j := 1, 0; i < len(x); i++ {
		bit := len(x) >> 1
		for j&bit != 0 {
			j ^= bit
			bit >>= 1
		}
		j ^= bit
		if i < j {
			x[i], x[j] = x[j], x[i]
		}
	}
	for width := 2; width <= len(x); width <<= 1 {
		angle := -2 * math.Pi / float64(width)
		if inverse {
			angle = -angle
		}
		step := cmplx.Exp(complex(0, angle))
		for base := 0; base < len(x); base += width {
			phase := complex(1, 0)
			for j := 0; j < width/2; j++ {
				u, v := x[base+j], x[base+j+width/2]*phase
				x[base+j], x[base+j+width/2] = u+v, u-v
				phase *= step
			}
		}
	}
	if inverse {
		for i := range x {
			x[i] /= complex(float64(len(x)), 0)
		}
	}
}
