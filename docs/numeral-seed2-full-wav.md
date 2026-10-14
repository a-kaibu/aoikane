# 数詞seed2の全256WAV比較

2026-10-13。継続実行していた `TestRandomNumeralSeed2OriginalWAV` は葵・茜とも正式終了0・PASS。全128文×両声の通常SDK/Go保存WAVと独立した原版採取WAVを再hashし、入力順、全サイズ/SHA256、全バイトを照合した。全256WAV、計39295450 PCMサンプルが一致した。

[全結果と検証対象hash](evidence/numeral-seed2-full-wav.json)に全入力を保持。読み一致による選別や途中終了はない。AMD64 Family25 Model33、threads6、AVX512 convolutions false、Go1.26.8、CGO=0での比較であり、任意入力・全設定・全CPUの証明ではない。

この比較は後続フロントエンド修正前から実行していた既存バイナリの結果。今回HEADでの全256件再実行として扱わない。テストバイナリのbuild metadataにはvcs.revisionがないため、実ファイルSHA256とbuild infoを保持した。
