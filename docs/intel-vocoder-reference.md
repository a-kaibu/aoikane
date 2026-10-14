# Intel Vocoder の保存原版比較

2026-10-10。FeatNet全118演算が一致したIntel観測にも、VocoderのJIT呼出し2件の
外部入力・最終FLOAT出力が保存されていた。これを層の途中に原版値を差し込まず比較した。

## 原版観測の確認

保存観測は `private/cpu-numeric-ci-37927408109/cpu-original-replica-3/trace`。
CPUはIntel64 Family 6 Model 207 Stepping 2、通常のMKL thread数は1。
観測済みWAVと、フックなしの原版WAVの実ファイルを再ハッシュし、どちらも
157964 bytes、SHA256 `6be20e8718dc8a1329426baa824b9dabef10315dddc2099a3ad11ffac6032d7d` と確認した。
`observation-validity.json` もequal/usable_as_numerical_reference=true。

JIT run 496と505のf0/bap/spcをstrideに従って取り出した。
入力のフレーム数は110/99、最終waveのFLOAT値は26400/23760、計50160個。
入力と最終出力を合わせた63954個の参照値をfixtureへ保存した。
途中のConvなどの値はこのテストの入力にはしない。

fixtureは `internal/onnx/testdata/vocoder_intel_original.json` と `.bin.gz`。
展開後の255816 bytesのSHA256は
`e9d441ac9c931dd18bff4c3b994a2ca4a6e491c22c49263e1525ff00d9d7fa79`。
原版wave2件を連結したSHA256は
`c5b7764e6ffe882ef9c0a830ef7a3abbad8a228810995f26e49e923199c74681`。

## 現時点の差

Goのone-thread AVX512 convolution/unary、FMA embedding/small-convolution/projectionを
まとめて使う構成では、Windows/Linuxで以下の不一致を同じ件数だけ検出した。

| JIT run | wave値数 | FLOATのビット不一致 | 最初の位置 |
| --- | ---: | ---: | ---: |
| 496 | 26400 | 18580 | 0 |
| 505 | 23760 | 17341 | 2 |

FMAフラグを外したrun 496の診断でも18596個が不一致だった。
既知のフラグを切り替えるだけでは一致していない。

`TestOriginalIntelVocoderReferenceStorage` は参照値の長さ・形状・SHA256を検証し成功する。
`TestOriginalIntelVocoderWholeGraph` は
`AOIKANE_INTEL_VOCODER_REFERENCE_AUDIT=1` で有効になるstrictな未解決監査。
現在は不一致として失敗する。誤差許容や既知差分件数による合格判定を加えない。
FeatNetの一致をIntelの文章→WAVの一致として扱わない。

## 未取得の証拠と次の観測

この古いIntel artifactにはTorchScriptファイルが残っていない。
イベントでVocoderの9836696 bytesというサイズは確認できるが、
この観測のTorchScript SHA256や各Convの入出力は検証できない。
CPUの計算順だけが原因とはまだ断定しない。原版モデルのハッシュと層の入出力が必要。

今後のために `tools/trace_models.py` は、実際に受信したTorchScript/ONNXの
長さを照合してからSHA256をイベントへ記録する。
CIが大きいモデルファイルをartifactから除外しても、モデル同一性を比較できるようになる。

この変更はローカルAMDの通常の原版観測で検証した。実ファイル6件のSHA256と
イベントのハッシュが一致し、観測WAVも保存済みフックなし原版と全バイト一致した。
WAVのSHA256は `d22ec328e5937a17f1f8159db0d6381d4e81826ffe57e712ff977c6f6539ff93`。
ローカルVocoderのTorchScript SHA256は
`f63fe6b24c0de3cac10c6720adc3002166f1856493d8f91105ca77c4e6f45737`。
これを古いIntel観測の未取得ハッシュとして代用しない。

今回はGitHub Actionsを新規実行していない。


## 2026-10-10: 早期終了の証拠とプロセス診断

GitHub run 37927408109 の失敗ログを取得し、replica 3のVocoder観測ステップを確認した。
開始は2026-10-09 12:05:03 UTC、12:05:12 UTCには
`No requested ATen operation outputs were observed` で終了している。
45秒の観測タイムアウトではない。残されたeventsは41件、最終イベントは入力tensorで、
JIT結果・ATen演算出力・probe.wavがない。
古い観測にはdetach reason、crash report、子プロセスstderrがなく、
異常終了の具体的な原因は未確定。タイムアウト延長だけで直るとは扱わない。
取得ログは `private/cpu-numeric-37927408109-failed.log`。

`tools/trace_models.py` は今後、`probe-process.json` にpid・detach reason・
crash情報・経過時間・イベント件数・最後のイベント・WAVの有無を保存する。
子プロセスstdout/stderrは `probe-stdout.log` / `probe-stderr.log` に保存する。
開始時に前回のprobe.wavを除去し、crash・想定外detach・WAV未生成を不完全な観測として失敗させる。
WAVの存在だけを通常原版一致の証拠にしない。独立WAVとの全バイト比較は引き続き必須。

この変更はローカルAMD Family 25 Model 33、通常threads=6の完全Vocoder観測で検証した。
`private/vocoder-process-observation/probe-process.json` に
`reason=process-terminated`、`crash=null`、`timed_out=false`、882 eventsが保存された。
全6モデルのevent SHA256は実ファイルと一致した。
Vocoder TorchScript SHA256は引き続き
`f63fe6b24c0de3cac10c6720adc3002166f1856493d8f91105ca77c4e6f45737`。

