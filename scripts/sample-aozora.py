#!/usr/bin/env python3
"""Bounded, reproducible public-domain Aozora samples for scheduled regression tests."""
import argparse
import csv
from datetime import datetime, timezone
import hashlib
from html.parser import HTMLParser
from io import BytesIO, StringIO
import json
from pathlib import Path
import random
import re
import time
from urllib.parse import urlparse
from urllib.request import Request, urlopen
from zipfile import ZipFile

INDEX = "https://www.aozora.gr.jp/index_pages/list_person_all_extended_utf8.zip"
AGENT = "aoikane-regression/1.0 (+https://github.com/a-kaibu/aoikane)"


def download(url, limit):
    if urlparse(url).hostname not in ("www.aozora.gr.jp", "aozora.gr.jp"):
        raise ValueError("only Aozora URLs are allowed")
    with urlopen(Request(url.replace("http:", "https:", 1), headers={"User-Agent": AGENT}), timeout=30) as response:
        if urlparse(response.url).hostname not in ("www.aozora.gr.jp", "aozora.gr.jp"):
            raise ValueError("unexpected download redirect")
        data = response.read(limit + 1)
    if len(data) > limit:
        raise ValueError("download exceeds size limit")
    return data


class MainText(HTMLParser):
    def __init__(self):
        super().__init__()
        self.depth = 0
        self.parts, self.rubies = [], []
        self.ruby = None
        self.reading, self.skip = False, 0

    def handle_starttag(self, tag, attrs):
        attrs = dict(attrs)
        if self.skip:
            if tag not in ("br", "img", "hr", "meta", "link", "input"):
                self.skip += 1
            return
        if tag == "div":
            if self.depth or attrs.get("class") == "main_text":
                self.depth += 1
        if not self.depth:
            return
        if tag in ("rp", "script", "style") or tag == "span" and attrs.get("class") == "notes":
            self.skip += 1
        elif tag == "ruby":
            self.ruby = {"base": "", "reading": ""}
        elif tag == "rt":
            self.reading = True
        elif tag == "br":
            self.parts.append("\n")
        elif tag == "img" and attrs.get("class") == "gaiji":
            match = re.search(r"第[34]水準([12])-(\d+)-(\d+)", attrs.get("alt", ""))
            char = "〓"
            if match:
                plane, row, cell = map(int, match.groups())
                try:
                    char = ((b"\x1b$(Q" if plane == 1 else b"\x1b$(P") + bytes([row+32, cell+32]) + b"\x1b(B").decode("iso2022_jp_2004")
                except (ValueError, UnicodeError):
                    pass
            self.parts.append(char)
            if self.ruby is not None and not self.reading:
                self.ruby["base"] += char

    def handle_endtag(self, tag):
        if self.skip:
            self.skip -= 1
            return
        if not self.depth:
            return
        if tag == "div":
            self.depth -= 1
        elif tag == "rt":
            self.reading = False
        elif tag == "ruby" and self.ruby is not None:
            self.rubies.append(self.ruby)
            self.ruby = None

    def handle_data(self, data):
        if not self.depth or self.skip:
            return
        if self.reading:
            if self.ruby is not None:
                self.ruby["reading"] += data
        else:
            self.parts.append(data)
            if self.ruby is not None:
                self.ruby["base"] += data


def extract(data, encoding):
    parser = MainText()
    parser.feed(data.decode(encoding or "cp932"))
    text = "".join(parser.parts)
    excerpts = [s.strip() for s in re.split(r"(?<=[。！？])|\n+", text)]
    excerpts = [s for s in excerpts if 20 <= len(s) <= 160 and re.search(r"[ぁ-んァ-ヶ一-龯]", s)]
    rubies = []
    for item in parser.rubies:
        base, reading = item["base"].strip(), item["reading"].strip()
        if 1 <= len(base) <= 20 and re.search(r"[㐀-龯]", base) and re.fullmatch(r"[ぁ-ゖァ-ヺー]{1,40}", reading):
            context = next((s for s in excerpts if base in s), base)
            rubies.append({"base": base, "reading": reading, "context": context})
    return excerpts, rubies


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--seed", default=datetime.now(timezone.utc).date().isoformat())
    parser.add_argument("--books", type=int, default=3)
    parser.add_argument("--samples", type=int, default=3)
    parser.add_argument("--output", required=True)
    args = parser.parse_args()
    if not 1 <= args.books <= 5 or not 1 <= args.samples <= 5:
        parser.error("books and samples must be between 1 and 5")
    rng = random.Random(args.seed)
    archive = download(INDEX, 8 << 20)
    with ZipFile(BytesIO(archive)) as bundle:
        entry = next(item for item in bundle.infolist() if item.filename.endswith(".csv"))
        if entry.file_size > 64 << 20:
            raise ValueError("index exceeds size limit")
        rows = list(csv.DictReader(StringIO(bundle.read(entry).decode("utf-8-sig"))))
    # Authors whose works were already public-domain in Japan before 1996.
    authors = {"000081", "000148", "000879", "000129"}
    candidates = {row["作品ID"]: row for row in rows if row["人物ID"] in authors and row["役割フラグ"] == "著者" and row["作品著作権フラグ"] == "なし" and row["人物著作権フラグ"] == "なし" and row["XHTML/HTMLファイルURL"]}
    candidates = sorted(candidates.values(), key=lambda row: row["作品ID"])
    rng.shuffle(candidates)
    books, errors = [], []
    for row in candidates[:args.books*3]:
        time.sleep(1)  # Polite bounded scheduled access, not a site-wide crawler.
        try:
            data = download(row["XHTML/HTMLファイルURL"], 2 << 20)
            encoding = row["XHTML/HTMLファイル符号化方式"].lower()
            excerpts, rubies = extract(data, "cp932" if encoding in ("shiftjis", "shift_jis") else encoding)
            if not excerpts or not rubies:
                continue
            books.append({"id": row["作品ID"], "title": row["作品名"], "author": row["姓"]+row["名"], "url": row["XHTML/HTMLファイルURL"], "sha256": hashlib.sha256(data).hexdigest(), "samples": rng.sample(excerpts, min(args.samples, len(excerpts))), "rubies": rng.sample(rubies, min(12, len(rubies)))})
            if len(books) == args.books:
                break
        except Exception as error:
            errors.append({"id": row["作品ID"], "error": str(error)})
    if len(books) != args.books:
        raise RuntimeError(f"not enough usable books: {errors}")
    output = Path(args.output)
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(json.dumps({"seed": args.seed, "indexSHA256": hashlib.sha256(archive).hexdigest(), "books": books, "downloadErrors": errors}, ensure_ascii=False, indent=2))
    print(json.dumps({"seed": args.seed, "books": [{"id": b["id"], "title": b["title"]} for b in books]}, ensure_ascii=False))


if __name__ == "__main__":
    main()
