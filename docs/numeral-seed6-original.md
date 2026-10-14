# 数詞seed6の全入力比較

2026-10-13。固定生成器seed6・count128・max_units16の全入力を葵・茜から通常SDKで採取。全256件の正式終了、DLL/入力/native report SHA256、全入力順、CP932整形読み、実WAVのサイズ/SHA256を検証した。

clean 68b7731、Go1.26.8、CGO=0、vcs.modified=falseの比較バイナリを固定し全256読みを比較。両声ともcase62・63・92・107の4文に差を発見。[全件比較とバイナリ情報](evidence/numeral-seed6-reading-baseline.json)に失敗入力を含めて保持。`AOIKANE_NUMERAL_SEED6_READING_AUDIT=1` の `TestRandomNumeralSeed6ReadingOriginal` は全128文×両声を読み、計8差で失敗することを確認した。入力の除外・置換はない。

- case62 `９千百千零Ａ千.−、千。`：原版のキューセンをGoはキュー／センへ分ける。
- case63 `.個8－0AA・５（五0零十。`：原版のハチタイをGoはハッタイにする。
- case92 `億５－万．〇０.。`：原版のマンとテンをGoはマンテンへ結合する。
- case107 `Ａ９Aの５十読む８万・2）八時。`：原版はハチマンの後を強い区切りにするが、Goはテンを加える。

原因と原版内部の処理は未調査。原版WAV採取の完全性を検証した結果であり、Goのseed6全WAV一致を意味しない。任意入力・全設定・全CPUの完全一致は未達。

## 比率のタイ接続修正

2026-10-14。[比率のタイ接続](hyphen-ratio-original.md)でcase63の読みと両声の全WAVが一致した。全128文×両声の比較を保持し、差はcase62・92・107の3文ずつ。

## 小単位と後続零の修正

2026-10-14。[小単位と後続零の再構築](small-unit-zero-original.md)でcase62の読みと両声の全WAVが一致した。全128文×両声を保持し、残る差はcase92・107の2文ずつ。

## 裸単位と小数点の修正

2026-10-14。[裸単位と小数点の再構築](bare-large-unit-point-original.md)でcase92の読みと両声の全WAVが一致した。全128文×両声を保持し、残る読み差はcase107の1文ずつ。

## 括弧の混在区切り列修正

2026-10-14。[括弧の混在区切り列再構築](bracket-connector-stream-original.md)でcase107の読みと両声の全WAVが一致した。全128文×両声の読み差はゼロになり、全件を通常回帰へ昇格。seed6全256WAVはまだ完走していない。
