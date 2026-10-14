# 大単位と助数詞の平板アクセント

2026-10-13。原版の `京個年` は `ケイカネン / 2-0-5` で平板になる。Goは `Tk=1` をC3助数詞だけに適用し、C1/C2では助数詞先頭に落ちを付けていた。助数詞アクセントのC1/C2分岐でも、ルール表のTk=1を平板として扱う。個年だけの文字列例外を追加せず、大単位や係数と助数詞のクラスを使う。

## 原版との比較

個年・カ年・か年と万〜垓、単位の前の数字を含む32文、C2クラスの亜型・泳法を含む12文、計44文×両声＝88読みが原版と一致した。全キャプチャの正常終了、入力hash、CP932整形読み、独立native report、保存元WAVの全サイズ・SHA256を確認し、通常回帰 `TestCounterFlatUnitReadingOriginal` に全件を固定した。

[原版観測](evidence/counter-flat-unit-akane-original.json)は3文すべての数詞dispatch・助数詞引数・全stageを検証し、独立SDK整形読みに一致した。フロントエンドのみの観測であり、波形比較は別に実施する。

seed2全128文×両声ではcase121を解消し、各声3→2文の差になった。残るcase7・95は大単位の整数片・助数詞アクセントと先頭小数点の分岐に関係し、未解決。追加residual32では各声15→10文の差になった。入力を削除せず、新しい不一致がないことを確認した。[全件の前後比較](evidence/counter-flat-unit-reading-comparison.json)。counter-flatの変更前の比較は既存のclean 7f126db comparator、residual32/seed2の変更前は4a30b95で保存した結果を用いる。

既存全ReadingOriginal回帰は90.797秒で通過。数詞・句処理の全source回帰も通過（句処理66.673秒）。変更パッケージのvetも通過。

任意文章・全設定・全CPUの完全一致はまだ未達。

## 波形の比較

`京個年。`、`2京個年。`、`21京個年。`、`京泳法。` の4文×両声＝8WAV、709,202 PCMサンプルが通常SDKと全バイト一致。全テストの終了PASS、全caseのサイズ・SHA256・最初の差分位置を検証し、保存原版/Go WAVは独立した通常SDKキャプチャの実ファイルとも一致した。[波形の全証拠](evidence/counter-flat-unit-wav.json)。

Go1.26.8 / Windows amd64 / CGO=0、AMD Family25 Model33、原版6スレッド、AVX512畳み込みなし。通常SDK比較は `AOIKANE_TEST_ENGINE` / `TestCounterFlatUnitOriginalWAV`。DLLなしの厳密参照再生は `AOIKANE_COUNTER_FLAT_UNIT_REFERENCE_AUDIT=1` / `TestCounterFlatUnitReferenceWAV`。参照 `internal/portable/testdata/counter_flat_unit_wave_original.json`。seed2全128文の厳密読みゲートは残るcase7・95の計4読み差で失敗し、失敗ログも保持している。

後続の[先頭点修正](leading-point-original.md)でcase95を解消し、seed2の残る差は両声ともcase7の1文になった。

Linux amd64でもGo1.26.8・CGO=0のportableバイナリで全8WAVが厳密原本参照に一致して終了（361.41s）。[Linux検証](evidence/counter-flat-unit-linux-wav.json)。
