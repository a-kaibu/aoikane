# Aoikane

文章から音声ファイルと字幕を作成するCLI / HTTP API。

## インストール

Goをインストールして、以下を実行する。

```sh
go install github.com/a-kaibu/aoikane/cmd/aoikane@latest
```

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
