#!/usr/bin/env python3
"""Install the pinned Pocket TTS 6.9M model and Alba voice for Aoikane."""

import argparse
import json
from pathlib import Path
import shutil
import struct
import subprocess
import tempfile

REPO = "kyutai/pocket-tts-without-voice-cloning"
MODEL_PATH = "languages/english_7m_refl_2026-10"
WEIGHTS_REV = "a7056f505a4a8ca72d8b5fb1440071d6f1a35739"
VOICE_REV = "08ffc58775e62aad9409511b9591c9af626270f3"


def download(directory):
    if not shutil.which("hf"):
        raise SystemExit("Install the Hugging Face CLI (hf), or supply --from-dir.")
    for files, revision in [
        (["model.safetensors"], WEIGHTS_REV),
        (["tokenizer.model", "embeddings/alba.safetensors"], VOICE_REV),
    ]:
        subprocess.run([
            "hf", "download", REPO, *(f"{MODEL_PATH}/{name}" for name in files),
            "--revision", revision, "--local-dir", str(directory),
        ], check=True)
    return directory / MODEL_PATH


def inference_weights(source):
    """Drop the reference-audio encoder from the official checkpoint."""
    data = source.read_bytes()
    if len(data) < 8:
        raise ValueError("truncated SafeTensors file")
    (size,) = struct.unpack_from("<Q", data)
    if size > len(data) - 8:
        raise ValueError("invalid SafeTensors header length")
    header = json.loads(data[8:8 + size])
    tensors = {}
    if "__metadata__" in header:
        tensors["__metadata__"] = header.pop("__metadata__")
    weights = bytearray()
    for name, tensor in header.items():
        if name.startswith(("mimi.encoder.", "mimi.encoder_transformer.", "mimi.downsample.")) or name == "flow_lm.speaker_proj_weight":
            continue
        begin, end = tensor["data_offsets"]
        if not 0 <= begin <= end <= len(data) - 8 - size:
            raise ValueError(f"invalid SafeTensors offsets for {name}")
        tensors[name] = {**tensor, "data_offsets": [len(weights), len(weights) + end - begin]}
        weights.extend(data[8 + size + begin:8 + size + end])
    encoded = json.dumps(tensors, separators=(",", ":")).encode("utf-8")
    encoded += b" " * (-len(encoded) % 8)
    return struct.pack("<Q", len(encoded)) + encoded + weights


def install(source, output):
    tokenizer = (source / "tokenizer.model").read_bytes()
    voice = (source / "embeddings/alba.safetensors").read_bytes()
    weights = inference_weights(source / "model.safetensors")
    output.mkdir(parents=True, exist_ok=True)
    (output / "voices").mkdir(exist_ok=True)
    # Publish the model last so a default CLI invocation can detect a new installation.
    for name, data in [("tokenizer.model", tokenizer), ("voices/alba.safetensors", voice), ("model.safetensors", weights)]:
        target = output / name
        with tempfile.NamedTemporaryFile(dir=target.parent, delete=False) as stream:
            temporary = Path(stream.name)
            stream.write(data)
        try:
            temporary.replace(target)
        finally:
            temporary.unlink(missing_ok=True)
    print(f"Installed English model and Alba voice in {output} ({len(weights):,} model bytes)")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", type=Path, default=Path("assets/english"))
    parser.add_argument("--from-dir", type=Path, help="use existing official model, tokenizer and embeddings/alba.safetensors")
    args = parser.parse_args()
    if args.from_dir:
        install(args.from_dir, args.output)
    else:
        with tempfile.TemporaryDirectory() as directory:
            install(download(Path(directory)), args.output)
