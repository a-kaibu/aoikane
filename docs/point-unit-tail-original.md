# 中点の後ろの単位列

2026-10-13。原版は `19・9万2` を小数として読まず、番地のジュ―キューノ／キューマン／ニとして読む。小数部の文法が数字＋任意の末尾1単位であることを判定し、内部単位や入れ子の点では番地・汎用列へ進む。先頭ゼロ、助数詞、ラベルがあれば番地を選ばず強い区切りを使う。英数字ラベルのnと、単位を含められる漢数字ラベルのJも区別した。電話接頭辞の末尾に単位列があるときは、数字対＋ノの前半を残して単位列を普通に読む。

`０９万２` の先頭ゼロの数字対は万まで1グループに保持する。後半の数字は一般数詞の片であり、二円などのwhole-counter特殊読みを適用しない。初回では漢数字ラベルと円の境界で差が出たため補足8文を採取して修正し、失敗結果も保持した。

追加24文＋補足8文、計32文×両声＝64読みが原版と一致。全キャプチャの正常終了、入力hash、CP932整形読み、native report、保存元WAVの全サイズ・SHA256を検証し、`TestPointUnitTailReadingOriginal` に全件固定した。[全比較](evidence/point-unit-tail-reading-comparison.json)。[原版の全6境界観測](evidence/point-unit-tail-akane-original.json)は数詞enter/leave・助数詞引数・全stageを検証し、独立SDK整形読みに一致した。観測はフロントエンドのみ。

seed3全128文×両声の読み差はcase72を解消し2→1文。新しい不一致はない。残るcase11は小数点の後の単位・数字片を連結してアクセント表で再解釈する経路であり、未解決。全128文を維持した厳密ゲートも残す。最終の既存全ReadingOriginal回帰は90.219秒で通過。数詞・句処理source全回帰も通過（句処理62.574秒）。変更パッケージのvetも通過。

## 波形比較

`19・9万2。`、`19・09万2。`、`電話19・9万2。`、seed3 case72の4文×両声＝8WAV、1,259,762 PCMサンプルが通常SDKと全バイト一致。正式な終了PASSと全caseのreportを確認し、保存原版/Go WAVを独立native採取ファイルにも照合。[全波形証拠](evidence/point-unit-tail-wav.json)。波形バイナリは最後の漢数字ラベル・特殊読み修正前にコンパイルし、その後の最終コードは全64読みと既存全読み回帰で確認した。

Go1.26.8 / Windows amd64 / CGO=0、AMD Family25 Model33、原版6スレッド、AVX512畳み込みなし。通常比較は `AOIKANE_TEST_ENGINE` / `TestPointUnitTailOriginalWAV`。DLL不要の厳密再生は `AOIKANE_POINT_UNIT_TAIL_REFERENCE_AUDIT=1` / `TestPointUnitTailReferenceWAV`。任意文章・全設定・全CPUでの完全一致は未達。

最後の漢数字ラベル・特殊読み修正を含むclean 2e61084の分離した純Go CLIで、`図十九・九万二。` と `０９万２円。` を葵で合成し、通常SDKの保存原版WAV全体に一致した（279,370 / 208,088 bytes）。CGO=0・portableタグ・vcs.modified=falseを確認。[最終CLI照合](evidence/point-unit-tail-final-cli-wav.json)。全128文厳密ゲートはcase11の両声計2読み差で失敗することも再確認した。

Linux amd64でも全8件の厳密原版WAV再生が一致して正常終了（803.44s）。Go1.26.8・CGO=0・portableタグ。[Linux照合](evidence/point-unit-tail-linux-wav.json)。
