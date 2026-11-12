import importlib.util
from pathlib import Path
import unittest

spec = importlib.util.spec_from_file_location("sample_aozora", Path(__file__).with_name("sample-aozora.py"))
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class ExtractTest(unittest.TestCase):
    def test_ruby_notes_and_gaiji(self):
        html = '''<html><h1>title</h1><div class="main_text"><ruby><rb>日本</rb><rp>（</rp><rt>にほん</rt><rp>）</rp></ruby>の文章を使って、短いテストの本文を作ります。<br />野原ノ松ノ林ノ<img class="gaiji" alt="第4水準2-86-78" />ノ<span class="notes">［入力者注］</span></div><div>footer</div></html>'''
        samples, rubies = module.extract(html.encode("utf-8"), "utf-8")
        self.assertEqual(len(samples), 1)
        self.assertTrue(samples[0].startswith("日本の文章"))
        self.assertNotIn("にほん", samples[0])
        self.assertEqual(rubies[0]["base"], "日本")
        self.assertEqual(rubies[0]["reading"], "にほん")
        parser = module.MainText()
        parser.feed(html)
        text = "".join(parser.parts)
        self.assertIn("林ノ䕃ノ", text)
        self.assertNotIn("入力者注", text)
        self.assertNotIn("footer", text)


if __name__ == "__main__":
    unittest.main()
