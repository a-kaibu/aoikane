# 先頭の単位と「零」の数詞列

2026-10-13。`兆零一％` のように単位から始まる数詞列で、原版の整数片と「零」の数字対を別のアクセント群として保持するようにした。

原版の `10b010` は先頭の単位を読み、`1043f0` は後続の「零」を含む数字対を読む。Goではこれらの音を整数表へまとめて再解析していたため、整数表にない `レー` が原因で数詞全体を拒否し、単語ごとの読みへ戻る場合があった。単位群を原版どおり平板化し、数字対と末尾の助数詞群をそのまま後ろへ接続する。小数の整数部に既に実装した処理を通常の数詞列にも共有した。

追加コーパスは12種類の数詞×4種類の末尾（なし・％・本・秒）の48文。茜・葵それぞれ全48件の読み・アクセント・区切りが通常SDKと一致した。全原版キャプチャのWAVとレポートを再ハッシュしてから、通常回帰 `TestUnitReiTailReadingOriginal` に全96件を固定した。[採取監査](evidence/unit-rei-tail-reading.json)。読取り観測4文では、整形読みを独立した通常SDK採取と照合し、各数詞呼出しと全ステージの終了時候補を保持した。[原版の処理記録](evidence/unit-rei-tail-akane-original.json)。この処理観測自体をGoの音声一致の証拠にはしない。

ランダム数詞seed1の全128文×両音声では、両方の声でcase119が一致へ変わり、新しい不一致は増えなかった。残る不一致は茜4件・葵3件。追加の数詞継続24文×両音声でも1件ずつ改善し、茜6件・葵4件が残る。一致・不一致の全400件を[比較記録](evidence/unit-rei-tail-reading-comparison.json)に残す。任意文章の完全一致は未達。

数詞・句処理の回帰、Windows CGO=0の全 `ReadingOriginal` 回帰、関連パッケージの静的検査は通過した。

通常SDKとGoのWAV比較 `TestUnitReiTailOriginalWAV` は `兆零一％。`・`十零一。`・`兆零１。`・`兆零一万。` の4文×両音声、全8件を完了した。茜306.795秒・葵305.814秒で正常終了し、全WAVがバイト単位で一致した。保存した両WAVを再読込みし、独立した通常SDK採取のWAV・長さ・SHA256とも照合した。[8WAVの基準](../internal/portable/testdata/unit_rei_tail_wave_original.json)。原版はAMD Family 25 Model 33・6スレッド・AVX512畳み込みなし。`AOIKANE_UNIT_REI_TAIL_REFERENCE_AUDIT=1` の `TestUnitReiTailReferenceWAV` はこの全8件をDLLなしで厳密比較する。全CPU・全設定での一致の証明には拡張しない。

Linux amd64・CGO=0でも全8WAVの厳密再実行が481.14秒で正常終了し、原版と全体一致した。Linuxの全 `ReadingOriginal` 回帰も通過した。[Linux検証](evidence/unit-rei-tail-linux-wav.json)。変更なしのコミット `40c406c` から再構築した通常CLIの埋込み純Go経路でも、`兆零１。` が茜179254バイト・葵178766バイトで原版全WAVと一致した。[CLI検証](evidence/unit-rei-tail-cli-wav.json)。
