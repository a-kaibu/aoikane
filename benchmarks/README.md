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

## モデル重みの保存精度

音響モデルと境界モデルの埋め込みについて、4 KiB以上の多次元FP32重みをFP16、FP8 E4M3FN、INT8、INT4に変換する。
FP8と整数形式はテンソルの第0軸の各要素にFP32のスケールを持たせ、残りの重みとボコーダーは変更しない。
CLIには変換後の重みをFP32に復元して渡すため、低精度演算の速度や実行時メモリは測らない。

```sh
python3 -m pip install numpy onnx ml-dtypes
./scripts/build-web.sh
go build -o ./aoikane ./cmd/aoikane
OPENBLAS_NUM_THREADS=1 python3 scripts/benchmark-precision.py \
  --binary ./aoikane --output /tmp/aoikane-precision
```

短文だけを先に測る場合は `--cases short --zstd-level 9` を指定する。
出力先には測定したケースのWAVとSRT、集計用の `results.json` を保存する。
Web版のサイズは、辞書の `sys.fea.gz` を可逆にzstdへ再圧縮したうえで、モデルと実行用ファイルを `tar.zst` にした場合のバイト数である。
サイズ測定用のファイルをそのままブラウザーで実行することはできない。

音声の比較では、FP32の発話区間について同時刻の対数メルスペクトルの二乗平均平方根誤差（RMSE）を測る。
この値には発音タイミングの差も含まれるため、字幕の時刻差や試聴と合わせて判断する。

| 重みの保存精度 | Web一式（zstd -9） | 短文の対数メルRMSE（葵 / 茜） |
| --- | ---: | ---: |
| FP32 | 251.1 MB | 0 / 0 dB |
| FP16 | 212.9 MB | 0.44 / 0.52 dB |
| FP8 E4M3FN | 192.3 MB | 3.05 / 3.36 dB |
| INT8 | 194.2 MB | 1.85 / 2.54 dB |
| INT4 | 183.2 MB | 14.41 / 15.39 dB |

FP16は辞書の再圧縮と外側のアーカイブをともに `zstd -19 --long=27` にすると、Web一式が193.7 MBになった。
短文と長文を合わせた6ケースで字幕の文面が一致し、時刻の最大差は14 msだった。
FP8、INT8、INT4の音声評価は短文のみで、聴感評価はしていない。

## OpenZLによる可逆圧縮

音響モデル、埋め込み、標準語辞書の主要ファイルを、zstdとOpenZLのバイト列用プロファイルで比較する。
ONNXと固定長の辞書ファイルは、データの型とレイアウトを指定するSDDL2プロファイルでも測る。
辞書の `sys.fea` は同じ展開済みバイト列を両方に渡し、すべての測定で復元後のバイト一致を検査する。

```sh
# OpenZL v0.3.0のzliと、Pythonのonnxパッケージを用意する
python3 scripts/benchmark-openzl.py --zli /path/to/zli --output /tmp/openzl-results.json
```

| ファイル | zstd -19 | OpenZL serial | OpenZL SDDL2 |
| --- | ---: | ---: | ---: |
| 音響モデル `featnet.onnx` | 66.74 MB | 66.87 MB | 60.22 MB |
| 埋め込み `emb.onnx` | 9.25 MB | 9.27 MB | 8.36 MB |
| 標準語辞書 `sys.dic` | 31.41 MB | 35.29 MB | 26.86 MB |
| 標準語辞書 `sys.tok` | 8.89 MB | 9.65 MB | 8.66 MB |
| 標準語辞書 `sys.fea` | 21.11 MB | 26.73 MB | 未測定 |

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
