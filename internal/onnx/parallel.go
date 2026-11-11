package onnx

import (
	"runtime"
	"sync"
	"sync/atomic"
)

// parallelFor runs body(i) for every i in [0, n). Callers only use it for
// iterations that write disjoint outputs, so scheduling never changes the
// numerical order of any value.
func parallelFor(n int, body func(i int)) {
	workers := min(runtime.GOMAXPROCS(0), n)
	if workers <= 1 {
		for i := 0; i < n; i++ {
			body(i)
		}
		return
	}
	var next atomic.Int64
	var wg sync.WaitGroup
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			for {
				i := int(next.Add(1) - 1)
				if i >= n {
					return
				}
				body(i)
			}
		}()
	}
	wg.Wait()
}

// parallelRows splits rows [0, rows) into contiguous blocks when the total
// work is large enough to amortize scheduling. Blocks write disjoint rows.
func parallelRows(rows, rowWork int, body func(lo, hi int)) {
	blocks := min(runtime.GOMAXPROCS(0), rows)
	if blocks <= 1 || rows*rowWork < 1<<16 {
		body(0, rows)
		return
	}
	parallelFor(blocks, func(b int) {
		body(b*rows/blocks, (b+1)*rows/blocks)
	})
}

// parallelElements splits [0, n) into contiguous element ranges.
func parallelElements(n int, body func(lo, hi int)) {
	const block = 1 << 14
	blocks := (n + block - 1) / block
	parallelRows(blocks, block, func(lo, hi int) {
		body(lo*block, min(n, hi*block))
	})
}
