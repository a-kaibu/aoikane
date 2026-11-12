package synthesis

import (
	"context"
	"errors"
	"sync"
)

// RenderMany preserves sentence boundaries and output ordering. Browser worker
// inference is asynchronous, so another sentence's acoustic model can execute
// while the worker pool renders an earlier sentence. Native execution retains
// its existing sequential scheduling and numerical profiles.
func (e *Engine) RenderMany(ctx context.Context, inputs []Input) ([][]byte, error) {
	parts := make([][]byte, len(inputs))
	if browserVocoderParallelism() < 2 || len(inputs) < 2 {
		for i, in := range inputs {
			part, err := e.Render(ctx, in)
			if err != nil {
				return nil, err
			}
			parts[i] = part
		}
		return parts, nil
	}
	e.mu.Lock()
	if e.featnet == nil || e.vocoder == nil {
		e.mu.Unlock()
		return nil, errors.New("portable synthesis engine is closed")
	}
	defer e.mu.Unlock()
	// Models and windows are immutable; each execution owns its own buffers.
	clones := make([]*Engine, min(2, len(inputs)))
	for i := range clones {
		clones[i] = &Engine{profile: e.profile, featnet: e.featnet, vocoder: e.vocoder, compressor: e.compressor}
	}
	errs := make([]error, len(inputs))
	var wg sync.WaitGroup
	for worker, clone := range clones {
		wg.Add(1)
		go func(worker int, clone *Engine) {
			defer wg.Done()
			for i := worker; i < len(inputs); i += len(clones) {
				parts[i], errs[i] = clone.Render(ctx, inputs[i])
			}
		}(worker, clone)
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}
	return parts, nil
}
