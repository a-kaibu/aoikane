#!/usr/bin/env python3
"""Compare lossless compressors on representative Aoikane assets."""

import argparse
from concurrent.futures import ThreadPoolExecutor
import filecmp
import gzip
import json
from pathlib import Path
import shutil
import subprocess
import tempfile

import onnx

ROOT = Path(__file__).resolve().parent.parent
INPUTS = ("synth/featnet.onnx", "boundmodel/emb.onnx", "dict/sys.dic", "dict/sys.tok", "dict/sys.fea.gz")


def describe(path):
    if path.suffix == ".onnx":
        data = path.read_bytes()
        graph = onnx.load_model_from_string(data).graph
        offset, lines = 0, []
        for i, t in enumerate(graph.initializer):
            raw = t.raw_data
            if t.data_type != onnx.TensorProto.FLOAT or len(raw) < 4096 or len(raw) % 4:
                continue
            start = data.index(raw, offset)
            if start > offset:
                lines.append(f"gap{i}: Bytes({start - offset})")
            lines.append(f"weights{i}: Float32LE[{len(raw) // 4}]")
            offset = start + len(raw)
        if offset < len(data):
            lines.append(f"tail: Bytes({len(data) - offset})")
        return "\n".join(lines) + "\n"
    if path.name == "sys.dic":
        if (path.stat().st_size - 37) % 4:
            raise ValueError("malformed dictionary trie")
        return "header: Bytes(37)\nunits: UInt32LE[]\n"
    if path.name == "sys.tok":
        if (path.stat().st_size - 50) % 9:
            raise ValueError("malformed dictionary tokens")
        return ("record Token() { context: UInt16LE, cost: Int16LE, form: UInt8, offset: UInt32LE }\n"
                "header: Bytes(50)\ntokens: Token[]\n")
    return None


def run(args, name):
    original = ROOT / "assets" / name
    with tempfile.TemporaryDirectory(dir=args.scratch) as temp:
        temp = Path(temp)
        source = original
        if original.suffix == ".gz":
            source = temp / original.stem
            with gzip.open(original, "rb") as src, source.open("wb") as dst:
                shutil.copyfileobj(src, dst)
        schema = describe(source)
        profiles = {"serial": ["--chunk-size", "128MiB"]}
        if schema:
            description = temp / "asset.sddl"
            description.write_text(schema)
            profiles["sddl2"] = ["--profile-arg", description, "--strict"]
        candidate = temp / "asset.zl"
        restored = temp / "restored"
        result = {"asset": name, "rawBytes": source.stat().st_size, "originalBytes": original.stat().st_size}
        for profile, extra in profiles.items():
            subprocess.run([args.zli, "compress", "--profile", profile, *extra, "-l", "12", source,
                            "-o", candidate], check=True, capture_output=True)
            subprocess.run([args.zli, "decompress", candidate, "-o", restored], check=True, capture_output=True)
            if not filecmp.cmp(source, restored, shallow=False):
                raise ValueError(f"OpenZL round-trip mismatch ({profile}): {name}")
            result[profile] = candidate.stat().st_size
            candidate.unlink()
            restored.unlink()
        zst = temp / "asset.zst"
        subprocess.run(["zstd", "-q", "-19", "--long=27", "-T1", source, "-o", zst], check=True)
        subprocess.run(["zstd", "-q", "-d", zst, "-o", restored], check=True)
        if not filecmp.cmp(source, restored, shallow=False):
            raise ValueError(f"zstd round-trip mismatch: {name}")
        result["zstd19"] = zst.stat().st_size
        return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--zli", default="zli")
    parser.add_argument("--scratch", type=Path)
    parser.add_argument("--jobs", type=int, default=2)
    parser.add_argument("--output", type=Path)
    args = parser.parse_args()
    if args.jobs < 1:
        parser.error("--jobs must be positive")
    try:
        with ThreadPoolExecutor(args.jobs) as pool:
            rows = []
            for row in pool.map(lambda name: run(args, name), INPUTS):
                rows.append(row)
                print(json.dumps(row), flush=True)
    except subprocess.CalledProcessError as err:
        if err.stderr:
            print(err.stderr.decode(errors="replace"), flush=True)
        raise
    if args.output:
        args.output.write_text(json.dumps(rows, indent=2) + "\n")


if __name__ == "__main__":
    main()
