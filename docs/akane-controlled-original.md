# 茜の公開設定と通常原版の完全WAV比較

2026-10-10、AMD64 Family 25 Model 33 Stepping 0で通常原版APIの公開設定を変更し、純Goの文章→WAVと比較した。tensor注入や層の差し替えは行っていない。

`tools/corpus/akane-controlled-requests.json` の8件は速度・高さ・音量・抑揚を個別に変更する4件、複合設定の短文と関西弁文、上下の追加設定2件。全設定範囲の網羅を意味しない。

Windowsの `TestControlledOriginalWAV` は8/8で全バイト一致した（193.62秒）。原版とGoの保存WAVを再読込し、RIFF/WAVE、48kHz・mono・PCM16、サイズ・SHA256・全バイト一致も確認した。合計517884 PCM16サンプル。

Linux amd64 CGO=0の `TestAkaneControlledReferenceWAV` は原版のWAVサイズとSHA256に8/8一致した（179.65秒）。入力は文章と公開設定のみで、原版の途中tensorを使用しない。

実行例（リポジトリルート、CGO_ENABLED=0）：

```powershell
$env:AOIKANE_TEST_ENGINE = 'C:\path\to\aoikane-dll\private\runtime'
go test ./internal/portable -run '^TestControlledOriginalWAV$' -count=1 -timeout 30m -v
$env:AOIKANE_CONTROLLED_REFERENCE_AUDIT = '1'
go test ./internal/portable -run '^TestAkaneControlledReferenceWAV$' -count=1 -timeout 30m -v
```

原版比較のprofileはローカル通常原版に合わせたthreads=6で固定。別CPUの通常profileを自動選択するテストではない。

原版期待値は `internal/portable/testdata/akane_controlled_original.json` に保存。原版DLL/Torch DLLハッシュ、CPU、設定、各WAVサイズ・SHA256、元reportのSHA256を残した。コーパスはGit checkoutの改行差を許すためCRLF→LFだけを正規化してハッシュ照合する。実際に観測したWindows原本ファイルの生ハッシュも `corpus_file_sha256` に保持しており、期待値をGo出力から生成していない。

ローカル証拠：`private/akane-controlled-original/report.json`、同ディレクトリの原版/Go WAV、`private/akane-controlled-original.log`、`private/akane-controlled-linux.log`。

Intel Vocoderの不一致、他AMDで確認済みの一部PCM差、任意文章・全設定の網羅は残る。GitHub Actionsは新規実行していない。ユーザー指定によりmacOS実機検証は対象外とする。
