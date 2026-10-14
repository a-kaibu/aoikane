# 混在数詞の中点と先頭単位

2026-10-13。seed5 case13 `0兆（個本兆九0・十９1八。` の原版はゼロの後に強い区切りを置くが、Goはノを挿入していた。

周辺20文×両声を通常SDKから採取し、全40件の正式終了、DLL SHA256、入力順/入力SHA256、native report SHA256、CP932読み、保存原版WAVのサイズ/SHA256を検証。全入力を通常 `TestMixedPointUnitReadingOriginal` に固定した。

[原版の3入力観測](evidence/mixed-point-unit-akane-original.json)では全stageと数詞enter/leaveのLIFO・戻り値を確認し、独立SDKの整形読みに一致した。`九0・十９1八` / `1・十九` の混在数詞は汎用列になり、`1・十9` の数字種が揃う列は従来の処理を維持する。観測はフロントエンドのみで波形証明ではない。

Goの中点処理で、先頭単位の判定より先に全数詞列の数字種混在を判定するようにした。単位で始まる後半だけを見て番地のノを入れる誤りを直し、既存の郵便・電話・ラベル接頭辞判定は維持する。

[全件比較](evidence/mixed-point-unit-reading-comparison.json)は追加40読みとseed5全128文×両声を保持する。追加40読みは一致、seed5の差は両声とも3→2文に減り、残るcase10・112は未解決。任意入力・全設定・全CPUでの完成は未達。

`TestMixedPointUnitOriginalWAV` は `1・十九。`・`九0・十９1八。`・元case13の3文を両声で通常SDK/Go WAV比較する。両声とも正式終了0・PASS。[全6WAVの証拠](evidence/mixed-point-unit-wav.json)で全SDK/Go WAVと独立採取WAVを再hashし全バイト照合。計972013 PCMサンプルが一致した。Go1.26.8、CGO=0、AMD64 Family25 Model33、threads6、AVX512 convolutions falseでの結果であり、全CPUの証明ではない。

数詞・文章処理の既存テストは通過（文章71.278秒）。全ReadingOriginal回帰も120.747秒で通過し、vetも通過した。Linuxの保存原版全6WAV再生比較を開始している。

Linux amd64の全6原版WAV再生比較も正式終了0・PASS、全WAVサイズ/SHA256一致。[Linuxの証拠](evidence/mixed-point-unit-linux-wav.json)。
