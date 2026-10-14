# Aoikane

文章読み上げをGoだけで実行するCLI / HTTP API。
辞書とモデルを埋め込み、`CGO_ENABLED=0`でビルドできる。元エンジンのDLL、ONNX Runtime、認証用ランタイムは不要。

## ビルドと使い方

```powershell
$env:CGO_ENABLED='0'
go build -trimpath -o aoikane.exe ./cmd/aoikane
.\aoikane.exe 台本.txt --voice aoi
.\aoikane.exe 台本.txt --voice akane --srt --vtt
.\aoikane.exe --text "こんにちは。" --output speech.wav
.\aoikane.exe --version
.\aoikane.exe --help
```

Linuxでは`CGO_ENABLED=0 go build -trimpath -o aoikane ./cmd/aoikane`。
通常ビルドがPure Go版で、ビルドタグの指定は不要。

`synth`は省略可能。入力ファイルと標準入力はUTF-8。入力を省略すると標準入力を読む。
`--voice aoi` / `--voice akane`で音声を切り替える。
`-o` / `--output`を省略すると入力名から音声の名前を決める。
`--srt` / `--vtt`も音声と同じ名前が既定で、明示的なパスも指定できる。
`--format mp3`、`--rate`、`--pitch`、`--volume`、`--emphasis`にも対応する。
モデル・辞書は埋め込みが既定。外部データは`--assets DIR`で指定する。

## HTTP API

```powershell
.\aoikane.exe serve --listen 127.0.0.1:8080
```

`GET /healthz`、`GET /v1/voices`、`POST /v1/synthesize`、`GET /openapi.json`。
任意で`--token` / `AOIKANE_TOKEN`を設定できる。

## 検証と開発状況

```sh
CGO_ENABLED=0 go test ./...
```

原版から採取した全入力の読みと波形参照、検証記録を保持している。
任意入力・全設定・全CPUの完全一致はまだ未達。直近の記録は
[零と単位の再構築](docs/rei-zero-unit-tail-original.md)、
[seed7の全入力比較](docs/numeral-seed7-original.md)、
[seed3の全256音声比較](docs/numeral-seed3-full-wav.md)を参照。
過去の[再構築記録](docs/engine-reconstruction.md)には移植途中のDLL・ONNX Runtime経路も含まれる。

原版DLLの呼び出し・認証・通常SDKとの直接比較は、非公開の別リポジトリ
[aoikane-dll](https://github.com/a-kaibu/aoikane-dll)に分離した。
このリポジトリはその25ad28f時点のPure Go実装を新しい履歴として取り込んでいる。

MP3エンコードは同梱した[shine-mp3](third_party/shine-mp3)を使用する。
変更点とライセンスは同ディレクトリを参照。
