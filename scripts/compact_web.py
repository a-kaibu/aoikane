"""Build the OpenZL assets for the downloadable Web bundle."""

import gzip
import json
from pathlib import Path
import shutil
import subprocess
import tarfile
import tempfile

from prepare_web import decompress, digest, restore_weights

MODELS = {"assets/synth/featnet.onnx", "assets/boundmodel/emb.onnx"}
TRIES = {"assets/dict/sys.dic", "assets/dict-kansai/sys.dic"}


def pack_fp16(data):
    import numpy as np
    import onnx

    model = onnx.load_model_from_string(data)
    weights = []
    for tensor in model.graph.initializer:
        if tensor.data_type != onnx.TensorProto.FLOAT or len(tensor.dims) < 2 or len(tensor.raw_data) < 4096:
            continue
        half = np.frombuffer(tensor.raw_data, dtype="<f4").astype("<f2")
        if not np.isfinite(half).all():
            raise ValueError(f"non-finite FP16 weights: {tensor.name}")
        tensor.raw_data = half.tobytes()
        tensor.data_type = onnx.TensorProto.FLOAT16
        weights.append(tensor.raw_data)
    if not weights:
        raise ValueError("model has no FP16 candidates")
    data = model.SerializeToString()
    offset, schema = 0, []
    for weight in weights:
        start = data.index(weight, offset)
        if start > offset:
            schema.append(f": Byte[{start - offset}];")
        schema.append(f": Float16LE[{len(weight) // 2}];")
        offset = start + len(weight)
    if offset < len(data):
        schema.append(f": Byte[{len(data) - offset}];")
    return data, "\n".join(schema)


def compress(data, schema=None):
    from openzl import ext

    compressor = ext.Compressor()
    compressor.set_parameter(ext.CParam.FormatVersion, ext.MAX_FORMAT_VERSION)
    compressor.set_parameter(ext.CParam.CompressionLevel, 12)
    if schema:
        successor = ext.graphs.Compress()(compressor)
        graph = ext.graphs.SDDL(description=schema, successor=successor)(compressor)
    else:
        # zstd.h: ZSTD_c_windowLog=101, ZSTD_c_enableLongDistanceMatching=160.
        graph = ext.graphs.Zstd(compression_level=19, zstd_params={101: 27, 160: 1})(compressor)
    compressor.select_starting_graph(graph)
    context = ext.CCtx()
    context.ref_compressor(compressor)
    packed = context.compress([ext.Input(ext.Type.Serial, data)])
    if decompress(packed) != data:
        raise ValueError("OpenZL round-trip mismatch")
    return packed


def transform(site, stage, files, manifest, *, online=False, pack_remaining=False):
    plan = []
    entries = {entry["path"]: entry for entry in manifest["files"]}
    if not MODELS | TRIES <= entries.keys():
        raise ValueError("compact distribution is missing a model or dictionary in the manifest")
    for path in files:
        if not path.is_file():
            continue
        name = path.relative_to(site).as_posix()
        if name == "manifest.json":
            continue
        dest = stage / name
        dest.parent.mkdir(parents=True, exist_ok=True)
        if online and name in entries and (name.endswith("/sys.fea.gz") or (
            pack_remaining and name.startswith("assets/") and name not in MODELS | TRIES
            and path.stat().st_size >= 256 * 1024
        )):
            data = path.read_bytes()
            uncompressed = gzip.decompress(data) if name.endswith("/sys.fea.gz") else data
            packed = compress(uncompressed)
            if len(packed) < len(data):
                if name.endswith("/sys.fea.gz"):
                    dest = dest.with_suffix("")
                    entries[name]["path"] = dest.relative_to(stage).as_posix()
                packed_path = dest.with_suffix(dest.suffix + ".zl")
                packed_path.write_bytes(packed)
                entries[name].update(size=len(packed), sha256=digest(packed), openzl={
                    "path": packed_path.relative_to(stage).as_posix(), "size": len(uncompressed),
                })
                continue
        if name not in MODELS | TRIES:
            shutil.copyfile(path, dest)
            continue
        data = path.read_bytes()
        fp16 = name in MODELS
        if fp16:
            data, schema = pack_fp16(data)
        else:
            if not data.startswith(b"\xffDar") or len(data) <= 37 or (len(data) - 37) % 4:
                raise ValueError(f"invalid dictionary trie: {name}")
            schema = f": Byte[37]; : UInt32LE[{(len(data) - 37) // 4}];"
        packed = compress(data, schema)
        packed_path = dest.with_suffix(dest.suffix + ".zl")
        packed_path.write_bytes(packed)
        if online:
            entries[name].update(size=len(packed), sha256=digest(packed), openzl={
                "path": packed_path.relative_to(stage).as_posix(), "size": len(data),
            })
            plan.append({"path": name})
            continue
        restored = restore_weights(data) if fp16 else data
        sha = digest(restored)
        plan.append({"path": name, "packedSHA256": digest(packed), "sha256": sha, "fp16": fp16})
        entries[name].update(size=len(restored), sha256=sha)
    if {entry["path"] for entry in plan} != MODELS | TRIES:
        raise ValueError("compact distribution is missing a model or dictionary")
    (stage / "manifest.json").write_text(json.dumps(manifest, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    if not online:
        (stage / "openzl-assets.json").write_text(json.dumps(plan, indent=2) + "\n", encoding="utf-8")
        shutil.copyfile(Path(__file__).with_name("prepare_web.py"), stage / "prepare_web.py")


def write_zstd_tar(stage, prefix, archive):
    temp = archive.with_suffix(".zst.tmp")
    try:
        with temp.open("wb") as output, subprocess.Popen(
            ["zstd", "-q", "-19", "--long=27", "-T2", "-c"], stdin=subprocess.PIPE, stdout=output
        ) as proc:
            def normalize(info):
                info.uid = info.gid = info.mtime = 0
                info.uname = info.gname = ""
                info.mode = 0o755 if info.isdir() else 0o644
                return info

            try:
                with tarfile.open(fileobj=proc.stdin, mode="w|") as tar:
                    tar.add(stage, arcname=prefix, filter=normalize)
            finally:
                proc.stdin.close()
            if proc.wait():
                raise RuntimeError("zstd compression failed")
        temp.replace(archive)
    finally:
        temp.unlink(missing_ok=True)


def write_archive(site, files, manifest, archive, prefix, readme, licenses):
    steps = """After extracting the archive, run the following from this directory:

    python3 -m pip install openzl==0.3.0 onnx==1.23.2
    python3 prepare_web.py

展開後、初回だけ上記のコマンドでモデルを準備してください。

"""
    with tempfile.TemporaryDirectory(dir=archive.parent) as directory:
        stage = Path(directory) / prefix
        stage.mkdir()
        transform(site, stage, files, manifest)
        for path, name in licenses:
            dest = stage / name
            dest.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(path, dest)
        (stage / "README.txt").write_text(steps + readme, encoding="utf-8")
        write_zstd_tar(stage, prefix, archive)
