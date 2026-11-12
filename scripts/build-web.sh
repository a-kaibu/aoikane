#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

GO="${GO:-go}"
OUT="${1:-dist}"
VERSION="$(git describe --tags --always --dirty 2>/dev/null || echo devel)"
GO_VERSION="$("$GO" env GOVERSION)"
GOEXPERIMENT_VALUE="${GOEXPERIMENT:-}"

# Go 1.27's simd experiment emits WebAssembly SIMD for the hot FMA kernels.
if [[ "$GO_VERSION" =~ ^go1\.(27|[2-9][0-9]) ]]; then
  case ",${GOEXPERIMENT_VALUE}," in
    *,simd,*) ;;
    *) GOEXPERIMENT_VALUE="${GOEXPERIMENT_VALUE:+${GOEXPERIMENT_VALUE},}simd" ;;
  esac
else
  echo "warning: ${GO_VERSION} does not provide WebAssembly SIMD; synthesis will be slow" >&2
fi

rm -rf "$OUT"
mkdir -p "$OUT/assets"

echo "Building aoikane.wasm (${VERSION}, ${GOEXPERIMENT_VALUE:-no experiment})"
GOEXPERIMENT="$GOEXPERIMENT_VALUE" GOOS=js GOARCH=wasm CGO_ENABLED=0 "$GO" build \
  -trimpath \
  -ldflags="-s -w -X main.version=${VERSION}" \
  -o "$OUT/aoikane.wasm" \
  ./cmd/aoikane-wasm

WASM_EXEC="$("$GO" env GOROOT)/lib/wasm/wasm_exec.js"
if [ ! -f "$WASM_EXEC" ]; then
  WASM_EXEC="$("$GO" env GOROOT)/misc/wasm/wasm_exec.js"
fi
cp "$WASM_EXEC" "$OUT/wasm_exec.js"

cp web/index.html web/app.js web/style.css web/favicon.svg "$OUT/"

for dir in boundmodel cabocha data dict dict-kansai rules synth; do
  cp -R "assets/${dir}" "$OUT/assets/${dir}"
done

python3 - "$OUT" "$VERSION" <<'PY'
import json
import os
import sys

out, version = sys.argv[1], sys.argv[2]
files = []
for root, _, names in os.walk(os.path.join(out, "assets")):
    for name in sorted(names):
        full = os.path.join(root, name)
        rel = os.path.relpath(full, out).replace(os.sep, "/")
        group = "akane" if rel.startswith("assets/dict-kansai/") else "shared"
        files.append({"path": rel, "size": os.path.getsize(full), "group": group})
files.sort(key=lambda item: item["path"])
manifest = {"version": version, "files": files}
with open(os.path.join(out, "manifest.json"), "w", encoding="utf-8") as handle:
    json.dump(manifest, handle, ensure_ascii=False, indent=2)
    handle.write("\n")
print(f"Manifest: {len(files)} files, {sum(item['size'] for item in files) / (1024 * 1024):.1f} MiB")
PY

echo "Output: ${OUT}"
du -sh "$OUT"
