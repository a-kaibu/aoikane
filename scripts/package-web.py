#!/usr/bin/env python3
"""Package a built static site, not a standalone WASM without its dependencies."""
import argparse
import hashlib
import json
from pathlib import Path
import re
import shutil
import zipfile


def digest(path):
    with path.open("rb") as stream:
        value = hashlib.sha256()
        for chunk in iter(lambda: stream.read(1 << 20), b""):
            value.update(chunk)
        return value.hexdigest()


def package(site, output, version):
    site, output = Path(site).resolve(), Path(output).resolve()
    version = version.removeprefix("v")
    if not re.fullmatch(r"\d+\.\d+\.\d+(?:[-+][A-Za-z0-9.-]+)*", version):
        raise ValueError("expected a release version such as v0.2.4")
    if output == site or output.is_relative_to(site):
        raise ValueError("output must be outside the site directory")
    for name in ("index.html", "app.js", "style.css", "aoikane.wasm", "wasm_exec.js", "manifest.json"):
        if not (site / name).is_file():
            raise ValueError(f"missing runtime file: {name}")
    with (site / "aoikane.wasm").open("rb") as stream:
        if stream.read(8) != b"\0asm\x01\0\0\0":
            raise ValueError("invalid WASM header")
    manifest = json.loads((site / "manifest.json").read_text())
    for entry in manifest["files"]:
        path = (site / entry["path"]).resolve()
        if not path.is_relative_to(site) or not path.is_file() or path.stat().st_size != entry["size"]:
            raise ValueError(f"missing or invalid model: {entry['path']}")
        if entry.get("sha256") and digest(path) != entry["sha256"]:
            raise ValueError(f"model checksum mismatch: {entry['path']}")
    files = sorted(site.rglob("*"))
    if any(path.is_symlink() for path in files):
        raise ValueError("site must not contain symlinks")
    output.mkdir(parents=True, exist_ok=True)
    prefix = f"aoikane_web_{version}"
    archive, wasm = output / f"{prefix}.zip", output / f"aoikane_{version}.wasm"
    checksums = output / f"{prefix}_checksums.txt"
    if any(path.exists() for path in (archive, wasm, checksums)):
        raise FileExistsError("release output already exists; use an empty output directory")
    readme = f"""aoikane WebAssembly {version}

This archive contains the WASM, matching Go runtime, UI, workers and both voices.
Serve this directory with a static HTTP server; do not open index.html as file://.

    python3 -m http.server 8000

Then open http://localhost:8000/ in a modern desktop browser.
Text synthesis runs locally. No synthesis server is required.
The standalone .wasm download still needs wasm_exec.js, model assets, and a JS host.
Optional kernels.wasm provides the fast SIMD path; keep it with the other files.

WASM、Goランタイム、画面、ワーカー、葵・茜のモデルを含む実行用一式です。
展開後、このフォルダーで上記のコマンドを実行してブラウザーで開いてください。
index.html のダブルクリックでは動きません。入力テキストは外部送信しません。
"""
    temporary = archive.with_suffix(".zip.tmp")
    with zipfile.ZipFile(temporary, "w", compression=zipfile.ZIP_DEFLATED) as bundle:
        def add(path, name):
            info = zipfile.ZipInfo(f"{prefix}/{name}", (1980, 1, 1, 0, 0, 0))
            info.external_attr = 0o100644 << 16
            info.compress_type = zipfile.ZIP_STORED if path.suffix == ".gz" else zipfile.ZIP_DEFLATED
            with path.open("rb") as src, bundle.open(info, "w") as dst:
                shutil.copyfileobj(src, dst, 1 << 20)
        for path in files:
            if path.is_file():
                if path.relative_to(site).as_posix() == "README.txt":
                    raise ValueError("README.txt is reserved for package instructions")
                add(path, path.relative_to(site).as_posix())
        root = Path(__file__).resolve().parent.parent
        for path in sorted((root / "internal").rglob("*LICENSE*")):
            if path.is_file():
                add(path, f"licenses/{path.relative_to(root).as_posix()}")
        info = zipfile.ZipInfo(f"{prefix}/README.txt", (1980, 1, 1, 0, 0, 0))
        info.external_attr = 0o100644 << 16
        bundle.writestr(info, readme.encode("utf-8"), compress_type=zipfile.ZIP_DEFLATED)
    temporary.replace(archive)
    shutil.copyfile(site / "aoikane.wasm", wasm)
    checksums.write_text("".join(f"{digest(path)}  {path.name}\n" for path in (archive, wasm)))
    return archive, wasm, checksums


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--site", default="dist")
    parser.add_argument("--output", required=True)
    parser.add_argument("--version", required=True)
    args = parser.parse_args()
    for path in package(args.site, args.output, args.version):
        print(path)
