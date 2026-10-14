# 数詞seed7の全入力比較

2026-10-14。固定生成器seed7・count128・max_units16の全入力を葵・茜の通常SDKから採取。正式終了、DLL・入力・native report SHA256、全入力順、CP932整形読み、実WAVのサイズ/SHA256を全256件で検証した。

clean 956b333、Go1.26.8、CGO=0、vcs.modified=falseの比較バイナリで全256読みを比較。両声とも次の3入力に差がある。[全件結果と比較バイナリ情報](evidence/numeral-seed7-reading-baseline.json)を保持し、入力を除外・置換しない。

- case12 `千0五−九零八日9．二一百。`：原版のハチニチにはアクセントがあるがGoは平板化する。
- case62 `、零億兆）（分零・零A億零〇兆零。`：後半の零を含む単位列と句の区切りが異なる。
- case103 `.〇・億の億分２日〇９の。`：原版は最初のゼロの後に区切りを置くが、Goはノを加える。葵では後続の区切りも異なる。

全128文×両声を `AOIKANE_NUMERAL_SEED7_READING_AUDIT=1` の `TestRandomNumeralSeed7ReadingOriginal` に保持する。原因となる内部処理は未調査。原版音声採取の完全性は検証したが、Goの全256WAV一致を意味しない。任意入力・全設定・全CPUの完全一致は未達。

2026-10-14。[零のペア読みと日数アクセント](rei-days-original.md)でcase12の読みと両声の全WAVが一致した。全128文×両声を保持し、読みの差はcase62・103の2文ずつ。

2026-10-14。[単位の後ろの零とペア読み](rei-zero-unit-tail-original.md)でcase62の読みと両声の全WAVが一致した。全128文×両声を保持し、残る読み差はcase103の1文ずつ。
