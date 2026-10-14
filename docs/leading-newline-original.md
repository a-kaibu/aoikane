# 先頭改行の空文を原版と一致させる

2026-10-12、seed7 / generator=1 / max_units=12のcase74
「\n）！é。é 😀A。」で、原版は空の読みを3つ、Goは2つ返していた。
原版graphの最初の形態素は空のSurface・空候補だった。
原因はSplitSentencesが先頭の改行を空文ごと捨てていたことだった。

原版の通常SDKで先頭改行、CR、CRLF、連続改行、前後の空白・タブ、句読点、
括弧、未知文字、通常の複数文を含む22文章×葵・茜=44件を追加比較した。
先頭改行は1つの空文を保持し、連続改行では追加しない。直後の句読点は
既存の改行後抑制の対象になる。空白のあとに先頭改行がある場合も同様。
SplitSentencesでこの空文を保持し、KanaTextが<S><F>として返すように修正した。
文字列そのものが空の場合は追加の読みを生成しない。合成APIの空白だけの入力拒否も維持する。

原版44件の読み期待値と採取元2回のSHA256を
internal/portable/testdata/leading_newline_reading_original.jsonに保存した。
TestLeadingNewlineReadingOriginalで常時比較し、既存の文分割テストにも境界を追加。
seed7の各128件は全読み一致し、TestRandomSeed7ReadingOriginalを追加した。
seed5〜7の計768件、既存の読み・文分割の回帰テストとgo vetも成功した。

## 完全WAV比較

行単位のコーパスでは先頭改行を含む1つのリクエストを表せないため、
TestLeadingNewlineOriginalWAVが改行を含む文字列をそのまま通常SDKへ渡す。
観測ツールなしの独立原版と、Windows amd64・CGO=0のGoで次の3文章×両音声を
完全WAV全バイト比較し、6件すべて一致した。計626640 PCM16サンプル。

- \n）！é。é 😀A。
- \nあ。
- \n。

上記\nは実際の改行。原版CPUはAMD64 Family 25 Model 33 Stepping 0、
通常threads=6、AVX512 convolutionなし。途中tensorは注入していない。
元reportと全原版WAVの長さ・SHA256を
internal/portable/testdata/leading_newline_wave_original.jsonへ保存した。
独立原版WAVは読み採取時の原版WAV長・SHA256とも一致していた。
Windowsログはprivate/leading-newline-wave-akane.log（46.74秒）と
private/leading-newline-wave-aoi.log（46.46秒）。

Linux amd64、CGO=0でも同じ6件すべてが原版WAV長・SHA256と一致した。
原版DLLや途中tensorは再現入力に使っていない。
ログはprivate/leading-newline-wave-linux.log（105.69秒）。
Windows embedded CLIも修正commit e993fb8から更新し、CGO_ENABLED=0と
vcs.modified=falseを確認した。茜「\nあ。」の180524 bytes / SHA256
02ad8964ace53160d77763a758293dacf3a26d0aa625d39a712cf60d680a2430が独立原版と一致した。
CLIログはprivate/leading-newline-cli.log。

```powershell
$env:CGO_ENABLED = '0'
$env:AOIKANE_TEST_ENGINE = 'C:\path\to\aoikane-dll\private\runtime'
$env:AOIKANE_DIFF_VOICE = 'KotonohaAkane_ns_48'
go test ./internal/portable -run '^TestLeadingNewlineOriginalWAV$' -count=1 -timeout 30m -v

# 原版DLLを使わず、保存した原版完全WAVを再現
$env:AOIKANE_LEADING_NEWLINE_REFERENCE_AUDIT = '1'
go test ./internal/portable -run '^TestLeadingNewlineReferenceWAV$' -count=1 -timeout 30m -v
```

既知Intel Vocoder等のCPU差と、任意文章・全CPU・全設定の完全一致の保証は未達。
768件の読み一致を768件すべてのWAV一致とは扱わない。

## 次のランダム比較

seed8 / generator=1 / max_units=12の128件×両音声を通常原版から取得した。
各126件の読みが一致したが、case18「漢（「－）～\n😀Ж「アあ。」と
case41「日1四（6七ａ声。」に相違が残る。前者の\nは実際の改行。
前者は原版だけマイナスの発音が残り、後者は原版の小文字aがエーだがGoはアールとなる。
原版参考はtools/compare_reading/testdata/seed8-{akane,aoi}-original.json。
相違を除外せず、厳密比較は非ゼロ終了する。seed8全件のGo WAV一致は未検証。
原版graphはprivate/reading-seed8-failures-akane-graph.jsonl。
