#!/usr/bin/env python3
"""Build online Pages assets or the compact CLI's embedded payload."""

import argparse
import hashlib
import json
from pathlib import Path
import shutil
import tempfile

from compact_web import transform, write_zstd_tar

ROOT = Path(__file__).resolve().parent.parent
CLI_ASSET_DIRS = ("boundmodel", "cabocha", "data", "dict", "dict-kansai", "rules", "synth")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--decoder", type=Path, required=True)
    parser.add_argument("--openzl-license", type=Path, required=True)
    modes = parser.add_mutually_exclusive_group(required=True)
    modes.add_argument("--pages", nargs=2, type=Path, metavar=("SITE", "OUT"))
    modes.add_argument("--cli", type=Path, metavar="OUT")
    args = parser.parse_args()

    if args.pages:
        site, out = (p.resolve() for p in args.pages)
        if out.is_relative_to(site) or site.is_relative_to(out):
            parser.error("use separate input and output directories")
        manifest = json.loads((site / "manifest.json").read_text(encoding="utf-8"))
        out.mkdir(parents=True)
        transform(site, out, sorted(site.rglob("*")), manifest, online=True, pack_remaining=True)
        shutil.copyfile(args.decoder, out / "openzl.wasm")
        shutil.copyfile(args.openzl_license, out / "OpenZL-LICENSE.txt")
        return

    files = sorted(p for name in CLI_ASSET_DIRS for p in (ROOT / "assets" / name).rglob("*") if p.is_file())
    manifest = {"files": [{"path": p.relative_to(ROOT).as_posix()} for p in files]}
    args.cli.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory() as work:
        stage = Path(work) / "payload"
        stage.mkdir()
        transform(ROOT, stage, files, manifest, online=True)
        bundle = args.cli / "models.tar.zst"
        write_zstd_tar(stage, ".", bundle)
        with bundle.open("rb") as stream:
            (args.cli / "models.sha256").write_text(hashlib.file_digest(stream, "sha256").hexdigest() + "\n", encoding="utf-8")
    shutil.copyfile(args.decoder, args.cli / "openzl.wasm")
    shutil.copyfile(args.openzl_license, args.cli / "OpenZL-LICENSE.txt")


if __name__ == "__main__":
    main()
