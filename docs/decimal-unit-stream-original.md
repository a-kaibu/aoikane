# 小数点後の単位と数字片

2026-10-13。seed3 case11の `十8十八` は、原版100a90で `十` → `８十` → `八` の順に読み、ジューハチジューハチという連結音をアクセント表で再解釈する。Goの小数汎用経路は文字種で先に切り、最後のジューハチを別グループにしていた。単位を含む部分は単位readerへまとめて渡し、単位でリセットされる文字種境界と連結音を再構築する。零の数字対は1043f0で別に扱うため、既存の零経路を保持した。初回の零末尾の差と失敗結果も保存した。

追加16文×両声＝32読みが原版と一致。全キャプチャの正常終了、入力hash、CP932整形読み、native report、保存元WAVの全サイズ・SHA256を検証し、`TestDecimalUnitStreamReadingOriginal` に全件固定。[全比較](evidence/decimal-unit-stream-reading-comparison.json)。[全3入力の原版観測](evidence/decimal-unit-stream-akane-original.json)は数詞enter/leave・助数詞引数・全stageを検証し、独立SDK整形読みに一致した。観測はフロントエンドのみ。

seed3全128文×両声の読み差はcase11を解消し1→0文。全128件を削らず、`TestRandomNumeralSeed3ReadingOriginal` を環境変数なしの通常回帰へ昇格した。最終の既存全ReadingOriginal回帰は132.435秒で通過。数詞・句処理source全回帰も通過（句処理86.691秒）。変更パッケージのvetも通過。読み一致は全256波形・任意文章・全設定・全CPUの一致を意味しない。

## 波形比較

`1．十8十八年。`、`1．十8十零年。`、seed3 case11の3文×両声＝6WAV、1,011,305 PCMサンプルが通常SDKと全バイト一致。正式な終了PASSと全caseのreportを確認し、保存原版/Go WAVを独立native採取ファイルにも照合。[全波形証拠](evidence/decimal-unit-stream-wav.json)。

Go1.26.8 / Windows amd64 / CGO=0、AMD Family25 Model33、原版6スレッド、AVX512畳み込みなし。通常比較は `AOIKANE_TEST_ENGINE` / `TestDecimalUnitStreamOriginalWAV`。DLL不要の厳密再生は `AOIKANE_DECIMAL_UNIT_STREAM_REFERENCE_AUDIT=1` / `TestDecimalUnitStreamReferenceWAV`。プロジェクト全体はまだ未完了。

Linux amd64でも全6件の厳密原版WAV再生が一致して正常終了（790.66s）。Go1.26.8・CGO=0・portableタグ。[Linux照合](evidence/decimal-unit-stream-linux-wav.json)。
