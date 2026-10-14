# 数詞の途中の算用数字係数

2026-10-13。原版100a90は `21百990` を `２１` → `百` → `９９０` の3片に分け、`ニジューイチｂヒャクキューヒャクキュージュー` を生成する。Goは最初の係数を百の位へ繰り上げてしまい、ニセンから読んでいた。末尾が単位の `21百` は別の先行分岐で `２１００` に展開し、この場合のニセンヒャクは正しい。

内部の小単位の前にある複数桁の算用数字を独立した整数片として読み、区切りの前の表アクセントを保持する。先頭ゼロと「零」を含む列は、既存の数字対・アクセント経路を保つ。初回実装で既存の零と先頭ゼロに回帰が出たため修正し、失敗ログも `private/interior-coefficient-v1-reading.log` に保存した。

[原版の処理観測](evidence/interior-coefficient-akane-original.json)は全3入力の数詞呼出し・全stage数・整形読みを検証し、独立した通常SDKキャプチャと一致した。フロントエンド観測であり、波形の証明は別に行う。

## 読みの比較

追加32入力×両声の全64原版キャプチャを保存し、元WAVのサイズ・SHA256とCP932整形読みを再検証した。今回の内部係数分岐では9入力×両声の差を解消。末尾の単位展開も含めた10入力×両声を `TestInteriorCoefficientReadingOriginal` の通常回帰に固定した。全32入力の比較も保持し、残る15文×両声の差を隠さない。厳密全件ゲートは `AOIKANE_RESIDUAL_NUMERAL_READING_AUDIT=1` / `TestResidualNumeralReadingOriginal`。

seed2の全128入力×両声ではcase35を解消し、各声4→3文の差になった。新たな不一致はない。[全件の前後比較](evidence/interior-coefficient-reading-comparison.json)。残るcase7・95・121は大単位・助数詞のアクセントと先頭小数点の分岐に関係し、引き続き未完了。

最終修正の既存全ReadingOriginal回帰は98.354秒で通過。数詞・句処理のsource全回帰も通過（句処理63.247秒）。変更パッケージのvetも通過。

## 波形の比較

`21百990。`、`21千9。`、`11百990。` の3文×両声＝6WAV、809,054 PCMサンプルが通常SDKと全バイト一致した。終了した全テストのPASS、全caseのサイズ・SHA256・差分位置を検証し、保存した原版/Go WAVを独立したSDKキャプチャの実ファイルとも比較した。[全波形の証拠](evidence/interior-coefficient-wav.json)。

Go1.26.8 / Windows amd64 / CGO=0、AMD Family25 Model33、原版6スレッド、AVX512畳み込みなし。通常比較は `AOIKANE_TEST_ENGINE` / `TestInteriorCoefficientOriginalWAV`。DLLなしの厳密参照再生は `AOIKANE_INTERIOR_COEFFICIENT_REFERENCE_AUDIT=1` / `TestInteriorCoefficientReferenceWAV`。参照は `internal/portable/testdata/interior_coefficient_wave_original.json`。任意文章・全設定・全CPUの完全一致は未達。

後続の[助数詞アクセント修正](counter-flat-unit-original.md)でcase121を解消し、seed2の残る差は両声ともcase7・95の2文になった。

Linux amd64でも全6件の厳密原本WAV再生が一致して正常終了（529.77s）。Go1.26.8・CGO=0・portableタグ。[Linux検証記録](evidence/interior-coefficient-linux-wav.json)。
