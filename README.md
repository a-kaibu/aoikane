# Aoikane

文章から音声ファイルと字幕を作成するCLI / HTTP API。

## インストール

Goをインストールして、以下を実行する。

```sh
go install github.com/a-kaibu/aoikane/cmd/aoikane@latest
```

取得エラーになる場合は、[配布ファイル](https://github.com/a-kaibu/aoikane/releases/latest)をダウンロードして利用できる。
CLIの配布ファイルはFP16の重みを含む圧縮モデルを内蔵し、初回の合成時に自動でキャッシュへ展開する。
配布版の利用にPythonやOpenZLの別途インストールは不要。
`go install` でソースからビルドした場合は、従来のFP32モデルを内蔵する。

`aoikane`が見つからない場合は、実行ファイルの保存先を`PATH`に追加する。
通常は`go env GOPATH`で表示される場所の`bin`。

## 使い方

```sh
aoikane 台本.txt --voice aoi
aoikane 台本.txt --voice akane --srt --vtt
aoikane 台本.txt --format mp3
aoikane --text "こんにちは。" --output speech.wav
aoikane --version
aoikane --help
```

入力ファイルと標準入力はUTF-8。入力を省略すると標準入力を読む。
`--voice aoi` / `--voice akane`で音声を切り替える。
`-o` / `--output`を省略すると入力名から音声の名前を決める。
`--srt` / `--vtt`も音声と同じ名前が既定で、明示的なパスも指定できる。
`--rate`、`--pitch`、`--volume`、`--emphasis`で速度・高さ・音量・抑揚を調整できる。

## HTTP API

```sh
aoikane serve --listen 127.0.0.1:8080
```

`GET /healthz`、`GET /v1/voices`、`POST /v1/synthesize`、`GET /openapi.json`。
任意で`--token` / `AOIKANE_TOKEN`を設定できる。

## ブラウザで使う

同じエンジンをWebAssemblyにして、GitHub Pagesなどの静的ホスティングで合成できる。
入力したテキストは外部へ送信されず、合成はブラウザ内で完結する。

ダウンロードは[GitHub Releases](https://github.com/a-kaibu/aoikane/releases/latest)の
`aoikane_web_<version>.tar.zst`（実行用一式）を利用する。
配布ファイルの音響モデルと埋め込みはFP16で保存し、辞書の索引とともにOpenZLで圧縮している。
展開したフォルダーで、初回だけモデルを準備してからHTTPサーバーを起動する。

```sh
version=0.0.0  # ダウンロードしたファイルのバージョンに置換
tar --zstd -xf "aoikane_web_${version}.tar.zst"
cd "aoikane_web_${version}"
python3 -m pip install openzl==0.3.0 onnx==1.23.2
python3 prepare_web.py
python3 -m http.server 8000
# http://localhost:8000/ を開く
```

`index.html` のダブルクリックでは動かない。
WASM本体だけなら `aoikane_<version>.wasm` も添付しているが、実行にはGoランタイムとモデル等が別途必要になる。
旧版のZIPは、展開後にモデルを準備せずHTTPサーバーを起動できる。

ソースからビルドする場合は、モデルの準備は不要。

```sh
./scripts/build-web.sh
python3 -m http.server 8000 -d dist
# http://localhost:8000/ を開く
```

`dist/`にWASM、HTML、モデル一式が出力される。
GitHub Pages版では、初回の合成時に声ごとの圧縮済みモデル（葵は約130 MB、茜はさらに約50 MB）を読み込み、ブラウザー内で自動展開する。
モデルはSHA-256で検証してCache Storageに明示保存し、リロード後も再ダウンロードせずに利用する。
アプリの更新だけでは再取得せず、内容が変わったモデルだけを更新する。
ブラウザーのデータ削除・容量制限・プライベートモードなどで保存を利用できない場合は再取得することがある。

WebAssembly SIMDを使うため、ビルドにはGo 1.27以降を推奨する。
Go 1.26でもビルドできるが、合成は大幅に遅くなる。
WASMターゲットとRelaxed SIMDに対応したClangがあれば、小さな専用カーネルもビルドする。
専用カーネルはGoのメモリを共有し、テンソルをコピーせずに演算する。
Clangがない場合やRelaxed SIMD非対応のブラウザでは、Goの実装を利用する。

CPUとメモリに余裕がある環境では、Web Workerで文と波形チャンクを並列処理する。
最大2つの音響モデル用ワーカーと6つの波形生成用ワーカーがモデルを個別に保持するため、
シングルスレッド版より数百MB〜1GB以上の追加メモリを使うことがある。デスクトップ環境を推奨する。
低メモリ・少コア環境ではワーカー数を制限し、ワーカーの初期化・実行に失敗した場合も
メインスレッドのエンジンで合成を続ける。テキストやモデルを外部サーバーへ送ることはない。

GitHub Pagesへ公開する場合は、リポジトリのSettings → PagesでSourceを
「GitHub Actions」に設定してから`main`へpushする。
`.github/workflows/pages.yml`がFP16とOpenZLを使用する静的サイトをビルドしてデプロイする。

ベンチマークは[benchmarks/README.md](benchmarks/README.md)を参照。
従来の2文に加え、日本国憲法前文と「雨ニモマケズ」の全文をブラウザー・CLIの両方で測定できる。
旧仮名のカタカナ「ヲ・ヰ・ヱ」が読みに残る場合は、発音時だけ「オ・イ・エ」として扱う。
