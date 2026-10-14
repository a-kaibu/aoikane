# 中点・「点」と前置きの数詞処理

2026-10-13。通常の原版SDKから茜・葵の読みとWAVを独立に採取し、中点と「点」の分岐をGoの文章経路へ接続した。

`1・2` の小数、`１・２・３` の番地、文字種が混ざった汎用列、`1点2` と `一点二` の促音の違いを保持する。番地は各項を整数として読み、ノと無音の区切りを付ける。電話番号は数字単位の読みを使い、途中のニ・ゴの長音と原版のポーズを保持する。図・表・問などの小数指定と郵便番号の前置きは、元の数詞規則ファイルから読み込む。

通常回帰 `TestPointNumeralReadingOriginal` と `TestPointPrefixReadingOriginal` は、それぞれ24文章×両音声の全96読みを比較する。失敗した入力を除外した集合ではない。各原版キャプチャの全WAV・レポートも再ハッシュしてから固定した。内部処理の読取り観測10文章は、別の通常SDK採取の整形読みと全件一致した。[観測記録](evidence/point-numeral-akane-original.json)に各数詞呼出しと全ステージの終了時グラフを残す。

既存数詞ランダムseed1の全128文章×両音声を再比較した結果、不一致は修正前の茜7・葵6から茜5・葵4に減った。追加の数詞継続24文章×両音声も全件保持し、修正前の茜21・葵20から茜7・葵5に減った。これらの残りを任意入力の対応済みとして扱わない。[全入力の比較記録](evidence/point-numeral-reading-comparison.json)には一致・不一致の両方を含む。

数詞・句処理の回帰、通常の全 `ReadingOriginal` 回帰、静的検査は通過した。先の小数・単位の8WAVもLinux amd64・CGO=0で原版全体と一致した。[Linux検証](evidence/decimal-unit-linux-wav.json)。CPUはAMD Family 25 Model 33、原版数値設定は6スレッド・AVX512畳み込みなし。

中点・前置きの最終WAV比較 `TestPointNumeralOriginalWAV` は7文章×両音声の全14件を終え、すべて原版とバイト単位で一致した。茜597.675秒・葵584.180秒でテストが終了し、両プロセスの終了も確認した。保存済みの原版・Go WAV全体を再読込みし、別途採取した通常SDKのWAV・長さ・SHA256とも照合した。[14WAVの基準](../internal/portable/testdata/point_numeral_wave_original.json)。`AOIKANE_POINT_NUMERAL_REFERENCE_AUDIT=1` の `TestPointNumeralReferenceWAV` は、この全14件をDLLなしで厳密比較する。任意文章・全CPU・全設定での完全一致は引き続き未達。

コミット `c3c957b` の変更なしのソースからCGO=0の通常CLIを再構築し、埋込み純Go経路で `電話1・2・3。` を合成した。茜282840バイト・葵282642バイトの全WAVが、通常SDK・独立採取・14件基準のすべてと一致した。[CLI検証](evidence/point-numeral-cli-wav.json)。Linux amd64・CGO=0の全14件再実行も954.90秒で終了し、すべて原版WAV全体と一致した。[Linux検証](evidence/point-numeral-linux-wav.json)。
