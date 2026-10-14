# 葵・茜の設定境界付近の原版WAV比較

2026-10-12、Windows amd64、AMD Family 25 Model 33、通常原版threads=6。
CGO_ENABLED=0のGoと通常原版の公開configuration setterを使い、文章「あ。」から
生成した完全WAVを比較した。原版のthread数の強制やtensor注入は行っていない。

`tools/corpus/controlled-boundary-requests.json` は以下を両音声で比較する8件。

| ケース | Rate | Pitch | Volume | Emphasis |
| --- | ---: | ---: | ---: | ---: |
| quiet | 1 | 1 | 0.001 | 1 |
| flat | 1 | 1 | 1 | 0.000001 |
| upper | 4 | 2 | 2 | 2 |
| slow-quiet | 0.5 | 0.5 | 0.001 | 2 |

Windowsの `TestControlledBoundaryOriginalWAV` は95.96秒で全8件がWAV全バイト一致した。
合計398880 PCM16サンプル。原版/Goの保存WAVも独立に再読込し、長さ・SHA256・
全バイト一致と48kHz mono PCM16を照合した。

低音量のquiet/slow-quietは、原版の非ゼロPCMが6067〜12564サンプル、
最大絶対値61〜94。無音同士の一致だけを検証したケースではない。

原版の期待値は `internal/portable/testdata/controlled_boundary_original.json`。
原版DLL/Torch DLLハッシュ、CPU、通常thread数、公開設定、元reportのハッシュ、
各原版WAV長・SHA256を保持した。コーパスの改行正規化ハッシュと実ファイルの生ハッシュも保存した。
期待値は原版WAVから生成し、Go出力のハッシュを期待値に流用していない。

元WAV/reportは `private/controlled-boundary-original/`、
ログは `private/controlled-boundary-original.log`。

```powershell
$env:CGO_ENABLED = '0'
$env:AOIKANE_TEST_ENGINE = 'C:\path\to\aoikane-dll\private\runtime'
go test ./internal/portable -run '^TestControlledBoundaryOriginalWAV$' -count=1 -timeout 30m -v

# 原版DLLを使わず保存された原版WAVを厳密再現
$env:AOIKANE_CONTROLLED_REFERENCE_AUDIT = '1'
go test ./internal/portable -run '^TestControlledBoundaryReferenceWAV$' -count=1 -timeout 30m -v
```

従来の茜8件も同じ比較helperで維持し、新しい8件は別コーパス/別fixtureとした。
Linux amd64 CGO=0でも8件すべてが原版のWAV長・SHA256と一致した（90.01秒）。
ログは `private/controlled-boundary-linux.log`。原版DLLや途中tensorを再現入力に使っていない。
共通helperへの変更後、従来の茜rateケースもWindowsで原版期待値との一致を再確認した。
ログは `private/controlled-helper-regression.log`。portableのgo vetも成功した。
新しいGHAは起動していない。Intel Vocoder等の既知CPU差は未解決。
この8件の一致を、任意文章・全CPU・全設定の保証として扱わない。
