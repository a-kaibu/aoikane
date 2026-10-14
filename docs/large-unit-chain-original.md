# 大単位を含むアクセント表の語幹

2026-10-13。seed5 case10 `２兆万1円．５は、Ｃ個．は9。` の原版はニチョーマンを1群にするが、Goはニチョー／マンに分けていた。

元DLLの通常SDKから12文＋周辺20文を両声で採取。全64件について正式終了・全入力順序・DLL/入力/native report SHA256・CP932読み・実WAVのサイズ/SHA256を検証した。`TestLargeUnitChainReadingOriginal` は `AOIKANE_LARGE_UNIT_CHAIN_READING_AUDIT=1` で全32文×両声を読み、未解決3文ずつで失敗する。入力の除外はない。

[原版の2入力観測](evidence/large-unit-chain-akane-original.json)は全stage・数詞LIFO・戻り値・独立SDKとの整形読み一致を確認した。観測はフロントエンドのみで、波形証明ではない。

原版のアクセント表はニチョーを独立した語として載せる。Goの音列照合はこれを「ニ＋単位兆」としたまま止めていた。表の語全体を語幹に戻し、次の大単位を1つ付加するようにした。全単位を無条件に連結せず、2億万・8兆万・2兆万兆などは通常の群境界を維持する。独立した表語幹には短いTkの助数詞処理を保持し、2兆万本／2兆万秒も一致した。

[全入力比較](evidence/large-unit-chain-reading-comparison.json)は全64件とseed5全256件を保持。新64件は58件一致、残りはゼロ境界2文と秒・区切り列1文が両声で不一致。00兆万／零兆万はclean d3e0d07の変更前比較にも同じ差がある。seed5の差は両声とも2→1文になり、case112が未解決。任意入力・全設定・全CPUでの完成は未達。

数詞・文章処理の既存テストと全ReadingOriginal回帰、vetは通過。`TestLargeUnitChainOriginalWAV` は2兆万・2兆万本・元case10の3文を通常SDKとGoで両声比較する。`AOIKANE_LARGE_UNIT_CHAIN_REFERENCE_AUDIT=1` の `TestLargeUnitChainReferenceWAV` は保存原版6WAVをDLLなしで再合成し全サイズ/SHA256を照合する。

Windows全6WAVは両声とも正式終了0・PASS。[全波形証拠](evidence/large-unit-chain-wav.json)で全SDK/Go WAVと独立採取WAVを再hashして全バイト照合。計779669 PCMサンプルが一致。Go1.26.8、CGO=0、AMD64 Family25 Model33、threads6、AVX512 convolutions falseでの結果。Linuxの全6WAV再生比較は継続中。全読み回帰は91.582秒、文章処理58.694秒で通過。

Linux amd64の全6WAV再生比較も正式終了0・PASS、全サイズ/SHA256一致。[Linuxの証拠](evidence/large-unit-chain-linux-wav.json)。
