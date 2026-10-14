# 括弧を含む混在区切り列

2026-10-14。seed6最後のcase107 `Ａ９Aの５十読む８万・2）八時。` は原版が8万・2）八の全列を処理するが、Goは括弧の前で切り、中点をテンにしていた。

通常SDKから混在区切り20文、先頭ゼロ・助数詞・コロンの境界8文、電話とコロン8文を両声で採取。全72件の正式終了、DLL/入力/native report SHA256、全入力順、CP932読み、実原版WAVのサイズ/SHA256を検証した。全36文×両声を通常 `TestBracketConnectorStreamReadingOriginal` に固定し一致した。

[原版case107の観測](evidence/bracket-connector-stream-akane-original.json)は全stage・数詞LIFO・内部戻り値・独立SDKとの整形読み一致を検証。100a90には８万・２）八が渡り、最後の時を全列へ適用する。観測はフロントエンドのみで、波形証明ではない。

Goの連続区切りスキャナーへ括弧とコロンを追加し、異なる区切りが連続しても全列を処理する。括弧がある汎用列は末尾助数詞の数詞キーを保つ。電話の中点・開き括弧・ハイフンに続く混在列は電話として読むが、閉じ括弧から始まる列は汎用読みにする。コロンと中点の組合せは右側小数の比率へ接続し、12：30・5を12タイ30テン5として読む。

[全比較](evidence/bracket-connector-stream-reading-comparison.json)は新72読みとseed6全256読みを保持しすべて一致した。seed6全128文×両声の `TestRandomNumeralSeed6ReadingOriginal` を環境変数なしの通常回帰へ昇格。seed6全128文の通常SDK/Go波形テスト `TestRandomNumeralSeed6OriginalWAV` も登録したが、全256WAVの完走は未実施。既知のゼロ大単位2文と範囲の小数助数詞、任意入力・全設定・全CPUの完全一致は未達。

数詞・文章処理の既存テストは通過（文章86.112秒）。全読み回帰と、関連4文×両声の通常SDK/Go全WAV比較を実行中。

seed6を通常回帰に含めた全ReadingOriginalテストは147.748秒で通過し、vetも通過した。Linuxの保存原版全8WAV再生比較を `AOIKANE_BRACKET_CONNECTOR_STREAM_REFERENCE_AUDIT=1` / `TestBracketConnectorStreamReferenceWAV` で開始している。

Windowsの全8WAVは両声とも正式終了0・PASS。[全波形証拠](evidence/bracket-connector-stream-wav.json)で全SDK/Go WAVと独立原版採取を再hash・全バイト照合して計1194072 PCMサンプルが一致した。Go1.26.8、CGO=0、AMD64 Family25 Model33、threads6、AVX512 convolutions falseでの結果であり全CPUの証明ではない。Linuxの全8WAV再生比較は継続中。

Linux amd64の全8WAV再生比較もログ末尾PASS、全8サイズ/SHA256一致を確認。[Linuxの証拠](evidence/bracket-connector-stream-linux-wav.json)。
