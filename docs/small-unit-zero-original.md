# 小単位の後に続く零の数詞処理

2026-10-14。seed6 case62 `９千百千零Ａ千.−、千。` のGoはキュー／センに分けていた。調査すると数詞の列全体を処理できず、辞書語の読みへ戻っていた。

通常SDKから周辺20文・先頭ゼロ8文・複数桁の係数8文を両声で採取し、全72件の正式終了、DLL/入力/native report SHA256、全入力順、CP932読み、実原版WAVのサイズ/SHA256を検証。全36文×両声を通常 `TestSmallUnitZeroReadingOriginal` に固定して一致した。

[原版の3入力観測](evidence/small-unit-zero-akane-original.json)は全stage・数詞LIFO・独立SDKとの整形読み一致を検証。原版は９千百千を10b010、零を1043f0へ渡す。零を含む2入力の100a90戻り値は0、その他の内部戻り値は1だった。戻り値をそのまま保持し、呼び出し側の意味は断定していない。最終の独立SDK読みには正しいキューセンが残っている。観測はフロントエンドのみで、波形証明ではない。

Goでは既存の「大単位の後に零」の分割を十百千にも広げた。前半を整数読みにし、後半を零の読みで処理する。前半にも零が後ろにある文脈を伝え、01千／09千の数字対と小単位、複数桁の係数の区切りを保つ。01千零はゼロイチ／セン／レーとなり、整数の位に変換してイチを失うことを防ぐ。

[全件比較](evidence/small-unit-zero-reading-comparison.json)は新72読みとseed6全128文×両声を保持。新72読みは一致、seed6の差は両声とも3→2文になりcase92・107が未解決。ゼロ大単位の既知2文も引き続き残る。任意入力・全設定・全CPUの完全一致は未達。

数詞・文章処理の既存テストは通過（文章62.419秒）。全ReadingOriginal回帰はテスト追加途中の原版入力ハッシュ不一致で失敗したため、現行ファイルの入力hash照合と追加72読みの通過を確認してから再実行している。通常SDK/Goの関連4文×両声の全WAV比較も実行中。

現行ファイルで再実行した全ReadingOriginal回帰は139.892秒で通過し、vetも通過。Windowsの全8WAVは両声とも正式終了0・PASS。[全波形証拠](evidence/small-unit-zero-wav.json)で全SDK/Go WAVと独立採取を再hash・全バイト照合し、計1005451 PCMサンプルが一致した。Go1.26.8、CGO=0、AMD64 Family25 Model33、threads6、AVX512 convolutions falseでの結果であり、全CPUの証明ではない。Linuxの全8WAV再合成は継続中。

Linux amd64の全8WAV再生比較もログ末尾PASS、全8サイズ/SHA256一致を確認。[Linuxの証拠](evidence/small-unit-zero-linux-wav.json)。
