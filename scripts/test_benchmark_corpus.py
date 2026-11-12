import json
from pathlib import Path
import unittest


class CorpusTest(unittest.TestCase):
    def test_full_texts_and_sources(self):
        root = Path(__file__).resolve().parent.parent / "benchmarks"
        cases = json.loads((root / "cases.json").read_text())
        by_id = {case["id"]: case for case in cases}
        self.assertEqual(len(cases), len(by_id))
        law = (root / by_id["constitution-preamble"]["file"]).read_text().strip()
        poem = (root / by_id["ame-ni-mo-makezu"]["file"]).read_text().strip()
        self.assertEqual(len(law.split("\n\n")), 4)
        self.assertTrue(law.startswith("日本国民は、正当に選挙された"))
        self.assertTrue(law.endswith("目的を達成することを誓ふ。"))
        self.assertEqual(len(poem.splitlines()), 37)
        self.assertIn("野原ノ松ノ林ノ䕃ノ", poem)
        self.assertIn("行ッテソノ稲ノ朿ヲ負ヒ", poem)
        self.assertTrue(poem.endswith("南無安立行菩薩"))
        for key in ("constitution-preamble", "ame-ni-mo-makezu"):
            self.assertTrue(by_id[key]["source"].startswith("https://"))


if __name__ == "__main__":
    unittest.main()