観測WAVはフックなし通常原版 `private/blas-dispatch-ordinary-aoi.wav` と全バイト一致し、
157964 bytes、78960 PCM16 samples、SHA256
`d22ec328e5937a17f1f8159db0d6381d4e81826ffe57e712ff977c6f6539ff93`。
観測を正常終了させたことで、数値・出力が変わっていないことも確認した。

全126演算のmanifestは46918503値、原版参照SHA256
`ac0fafabb95f349de2c558f8ae7fe09bfefefcd3a2e06934c008277eddfa72ce`。
Windows CGO=0の連続Go Vocoder実行で全46918503値と最終waveの26400値がビット一致した。
原版の層出力をGo入力として差し替えていない。
manifest/監査結果は `private/vocoder-process-observation/`、ログは
`private/vocoder-process-manifest.log` / `private/vocoder-process-continuous.log`。
このローカル6-threadの成功を、Intel 1-threadの未解決差の解消として扱わない。

起動失敗も実プロセスで検証した。存在しないruntimeを指定し、事前に前回WAVを置いた状態で
観測を開始すると、終了コード1・wav_exists=falseとなり、前回WAVは残らなかった。
stderrにはruntime設定ファイルが見つからない原因が保存された。
検証先は `private/probe-process-failure/`。crash reportが届く場合の分岐は、
この起動失敗では通っていないため実クラッシュでの確認は未実施。
今回はGitHub Actionsを新規実行していない。

Linux amd64 CGO=0でも同じ通常原版参照から連続実行し、
126演算の46918503値と最終wave 26400値がすべてビット一致した。
ログは `private/vocoder-process-continuous-linux.log`。
この検証は初回Vocoder呼出しのモデル入力から最終出力までを対象とし、
任意文章・全呼出し・Intel環境についての全WAV一致を証明するものではない。
保存済みCI Vocoder traceも再検索し、既存Intel replica 3に完全な層観測がないことを再確認した。


## 2026-10-12: 1台限定のIntel再観測を準備

`.github/workflows/intel-vocoder-original.yml` は手動起動のみ、windows-2022を1台、
jobの上限15分に限定した。割り当てられたCPUがIntel以外なら、runtime復元や
数値比較を行わずhost.jsonだけを残す。このスキップを数値一致として扱わない。
runnerのCPU種類は指定できないため、必ずIntelを取得できるジョブではない。

Intelではフックなしの通常原版CLI WAVを先に生成する。ATenフックなしのモデル観測、
完全Vocoder演算観測の両WAVが独立原版と全バイト一致した場合に限り、
モデルの実ハッシュ・全Conv重み・通常threads=1を検証してmanifestを作る。
threadsを原版へ強制するオプションやtensor注入は使用しない。
不完全観測・原版WAV不一致・未対応thread数は失敗させる。

実際のVocoder TorchScript（model-2.pt）、native tensor、イベント、WAV、
probe-process.jsonとstdout/stderr、厳密比較reportを3日間のartifactとして保存する。
credentials/runtimeはartifact対象に含めない。

`tools/verify_original_vocoder -intel-profile` は既存のone-thread Intel再構築設定
（AVX512 convolution/unaryとFMA embedding/small convolution/projection）を使い、
モデル入口の3入力から全演算を連続実行する。原版の途中値は比較対象であり、
Goの入力として差し替えない。最初の不一致を保存し、1ビットでも異なれば終了コード1。
Intel設定を指定した事実を一致の証拠にしない。レポートにもintel_profileを記録する。

ツールのテストとgo vet、actionlint v1.7.11によるジョブ構文検証が成功した。
通常ローカル6-threadの126演算・46918503値と最終26400値も再比較し、
全ビット一致を維持した。ログは `private/vocoder-controls-cli-default.log`、
reportは `private/vocoder-process-observation/controls-cli-default-report.json`。
この新ジョブはまだGitHub Actionsで実行しておらず、新しいIntelの観測値は未取得。
Intel Vocoderの既知差は未解決。

既存の `cpu-original.yml` の手動入力modeに `intel-vocoder` を追加し、
そのmodeでは4台のcpu-kernelsを起動せず、新しい1台の観測ジョブだけを呼び出す。
既存のfull modeの挙動は維持する。新workflowのworkflow_callにも対応した。
公開時のpushで自動比較を一斉起動しないよう、この変更のcommitには[skip ci]を付ける。
workflow_dispatchはこの指定とは独立して手動実行する。


### 手動実行結果: runner起動前のアカウント制限

変更5702c11をpushし、既存cpu-original.ymlをmode=intel-vocoderで手動起動した。
run: https://github.com/a-kaibu/aoikane-dll/actions/runs/38177529951 。
4台のcpu-kernels jobはskippedになり、1台のIntel観測jobだけが対象になった。
ただしIntel観測jobはstepsなし・runner_name空のままfailureで終了した。
GitHub check annotationは以下の内容：

> The job was not started because recent account payments have failed or your spending limit needs to be increased.

GitHubのアカウント制限による起動前の失敗で、CPUの割当ても新規モデル観測もない。
数値一致/不一致やworkflowの実処理成功を示す結果ではない。
ログも生成されていない。再実行を繰り返していない。
新しいIntel原版の層データを得るには、この制限の解消かIntel環境の別の原版観測が必要。
ローカルAMDの原版比較・再構築は継続できる。
