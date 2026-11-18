import importlib.util
import json
from pathlib import Path
import shutil
import subprocess
import tarfile
import tempfile
import unittest
import zipfile

spec = importlib.util.spec_from_file_location("package_web", Path(__file__).with_name("package-web.py"))
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class PackageTest(unittest.TestCase):
    def test_runtime_and_models_are_included(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            site = root / "site"
            site.mkdir()
            for name in ("index.html", "app.js", "style.css", "wasm_exec.js", "renderer-worker.js"):
                (site / name).write_text("test")
            (site / "aoikane.wasm").write_bytes(b"\0asm\x01\0\0\0")
            (site / "assets").mkdir()
            (site / "assets/model.bin").write_bytes(b"model")
            (site / "manifest.json").write_text(json.dumps({"files": [{"path": "assets/model.bin", "size": 5}]}))
            archive, wasm, checksums = module.package(site, root / "out", "v1.2.3")
            with zipfile.ZipFile(archive) as bundle:
                self.assertIsNone(bundle.testzip())
                self.assertIn("aoikane_web_1.2.3/assets/model.bin", bundle.namelist())
                self.assertIn("aoikane_web_1.2.3/renderer-worker.js", bundle.namelist())
                self.assertIn(b"python3 -m http.server 8000", bundle.read("aoikane_web_1.2.3/README.txt"))
            self.assertEqual(wasm.read_bytes(), (site / "aoikane.wasm").read_bytes())
            self.assertIn(module.digest(archive), checksums.read_text())
            with self.assertRaises(FileExistsError):
                module.package(site, root / "out", "v1.2.3")
            (site / "assets/model.bin").unlink()
            with self.assertRaises(ValueError):
                module.package(site, root / "missing", "v1.2.3")

    def test_invalid_version(self):
        with self.assertRaises(ValueError):
            module.package("dist", "/tmp/unused", "../../bad")

    @unittest.skipUnless(shutil.which("zstd") and importlib.util.find_spec("openzl")
                         and importlib.util.find_spec("onnx"), "compact packaging dependencies unavailable")
    def test_compact_round_trip(self):
        import numpy as np
        import onnx
        from compact_web import MODELS, TRIES
        from prepare_web import prepare

        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            site = root / "site"
            site.mkdir()
            for name in ("index.html", "app.js", "style.css", "wasm_exec.js"):
                (site / name).write_text("test")
            (site / "aoikane.wasm").write_bytes(b"\0asm\x01\0\0\0")
            x = np.linspace(-1, 1, 1024, dtype=np.float32).reshape(128, 8)
            tensor = onnx.numpy_helper.from_array(x, name="weights")
            graph = onnx.helper.make_graph([], "fixture", [], [], initializer=[tensor])
            model = onnx.helper.make_model(graph).SerializeToString()
            trie = b"\xffDar" + b"\x00" * 33 + bytes(range(32)) * 128
            files = []
            for name in sorted(MODELS | TRIES):
                target = site / name
                target.parent.mkdir(parents=True, exist_ok=True)
                target.write_bytes(model if name in MODELS else trie)
                files.append({"path": name, "size": target.stat().st_size, "sha256": module.digest(target)})
            (site / "manifest.json").write_text(json.dumps({"files": files}))
            archive, wasm, checksums = module.package(site, root / "out", "v1.2.3", compact=True)
            self.assertTrue(archive.name.endswith(".tar.zst"))
            self.assertIn(module.digest(archive), checksums.read_text())
            self.assertEqual(wasm.read_bytes(), (site / "aoikane.wasm").read_bytes())
            with subprocess.Popen(["zstd", "-dc", archive], stdout=subprocess.PIPE) as proc:
                with tarfile.open(fileobj=proc.stdout, mode="r|") as bundle:
                    bundle.extractall(root, filter="data")
                self.assertEqual(proc.wait(), 0)
            installed = root / "aoikane_web_1.2.3"
            self.assertTrue((installed / "prepare_web.py").is_file())
            for name in MODELS | TRIES:
                self.assertFalse((installed / name).exists())
                self.assertTrue((installed / (name + ".zl")).is_file())
            corrupted = installed / (sorted(MODELS)[0] + ".zl")
            packed = corrupted.read_bytes()
            corrupted.write_bytes(b"broken")
            with self.assertRaisesRegex(ValueError, "compressed asset checksum mismatch"):
                prepare(installed)
            corrupted.write_bytes(packed)
            prepare(installed)
            prepare(installed)  # Idempotent.
            manifest = json.loads((installed / "manifest.json").read_text())
            for entry in manifest["files"]:
                path = installed / entry["path"]
                self.assertEqual(path.stat().st_size, entry["size"])
                self.assertEqual(module.digest(path), entry["sha256"])
                if entry["path"] in TRIES:
                    self.assertEqual(path.read_bytes(), trie)
                else:
                    array = onnx.numpy_helper.to_array(onnx.load(path).graph.initializer[0])
                    np.testing.assert_array_equal(array, x.astype(np.float16).astype(np.float32))


if __name__ == "__main__":
    unittest.main()
