# Pocket TTS英語合成の検証

Kyutaiの学習済み小型モデルとAlbaの事前計算済み声質を使用した。Go試作は `CWBudde/go-pocket-tts` の `4b72d3af8be4676168712fec6077db22d8909fd5` を基に、低ランク埋め込み、GQA、潜在射影、Mimiデコーダーの設定を追加した。GQAは既存の演算を使うため、読み込み時に共有重みと声質キャッシュを各クエリヘッドに展開する。モデルの再学習は不要。 [公式モデル](https://kyutai.org/blog/2026-10-20-pocket-tts-small/) [Go実装](https://github.com/CWBudde/go-pocket-tts/tree/4b72d3af8be4676168712fec6077db22d8909fd5)

## 2026年11月24日の測定

Ryzen 5 5600X、Windows amd64、Chrome 158.0.8127.37、[共通英文](sample.txt)。Goの合成は各3回の中央値。CLIは起動・モデル読込・WAV書き込み込みで6ワーカー。WASMは1ワーカーで、モデル読込後の推論を計測した。公式Rust版の速度はモデル読込後の合成ログから集計した。

| モデル | Go CLI | Go WASM | 公式Rust |
| --- | ---: | ---: | ---: |
| 6.9M REFL | 8.38倍速 | 1.08倍速 | 16.20倍速 |
| 1.9M REFL | 20.17倍速（参考） | 2.34倍速（参考） | 26.11倍速 |

6.9MはCLIの12倍速とWASMの8倍速の目標をどちらも下回った。1.9MのGo試作では同じ英文が28.4秒になり、Rust版の11.56秒と大きく異なる。1.9MのGo欄は生成した音声長を分子に使った参考値である。6.9MのGo試作は1ワーカーの計測で約8割の時間をMimiデコードに費やした。葵・茜の声質は利用許諾のある参照音声と事前抽出データを用意してから評価する。 [生データ](../results/2026-11-24-pocket-tts.json)

公開のSafetensorsファイルは6.9M版が66.5 MB、1.9M版が46.7 MBだった。配布サイズの評価には、推論で使うテンソルの抽出と保存形式の比較が必要になる。 [公式配布先](https://huggingface.co/kyutai/pocket-tts-without-voice-cloning/tree/main/languages)

## 再現手順

AoikaneのリポジトリルートからPowerShellで実行する。別のシェルを使う場合は環境変数の設定を読み替える。

```powershell
$root = $PWD.Path
$work = Join-Path $root 'dist/pocket-tts-repro'
$src = Join-Path $work 'source'
$web = Join-Path $work 'web'
New-Item -ItemType Directory $web -Force | Out-Null
git -c core.autocrlf=false clone https://github.com/CWBudde/go-pocket-tts.git $src
git -C $src checkout --detach 4b72d3af8be4676168712fec6077db22d8909fd5
git -C $src apply (Join-Path $root 'benchmarks/pocket-tts/go-pocket-tts-small.patch')
$env:CGO_ENABLED = '0'

Push-Location $src
try {
    go test . ./internal/modelcfg ./internal/native ./internal/nativert
    go build -o (Join-Path $web 'go-probe.exe') ./cmd/aoikane-probe
    $env:GOOS = 'js'
    $env:GOARCH = 'wasm'
    go build -o (Join-Path $web 'go-probe.wasm') ./cmd/aoikane-probe
} finally {
    Remove-Item Env:GOOS,Env:GOARCH
    Pop-Location
}
Copy-Item (Join-Path (go env GOROOT) 'lib/wasm/wasm_exec.js') $web
Copy-Item benchmarks/pocket-tts/index.html $web

$repo = 'kyutai/pocket-tts-without-voice-cloning'
$model = 'english_7m_refl_2026-10'
$path = "languages/$model"
$models = Join-Path $web 'model'
$assets = Join-Path $models $path
hf download $repo "$path/model.safetensors" --revision a7056f505a4a8ca72d8b5fb1440071d6f1a35739 --local-dir $models
hf download $repo "$path/tokenizer.model" "$path/embeddings/alba.safetensors" --revision 08ffc58775e62aad9409511b9591c9af626270f3 --local-dir $models
Invoke-WebRequest "https://raw.githubusercontent.com/kyutai-labs/pocket-tts/v4.0.1/pocket_tts/config/$model.yaml" -OutFile (Join-Path $assets 'config.yaml')
python benchmarks/pocket-tts/benchmark-cli.py --binary (Join-Path $web 'go-probe.exe') --assets $assets --workers 6 --sample-output (Join-Path $work 'go.wav')
python -m http.server 8765 --bind 127.0.0.1 --directory $web
```

別の端末からPlaywrightを使って測定する。既存のChromeを利用する場合は `BROWSER_CHROMIUM_PATH` を設定する。1.9Mの再現時はモデルIDを `english_2m_refl_2026-10` に変更する。

```powershell
npm install --prefix dist/pocket-tts-repro --no-save playwright
$env:PLAYWRIGHT_MODULE = (Resolve-Path dist/pocket-tts-repro/node_modules/playwright).Path
node benchmarks/pocket-tts/benchmark-web.cjs http://127.0.0.1:8765/ english_7m_refl_2026-10
```

公式Rust版の比較にはKyutaiのv4.0.1配布バイナリを使用する。 [公式リリース](https://github.com/kyutai-labs/pocket-tts/releases/tag/v4.0.1)

```powershell
$text = (Get-Content benchmarks/pocket-tts/sample.txt -Raw).Trim()
pocket-tts generate --language english_7m_refl_2026-10 --voice alba --text $text --output-path dist/pocket-tts-repro/rust.wav
```
