# ハイフン列の内部にある秒

2026-10-13。seed5最後のcase112 `9−千二兆千秒8兆1時。` は原版がセンビョーの表アクセントとハッチョーを保つのに、Goは秒で列を切り後半の数詞処理を失っていた。

通常SDKから周辺20文・ゼロ境界12文・助数詞8文を両声で採取。全80件の正式終了、DLL/入力/native report SHA256、入力順、CP932整形読み、実原版WAVの全サイズ/SHA256を検証した。全40文×両声を通常 `TestConnectorSecondsReadingOriginal` に固定して一致した。

[原版の3入力観測](evidence/connector-seconds-akane-original.json)で全stage・数詞LIFO・戻り値・独立SDKとの整形読み一致を確認。原版100a90への入力は９－千二兆千秒８兆１の全列で、助数詞は最後の時だけに付く。観測はフロントエンドのみで、波形証明ではない。

Goではハイフン列に続く秒と数詞を同じ汎用列へ取り込む。秒前の数詞は既存の秒接続処理を共用し、末尾助数詞は全列に適用する。末尾が1桁の場合は汎用の語キー、複数桁の数字の場合は数詞の最終キーを保持するため、8本はハチホン、03本／13本はサンボンになる。通常の小数秒や範囲の既存処理は維持する。

[全件比較](evidence/connector-seconds-reading-comparison.json)は新80読みとseed5全256読みを保持してすべて一致。seed5の差は両声とも1→0文になり、全128文×両声の `TestRandomNumeralSeed5ReadingOriginal` を環境変数なしの通常回帰へ昇格した。seed5全128文の通常SDK/Go波形比較 `TestRandomNumeralSeed5OriginalWAV` も追加したが、全256WAVの完走は未実施。任意入力・全設定・全CPUでの完全一致は未達。

数詞・文章処理の既存テストは通過（文章64.663秒）。全ReadingOriginal回帰と、関連4文×両声の通常SDK/Go全WAV比較を実行中。Linuxの保存原版8WAVを使う `AOIKANE_CONNECTOR_SECONDS_REFERENCE_AUDIT=1` / `TestConnectorSecondsReferenceWAV` も開始した。

大単位の全32文×両声の厳密監査も再実行し、chain12は全件一致、boundary20は既知の00兆万／零兆万が両声で残る計4読み差で失敗することを確認した。入力を除外せず保持しており、seed5の読み一致は任意入力の完成を意味しない。

Windowsの全8WAVは両声とも正式終了0・PASS。[全波形証拠](evidence/connector-seconds-wav.json)で全SDK/Go WAVと独立採取WAVを再hash・全バイト照合し、計1111754 PCMサンプルが一致。Go1.26.8、CGO=0、AMD64 Family25 Model33、threads6、AVX512 convolutions falseでの結果であり、全CPUの証明ではない。全ReadingOriginal回帰は115.688秒で通過し、vetも通過した。Linuxの全8WAV比較は継続中。

Linux amd64の全8WAV再生比較もログ末尾PASS、全8サイズ/SHA256一致を確認。[Linuxの証拠](evidence/connector-seconds-linux-wav.json)。
