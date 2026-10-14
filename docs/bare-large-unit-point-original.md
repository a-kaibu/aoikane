# 区切り列の裸の大単位と小数点

2026-10-14。seed6 case92 `億５－万．〇０.。` の原版はマンとテンを分けるが、Goはマンテンへ結合していた。

通常SDKから裸単位と小数点20文、右側の小数範囲8文、電話の点区切り8文、電話の単位境界4文を両声で採取。全80件の正式終了、DLL/入力/native report SHA256、入力順、CP932読み、保存原版WAVのサイズ/SHA256を検証した。最初の36文×両声は通常 `TestBareLargeUnitPointReadingOriginal` で全件一致。単位境界4文は `AOIKANE_DOTTED_PHONE_UNIT_READING_AUDIT=1` / `TestDottedPhoneUnitReadingOriginal` に全件を保持し、電話1−万.03本の濁音が両声で残る計2差で失敗する。変更前clean68b7731にも同じ濁音差がある。

[原版case92の観測](evidence/bare-large-unit-point-akane-original.json)は全stage・数詞LIFO・内部戻り値・独立SDKとの整形読み一致を検証。億５－万．〇０の全列が100a90へ渡る。観測はフロントエンドのみで波形証明ではない。

Goでは汎用の区切り列にあることを数詞へ伝え、裸の大単位でもテンを別語にする。単独の万.〇０は従来のマンテンを保持する。また数字種が揃うハイフン列は右側が小数の場合も範囲のカラで処理する。電話接頭辞があり最初が数字の列は点を番号区切りとして先に処理し、混在数字種や中間の裸単位も維持する。電話万−2.3のように最初が単位なら通常の範囲読みを保つ。

[全比較](evidence/bare-large-unit-point-reading-comparison.json)に新80件とseed6全256件を保持。seed6は両声とも2→1文の差になりcase107が残る。新しく採取した万.03本の濁音差、既知のゼロ大単位2文、任意入力・全設定・全CPUの完全一致は未達。

現行の数詞・文章処理の既存テストは通過（文章87.975秒）。全読み回帰と、関連6文×両声の通常SDK/Go全WAV比較を実行中。

現行の全ReadingOriginal回帰は147.386秒で通過、vetも通過した。追加36文×両声の72読みは全件一致。関連6文×両声の全WAV比較とLinux再生比較は継続中。

Windowsの全12WAVは両声とも正式終了0・PASS。[全波形証拠](evidence/bare-large-unit-point-wav.json)で全SDK/Go WAVと独立原版採取を再hash・全バイト照合して計1566671 PCMサンプルが一致した。Go1.26.8、CGO=0、AMD64 Family25 Model33、threads6、AVX512 convolutions falseでの結果。Linuxの全12WAV再生比較は継続中で、全CPUの証明ではない。

Linux amd64の全12WAV再生比較もログ末尾PASS、全12サイズ/SHA256一致を確認。[Linuxの証拠](evidence/bare-large-unit-point-linux-wav.json)。

2026-10-14。[範囲小数の助数詞再構築](range-fraction-counter-original.md)で電話1−万.03本の差も解消。単位境界全4文×両声を通常回帰へ昇格した。変更前の全結果は保持する。
