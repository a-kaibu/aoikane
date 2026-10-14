# 零と文字種が混在する数詞の原版観測

`tools/probe_graph.py` の `PROBE_NUMERAL_SEGMENTS=1` は原版の
100a90 / 1043f0 / 10b010 / 107920 / 109320 の呼び出しを読み取り専用で観測する。
CP932の入力、呼び出し元RVA、読みと表記の累積出力を記録する。
引数のraw_argsは未解釈の値であり、すべてを有効な関数引数とは扱わない。

葵の5入力について18呼び出しのenter/leaveが対応し、各入力でformat/doneを1回確認。
errorイベントなし。Python構文と置換済みJavaScript構文の検査も成功した。
診断はformatter到達時にプロセスを終了するため、この観測だけではWAV一致を証明しない。
元traceのSHA256、対象DLLのSHA256、全18呼び出しの復号済み出力を
`docs/evidence/mixed-rei-segments.json` に保存した。

## 観測した累積出力

| 入力 | 100a90の読み | 数詞の表記 |
| --- | --- | --- |
| 一零67。 | イチレーｂロクジューナナ | 一零六十七 |
| 一零０３人。 | イチレーｂゼロサン | 一零〇三 |
| 一零十67。 | イチレーｂジューロクジューナナ | 一零十六十七 |
| 一零67十。 | イチレーｂロクジューナナｂジュー | 一零六十七十 |
| 一零12.3。 | イチレーｂジューニテンｂサン | 一零十二．三 |

すべてで一零の区間は1043f0へ渡され、イチレーと元の表記一零を保持する。
その後のArabic数字区間は10b010へ渡される。
100a90の逆コンパイルでも、Jap区間に零がある場合に1043f0へ進む条件と、
文字種境界でｂを追加する処理を確認した（零の定数RVA244a68、ｂは38e738）。
単位の判定文字列RVA244b5cは十百千。

単位→数字では読みを直接連結するが、複数Arabic数字→十では区切る。
そのため一零十67と一零67十は、単純な数値評価や同じ文字種の区間分割では再現できない。
一零０３の表記には最終桁三が残る。助数詞の処理でこの表記を参照する可能性を
引き続き調べる。すべての単位・助数詞分岐を再現したという証拠ではない。

## 区間の桁読みを保持する実装修正

混在数詞のdigitTypeRunsに零がある場合、数値評価ではなく既存の区間読みを使う。
connectorDecimalでも、後続の文字種区間がある零の区間を保持する。
一零67と一零12.3の一零がジューになる差を修正した。
単位間の再分割・読みの再結合や、混在区間末尾の先頭ゼロ＋助数詞はまだ未再現。

48文章をすべて比較し、茜・葵とも不一致が31件から10件へ減った。
各比較は残る不一致により終了コード1。結果に応じた除外・許容差はない。
残る番号は10,11,25〜32（一零０３人/本、一零十67、一零67十と助数詞）。
`AOIKANE_MIXED_REI_READING_AUDIT=1` のTestMixedReiReadingOriginalAuditは
全48×2件を監査し、現状は20件の不一致で失敗する。これを成功扱いにしない。

seed15の128文章×両音声は全読み一致となり、通常回帰へ追加した。
seed5〜15の2816件と追加読みの回帰は35.057秒で成功。
numeral/phraseの全回帰は0.546秒/63.577秒で成功。go vetも成功。
ログはprivate/mixed-rei-reading-regression.log、mixed-rei-source-regression.log、mixed-rei-vet.log。

フック・tensor差し替えなしの通常原版SDKとGoを比較し、茜の次の3文章は
WAV全バイト一致（345.09秒、543326 PCM16サンプル）：

- 日五月8Ａ一零67人二生開（0人ａー七木ｂ。
- 一零67。
- 一零12.3。

