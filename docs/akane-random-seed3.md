# 茜: ランダム入力 seed 3 の原版比較

茜（関西弁）の純Go合成は通常CLIで利用できる。

```powershell
.\dist\aoikane-purego.exe synth --portable embedded --voice KotonohaAkane_ns_48 --text "今日はええ天気やな。" --output akane.wav
```

CGO_ENABLED=0。モデル・辞書はembedded資産を利用し、原版DLLを呼ばない。
任意の文章・全CPU・全設定での原版完全再現は、引き続き未達。

## 追加した比較

Windows amd64、AMD Family 25 Model 33、原版threads=6で、
seed=3、generator_version=1、count=8、max_units=12の通常原版API比較を実施した。
英字混じりの長文・全角英数字・記号・絵文字を含む全8件でWAV全バイトが一致した。
PCM16の1138884サンプルが一致し、実行時間は586.69秒。
ランダムseed 1/2/3を合わせて、Windowsでは24件・2646043サンプルが一致した。
この件数は検証した入力の範囲を表し、任意文章の保証ではない。

原版reportは `private/akane-random-seed3-local/KotonohaAkane_ns_48/report.json`。
原版のWAV長・SHA256を保存したfixtureは
`internal/portable/testdata/akane_random_seed3_original.json`。
期待値をGo出力から作っていない。原版のCPU・DLLハッシュ・設定・生成seedも保存した。

`TestAkaneRandomSeed3ReferenceWAV` は同じ入力を純Goで再合成し、
WAV長・SHA256が異なる場合は必ず失敗する。DLLなしのLinuxでの再現にも使える。

```powershell
$env:CGO_ENABLED = '0'
$env:AOIKANE_AKANE_RANDOM_REFERENCE_AUDIT = '1'
go test ./internal/portable -run '^TestAkaneRandomSeed3ReferenceWAV$' -timeout 30m -v
```

長文を含むため監査はopt-inで、通常のgo testではスキップする。
スキップを原版一致の証拠として扱わない。今回はGitHub Actionsを新規実行していない。

Linux amd64 CGO=0で保存原版との全8件比較も成功した（559.28秒）。
ログは `private/akane-random-seed3-linux.log`。
seed 1/2の既存16件と合わせ、Windows/Linuxで24件・2646043 PCM16サンプルが一致した。
Linuxでは原版DLLを実行せず、Windows通常原版から保存したWAV長・SHA256と比較した。
関連パッケージの `go vet ./internal/portable` と `git diff --check` も成功した。
