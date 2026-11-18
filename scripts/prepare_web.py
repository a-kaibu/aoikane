#!/usr/bin/env python3
"""Prepare a compact Aoikane Web release for ordinary static hosting."""

import hashlib
import json
from pathlib import Path
import tempfile


def digest(data):
    return hashlib.sha256(data).hexdigest()


def restore_weights(data):
    import numpy as np
    import onnx

    model = onnx.load_model_from_string(data)
    for tensor in model.graph.initializer:
        if tensor.data_type == onnx.TensorProto.FLOAT16:
            tensor.raw_data = np.frombuffer(tensor.raw_data, dtype="<f2").astype("<f4").tobytes()
            tensor.data_type = onnx.TensorProto.FLOAT
    return model.SerializeToString()


def decompress(data):
    from openzl import ext

    outputs = ext.DCtx().decompress(data)
    if len(outputs) != 1 or outputs[0].type != ext.Type.Serial:
        raise ValueError("expected one OpenZL byte stream")
    return outputs[0].content.as_nparray().tobytes()


def prepare(root):
    root = Path(root).resolve()
    plan = json.loads((root / "openzl-assets.json").read_text(encoding="utf-8"))
    for entry in plan:
        dest = (root / entry["path"]).resolve()
        packed = dest.with_suffix(dest.suffix + ".zl").resolve()
        if not dest.is_relative_to(root / "assets") or not packed.is_relative_to(root / "assets"):
            raise ValueError("invalid compact asset path")
        expected = entry["sha256"]
        if dest.is_file() and digest(dest.read_bytes()) == expected:
            continue
        compressed = packed.read_bytes()
        if digest(compressed) != entry["packedSHA256"]:
            raise ValueError(f"compressed asset checksum mismatch: {packed.name}")
        data = decompress(compressed)
        if entry["fp16"]:
            data = restore_weights(data)
        if digest(data) != expected:
            raise ValueError(f"restored asset checksum mismatch: {dest.name}")
        with tempfile.NamedTemporaryFile(dir=dest.parent, delete=False) as tmp:
            tmp_path = Path(tmp.name)
        try:
            tmp_path.write_bytes(data)
            tmp_path.replace(dest)
        finally:
            tmp_path.unlink(missing_ok=True)


if __name__ == "__main__":
    try:
        prepare(Path(__file__).resolve().parent)
    except ImportError as error:
        raise SystemExit(f"Missing dependency: {error}\nRun: python3 -m pip install openzl==0.3.0 onnx==1.23.2") from None
    print("Models prepared. Start the site with: python3 -m http.server 8000")
