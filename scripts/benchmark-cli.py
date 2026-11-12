#!/usr/bin/env python3
"""Benchmark the same corpus through the CLI, including per-process startup."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import statistics
import subprocess
import tempfile
import time
import wave

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("--binary", required=True)
parser.add_argument("--voices", default=os.getenv("BENCH_VOICES", "aoi,akane"))
parser.add_argument("--cases", default=os.getenv("BENCH_CASES", ""))
parser.add_argument("--runs", type=int, default=int(os.getenv("BENCH_RUNS", "3")))
args = parser.parse_args()
if not 1 <= args.runs <= 100:
    parser.error("runs must be between 1 and 100")
root = Path(__file__).resolve().parent.parent / "benchmarks"
cases = json.loads((root / "cases.json").read_text())
selected = set(filter(None, args.cases.split(",")))
if selected - {case["id"] for case in cases}:
    parser.error("unknown benchmark case")
cases = [case for case in cases if not selected or case["id"] in selected]
voices = args.voices.split(",")
if not cases or any(voice not in ("aoi", "akane") for voice in voices):
    parser.error("invalid case/voice selection")

with tempfile.TemporaryDirectory() as directory:
    failed = False
    output = Path(directory) / "output.wav"
    def synth(voice, text):
        started = time.perf_counter()
        result = subprocess.run([args.binary, "--voice", voice, "--text", text, "-o", str(output)], capture_output=True)
        elapsed = (time.perf_counter() - started) * 1000
        if result.returncode:
            raise RuntimeError(result.stderr.decode(errors="replace"))
        with wave.open(str(output)) as audio:
            seconds = audio.getnframes() / audio.getframerate()
        data = output.read_bytes()
        return {"ms": elapsed, "hash": hashlib.sha256(data).hexdigest(), "bytes": len(data), "audioSeconds": seconds}

    for voice in voices:
        synth(voice, "こんにちは。今日はいい天気ですね。")
        for case in cases:
            text = (root / case["file"]).read_text().strip() if "file" in case else case["texts"][voice]
            try:
                runs = [synth(voice, text) for _ in range(args.runs)]
            except RuntimeError as error:
                failed = True
                print(json.dumps({"id": case["id"], "title": case["title"], "voice": voice, "error": str(error)}, ensure_ascii=False), flush=True)
                continue
            if len({run["hash"] for run in runs}) != 1:
                raise RuntimeError(f"non-deterministic output: {case['id']}")
            elapsed = statistics.median(run["ms"] for run in runs)
            print(json.dumps({"id": case["id"], "title": case["title"], "voice": voice, "characters": len(text), "textSHA256": hashlib.sha256(text.encode("utf-8")).hexdigest(), "includesStartup": True, "medianMs": elapsed, "audioSeconds": runs[0]["audioSeconds"], "medianRTF": elapsed / 1000 / runs[0]["audioSeconds"], "runs": runs}, ensure_ascii=False), flush=True)
if failed:
    raise SystemExit(1)
