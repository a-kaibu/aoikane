# 秒パターンの先頭ゼロ再構築

2026-10-13。seed5のcase81 `A日．月人１月声0０秒9声。` にあったゼロビョー／レービョーの差を調べた。

通常SDKから周辺18文と単独ゼロ8文を両声で採取。全52件について正式終了・DLL SHA256・入力SHA256・入力順・CP932読み・native report SHA256・保存WAVサイズ/SHA256を検証した。全件を通常の `TestZeroSecondsReadingOriginal` へ固定し、入力を除外せず一致した。

[原版の5入力観測](evidence/zero-seconds-akane-original.json)は全stageの釣り合い、数詞enter/leaveのLIFO、戻り値、独立SDKとの整形読み一致を検証。数字種が揃う `00秒9` / `〇〇秒九` は時間パターンの100a90で００をレーへ評価してから1043f0へ進む。混在する `00秒九` / `〇〇秒9` は全列を汎用100a90へ渡し、００をゼロゼロとして維持する。単独 `000秒` は整数ゼロ飛ばしを使うが短い特殊読みへ再一致せず、ゼロビョーになる。

Goでは時間パターンの整数部を先にゼロ飛ばしし、混在する数字だけの秒列はフラグメントとして読む。裸の単位や零京の既存処理を保つ。3桁以上の全ゼロ整数はゼロ飛ばしで短縮しても短い特殊読みに再一致させない。

[全比較](evidence/zero-seconds-reading-comparison.json)は新26文×両声の全52読み一致と、seed5全128文×両声を保持。seed5の差は4→3文、残りcase10・13・112は未解決。全128文を読む失敗する厳密ゲートを維持する。任意入力・全設定・全CPUの完全一致は未達。

数詞・文章処理の既存テストと全ReadingOriginal回帰は通過（文章70.947秒、全読み106.019秒）。`TestZeroSecondsOriginalWAV` は `00秒9。`・`00秒九。`・`000秒。`・元case81の4文を通常SDKとGoで両声比較する。`AOIKANE_ZERO_SECONDS_REFERENCE_AUDIT=1` の `TestZeroSecondsReferenceWAV` は保存原版8WAVをDLLなしで再合成して全体hash/サイズを照合する。

Windows全8WAVは両声とも正式終了0・PASS。[全波形証拠](evidence/zero-seconds-wav.json)は保存SDK/Go WAVと独立採取WAVを再hashし全バイト照合、計945881 PCMサンプルが一致した。AMD64 Family25 Model33、threads6、AVX512 convolutions false、Go1.26.8、CGO=0での結果であり、全CPUの証明ではない。Linux amd64の全8WAV再生比較も正式終了0・PASS、全体サイズ/SHA256一致。[Linuxの証拠](evidence/zero-seconds-linux-wav.json)。
