import importlib.util
import json
from pathlib import Path
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


if __name__ == "__main__":
    unittest.main()
