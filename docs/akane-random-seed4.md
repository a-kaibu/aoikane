# 茜: ランダム入力 seed 4 の原版比較

2026-10-10、Windows amd64、AMD Family 25 Model 33、通常原版threads=6、CGO_ENABLED=0。
seed=4、generator_version=1、count=8、max_units=12で通常原版APIと文章からの純Go合成を比較した。
全8件でWAV全バイトが一致した（445.24秒）。原版へのtensor注入・強制threads設定は行っていない。

8件はひらがな列、漢字/数字/括弧の混在、括弧だけの入力、小数を含む文、数字単独、
記号/半角カナ/英字の混在、単語を連ねた文を含む。各原版とGoの保存WAVも独立に再読込し、
サイズ・SHA256・全バイト一致と48kHz mono PCM16を確認した。今回884875サンプル。
Windowsのseed 1/2/3/4合計は32件・3530918 PCM16サンプルで完全一致。

原版report: `private/akane-random-seed4-local/KotonohaAkane_ns_48/report.json`。
reportのSHA256: `85003b69e737b55b5357021a70ddcf22c8f2507de7802cd96d5756286d7bc4c2`。
通常原版WAVの長さ・SHA256を `internal/portable/testdata/akane_random_seed4_original.json` に保存した。
DLLハッシュは比較終了後に同じruntimeの実ファイルから再確認した。
期待値はGo出力から生成せず、元reportと原版WAVの長さ・ハッシュを照合して保存した。

```powershell
$env:CGO_ENABLED = '0'
$env:AOIKANE_TEST_ENGINE = 'C:\path\to\aoikane-dll\private\runtime'
$env:AOIKANE_DIFF_VOICE = 'KotonohaAkane_ns_48'
$env:AOIKANE_DIFF_SEED = '4'
$env:AOIKANE_DIFF_COUNT = '8'
$env:AOIKANE_DIFF_MAX_RUNES = '12'
go test ./internal/portable -run '^TestRandomOriginalWAV$' -count=1 -timeout 30m -v

# 原版DLLなしの厳密再現
$env:AOIKANE_AKANE_RANDOM_REFERENCE_AUDIT = '1'
go test ./internal/portable -run '^TestAkaneRandomSeed4ReferenceWAV$' -count=1 -timeout 30m -v
```

再現テストは保存した原版の全WAV長とSHA256に一致しなければ失敗する。
Linux amd64 CGO=0でも全8件が原版のWAV長・SHA256と一致した（432.90秒）。
ログは `private/akane-random-seed4-linux.log`。Windows/Linuxのseed 1/2/3/4合計は
32件・3530918 PCM16サンプルで一致。任意文章・全CPU・全設定の保証ではない。
Intel Vocoderと他AMDの既知差の解消は残る。GHAは新規実行していない。
ユーザー指定によりmacOS実機検証は対象外。
