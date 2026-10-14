# 小数点後の大単位列

2026-10-13。`万万。` は一つの数詞群になるが、`1．万万。` では原版が先頭の `万` と後続の `万` を分ける。通常の整数列の結合を維持し、小数点以降の大単位だけの列では先頭の単位を独立させ、残りを既存の整数表照合へ渡すようにした。`1．万万万。` は全単位を分離せず、`マン / マンマン` の二群になる。助数詞は最後の群に接続する。

追加24文と拡張32文、計56文×茜・葵の全112読み・アクセント・区切りが通常SDKと一致した。全ケースの原版WAVとレポートを再ハッシュし、全112件を `TestPointUnitPairsReadingOriginal` の通常回帰へ固定した。小単位、数字を含む列、三つの単位、二つの点、末尾の助数詞も含み、結果による入力の除外はしていない。読取り観測7文でも、全整形読みを独立した通常SDK採取と照合し、全数詞呼出しと各ステージの終了候補を保存した。[原版処理](evidence/point-unit-pairs-akane-original.json)。この観測自体はGoのWAV一致の証拠にはしない。

数詞ランダムseed1の全128文×両音声ではcase23が両方の声で一致へ変わり、新たな不一致はなかった。残る不一致は茜3件・葵2件。追加の数詞継続24文×両音声ではcase22・23が一致へ変わり、茜4件・葵2件が残る。追加112件と、この一致・不一致を含む全比較416件を[比較記録](evidence/point-unit-pairs-reading-comparison.json)に残した。任意文章の完全一致は未達。

全数詞・句処理回帰、既存の全 `ReadingOriginal` 回帰、新しい112件の回帰、関連パッケージの静的検査は通過した。通常SDKとGoの最終音声比較 `TestPointUnitPairsOriginalWAV` は、`1．万万。`・`1．万億Ｃ。`・`1．万万万本。`・`1．億万億秒。` の4文×両音声、全8WAVがバイト単位で一致して正常終了した。茜262.372秒・葵262.021秒。保存した全原版・Go WAVを再読込みし、独立した通常SDK採取のWAV・長さ・SHA256とも照合した。[8WAVの基準](../internal/portable/testdata/point_unit_pairs_wave_original.json)。原版の設定はAMD Family 25 Model 33・6スレッド・AVX512畳み込みなし。

`AOIKANE_POINT_UNIT_PAIRS_REFERENCE_AUDIT=1` の `TestPointUnitPairsReferenceWAV` は、この全8件をDLLなしで厳密比較する。この原版プロファイル・入力集合の証拠を、全CPU・全設定への証明には拡張しない。

Linux amd64・CGO=0での全8WAV再実行も511.95秒で正常終了し、すべて原版全体と一致した。[Linux検証](evidence/point-unit-pairs-linux-wav.json)。変更なしのコミット `a6b0338` から再構築したCGO=0の通常CLIでも、埋込み純Go経路で `1．万万万本。` を合成し、茜・葵とも251162バイトの原版WAV全体と一致した。[CLI検証](evidence/point-unit-pairs-cli-wav.json)。