この独立原版3件のSHA256と長さは、既存48件の原版保存WAVとも一致した。
同じ3文章の両音声6件について、既存の通常SDK全合成のWAV実ファイルと
native reportのSHA256を再確認してtestdata/mixed_rei_wave_original.jsonへ保存。
葵のフックなし通常原版3件も351.70秒で全WAV一致した。
両音声6件で1093127 PCM16サンプル。保存参照のreport_sha256を独立比較2件へ更新し、
以前のformatter観測のnative report SHA256はcapture_report_sha256として区別して保持。
Linux amd64 CGO=0の6件再現は689.10秒で全成功した（実行バイナリは44c9cbc時点のソース）。AOIKANE_MIXED_REI_REFERENCE_AUDIT=1で全文から
Go合成し、全WAVの長さとSHA256を比較する。

進行中ログはprivate/mixed-rei-step2-aoi-wave.log、mixed-rei-step2-linux-reference.log。
茜の独立比較ログはprivate/mixed-rei-step2-akane-wave.log。
既知CPU差と未検証入力・設定も継続課題であり、任意文章の完全一致は未達。

## 助数詞の区間モードの違い

10f0a0の入力を読み取り専用で追加観測した。4入力は各1回format/doneへ到達し、
errorなし。全入力引数をdocs/evidence/mixed-rei-counter-context.jsonへ保存。
param11/param12は逆コンパイルのchar引数の下位8ビットであり、上位ビットを意味づけしない。

| 入力 | 数詞の表記 | 助数詞の入力読み | param11 | param12 |
| --- | --- | --- | --- | --- |
| 一零０３本。 | 一零〇三 | ボン | 1 | 1 |
| ０３本。 | 〇三 | ホン | 0 | 1 |
| 一零０３人。 | 一零〇三 | ニン | 1 | 1 |
| ０３人。 | 〇三 | ニン | 0 | 1 |

一零０３の数詞アクセントは2-3-4,s0,2-3-4、０３は2-3-4。
10f0a0の逆コンパイルでは表記の末尾文字を取り出し、125b90へtype=6で渡す。
param11が有効な場合に助数詞のTn=/Tw=の照合へ進む分岐を確認。
ボンへの変更は10f0a0入口ですでに入力読みへ反映されているため、その前の
読み分けも追う必要がある。すべての桁読みへ最終桁のkeyを付ける修正は、
０３本などの原版ホンを壊すので行わない。

別の候補f8940も逆コンパイルしたが、これは末尾ゼロをレーへ置換する関数で、
一般的なアクセント再分割や助数詞の変化を処理する関数ではなかった。

44c9cbcのclean treeからdist/aoikane-purego.exeを更新した。
Go 1.26.8、CGO_ENABLED=0、GOAMD64=v1、vcs.modified=falseを確認。
--portable embeddedで茜「一零12.3。」を合成し、独立原版WAVと全バイト一致。
237114 bytes、SHA256 `25c368feaefaa54d485b631581ef357cbe524bd78bec335f88dffecc1e0d5057`。
元のSDK用dist/aoikane-dev.exeは更新していない。

## 小単位の区切りと助数詞の数詞モード

100a90の小単位区間を再現し、複数桁→十百千と文字種の切り替わりにはｂを残す。
単位→数字は音を連結してテーブルを先頭から照合し、一零十67のジュー＋ロクジュー＋ナナを
ジューロク／ジューナナへ再分割する。11〜19の最終桁とPartsを保持して助数詞へ渡す。
大単位の処理は既存経路を継続使用し、今回これらを全面的に再検証したとは扱わない。

桁読みのfragmentにのみ末尾の数字keyを付け、一零０３人/本のアクセントとボンを再現。
単独０３本などの桁読みのkeyは従来どおり保持しない。
追加48文章×両音声が全96件一致となり、通常の読み回帰へ移した。
TestMixedReiReadingOriginalは全件を保持し、任意の1件でも違えば失敗する。

