# 合成ベンチマーク

`cases.json` の全ケースを葵・茜で測定する。
日本国憲法前文はe-Gov法令検索の旧仮名表記を維持した全文4段落である。
「雨ニモマケズ」は青空文庫の本文全体を使用し、末尾の題目7行も含む。
入力者注・ルビは除き、外字画像のJIS X 0213第4水準2-86-78をUnicodeの「䕃」に置き換えた。
いずれもパブリックドメインの文章であり、出典URLは `cases.json` に記録している。

```sh
# Playwrightを利用できるNode.js環境で実行する
node scripts/benchmark-web.cjs http://localhost:8000/

# 長文だけ、葵だけを各3回測定する
BENCH_CASES=constitution-preamble,ame-ni-mo-makezu BENCH_VOICES=aoi node scripts/benchmark-web.cjs http://localhost:8000/

# 同じ文章をCLIでも測定する（プロセス起動・モデル読込を含む）
python3 scripts/benchmark-cli.py --binary ./aoikane
```

既定は各3回の中央値で、`BENCH_RUNS` で回数を指定できる。
モデル読込と最初のウォームアップ合成は別記し、時間・文字数・音声長・RTF・WAVのSHA-256を出力する。
RTFは合成時間を音声長で割った値で、1未満なら実時間より速い。
`PLAYWRIGHT_MODULE` と `BROWSER_CHROMIUM_PATH` でPlaywrightの場所・Chromiumの実行ファイルも指定できる。

## 青空文庫のランダム回帰テスト

`.github/workflows/aozora.yml` は毎週・手動実行に対応し、著作権保護期間が終了している作品から3作品を抽出する。
既定の対象著者は宮沢賢治・夏目漱石・芥川龍之介・森鴎外で、一覧の著作権フラグも検査する。
各作品の短い抜粋3件を葵・茜で読み・発音列へ変換し、最初の抜粋は実際の音声合成も行う。
変換・合成の失敗はGHAを失敗させる。

ルビは正解候補として音素列を比較する。
長音の「オウ／オー」「エイ／エー」と旧かなの母音を正規化するが、比較対象は単語単独の読みである。
当て字・固有名詞・文脈依存・歴史的な表記があるため、ルビとの差分だけではエンジンの誤りと断定しない。
ルビ比較は標準語の葵で行い、アクセントや全文の読みの正しさを保証するものではない。
読みの差分は出典・原文の周辺文脈とともに確認用レポートへ記録し、既知の正しい読みを固定した既存テストとは分ける。
任意のLLMに自己採点させたり、秘密鍵や有料APIを使ったりはしない。

seed・作品ID・ソースのSHA-256・抽出済み本文・レポートをGHA artifactに保存する。
再現にはartifactの `corpus.json` をそのまま使う（一覧が更新されるため、seedだけでなく入力も固定する）。

```sh
python3 scripts/sample-aozora.py --seed 2026-11-13 --output /tmp/opencode/aozora/corpus.json
AOIKANE_AOZORA_CORPUS=/tmp/opencode/aozora/corpus.json AOIKANE_AOZORA_REPORT=/tmp/opencode/aozora/report.json AOIKANE_AOZORA_SYNTH=1 go test ./internal/portable -run '^TestAozoraCorpus$' -count=1 -v
```
