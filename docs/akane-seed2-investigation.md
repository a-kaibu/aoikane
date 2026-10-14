# 茜: ランダム比較 seed 2 の未解決差分

2026-10-10。HEAD `2e50345` 時点。修正の完了報告ではなく、次の原版比較に必要な調査記録。

## 比較条件と結果

`TestRandomOriginalWAV`、CGO_ENABLED=0、KotonohaAkane_ns_48、seed=2、count=8、max-runes=12。
原版はローカルの通常エンジン実行。8件中5件がWAV全バイト一致、3件が不一致。
保存先は `private/akane-random-seed2-local/KotonohaAkane_ns_48/report.json`。
全8件一致という判定に変更してはいけない。

| 入力 | 原版WAV bytes | Go WAV bytes | 調査結果 |
| --- | ---: | ---: | --- |
| ` －「（\n？。` | 81644 | 163244 | 原版は空の読み1件、Goは2件。全PCMはゼロで、無音長が異なる。 |
| `のゆなむぷへぜつ。` | 215328 | 209444 | Goは末尾の「^ツ」の上昇記号を消している。 |
| `！（A０0。` | 254542 | 228416 | 原版は数字をゼロゼロ、Goはゼロと読む。 |

## アクセント差分の位置

`private/akane-seed2-failed-full-graphs.jsonl` は通常の茜エンジンに
`tools/probe_graph.py` を付けた観測。PROBE_MULTI=100で最初の読み取得後も実行を継続し、
PROBE_STAGES=137520,135e80で関西弁のchunk/BOUND処理も取得した。

「のゆなむぷへぜつ。」はBuild前のMorph候補・選択・Kind・Kind2・ChunkがGoと一致する。
原版のgraph_builtとgraph_afterでは、末尾「つ」のPWordはともに
Up=1、Down=1、Moras=1。原版のformatは `<S>ノ|0^ユ!ナ|0^ムプエ!ゼ|0^ツ<F>`。

`internal/phrase/format.go` の `pwordText` は、末尾のfallを消す際、
rise=1かつ単一モーラならriseも消している。この処理は上記の関西弁原版と一致しない。
BOUNDやChunk処理の差として扱わず、出力処理を原版のMode別挙動と比較する。
標準語の既存挙動を保つべきか、単一モーラ一般の条件が不足しているかは追加観測が必要。

## 数字差分の位置

「！（A０0。」の2件目の原版graph_inには、正規化後の「００」が1つのMorphとして存在する。
POS=0x1060000、候補はReading/Pronunciation=ゼロゼロ、Accent=2-3-4、Connection=*。
graph_builtのMorphもUp=2、Down=3、Moras=4。
原版の2件目のformatは `<S>^エ!ー|0ゼ^ロゼ!ロ<F>`。
混在する文字幅だけを原因と断定せず、先頭ゼロ・数字の連続・英字直後の規則を比較する。

## 次の検証

1. 関西弁の単一モーラ末尾の上昇記号を追加の原版入力で観測し、出力処理を修正する。
2. 先頭ゼロの連続読みを、英字あり/なし、全角/半角、数字の長さ別に原版と比較する。
3. 改行・開き括弧・空文の分割条件を原版と比較する。既存の13件の文境界比較を維持する。
4. 原版由来の回帰fixtureを追加し、失敗3件のWAV全バイト一致をWindows/Linuxで確認する。
5. seed 2 全8件を再比較し、既存のseed 1全8件も維持してから小さい単位でcommitする。

## 単一モーラの修正と検証

茜「つ。」「ぜつ。」「むぷへぜつ。」を原版から追加観測した。
いずれも末尾の `^ツ` を保持する。逆コンパイル済み16d400は、
Mode==1を第五引数として受け、標準語と異なる末尾記号の処理を行う。
Goのformat処理をMode別にし、末尾fallを省略しても関西弁の単一モーラriseを保持する。

Windowsの通常原版APIとの再比較で「のゆなむぷへぜつ。」は全215328バイト一致した。
Linux amd64のCGO=0ビルドでも保存済み原版WAVの全バイト相当のSHA256が一致した。
SHA256: `bbd8fa59115d1b5dc2e42996261d93c37199c3bef4b9efd8d9ff4dd5aa49edfe`。
新規の読み4件、既存の茜読み85件、原版Build比較326件、文境界13件も一致した。
`go vet ./internal/phrase ./internal/portable` は成功した。

新規fixtureは `internal/portable/testdata/akane_single_mora_original.json`。
保存原版WAVの再現テストは `TestAkaneSingleMoraOriginalWAV`、
`AOIKANE_AKANE_RANDOM_REFERENCE_AUDIT=1` で有効になる。
一時調査コード2ファイルはprivateへ移動済み。
途中で停止した自動承認レビューの利用上限は解消し、実行を再開できた。
## 先頭ゼロの修正と検証

茜と葵の原版追加観測で、助数詞のない「00」「000」「01」「007」、
英字直後の「A00」「A01」は先頭ゼロを保持し2桁ずつ読むことを確認した。
幅の混在は正規化済みで、この差の原因ではなかった。
助数詞のない全角数字列が複数桁で先頭ゼロを持つ場合は、既存の桁読み処理を使用する。

