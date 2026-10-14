# 未知の1文字ひらがなの原版比較

2026-10-12、seed=5 / generator=1 / max_units=12 のランダム文章128件を
葵・茜それぞれの通常SDKで合成し、formatterを観測して読み全体を比較した。
各音声3件の相違のうち、case 76「きぱこごてみんあながぐげ。」で
未知の「ぐ」がGo側だけ欠落していた。

原版graph_inでは「ぐ」はPOS=0xd030000、空の読み候補のままSurfaceが保持され、
後段の1a9ea0で発音「グ」、Up=0 / Down=0 / Moras=1になっていた。
GoはunknownUnitでSurfaceを空白に変えていたため、既存のa9ea0相当処理が動かなかった。
未知の1文字ひらがなのSurfaceを保持して修正した。未知カタカナのアクセントを転用していない。

修正後の読み比較は各128件中126件一致。残るcase 105の長音選択と
case 125の数字・英字のアクセント／境界は厳密比較で失敗するまま保持している。
256件のWAVが一致したという意味ではない。

修正対象の文章は、観測ツールを使わない独立の通常原版SDKとGoで再合成し、
Windowsで完全WAV全バイト一致した（茜91.24秒、葵100.64秒）。
原版CPUはAMD64 Family 25 Model 33 Stepping 0、通常threads=6、AVX512 convolutionなし。
DLL SHA256は711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17。

| 音声 | WAV bytes | 原版WAV SHA256 |
| --- | ---: | --- |
| 茜 | 253046 | 72e9b2818f28cf03ab72a7ffb803c2bb0bdf0db9b1802a37b78358ef11c073a2 |
| 葵 | 285702 | d6e14f048fa4baed73d14f77bbc87995feaa10816ca4cbda983ca5115c01c940 |

Linux amd64 CGO=0で、原版DLLや中間tensorを使わず両音声のWAV長・SHA256も一致した
（150.82秒、private/hiragana-wave-linux.log）。
期待値は原版WAVから採取し、原版reportと読みcaptureのSHA256を
internal/portable/testdata/unknown_hiragana_original.jsonに保存した。
読み比較はTestUnknownHiraganaOriginalで常時実行する。
完全WAVの保存原版再現は以下で実行する。

```powershell
$env:CGO_ENABLED = '0'
$env:AOIKANE_UNKNOWN_HIRAGANA_REFERENCE_AUDIT = '1'
go test ./internal/portable -run '^TestUnknownHiraganaOriginal$' -count=1 -timeout 20m -v
```

Windows元比較ログはprivate/hiragana-wave-akane.logとprivate/hiragana-wave-aoi.log。
保存WAVはprivate/hiragana-wave-akane/KotonohaAkane_ns_48/とprivate/hiragana-wave-aoi/。
既存の読み・長音・未知英語の回帰比較とportableのgo vetも成功した。
Intel Vocoderなど既知のCPU差は未解決であり、この修正で完成とは扱わない。
