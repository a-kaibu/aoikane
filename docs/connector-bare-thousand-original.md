# コネクタ内の裸の千と助数詞

2026-10-13。原版の `9.千分` はセンプンが平板になるが、Goは単独の千と同じ調整でセン!プンにしていた。原版100a90/10b010が再構築した裸の単位は数詞片の助数詞モードを保持するため、コネクタのfragment readerで単位のnumericFragmentを設定する。単独の `千分` は従来の別調整を維持する。

追加境界 `9.千・2分` は、既に小数点を含む前半へ中点を結合して小数に変換しない。汎用の強い区切りを選び、センテンという余分なテンを加えない。

全16文×両声＝32読みが原版と一致。全キャプチャの正常終了、入力hash、CP932、native report、保存原版WAVのサイズ・SHA256を検証し、`TestConnectorBareThousandReadingOriginal` に固定。[全件比較](evidence/connector-bare-thousand-reading-comparison.json)。[原版の全3境界観測](evidence/connector-bare-thousand-akane-original.json)は数詞enter/leave・助数詞引数・全stageを検証し、独立SDK整形読みに一致した。観測はフロントエンドのみ。

seed4全128文×両声の読み差はcase26を解消し2→1文。新しい不一致はない。残るcase8は横棒列の先頭ゼロで未解決。全128文を維持した厳密ゲートは両声計2読み差で失敗し、失敗ログも保持した。最終の既存全ReadingOriginal回帰は99.536秒で通過。数詞・句処理source全回帰も通過（句処理63.182秒）。変更パッケージのvetも通過。

## 波形比較

`9.千分。`、`9.千・2分。`、seed4 case26の3文×両声＝6WAV、713,822 PCMサンプルが通常SDKと全バイト一致。全終了PASSと全case結果を確認し、保存原版/Go WAVを独立native採取ファイルにも照合。[全波形証拠](evidence/connector-bare-thousand-wav.json)。

Go1.26.8 / Windows amd64 / CGO=0、AMD Family25 Model33、原版6スレッド、AVX512畳み込みなし。通常比較は `AOIKANE_TEST_ENGINE` / `TestConnectorBareThousandOriginalWAV`。DLL不要の厳密再生は `AOIKANE_CONNECTOR_BARE_THOUSAND_REFERENCE_AUDIT=1` / `TestConnectorBareThousandReferenceWAV`。任意文章・全設定・全CPUの完全一致は未達。

Linux amd64でも全6件の厳密原版WAV再生が一致して終了（409.60s）。Go1.26.8・CGO=0・portableタグ。[Linux照合](evidence/connector-bare-thousand-linux-wav.json)。
