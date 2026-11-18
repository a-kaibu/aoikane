#!/usr/bin/env python3
"""Compare weight storage formats without changing the inference engine."""

import argparse
from concurrent.futures import ThreadPoolExecutor
import filecmp
import gzip
import json
from pathlib import Path
import re
import shutil
import subprocess
import tempfile
import wave

import ml_dtypes
import numpy as np
import onnx

ROOT = Path(__file__).resolve().parent.parent
MODELS = ("synth/featnet.onnx", "boundmodel/emb.onnx")
MODES = ("fp32", "fp16", "fp8", "int8", "int4")
DICTIONARIES = tuple(Path("assets") / name / "sys.fea.gz" for name in ("dict", "dict-kansai"))


def encode(x, mode):
    """Return packed bytes, scales, and the weights reconstructed from them."""
    if mode == "fp16":
        raw = x.astype("<f2").tobytes()
        return raw, b"", np.frombuffer(raw, "<f2").astype("<f4").reshape(x.shape)
    # Each axis-0 row (output channel or embedding vector) gets a scale.
    limit = {"fp8": 448, "int8": 127, "int4": 7}[mode]
    scale = np.max(np.abs(x), axis=1, keepdims=True) / np.float32(limit)
    scale[scale == 0] = 1
    normalized = np.clip(x / scale, -limit, limit)
    if mode == "fp8":
        q = normalized.astype(ml_dtypes.float8_e4m3fn)
    else:
        q = np.rint(normalized).astype(np.int8)
    if mode == "int4":
        flat = q.ravel().view(np.uint8) & 15
        if flat.size % 2:
            flat = np.append(flat, np.uint8(0))
        packed = (flat[::2] | (flat[1::2] << 4)).tobytes()
        data = np.frombuffer(packed, np.uint8)
        nibbles = np.stack((data & 15, data >> 4), axis=1).ravel()[:x.size]
        decoded = (nibbles.astype(np.int8) ^ 8) - 8
    else:
        packed = q.tobytes()
        decoded = np.frombuffer(packed, q.dtype)
    scales = scale.astype("<f4", copy=False).tobytes()
    restored = decoded.astype("<f4").reshape(x.shape) * np.frombuffer(scales, "<f4")[:, None]
    return packed, scales, restored


def convert(source, runtime, compact, mode):
    m = onnx.load(source)
    runtime.parent.mkdir(parents=True, exist_ok=True)
    compact.parent.mkdir(parents=True, exist_ok=True)
    packed = {}
    for i, tensor in enumerate(m.graph.initializer):
        if tensor.data_type != onnx.TensorProto.FLOAT or len(tensor.dims) < 2 or len(tensor.raw_data) < 4096:
            continue
        x = np.frombuffer(tensor.raw_data, dtype="<f4").reshape(tensor.dims[0], -1)
        raw, scale, restored = encode(x, mode)
        if not np.isfinite(restored).all():
            raise ValueError(f"non-finite weights in {source}: {tensor.name}")
        tensor.raw_data = restored.tobytes()
        packed[i] = (raw, scale)
    onnx.save(m, runtime)
    # A size fixture, not an executable ONNX graph. The initializer order and
    # dimensions identify all packed tensors, with row scales in a sidecar.
    for i, (raw, _) in packed.items():
        m.graph.initializer[i].raw_data = raw
    onnx.save(m, compact)
    if mode != "fp16":
        compact.with_suffix(".scales").write_bytes(b"".join(scales for _, scales in packed.values()))


def setup(args, mode):
    runtime = args.output / "runtime" / mode
    compact = args.output / "size-fixtures" / mode
    runtime.mkdir(parents=True)
    for source in sorted((ROOT / "assets").iterdir()):
        if not source.is_dir():
            continue
        if mode != "fp32" and source.name in {"synth", "boundmodel"}:
            (runtime / source.name).mkdir()
            for p in source.iterdir():
                name = f"{source.name}/{p.name}"
                if name in MODELS:
                    convert(p, runtime / name, compact / name, mode)
                else:
                    (runtime / name).symlink_to(p)
        else:
            (runtime / source.name).symlink_to(source, target_is_directory=True)
    return runtime, compact


