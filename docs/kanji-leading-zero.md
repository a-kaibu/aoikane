# 先頭〇を含む数字の原版比較

茜の通常原版フロントエンドとの比較で、「〇一」「〇〇一人」の先頭〇が
Goで失われる差を確認した。「〇四日」も特殊読みのヨッカへ進めず、ヨンニチだった。

純Goのnumeral.Readを修正し、同じ文字種の先頭ゼロ数字を2桁ずつの読みグループにする。
日付・時刻のゼロ省略パターンは全角０に加え、漢数字〇にも適用する。
混在文字種や単位付き入力では、その境界を保持して読む。
「〇一十」はゼロイチ / ジュウ、「０〇一」はゼロ / ゼロイチになる。
数字の一部を分割して読んだ際には、全体の数字を対象とする特殊読み・ゼロ省略を
再適用しない。「〇１人」はゼロ / イチニンで、ヒトリにならず、
「０〇四日」はゼロ / ゼロヨンニチで、ヨッカにならない。

通常原版から得た32件の整形読み・アクセントはWindows CGO=0で全件一致した。
`internal/portable/testdata/kanji_leading_zero_original.json` に期待値と
原版DLL・規則・観測ファイルのSHA256を保存し、
`TestKanjiLeadingZeroReadingOriginal` で不一致を必ず失敗にする。

## WAV全体の比較

原版の通常APIを直接呼び、WindowsのGo合成と7件を全バイト比較してすべて一致した。
このWAV比較にはFridaやモデル入力の差し替えを使わない。
原版側の長さ・SHA256を保存したfixtureは
`internal/portable/testdata/kanji_leading_zero_wav_original.json`。
通常原版CPUはAMD Family 25 Model 33、threads=6。

| 入力 | WAV bytes | 原版 SHA256 |
| --- | ---: | --- |
| 〇〇一人。 | 189268 | `04dec38cdcfa749c8a7aaa0e710c1821a34bb51e86e92d89cb811c7508a1766c` |
| 〇一六本。 | 189666 | `a546fb5d05bfd2514c3f321e418b1e7aa06925f8df67ba184dc3b2d3b266aa79` |
| 〇四日。 | 136876 | `ae301cbeb5e4bc2b3b93320e5eee6c850f443e13abdc5e761b241e64ebf8a96b` |
| ０〇四日。 | 198338 | `a6421535ea65883983cfe5754b87fb7f83817b4ad058729bb3ff8023e41121c7` |
| 〇一十。 | 176236 | `789d42590aa57e50f203368ca67390c630492d80ec26575f415ee5c2a7da5d0d` |
| 01十。 | 153164 | `10fe454facacb6352bd58aa1d0cf7ebf4d98e260f693832076b8c9b5730ba265` |
| 0万。 | 137308 | `c272f344827c5cff01802669ead7ec261569b9aa077e1fb6ade01005e1e34b89` |

`TestKanjiLeadingZeroOriginalWAV` は
`AOIKANE_AKANE_RANDOM_REFERENCE_AUDIT=1` で有効になるstrictな再現テスト。
WAV長・SHA256の両方を比較する。期待値はGo出力から生成していない。
Windowsログは `private/kanji-leading-zero-wav.log` と
`private/kanji-leading-zero-mixed-wav.log`。

更新したCGO=0 CLI `dist/aoikane-purego.exe` でもembedded資産から「〇四日」を合成し、
136876 bytes・原版SHA256と一致した。出力は `private/akane-kanji-zero-latest-cli.wav`。

茜の既存85文、単一モーラ4件、全角先頭ゼロ10件、助数詞20件、改行21件の読み比較も成功。
既存のBuild原版比較326件、混在数字15件・語グラフ3件、小数22件と関連3パッケージのvetも成功した。
既存numeralの診断用oracleテストは差をログに出すだけで失敗にしない。
plain 1件・counter 4件・pattern 1件の差が残り、その終了コード0を完全一致の証拠に使わない。

任意文章・全CPU・全設定の完全再現は引き続き未達。今回はGitHub Actionsを新規実行していない。

全角数字の単位付き入力ではゼロだけを別グループにし、「０１十」はゼロ / ジュウと読む。
「０万」「〇万」では単位を保持し、暗黙の一に置き換えない。
追加8件はこの計算順と日付の特殊読みの抑止を原版と比較したもの。
漢数字の24件と合わせ、32件をstrictな読みテストへ保存した。
追加2件のWindows WAV比較ログは `private/kanji-leading-zero-units-wav.log`。
最終CLIのembedded資産による「0万」も137308 bytes・原版SHA256と一致した。
macOS arm64はCGO=0のCLIビルド成功のみを確認し、音声比較は未実施。

Linux amd64 CGO=0でも保存原版とのWAV7件と読み32件が一致した。
初回5件は220.66秒、追加2件は61.26秒。ログは
`private/kanji-leading-zero-linux.log` と `private/kanji-leading-zero-units-linux.log`。
LinuxはDLLを実行せず、Windows通常原版から保存した長さ・SHA256と比較した。
追加2件と読み32件は最終コードで再ビルドしたLinuxバイナリで確認した。
初回5件の後の変更は、全角先頭ゼロと単位・単独ゼロと大きい単位の処理を追加したもの。
