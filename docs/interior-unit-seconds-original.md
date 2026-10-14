# 途中の大単位と秒の結合

2026-10-13。seed3 case124 `８億京5二秒9九5％読むは。` の原版100a90は、秒を直前の二に付けてニビョーの1グループにする。Goは先行整数に大単位が1つでもあれば秒を別グループにしていた。秒の直前が大単位で終わる場合と、途中に大単位があるだけで数字で終わる場合を区別する。後者は通常の助数詞処理で前の数字に秒を結合する。

原版の根拠は[seed3の全3入力観測](evidence/numeral-seed3-residual-akane-original.json)のcase124。数詞dispatch、助数詞引数、全stageと独立SDK formatterとの一致を検証済み。今回追加した16文×両声＝32読みはすべて原版と一致。全キャプチャの正常終了、入力hash、CP932整形読み、native report hash、保存元WAVのサイズとSHA256を検証し、`TestInteriorUnitSecondsReadingOriginal` に全件固定した。

seed3全128文×両声の読み差は3→2文に減り、新しい不一致はない。残るcase11・72を削らず全件比較を保持。[前後の全比較](evidence/interior-unit-seconds-reading-comparison.json)。既存全ReadingOriginal回帰は88.689秒で通過。数詞・句処理source全回帰も通過（句処理60.734秒）。任意文章・全設定・全CPUでの完全一致は未達。

## 波形比較

`1京2秒3。` と `８億京5二秒9九5％読むは。` の2文×両声＝4WAV、630,860 PCMサンプルが通常SDKと全バイト一致。正式な終了PASSと全case結果を検証し、保存原版/Go WAVを独立native採取ファイルにも照合。[全波形証拠](evidence/interior-unit-seconds-wav.json)。変更パッケージのvetも通過。

Go1.26.8 / Windows amd64 / CGO=0、AMD Family25 Model33、原版6スレッド、AVX512畳み込みなし。通常比較は `AOIKANE_TEST_ENGINE` / `TestInteriorUnitSecondsOriginalWAV`。DLL不要の厳密再生は `AOIKANE_INTERIOR_UNIT_SECONDS_REFERENCE_AUDIT=1` / `TestInteriorUnitSecondsReferenceWAV`。seed3の全128文ゲートは残るcase11・72の両声計4読み差で失敗し、全件と失敗ログを保持している。

Linux amd64でも全4件の厳密原版WAV再生が一致して終了（399.45s）。Go1.26.8・CGO=0・portableタグ。[Linux照合](evidence/interior-unit-seconds-linux-wav.json)。
