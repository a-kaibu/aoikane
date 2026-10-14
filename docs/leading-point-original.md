# 先頭点の汎用数字列

2026-10-13。原版は `．5` の点を数詞に含めないが、`．5．2` は先頭のテンを残し、1043f0の数字対で読む。Goは先頭点を捨て通常の小数として読み、ゴーテンという長音にしていた。先頭点＋後続点を収集し、短い単独数字・数字対・アクセント表のFlagに従ってテンを付ける。混在する文字種は先頭点を外す。単位列が数字対の文法を外れるときも先頭点を除いて既存の汎用数詞経路へ戻す。末尾単位や中点との組合せも原版観測から再構築した。

[原版の全6観測](evidence/leading-point-akane-original.json)は数詞enter/leaveと全stageを検証し、独立SDK整形読みに一致した。フロントエンド観測であり、波形証拠は別に保存する。

追加32文×両声＝64読みが原版と一致。全キャプチャの正常終了、入力hash、CP932、native report、保存元WAVの全サイズ・SHA256を確認し、通常回帰 `TestLeadingPointReadingOriginal` に全件固定した。seed2全128文×両声ではcase95を解消し、新しい不一致はなく、各声2→1文の差になった。[全比較](evidence/leading-point-reading-comparison.json)。残るcase7は `京・9千万円2兆零十、。` の大単位・助数詞アクセント・零の整数片であり未解決。全件厳密読みゲートはこの計2読み差で失敗し、失敗ログも保持している。

初回の単位列fallback漏れで既存の `.9.億` とseed2 case102に回帰が出た。失敗ログを残して修正し、最終の全ReadingOriginal回帰は91.882秒で通過。数詞・句処理全source回帰も通過（句処理67.410秒）。変更パッケージのvetも通過。

## 波形比較

`．5．2。`、`．12．34。`、`．万．2。`、`.５．2個秒．9月。` の4文×両声＝8WAV、901,389 PCMサンプルが通常SDKと全バイト一致。終了PASSと全case結果を確認し、保存原版/Go WAVは独立した通常SDK採取ファイルとも一致。[全波形証拠](evidence/leading-point-wav.json)。波形比較のバイナリは最後の単位列fallback修正前にコンパイルしており、その後の最終コードは全64読みと既存全読み回帰で検証した。

Go1.26.8 / Windows amd64 / CGO=0、AMD Family25 Model33、原版6スレッド、AVX512畳み込みなし。通常比較は `AOIKANE_TEST_ENGINE` / `TestLeadingPointOriginalWAV`。DLL不要の厳密参照再生は `AOIKANE_LEADING_POINT_REFERENCE_AUDIT=1` / `TestLeadingPointReferenceWAV`。任意文章・全設定・全CPUの完全一致は未達。

最終fallback修正を含むclean 94c5539の分離した純Go CLIでも、seed2 case95を葵で合成し、通常SDKの保存WAV全体と一致した（296,092 bytes）。CGO=0、portableタグ、vcs.modified=falseを確認。[最終CLIの照合](evidence/leading-point-final-cli-wav.json)。

後続の[大単位と零の修正](large-zero-counter-original.md)でcase7も解消し、seed2全128文×両声の整形読みは全件一致した。全件ゲートを通常回帰へ移した。

Linux amd64でも全8件の厳密原版WAV再生が一致して正常終了（513.54s）。Go1.26.8・CGO=0・portableタグ。[Linux証拠](evidence/leading-point-linux-wav.json)。