固定64文章（16数詞×助数詞なし/人/本/Ｃ）を追加して通常SDKの全合成から読みとWAVを採取した。
両音声の全talk成功・正常終了・全WAVの長さ/SHA256・native report SHA256を再確認。
corpus SHA256はd5064eab0145aafeba64c2044121c31fc9645fa8089be677203f38675979d2ed。
元参考はtools/compare_reading/testdata/mixed-rei-unit-64-{akane,aoi}-original.json。

小単位の最初の試作は各64件中15件不一致だった。全件を残したまま追加観測し、
十０３のような単位後の先頭ゼロは、ｂがなくても独立した桁読みアクセントを保つことを確認。
零一十０３では前段で零一が分かれ、100a90への入力は十０３になる。
そのため零を含む混在数詞だけを修正しても足りず、単位後の先頭ゼロも同じ経路にした。

一般数詞区間内の千＋助数詞は千のTk行を使用する（人/本で平板）。
単独の裸千の既存分岐を区別して保持する。
十九のテーブルReadがキュウではなくキューで終わる場合も、九の読み分けを適用する。
二零十49人のジューヨン／ジュークニンを再現した。
5入力のnative fragment/counter観測とtrace SHA256はdocs/evidence/mixed-rei-unit-branches.json。

二度目の試作では追加64文章×両音声も全128件が読み一致し、通常回帰へ追加した。
TestMixedReiUnitReadingOriginalには結果による除外も誤差許容もない。

独立した通常原版SDKとの一零０３人、一零０３本、一零十67人、一零67十本の
茜4件は289.22秒、葵4件は361.23秒で全WAV一致した（計993140 PCM16サンプル）。
新たに見つけた単位後の先頭ゼロ・千・十九の4文章も、独立通常原版との比較で
茜375.84秒、葵306.39秒で全WAV一致した（計1029065 PCM16サンプル）。
全224件の読み一致を全224件のGo WAV一致とは扱わない。
CPU差と未検証の入力・設定は継続課題。

最終のseed5〜15の2816件と追加224件を含む読み回帰は46.083秒で成功。
numeral/phraseの全回帰は0.750秒/73.222秒で成功、go vetも成功。
ログはprivate/mixed-rei-units-v2-{reading,source}-regression.logとmixed-rei-unit-final-vet.log。

保存原版との全WAV再現を独立した2組（各8件）で追加した：
AOIKANE_MIXED_REI_UNIT_COUNTER_REFERENCE_AUDIT=1と
AOIKANE_MIXED_REI_UNIT_BRANCHES_REFERENCE_AUDIT=1。
それぞれtestdata/mixed_rei_unit_{counter,branches}_wave_original.jsonの全文から
Go合成して全WAV長さ/SHA256を比較する。
保存参照の原版WAV/native reportは再ハッシュ済み。
独立フックなし比較の全16件が完了し、原版とGo双方の保存WAV実ファイルを
レポートと参照の長さ/SHA256に照合した。report_sha256を独立比較4レポートへ更新し、
元の観測時native report SHA256はcapture_report_sha256として保持した。

Linux amd64 CGO=0でもf03cf70のソースから作ったテスト実行ファイルにより、
助数詞8件558.87秒、小単位分岐8件535.82秒で全WAVの長さ/SHA256が一致した。
全文から合成しており、中間tensorの差し替えはない。
ログはprivate/mixed-rei-units-f03cf70-linux-reference.log。
Windowsの独立比較ログはprivate/mixed-rei-unit-{counter,branches}-{akane,aoi}-wave.log。

f03cf70のclean treeから生成したdist/aoikane-purego.exeについても、
Go 1.26.8、CGO_ENABLED=0、GOAMD64=v1、vcs.modified=falseを確認。
--portable embeddedで茜「一零０３本。」を合成し、独立原版の全WAV実ファイルと一致。
221762 bytes、SHA256 `154bf9b1ae2021a678b7fca8cfa2d30d2a7796e028fd8dd5848ef477ff5acf8a`。
ログはprivate/mixed-rei-units-cli-{akane,build-info}.log。
これは上記入力とCPU設定での証明であり、任意文章・任意CPU・全設定の完成を意味しない。
