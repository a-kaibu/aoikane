# 改行直前の末尾空白を保持する

seed18 / generator_version=1 / max_units=24 / count=128を両音声の通常原版SDKで採取。
全256件のtalk成功・正常detach・crash/timeout/エラーなしを確認し、全原版WAV実ファイルと
native report SHA256を再検証した。corpus SHA256は
`830945b988184ac818173d53102728382374cd36ecc9118f99604124c71e9f5b`。
参考はtools/compare_reading/testdata/seed18-{akane,aoi}-original.json。
茜は全128件一致、葵はcase106の1件が異なった。
入力は「「漢あ \n（0ｱ😀ｱ〜！Ж。」（実改行）。第1文の原版はカンとアの間に|0があり、Goは連結していた。

読み取り専用の3文の段階観測では、原版CaboCha入口の第1文に末尾全角空白の語が残る。
GoのSplitSentencesが末尾をTrimSpaceで落とすとこの語が消え、標準語の境界予測が変わる。
原版trace private/seed18-aoi-space-stages.jsonlのSHA256は
`22cac6bf7c4e9a171241a15b4961fb68434cf31f2c3c196ba40a1607d5eb1aea`。
全3文のformatを観測し、errorなし。この整形後終了の診断は全WAV一致の証明とは区別する。

最初の試作で先頭空白まで保持したところ、既存のランダム読み回帰に差が発生した。
先頭は従来どおり除き、末尾の水平空白を保持する形に修正した。
空白だけの範囲は引き続き空文の既存扱いとし、発音する語は付け足さない。
修正後の既存全読み回帰は57.818秒で成功。
さらに改行前の空白なし/半角空白/タブ/全角空白を含む32文章×両音声の64件が読み一致。
全talk成功・正常detach・全原版保存WAVの長さ/SHA256・native report SHA256も再検証。
corpus SHA256 `b9398dca34446dedf764820dc88f885132c5a7bf0c46b97f5a4cb25af81096b9`。
参考はtools/compare_reading/testdata/space-tail-32-{akane,aoi}-original.json。
TestRandomSeed18LongReadingOriginalとTestSpaceTailReadingOriginalで、結果による除外なしに
全件を通常回帰へ追加し、5.314秒で全320件が一致した。seed5〜18の対象は計3584件。
phrase/portableのgo vetも成功。phrase全回帰の最終試作も104.459秒で成功。
ログはprivate/space-tail-v2-{reading,phrase}-regression.log、space-tail-final-reading-regression.log、
space-tail-vet.log。初回の先頭空白保持の失敗ログも保存し、最終結果と混同しない。

TestSpaceTailOriginalWAVで問題の全文、短縮した末尾空白＋改行、空白なしの対照文を
両音声の独立通常原版SDKとGoで全WAV比較中。フック・tensor差し替えなしで全文から合成。
ログはprivate/space-tail-{akane,aoi}-wave.log。問題の全文は茜435286 bytes、葵438582 bytesで全WAV一致。
全6件の完了はまだ主張しない。
既知CPU差と任意文章・全設定の再現保証は残る。


独立通常原版とのWindows全6件は、茜290.94秒、葵296.33秒で全WAV一致した。
全原版とGoの保存WAVの長さ/SHA256をレポートへ再照合し、計910678 PCM16サンプル。
参照はtestdata/space_tail_wave_original.json。AOIKANE_SPACE_TAIL_REFERENCE_AUDIT=1で
全文からDLLなしで合成し全WAVを比較できる。Linux amd64 CGO=0の6件も実行中。

清浄なec7676eのソースから更新したdist/aoikane-purego.exeはGo 1.26.8、CGO_ENABLED=0、
GOAMD64=v1、vcs.modified=false。--portable embeddedで葵の問題の全文を合成し、
独立通常原版の保存WAVと全バイト一致した。438582 bytes、SHA256
`39bbfa740333133ef8891c263151fe033d240bbbaeace247708492151478beb5`。
ログはprivate/space-tail-cli-{aoi,build-info}.log。


Linux amd64 CGO=0のTestSpaceTailReferenceWAVも全6件が原版WAVと一致し、334.57秒で正常終了。ログはprivate/space-tail-linux-reference.log。