通常原版APIとのWindows再比較で「！（A０0。」の全254542バイトが一致した。
Linux amd64 CGO=0でも保存原版WAVのSHA256が一致した。
SHA256: `d2cacebf15d4c3688e178933a82e5d9d278f1b628994a0b13ce57ae823e4f7ae`。
新規の読み10件、既存の茜85件が一致し、numeralパッケージの既存テストは終了コード0で終了した。
`go vet ./internal/numeral ./internal/portable` も成功した。

fixtureは `internal/portable/testdata/leading_zero_reading_original.json`、
WAV再現テストは `TestLeadingZeroOriginalWAV`。
追加観測では助数詞付き「00人」「01人」も先頭ゼロを保持することを確認したが、
その助数詞付き処理は未修正。

## 改行の修正と検証

追加の通常原版観測21件では、改行直後の句読点が読み飛ばされる一方、
改行と句読点の間にスペース・全角スペース・タブがあると別の空文になる。
二重改行、CR、発音のある文字、開き括弧、改行後の通常語も比較した。
明示的な句読点の後の空白とは挙動が異なるため、SplitSentencesには別状態を持たせた。

Windows原版APIとの再比較でseed 2 case 2の全81644バイトが一致した。
Linux amd64 CGO=0でも保存原版のSHA256が一致した。
SHA256: `ca67ad155f7a6f0f81c54f8613e086e45a5e4ba3d9d6b5c49979f7cf89a7a3bc`。
新規の原版読み21件はWindows/Linuxで一致した。
既存13件を含む文境界34件、原版Build比較326件、茜の既存85件と新規4件、
先頭ゼロ10件の読み比較、関連パッケージのvetも成功した。

原版のseed 2全8件のWAV長・SHA256を、失敗した原版比較reportから直接保存した
`internal/portable/testdata/akane_random_seed2_original.json` を追加した。
Go出力から期待値を作らず、全8件をstrictに再現する
`TestAkaneRandomSeed2ReferenceWAV` を既存のopt-in監査へ追加した。

Windowsの通常原版とのseed 2全8件再比較は8/8、全822483 PCM16サンプルが一致した。
再実行した原版の全8件のWAV長・SHA256も、保存済み原版fixtureと同一だった。
Linux amd64 CGO=0の保存原版seed 1/2計16件の再現も16/16で成功した。
これらは次節の助数詞追加移植の前にビルドした実行ファイルでの検証。
ログは `private/akane-seed2-all-fixed.log`、`private/akane-random-seeds-linux.log`。

## 助数詞付き先頭ゼロの修正と検証

先頭ゼロの連続数字は2桁ずつのグループを保持し、助数詞を最後のグループだけに接続する。
グループに整数の最終桁キーを与えない。原版の「001人」はイチニンで、
ヒトリにならず、「016本」はロクホンでロッポンにならない。

原版bancharcxの[パターン]にある `整数ゼロ飛ばし` は、
年・月・日・時・分・秒・期・年度について先頭ゼロを省く指定を持つ。
Goのルールローダでこの条件のリテラル助数詞を読み取り、特殊読みより先に適用する。
「04日」はヨッカ、「00年」はゼロネンとなる。全角数字の条件を変更し、
未観測の漢数字先頭〇などへはこの移植の証拠を拡張しない。

原版から得た助数詞付きの読み20件はWindows/Linuxで全件一致した。
通常原版APIで「001人」「016本」「04年」を合成し、WindowsのGo WAVと全バイト一致した。
その原版のWAV長・SHA256を保存したfixtureは
`internal/portable/testdata/leading_zero_counter_wav_original.json`。
Linux CGO=0での3件の再現も全件一致した。

| 入力 | WAV bytes | 原版/Go共通 SHA256 |
| --- | ---: | --- |
| 001人。 | 189268 | `04dec38cdcfa749c8a7aaa0e710c1821a34bb51e86e92d89cb811c7508a1766c` |
| 016本。 | 189666 | `a546fb5d05bfd2514c3f321e418b1e7aa06925f8df67ba184dc3b2d3b266aa79` |
| 04年。 | 132602 | `f1b707716cbedf6c2cd69c6ab8a0cbaa95524c816b6b38222b847b250e0db775` |

関連パッケージのvet、新規20件と既存の茜85件・単一モーラ4件・先頭ゼロ10件・改行21件の
strictな読み比較が成功した。macOS arm64はCGO=0でのCLIビルド成功のみを確認し、
実機音声比較は行っていない。Windowsの `dist/aoikane-purego.exe` もCGO=0で更新し、
embedded資産からの空文WAVが原版の81644バイト・SHA256に一致することを確認した。

既存numeralの診断用 `TestPlainNumbers`/`TestCounterNumbers` は、差をログに出すだけで
失敗にしない。今回確認したログにも、英語読みの文脈や「1日」の日付文脈に起因する
plain 1件・counter 4件の差が残っている。その終了コード0を原版完全一致の証拠として使わない。
今回追加した読み/WAVテストは、不一致を必ず失敗にする。

任意の文章・全CPU・全設定について原版と同一であることは、引き続き未達。
