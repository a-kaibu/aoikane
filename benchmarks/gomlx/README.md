# GoMLXでAoikaneの日本語ONNXを実行する

Aoikaneの実モデルを `onnx-gomlx v0.5.36`、`gomlx v0.28.39`、`compute v0.1.33` のGoバックエンドで検証した。Windows amd64、Ryzen 5 5600X、Go 1.27.4、`CGO_ENABLED=0`、`GOEXPERIMENT=simd`。js/wasmへのクロスビルドも確認した。 [GoMLXバックエンドの仕様](https://gomlx.github.io/docs/backends/)

| ONNX | 公開版 | Round・EyeLike追加後 |
| --- | --- | --- |
| emb | 実行成功 | 実行成功 |
| bou | 実行成功 | 実行成功 |
| pau | EyeLike未対応 | 実行成功 |
| Vocoder | Round未対応 | 実行成功 |
| FeatNet | Tileのrepeatsを静的に評価する際、サブグラフが参照するphnを解決できない | 同じエラー |

表は動的軸を8に置いた短い合成入力で確認した。Vocoderではさらに現行Aoikaneの葵の発話「おはよう。」から音響特徴量を取得し、同じ特徴量から48 kHzの波形を生成した。

| Vocoderの入力長 | 現行Aoikane | GoMLX |
| --- | ---: | ---: |
| 110フレーム | 93 ms | 3,640 ms |
| 69フレーム | 76 ms | 1,818 ms |

現行は特徴量取得時の各呼び出し時間。GoMLXは同じ入力でグラフをコンパイルした後、3回呼び出した中央値。計測範囲はVocoderのみ。VocoderのWAVは既存のコンプレッサと最終出力処理を通す前の波形を試聴するために保存した。日本語全体の品質・字幕・WASM実行速度はFeatNetの変換が通ってから評価する。 [各試行の記録](../results/2026-11-24-gomlx.json)

## 再現手順

AoikaneのリポジトリルートからPowerShellで実行する。

```powershell
$root = $PWD.Path
$work = Join-Path $root 'dist/gomlx-probe'
$bench = Join-Path $root 'benchmarks/gomlx'
$models = Get-ChildItem assets/synth,assets/boundmodel -Filter '*.onnx' |
    Select-Object -ExpandProperty FullName
New-Item -ItemType Directory $work -Force | Out-Null

$env:GOTOOLCHAIN = 'go1.27.4'
$env:CGO_ENABLED = '0'
$env:GOEXPERIMENT = 'simd'
Push-Location $bench
go build -o (Join-Path $work 'gomlx-probe.exe') .
Pop-Location
& (Join-Path $work 'gomlx-probe.exe') @models
```

互換パッチを使う場合はGoMLXの変換ライブラリを別ディレクトリに展開する。検証用の別のgo.modにローカルのソースを指定する。

```powershell
$src = Join-Path $work 'onnx-gomlx'
$mod = Join-Path $work 'compat.mod'
git -c core.autocrlf=false clone --depth 1 --branch v0.5.36 https://github.com/gomlx/onnx-gomlx.git $src
git -C $src apply (Join-Path $bench 'onnx-compat.patch')
Copy-Item (Join-Path $bench 'go.mod') $mod
Copy-Item (Join-Path $bench 'go.sum') (Join-Path $work 'compat.sum')
$exe = Join-Path $work 'gomlx-compat.exe'
Push-Location $bench
go mod edit "-modfile=$mod" "-replace=github.com/gomlx/onnx-gomlx=$src"
go build "-modfile=$mod" -o $exe .
Pop-Location
& $exe @models
```

実際の発話で測定する場合は、Goのオーバーレイを使って現行Vocoderの入出力を作業ディレクトリに保存する。

```powershell
$original = (Resolve-Path internal/synthesis/engine.go).Path
$capture = Join-Path $work 'capture_engine.go'
Copy-Item $original $capture
git apply --ignore-space-change --directory=dist/gomlx-probe benchmarks/gomlx/capture-overlay.patch
$overlay = Join-Path $work 'overlay.json'
@{ Replace = @{ $original = $capture } } | ConvertTo-Json |
    Set-Content -Encoding utf8NoBOM $overlay
$fixtures = Join-Path $work 'fixtures'
New-Item -ItemType Directory $fixtures -Force | Out-Null
$env:AOIKANE_VOCODER_FIXTURES = $fixtures
$env:GOTOOLCHAIN = 'local'
Remove-Item Env:GOEXPERIMENT
go build "-overlay=$overlay" -o (Join-Path $work 'capture.exe') ./cmd/aoikane
& (Join-Path $work 'capture.exe') --voice aoi --text 'おはよう。' -o (Join-Path $work 'aoikane.wav')
$vocoder = (Resolve-Path assets/synth/vocoder.onnx).Path
Get-ChildItem $fixtures -Filter '*.json' | ForEach-Object {
    & $exe --vocoder-fixture $_.FullName $vocoder
}
```