def compress_dictionary(source, target, zstd):
    with gzip.open(source, "rb") as src, target.open("wb") as dst:
        with subprocess.Popen(zstd, stdin=subprocess.PIPE, stdout=dst) as proc:
            shutil.copyfileobj(src, proc.stdin)
            proc.stdin.close()
            if proc.wait():
                raise RuntimeError(f"zstd failed: {source}")


def bundle_size(args, compact, dictionaries):
    # Stage copies as symlinks and let tar dereference them. All modes have the
    # same web runtime, licenses aside, and the same losslessly repacked dictionaries.
    with tempfile.TemporaryDirectory(dir=args.output) as temp:
        stage = Path(temp)
        for source in sorted(args.site.rglob("*")):
            if not source.is_file():
                continue
            name = source.relative_to(args.site)
            if name in dictionaries:
                source = dictionaries[name]
                name = name.with_suffix(".zst")
            elif name.parts[0] == "assets" and (compact / Path(*name.parts[1:])).is_file():
                source = compact / Path(*name.parts[1:])
            target = stage / name
            target.parent.mkdir(parents=True, exist_ok=True)
            target.symlink_to(source)
        for sidecar in compact.rglob("*.scales"):
            (stage / "assets" / sidecar.relative_to(compact)).symlink_to(sidecar)
        with subprocess.Popen(["tar", "--sort=name", "-chf", "-", "-C", stage, "."], stdout=subprocess.PIPE) as tar:
            with subprocess.Popen(args.zstd, stdin=tar.stdout, stdout=subprocess.PIPE) as proc:
                tar.stdout.close()
                size = sum(len(chunk) for chunk in iter(lambda: proc.stdout.read(1 << 20), b""))
                zstd_status, tar_status = proc.wait(), tar.wait()
                if zstd_status or tar_status:
                    raise RuntimeError("bundle size compression failed")
    return size


def synth(args, mode, runtime, case, voice):
    text = (ROOT / "benchmarks" / case["file"]).read_text().strip() if "file" in case else case["texts"][voice]
    out = args.output / "wav" / mode / f"{case['id']}-{voice}.wav"
    out.parent.mkdir(parents=True, exist_ok=True)
    subprocess.run([args.binary, "--assets", runtime, "--voice", voice, "--text", text,
                    "--srt", "-o", out], check=True, stdout=subprocess.DEVNULL)
    return out


def pcm(path):
    with wave.open(str(path)) as w:
        if w.getsampwidth() != 2 or w.getnchannels() != 1:
            raise ValueError(f"expected mono PCM16: {path}")
        return w.getframerate(), np.frombuffer(w.readframes(w.getnframes()), "<i2").astype(np.float32) / 32768


CUE = re.compile(r"(\d+):(\d+):(\d+),(\d+)")


def cues(path):
    times, text = [], []
    for block in path.with_suffix(".srt").read_text().strip().split("\n\n"):
        lines = block.splitlines()
        times.extend(((int(h) * 60 + int(m)) * 60 + int(s)) * 1000 + int(ms)
                     for h, m, s, ms in CUE.findall(lines[1]))
        text.append("\n".join(lines[2:]))
    return np.array(times), text


def mel_power(x, sr):
    # Use short windows so the score does not require sample/phase alignment.
    n = round(0.025 * sr)
    hop = round(0.010 * sr)
    b = 1 << (n - 1).bit_length()
    frames = np.lib.stride_tricks.sliding_window_view(x, n)[::hop]
    power = abs(np.fft.rfft(frames * np.hanning(n), n=b)) ** 2
    mel = lambda f: 2595 * np.log10(1 + f / 700)
    bands = np.linspace(mel(80), mel(min(8000, sr / 2)), 42)
    freq = mel(np.fft.rfftfreq(b, 1 / sr))[:, None]
    filters = np.maximum(0, np.minimum(freq - bands[:-2], bands[2:] - freq) / (bands[1] - bands[0]))
    return power @ filters


