# 汎用区切り列の先頭ゼロ

2026-10-13。seed4 case8の原版100a90は、後半の `０１２` を10b010に渡してゼロ／ジューニと読む。Goの汎用列は後半を通常の数字対として読んでゼロイチ／ニにしていた。汎用列の最後をconnector fragmentへ渡し、数値として読む後半を再現する。前半の先頭ゼロは1043f0の数字対を維持し、区切り前の単独二・五の長音を保持する。`012-一` はゼロイチ／ニーであり、`一-012` の後半とは異なる。

追加17文＋長音境界6文、計23文×両声＝46読みが原版と一致。全キャプチャの正常終了、入力hash、CP932、native report、保存元WAVのサイズ・SHA256を検証し、`TestGenericZeroStreamReadingOriginal` に固定。[全比較](evidence/generic-zero-stream-reading-comparison.json)。[原版の3境界観測](evidence/generic-zero-stream-akane-original.json)は数詞enter/leave・助数詞引数・全stageを検証し、独立SDK整形読みに一致した。seed4の元入力の観測は[全3残差観測](evidence/numeral-seed4-residual-akane-original.json)にも保持する。観測はフロントエンドのみ。

seed4全128文×両声の読み差はcase8を解消し1→0文。全128件を削らず、`TestRandomNumeralSeed4ReadingOriginal` を環境変数なしの通常回帰へ昇格した。最終の全ReadingOriginal回帰は96.594秒で通過。数詞・句処理source全回帰も通過（句処理59.976秒）。変更パッケージのvetも通過。全256波形・任意入力・全設定・全CPUでの一致は未証明。

## 波形比較

`一-012本。`、`012-一。`、seed4 case8の3文×両声＝6WAV、868,381 PCMサンプルが通常SDKと全バイト一致。全終了PASSと全case結果を確認し、保存原版/Go WAVを独立native採取ファイルにも照合。[全波形証拠](evidence/generic-zero-stream-wav.json)。

Go1.26.8 / Windows amd64 / CGO=0、AMD Family25 Model33、原版6スレッド、AVX512畳み込みなし。通常比較は `AOIKANE_TEST_ENGINE` / `TestGenericZeroStreamOriginalWAV`。DLL不要の厳密再生は `AOIKANE_GENERIC_ZERO_STREAM_REFERENCE_AUDIT=1` / `TestGenericZeroStreamReferenceWAV`。プロジェクト全体は未完了。

Linux amd64でも全6件の厳密原版WAV再生が一致して正常終了（594.39s）。Go1.26.8・CGO=0・portableタグ。[Linux照合](evidence/generic-zero-stream-linux-wav.json)。
