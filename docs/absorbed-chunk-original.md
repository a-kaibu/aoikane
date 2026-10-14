# 数詞結合時の係り先を原版に合わせる

seed16 / generator_version=1 / max_units=24 / count=128を両音声の通常原版SDKで採取した。
corpus SHA256 `1bdddd32b467d0350e6d1c8fb6dbe8769c354044cd43ec7581ebe84910121ee9`。
全256件のtalk成功、正常detach、crash/timeout/エラーなし、保存原版WAVの長さとSHA256、
native report SHA256を検証した。参考はtools/compare_reading/testdata/seed16-{akane,aoi}-original.json。

葵は全128件一致、茜はcase27「読む行くに一人はがと3.5ABC読む開ける今日。」の1件が不一致。
原版の「イクニ|0」に対してGoは「イクニ$2_2」で、「に」の境界が原版Kind=0、Go=4だった。
読みの比較ツールは不一致を終了コード1で報告し、全128件を保持した。

読み取り専用の原版段階観測では、CaboCha後の文節0の係り先は3。
BANCHA後に小数の結合で文節3を削除しても、文節0の係り先は3のまま残る。
一方、文節3を結合する文節2自身の係り先は3から4に更新される。
Goは削除された文節への依存をすべて追っていたため、文節0も4に変更していた。
この違いが関西弁の文節距離・モーラ範囲の推定に影響し、余分な区切りになった。

Ghidraの128370では、隣の文節を結合する際に、結合元自身の+0x34が
結合される隣の+0x28と一致した場合だけ、その隣の+0x34へ更新する。
18fbe0は文節を削除するが、他文節の+0x34を全探索して書き換える処理はない。
GoのcontractNumeralChunksを、自身が吸収した文節への依存だけ更新する処理へ修正した。
依存先の元番号と、生存文節の飛び番は保持する。
原版の3段階の文節属性とtrace SHA256はdocs/evidence/absorbed-chunk-dependencies.jsonに保存。
この診断は整形後に終了しており、音声全体の一致の証明とは区別する。

修正後はseed16両音声の全256件が読み一致した。
さらに前置の動詞・助詞・名詞と小数/整数を変えた32文章×両音声の全64件が読み一致。
全talk成功、正常detach、エラーなしと全原版WAVの長さ/SHA256を再確認した。
corpus SHA256 `60975ff6cfce5725ccb18febd289ce41815232d46395c766cc0b73592ea93957`。
参考はtools/compare_reading/testdata/absorbed-chunk-32-{akane,aoi}-original.json。
TestRandomSeed16LongReadingOriginalとTestAbsorbedChunkReadingOriginalで全件を通常回帰に追加。
結果による入力除外、誤差許容、文章専用の置換はない。

既存の原版小数文節回帰と新旧の全読み回帰が成功（portable 49.250秒）。
phrase全回帰68.804秒、phrase/portableのgo vetも成功した。
ログはprivate/absorbed-chunk-{reading,phrase}-regression.log、absorbed-chunk-vet.log。

次の3文章×両音声6件の独立通常原版SDKとGoの全WAV比較が成功：

- 読む行くに一人はがと3.5ABC読む開ける今日。
- 行くに一人はがと3.5ABC読む開ける今日。
- 読む行くに一人はがと35ABC読む開ける今日。

茜の独立通常原版3件は590.58秒で全WAV一致（702972 PCM16サンプル）。
原版とGo双方の保存WAVを参照・レポートの長さ/SHA256へ再照合した。
葵も705.68秒で全3件が一致し、両音声6件で1525649 PCM16サンプル。
全原版とGo WAV実ファイルの長さ/SHA256を再照合し、report_sha256を独立比較2件へ更新。
以前の観測時native report SHA256はcapture_report_sha256へ保持した。
Linux amd64 CGO=0の全6件も1264.46秒で成功。全文から生成し全WAVの長さ/SHA256が一致。
TestAbsorbedChunkOriginalWAVはフック・tensor差し替えなしで全文から比較する。
保存原版6件のDLL不要再現はAOIKANE_ABSORBED_CHUNK_REFERENCE_AUDIT=1で実行でき、
Linux amd64 CGO=0の実行も全件成功した。参照はtestdata/absorbed_chunk_wave_original.json。
ログはprivate/absorbed-chunk-{akane,aoi}-wave.log、absorbed-chunk-linux-reference.log。
今回の一致は採取した入力の範囲。任意文章・全CPU・全設定の完成は未達で、既知CPU差も残る。


7d6b819のclean treeからdist/aoikane-purego.exeを更新した。
Go 1.26.8、CGO_ENABLED=0、GOAMD64=v1、vcs.modified=falseを確認。
--portable embeddedで茜「読む行くに一人はがと3.5ABC読む開ける今日。」を全文から合成し、
独立通常原版のWAV実ファイルと全バイト一致した。
478724 bytes、SHA256 `e1d9a41557e0ddc3d67790719bc9788962d85e59b60f43554a21380ebf1c18d8`。
ログはprivate/absorbed-chunk-cli-{akane,build-info}.log。
元のSDK用dist/aoikane-dev.exeは更新していない。
