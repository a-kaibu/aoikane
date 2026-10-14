# 比率のタイと促音化

2026-10-14。seed6 case63 `.個8－0AA・５（五0零十。` の原版ハチタイをGoはハッタイにしていた。

通常SDKから周辺20文を両声で採取し、全40件の正式終了、DLL/入力/native report SHA256、入力順、CP932読み、保存元WAVのサイズ/SHA256を検証した。全入力を通常 `TestHyphenRatioReadingOriginal` に固定し、一致した。

[原版の3入力観測](evidence/hyphen-ratio-akane-original.json)は全stage・数詞LIFO・戻り値・独立SDKとの整形読み一致を確認。観測はフロントエンドのみ。原版は1タイ／8タイを促音化せず、10タイはジュッタイにする。

Goの比率接続を既存の対比用Tn=JAAの処理へ接続した。一般の助数詞のTn=HAAによる一／八の促音化をここへ持ち込む誤りを直す。また文字として対を含む列の末尾助数詞も取り込み、8対0本のゼロホンを一緒に処理する。

[全件比較](evidence/hyphen-ratio-reading-comparison.json)に新40読みとseed6全128文×両声を保持。新40読みは一致。seed6は両声とも4→3差になり、case62・92・107が未解決。全件失敗監査は引き続き保持する。任意入力・全設定・全CPUの完全一致は未達。

[全8WAV](evidence/hyphen-ratio-wav.json)は8−0・10−0・8対0本・元case63を両声で通常SDK/Go比較。両ログ末尾PASS、保存全SDK/Go WAVと独立原版採取を再hash・全バイト照合して959102 PCMサンプルが一致した。ツールセッション切替で終了コードは失われたため、ログ末尾とプロセスが存在しないことを確認している。Go1.26.8、CGO=0、AMD64 Family25 Model33、threads6、AVX512 convolutions falseでの結果。全CPUの証明ではない。

数詞・文章処理の既存テスト（文章63.952秒）、全ReadingOriginal回帰（108.886秒）、vetは通過。`AOIKANE_HYPHEN_RATIO_REFERENCE_AUDIT=1` の `TestHyphenRatioReferenceWAV` でLinuxの全8WAV再生比較を開始した。

Linux amd64の全8WAV再生比較も正式終了0・PASS、全8サイズ/SHA256一致。[Linuxの証拠](evidence/hyphen-ratio-linux-wav.json)。
