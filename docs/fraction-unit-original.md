# 小数部の数字対と末尾単位

2026-10-13。新しい固定seed2の数詞128文×両声を通常SDKへ照合すると、両声とも10文の読み差があった。全件を採取・保存し、各WAV・原版report・DLL・コーパスSHA-256を再検証した。

今回修正したのは `89.52京` 型の小数部。原版は `1043f0` の数字対処理を通り、`５２京` を `ゴーニケイ` と読む。Goは小数部の単位許可を1桁に限定しており、汎用整数経路で `ゴジュー / ニケイ` になっていた。複数桁にも末尾の単位を許可し、前半は数字対＋s0、単位直前だけ短い数字の読みを使う。

末尾の音列にはアクセント表の最長先頭一致とモーラ数、数字対・単位接続のフラグを適用する。`52千` は `ゴー / ニセン`、`12千` は表の `イチニセン`、`52兆` は `ゴー / ニチョー`。`12兆` は `イチニチ` のモーラ数が先に適用されて末尾 `ー` が別群になり、元エンジンのこの出力も保持する。

小数処理の生の数字＋単位と、整数部に単位や文字通りの零がある場合の汎用経路を分ける。前者の `89.8百` は `ハチ / ヒャク`、後者の `万．八百` は `ハッピャク`。後者の促音・濁音処理を変更しない。英数字の単位付き整数部は小数表の lower-case n から外れ、汎用整数経路に残る。日本語の `一万．五二京` は整数部に付いた大単位とテンを別群に保持する。

## 原版比較

追加11文、単位8種×桁組6種の48文、濁音・促音になる数字と助数詞を含む60文、計119文×両声＝238件が整形読みまで一致。すべて通常回帰 `TestFractionUnitReadingOriginal` に固定した。[全比較](evidence/fraction-unit-reading-comparison.json)。原版11入力の全数詞呼出しと繰返しステージを保持し、最終formatterを独立の通常SDK採取へ照合した。[原版の処理](evidence/fraction-unit-akane-original.json)。

`89.52京。`、`89.52千。`、`89.12兆。`、`A−年２月１日８9.５2京読む。` の4文×両声＝8WAV、1,295,446 PCMサンプルがフックなし通常SDKと全バイト一致。保存した原版/Goファイルとreportを再ハッシュし、別採取のSDK WAVそのものにも一致した。参照 `internal/portable/testdata/fraction_unit_wave_original.json`。茜463.98秒、葵484.44秒で正常終了。Go1.26.8 / Windows amd64 / CGO=0、AMD Family 25 Model 33、元処理6スレッド、AVX512畳み込みなし。

source全回帰と既存＋追加の全ReadingOriginal回帰が通過（後者100.916秒）。変更パッケージのvetも通過。`AOIKANE_TEST_ENGINE` と `TestFractionUnitOriginalWAV` で通常SDKと比較する。DLL不要の厳密参照再生は `AOIKANE_FRACTION_UNIT_REFERENCE_AUDIT=1` と `TestFractionUnitReferenceWAV`。

## 未完了

seed2はcase48が両声で改善し、新しい不一致はない。残りはcase7・35・73・90・91・95・111・119・121の9文×両声。全128文の採取はフィルタせず `tools/compare_reading/testdata/random-numeral-seed2-{akane,aoi}-original.json` に保存した。`AOIKANE_NUMERAL_SEED2_READING_AUDIT=1` の `TestRandomNumeralSeed2ReadingOriginal` は全件を照合し、現時点ではこの18読み差で失敗することも確認した。通常テストで成功率や完成度へ読み替えない。

残る9文の原版数詞dispatch・読選択・前アクセント処理も全呼出しを保存し、独立SDK formatterへ検証済み。[残差の処理観測](evidence/seed2-residual-akane-original.json)。これ自体はGo一致の証明ではない。次の対象は中点と後続コネクタの継続範囲、先頭小数点、単位整数片、助数詞のアクセント。別CPU・任意文章・全設定の完全一致も未達。

Linux amd64 / CGO=0で同じ厳密8WAV再生も全件正常終了した（WSLのディスクがread-only fallbackになった際の警告とGoの正常終了を区別し、8件のwhole-WAV比較・各case PASS・最終PASSを確認）。
