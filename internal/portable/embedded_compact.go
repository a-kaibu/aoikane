//go:build !(js && wasm) && aoikane_compact

package portable

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/a-kaibu/aoikane/assets"
	"github.com/klauspost/compress/zstd"
	"github.com/tetratelabs/wazero"
)

const maxDecodedAsset = 96 << 20

func embeddedKey() (string, error) {
	id := strings.TrimSpace(assets.BundleID)
	if _, err := hex.DecodeString(id); err != nil || len(id) != 64 {
		return "", fmt.Errorf("invalid embedded model bundle ID")
	}
	return "compact-" + id, nil
}

func materializeEmbedded(dir string) error {
	r, err := zstd.NewReader(bytes.NewReader(assets.Bundle), zstd.WithDecoderConcurrency(1), zstd.WithDecoderMaxWindow(128<<20))
	if err != nil {
		return err
	}
	defer r.Close()
	tr := tar.NewReader(r)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		name := strings.TrimPrefix(strings.TrimSuffix(h.Name, "/"), "./")
		if !fs.ValidPath(name) || strings.Contains(name, "\\") || !filepath.IsLocal(filepath.FromSlash(name)) {
			return fmt.Errorf("invalid embedded path: %s", h.Name)
		}
		if name == "assets" {
			if h.Typeflag != tar.TypeDir {
				return fmt.Errorf("invalid embedded assets directory")
			}
			continue
		}
		name = strings.TrimPrefix(name, "assets/")
		target := filepath.Join(dir, filepath.FromSlash(name))
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(f, tr)
			closeErr := f.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		default:
			return fmt.Errorf("invalid embedded entry type: %s", h.Name)
		}
	}
	return decodeEmbedded(dir)
}

func decodeEmbedded(dir string) error {
	var manifest struct {
		Files []struct {
			Path   string
			OpenZL *struct {
				Path string
				Size uint32
			}
		}
	}
	b, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, &manifest); err != nil {
		return err
	}
	ctx := context.Background()
	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)
	// The decoder has no host I/O. The lowered module retains its 64-bit
	// C pointer ABI, so use explicit stubs instead of the 32-bit WASI ABI.
	const badFD = uint32(8)
	if _, err := r.NewHostModuleBuilder("wasi_snapshot_preview1").
		NewFunctionBuilder().WithFunc(func(uint32) uint32 { return badFD }).Export("fd_close").
		NewFunctionBuilder().WithFunc(func(uint32, uint64, uint64, uint64) uint32 { return badFD }).Export("fd_write").
		NewFunctionBuilder().WithFunc(func(uint32, uint64, uint32, uint64) uint32 { return badFD }).Export("fd_seek").
		Instantiate(ctx); err != nil {
		return err
	}
	if _, err := r.NewHostModuleBuilder("env").NewFunctionBuilder().WithFunc(func(uint64) {}).
		Export("emscripten_notify_memory_growth").Instantiate(ctx); err != nil {
		return err
	}
	m, err := r.InstantiateWithConfig(ctx, assets.OpenZL, wazero.NewModuleConfig().WithStartFunctions("_initialize"))
	if err != nil {
		return err
	}
	alloc, free, decode := m.ExportedFunction("aoikane_alloc"), m.ExportedFunction("aoikane_free"), m.ExportedFunction("aoikane_decode")
	mem := m.Memory()
	if alloc == nil || free == nil || decode == nil || mem == nil {
		return fmt.Errorf("incomplete embedded OpenZL decoder")
	}
	write := func(target string, packed []byte, size uint32) error {
		p, err := alloc.Call(ctx, uint64(len(packed)))
		if err != nil {
			return fmt.Errorf("OpenZL input allocation failed: %w", err)
		}
		if p[0] == 0 {
			return fmt.Errorf("OpenZL input allocation failed")
		}
		defer free.Call(ctx, p[0])
		q, err := alloc.Call(ctx, uint64(size))
		if err != nil {
			return fmt.Errorf("OpenZL output allocation failed: %w", err)
		}
		if q[0] == 0 {
			return fmt.Errorf("OpenZL output allocation failed")
		}
		defer free.Call(ctx, q[0])
		if !mem.Write(uint32(p[0]), packed) {
			return fmt.Errorf("invalid OpenZL input address")
		}
		result, err := decode.Call(ctx, q[0], uint64(size), p[0], uint64(len(packed)))
		if err != nil {
			return err
		}
		if result[0] != 1 {
			return fmt.Errorf("OpenZL decode failed: %s", filepath.Base(target))
		}
		data, ok := mem.Read(uint32(q[0]), size)
		if !ok {
			return fmt.Errorf("invalid OpenZL output address")
		}
		return os.WriteFile(target, data, 0o644)
	}
	for _, f := range manifest.Files {
		if f.OpenZL == nil {
			continue
		}
		if !fs.ValidPath(f.Path) || !strings.HasPrefix(f.Path, "assets/") || strings.Contains(f.Path, "\\") || !filepath.IsLocal(filepath.FromSlash(f.Path)) || f.OpenZL.Path != f.Path+".zl" || f.OpenZL.Size == 0 || f.OpenZL.Size > maxDecodedAsset {
			return fmt.Errorf("invalid OpenZL asset: %s", f.Path)
		}
		target := filepath.Join(dir, filepath.FromSlash(strings.TrimPrefix(f.Path, "assets/")))
		packedPath := target + ".zl"
		data, err := os.ReadFile(packedPath)
		if err != nil {
			return err
		}
		if err := write(target, data, f.OpenZL.Size); err != nil {
			return err
		}
		if err := os.Remove(packedPath); err != nil {
			return err
		}
	}
	return os.Remove(filepath.Join(dir, "manifest.json"))
}
