# Aoikane

文章から音声ファイルと字幕を作成するCLI / HTTP API。

## インストール

Goをインストールして、以下を実行する。

```sh
go install github.com/a-kaibu/aoikane/cmd/aoikane@latest
```

取得エラーになる場合は、[配布ファイル](https://github.com/a-kaibu/aoikane/releases/latest)をダウンロードして利用できる。

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

```sh
./scripts/build-web.sh
python3 -m http.server 8000 -d dist
# http://localhost:8000/ を開く
```

`dist/`にWASM、HTML、モデル一式が出力される。
初回の合成時に声ごとのモデル（葵は約268 MB、茜はさらに約97 MB）を読み込む。
2回目以降はブラウザのキャッシュを利用する。

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
「GitHub Actions」に設定してから`main`へpushする。`.github/workflows/pages.yml`が
`dist/`をビルドしてデプロイする。
