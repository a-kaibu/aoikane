# 先頭ゼロと大単位の連結

2026-10-14。00兆万・零兆万の読みが原版と異なっていた。先頭がゼロの数字列では最初の大単位までを先に読み、残りの単位と助数詞へ接続する。漢字の零の後ろに大単位が連続する場合も、最初の単位と残りを分けて原版のアクセント境界を保つ。

通常SDKから新28文を葵・茜で採取し、正式終了・入力SHA256・DLL・native report・CP932読み・実WAVのサイズ/SHA256を検証した。新28文×両声と既存の大単位32文×両声はすべて一致。[全読み比較](evidence/zero-large-unit-chain-reading-comparison.json)に入力ごとの結果を保持する。既存32文も環境変数ゲートを外し、全入力を通常回帰へ昇格した。

[茜の内部観測](evidence/zero-large-unit-chain-akane-original.json)では00兆万と零兆万の数詞処理と整形後の読みを確認した。内部観測はフロントエンドの証拠であり波形比較とは別である。

Windowsでは00兆万、零兆万、00兆万本、01兆万の4文×両声を通常SDKとGoで合成し、両テストが終了0・PASS。[全WAV証拠](evidence/zero-large-unit-chain-wav.json)は実ファイルを再hashし、SDK/Goと独立SDK採取を全バイト照合した計784910 PCMサンプルの一致を保持する。原版DLL SHA256、CPU、threads6、AVX512 convolutions false、比較バイナリSHA256も記録する。

Go1.26.8、CGO=0、AMD64 Family25 Model33の結果。数詞・文章処理の回帰は通過（文章71.744秒）、全ReadingOriginal回帰は132.759秒で通過、vetも通過した。[Linuxの同8WAV参照比較](evidence/zero-large-unit-chain-linux-wav.json)も全件一致し、518.56秒でPASS、対象WSLプロセスの終了を確認した。元の起動ハンドルは引き継げず終了コードは独立には保存していない。任意入力・全設定・全CPUの一致は未達であり、有限コーパスの一致を完成とは扱わない。
