// Copyright 2016-2021 Intel Corporation
// Copyright 2026 aoikane contributors
// SPDX-License-Identifier: Apache-2.0
// Adapted to Go and the observed original engine's cache/thread constants from
// oneDNN v2.7 src/cpu/gemm_convolution_utils.cpp. See ONEDNN-LICENSE.
package onnx

import "math"

// Original oneDNN 2.7.3's batch-one NCSP im2col cache heuristic. Numeric
// cache/thread constants describe the observed original engine, not this host.
// The search follows gemm_convolution_utils.cpp (oneDNN, Apache-2.0).
type originalDNNLPlan struct{ threadsOC, oc, sp, ic int }

func originalDNNLWidePlan(in, out, kernel, frames int) originalDNNLPlan {
	return originalDNNLStridedPlan(in, out, kernel, frames, 1)
}

func originalDNNLStridedPlan(in, out, kernel, frames, stride int, profile ...int) originalDNNLPlan {
	threads := originalThreadCount(profile)
	const cache, simd = 131072, 8 // 512 KiB / sizeof(float)
	up := func(n, d int) int { return (n + d - 1) / d }
	round := func(n, d int) int { return up(n, d) * d }
	clamp := func(n, lo, hi int) int { return min(hi, max(lo, n)) }
	pow := func(x, y float32) float32 { return float32(math.Pow(float64(x), float64(y))) }
	best := originalDNNLPlan{1, 1, 1, in}
	bestScore := float32(0)
	try := func(nt, oc, sp int) {
		maxOC, minOC := up(out, nt), max(1, out/nt)
		maxSP := up(frames, threads/nt)
		oc = clamp(oc, 8, maxOC)
		sp = clamp(sp, simd, maxSP)
		minSP := up(frames, up(threads, nt))
		maxWork := maxOC * maxSP
		if out%nt <= threads%nt {
			maxWork = max(maxOC*minSP, minOC*maxSP)
		}
		minSP = max(1, frames/up(threads, nt))
		minWork := minOC * minSP
		if threads%nt <= out%nt {
			minWork = min(maxOC*minSP, minOC*(frames/(threads/nt)))
		}
		balance := float32(minWork) / float32(maxWork)
		maxIC := 0
		if sp >= maxSP {
			maxIC = cache / (stride*sp + kernel*sp)
			if oc < maxOC {
				maxIC = min(maxIC, (cache-oc*sp)/(kernel*sp+kernel*oc))
			}
		} else {
			maxIC = (cache - oc*sp) / (stride*sp + kernel*sp + kernel*maxOC)
		}
		if maxIC < in {
			maxIC = (cache - oc*sp) / (stride*sp + kernel*sp + kernel*maxOC)
		}
		cost := 1
		if maxIC < 1 {
			cost = 20
		}
		maxIC = max(1, maxIC)
		ic := max(1, in/up(in, maxIC))
		nb := up(in, ic)
		kb := ic * kernel
		outOps := cost * maxOC * round(maxSP, simd)
		if ic != in {
			outOps *= 2*nb - 1
		}
		inputOps := cost * kernel * in * round(maxSP*stride, simd)
		colTail := float32(round(sp, simd)) / float32(sp)
		colOps := int(float32(cost) * (float32(kernel*in*round(maxSP*stride, simd))*colTail + float32(kernel*in*round(maxSP*stride, simd))*colTail))
		if stride != 1 {
			colOps = int(float32(colOps) * 8)
		}
		weightOps := cost * maxOC * round(in*kernel, simd)
		mem := float32(maxSP) / simd * float32(maxOC) * float32(in*kernel) / float32(inputOps+colOps+weightOps+outOps)
		ocBalance := float32(maxOC) / float32(round(maxOC, oc))
		spBalance := float32(maxSP) / float32(round(maxSP, sp))
		icBalance := float32(in) / float32(round(in, ic))
		regBalance := float32(sp) / float32(round(sp, 3*simd))
		gemm := float32(sp) * float32(oc) * float32(kb) / (float32(maxOC) * float32(maxSP) * float32(in*kernel))
		calc := float32(sp) / simd * float32(oc) * float32(kb) / float32(round(sp, simd)*kb+oc*round(kb, simd)+oc*round(sp, simd))
		score := pow(balance, 8) * ocBalance * spBalance * icBalance * regBalance * mem * pow(gemm, .5) * calc
		score = pow(score, 1/14.5)
		if score > bestScore {
			bestScore = score
			best = originalDNNLPlan{nt, oc, sp, ic}
		}
	}
	explore := func(nt, oc, sp int) {
		try(nt, oc, sp)
		try(nt, oc/8*8, sp/48*48)
		try(nt, round(oc, 8), sp/48*48)
		try(nt, round(oc, 8), round(sp, simd))
		try(nt, round(oc, 8), round(sp, 48))
	}
	for nt := 1; nt <= threads; nt++ {
		maxOC, maxSP := up(out, nt), up(frames, threads/nt)
		try(nt, maxOC, maxSP)
		if best.oc == maxOC && best.sp == maxSP && best.ic == in {
			continue
		}
		a, b := kernel+stride, kernel*maxOC
		oc := clamp((cache-a*in*simd-b*in)/simd, 8, maxOC)
		sp := clamp((cache-b*in)/(a*in+oc), simd, maxSP)
		explore(nt, oc, sp)
		maxIC := max(1, cache/(2*8*a+2*b))
		if maxIC < in {
			ic := in / max(1, in/maxIC)
			sp = max(1, (cache-2*b*ic)/(2*ic*a))
			oc = cache / 2 / sp
			if oc > maxOC {
				oc = maxOC
				sp = clamp((cache-b*ic)/(a*ic+oc), simd, maxSP)
			} else if sp > maxSP {
				sp = maxSP
				oc = clamp((cache-a*ic*sp-b*ic)/sp, 8, maxOC)
			}
			explore(nt, oc, sp)
		}
		oc = maxOC
		sp = max(simd, ((cache-b*in)/(a*in+oc))/simd*simd)
		explore(nt, oc, sp)
	}
	return best
}
