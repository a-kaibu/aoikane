package onnx

import "testing"

func TestOriginalDNNLWideObservedPlans(t *testing.T) {
	// Plans reconstructed from actual engine GEMM calls for two sentences.
	for _, c := range []struct{ frames, kernel, sp, ic int }{
		{389, 21, 65, 64}, {389, 31, 65, 32}, {389, 41, 65, 32}, {389, 11, 65, 64}, {389, 15, 65, 64},
		{4693, 21, 72, 64}, {4693, 31, 783, 4}, {4693, 41, 783, 2}, {4693, 11, 144, 64}, {4693, 15, 783, 8},
	} {
		got := originalDNNLWidePlan(64, 16, c.kernel, c.frames)
		want := originalDNNLPlan{1, 16, c.sp, c.ic}
		if got != want {
			t.Errorf("frames=%d kernel=%d: got %+v want %+v", c.frames, c.kernel, got, want)
		}
	}
}
