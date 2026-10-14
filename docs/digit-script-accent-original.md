# 数字の文字種境界のアクセント

2026-10-13。原版の `80五` はハチジュ!ー／ゴとなる。Goは通常整数の内部継続と同じFlag処理を使い、文字種が変わる境界でもハチジューを平板にしていた。Jap/Engの数字片の最後に既存のfragment boundaryを設定し、そこで数詞片の表アクセントを維持する。通常の同一文字種の整数内の平板化は続ける。

追加16文×両声＝32読みが原版と一致。seed4全128文×両声の原版採取も完了し、全288キャプチャの正常終了、入力hash、CP932、native report、保存元WAVのサイズ・SHA256を検証した。32読みは `TestDigitScriptAccentReadingOriginal` に通常回帰として固定した。[全件の前後比較](evidence/digit-script-accent-reading-comparison.json)。seed4の読み差はcase127を解消し3→2文、両声とも新しい差はない。

[seed4の全3失敗入力の原版観測](evidence/numeral-seed4-residual-akane-original.json)は数詞enter/leave・助数詞引数・全stageを検証し、独立SDK整形読みに一致した。観測はフロントエンドのみ。残るcase8は汎用横棒列の先頭ゼロ、case26は小数点後の裸の千と分のアクセントに関係する。全128文を保持した `AOIKANE_NUMERAL_SEED4_READING_AUDIT=1` / `TestRandomNumeralSeed4ReadingOriginal` は、この両声計4読み差で失敗する。失敗例を削除・置換していない。

既存全ReadingOriginal回帰は105.247秒で通過。数詞・句処理source全回帰も通過（句処理70.837秒）。変更パッケージのvetも通過。

## 波形比較

`80五。` とseed4 case127 `時時8０五。` の2文×両声＝4WAV、383,520 PCMサンプルが通常SDKと全バイト一致。正式な終了PASSと全case結果を確認し、保存原版/Go WAVを独立native採取ファイルにも照合。[波形証拠](evidence/digit-script-accent-wav.json)。

Go1.26.8 / Windows amd64 / CGO=0、AMD Family25 Model33、原版6スレッド、AVX512畳み込みなし。通常比較は `AOIKANE_TEST_ENGINE` / `TestDigitScriptAccentOriginalWAV`。DLL不要の厳密再生は `AOIKANE_DIGIT_SCRIPT_ACCENT_REFERENCE_AUDIT=1` / `TestDigitScriptAccentReferenceWAV`。任意文章・全設定・全CPUでの完全一致は未達。

[コネクタ内の裸の千の修正](connector-bare-thousand-original.md)でcase26を解消し、seed4の残る差は両声ともcase8の1文になった。

Linux amd64でも全4件の厳密原版WAV再生が一致して終了（223.29s）。Go1.26.8・CGO=0・portableタグ。[Linux照合](evidence/digit-script-accent-linux-wav.json)。

[汎用列の先頭ゼロ修正](generic-zero-stream-original.md)でcase8も解消し、seed4全128文×両声の読みは全件一致した。全件ゲートを通常回帰へ移した。