def compare(reference, changed):
    sr, x = pcm(reference)
    other_sr, y = pcm(changed)
    if other_sr != sr:
        raise ValueError("sample rate changed")
    t, text = cues(reference)
    u, other_text = cues(changed)
    a, b = mel_power(x, sr), mel_power(y, sr)
    n = min(len(a), len(b))
    a, b = a[:n], b[:n]
    # Reference-active frames only; floor quiet bins at -60 dB relative to peak.
    peak = a.max()
    energy = a.sum(axis=1)
    active = energy > 1e-4 * energy.max()
    if not active.any():
        raise ValueError(f"silent reference: {reference}")
    floor = peak * 1e-6
    error = 10 * np.log10(np.maximum(a[active], floor) / np.maximum(b[active], floor))
    return {"refSeconds": round(len(x) / sr, 4), "durationDeltaMs": round((len(y) - len(x)) * 1000 / sr, 2),
            "subtitleTextEqual": text == other_text,
            "maxCueDeltaMs": int(np.max(abs(t - u))) if text == other_text and len(t) else None,
            "melRMSEdB": round(float(np.sqrt(np.mean(error ** 2))), 3)}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", type=Path, required=True)
    parser.add_argument("--site", type=Path, default=ROOT / "dist")
    parser.add_argument("--output", type=Path, required=True, help="new directory for results and WAVs")
    parser.add_argument("--jobs", type=int, default=1)
    parser.add_argument("--cases", default="", help="comma-separated case IDs (default: all)")
    parser.add_argument("--zstd-level", type=int, default=15)
    args = parser.parse_args()
    if args.jobs < 1:
        parser.error("--jobs must be positive")
    args.binary, args.site, args.output = [p.resolve() for p in (args.binary, args.site, args.output)]
    if not 1 <= args.zstd_level <= 19:
        parser.error("--zstd-level must be between 1 and 19")
    args.zstd = ["zstd", "-q", "-T1", f"-{args.zstd_level}", "--long=27", "-c"]
    if not (args.site / "aoikane.wasm").is_file():
        parser.error("--site must point to a built Web distribution")
    for path in [*(Path("assets") / name for name in MODELS), *DICTIONARIES]:
        if not (args.site / path).is_file() or not filecmp.cmp(args.site / path, ROOT / path, shallow=False):
            parser.error(f"rebuild --site: stale or missing {path}")
    cases = json.loads((ROOT / "benchmarks/cases.json").read_text())
    if args.cases:
        chosen = set(args.cases.split(","))
        if chosen - {case["id"] for case in cases}:
            parser.error("unknown benchmark case")
        cases = [case for case in cases if case["id"] in chosen]
    args.output.mkdir(parents=True)
    dictionaries = {}
    for path in DICTIONARIES:
        target = args.output / f"{path.parent.name}-sys.fea.zst"
        compress_dictionary(args.site / path, target, args.zstd)
        dictionaries[path] = target
    results, reference = {}, {}
    with ThreadPoolExecutor(args.jobs) as pool:
        for mode in MODES:
            runtime, compact = setup(args, mode)
            jobs = {(case["id"], voice): pool.submit(synth, args, mode, runtime, case, voice)
                    for case in cases for voice in ("aoi", "akane")}
            rows = []
            for (case, voice), job in jobs.items():
                path = job.result()
                key = (case, voice)
                if mode == "fp32":
                    reference[key] = path
                rows.append({"case": case, "voice": voice, **compare(reference[key], path)})
            results[mode] = {"cases": rows}
            print(json.dumps({"mode": mode, "cases": rows}), flush=True)
    for mode in MODES:
        results[mode]["bundleBytes"] = bundle_size(args, args.output / "size-fixtures" / mode, dictionaries)
    report = {"zstd": args.zstd, "dictionariesBytes": {p.parent.name: v.stat().st_size for p, v in dictionaries.items()},
              "models": MODELS, "modes": results}
    (args.output / "results.json").write_text(json.dumps(report, indent=2) + "\n")


if __name__ == "__main__":
    main()
