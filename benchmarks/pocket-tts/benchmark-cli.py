#!/usr/bin/env python3
"""Measure the Pocket TTS Go probe, including process startup and WAV output."""

import argparse
import json
from pathlib import Path
import statistics
import subprocess
import tempfile
import time
import wave


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", required=True, type=Path)
    parser.add_argument("--assets", required=True, type=Path)
    parser.add_argument("--workers", type=int, default=1)
    parser.add_argument("--runs", type=int, default=3)
    parser.add_argument("--sample-output", type=Path)
    args = parser.parse_args()
    if args.workers < 1 or args.runs < 1:
        parser.error("--workers and --runs must be positive")

    text = Path(__file__).with_name("sample.txt").read_text(encoding="utf-8").strip()
    runs = []
    with tempfile.TemporaryDirectory() as temp:
        wav = Path(temp) / "speech.wav"
        command = [
            str(args.binary.resolve()), "--assets", str(args.assets.resolve()),
            "--workers", str(args.workers), "--text", text, "--out", str(wav),
        ]
        for _ in range(args.runs):
            start = time.perf_counter()
            subprocess.run(command, check=True, stdout=subprocess.PIPE)
            seconds = time.perf_counter() - start
            with wave.open(str(wav)) as audio:
                if (audio.getframerate(), audio.getnchannels(), audio.getsampwidth()) != (24000, 1, 2):
                    raise ValueError("expected a 24 kHz mono PCM16 WAV")
                duration = audio.getnframes() / audio.getframerate()
                if duration <= 0:
                    raise ValueError("empty audio")
            runs.append({"seconds": seconds, "audioSeconds": duration, "rtf": seconds / duration})
        if args.sample_output:
            args.sample_output.write_bytes(wav.read_bytes())

    print(json.dumps({
        "model": args.assets.name,
        "workers": args.workers,
        "includesStartup": True,
        "medianRTF": statistics.median(run["rtf"] for run in runs),
        "runs": runs,
    }, indent=2))


if __name__ == "__main__":
    main()
