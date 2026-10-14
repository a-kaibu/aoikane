# 括弧に挟まれたマイナスの原版再現

## 修正

seed8 case18 `漢（「－）～\n😀Ж「アあ。` では、Goだけが最初の文の「マイナス」を落としていた。
辞書解析は U+2212 `−` を返すが、原版BANCHAのCP932表記と読み規則は U+FF0D `－` を使う。
規則照合用のsurface/baseをCP932表記に戻し、読みのない候補も原版の `＊` に合わせた。
これにより「開き括弧・マイナス・閉じ括弧」の既存規則が適用される。
無音候補を先に合わせることは、英字直後のマイナスや `10%-20%` を発音しない既存規則にも必要。
文章別の特例や許容差は追加していない。

## 原版比較

- 21文章×葵・茜の42件を通常の原版APIで合成し、formatterを観測。全読みが最終Goと一致。
- 原版の正常終了、全ケース完了、実際のWAV長・SHA256をcapture工具が確認。
- 読み参照: `internal/portable/testdata/sign_bracket_reading_original.json`。
- 3文章×両音声の6件を、フックなし通常原版APIと独立比較。WAV全バイト一致、計749040 PCM16サンプル。
- 文章: 上記seed8 case18、`漢（－）。`、`漢「－」。`。
- 独立原版のWAVはformatter観測時に保存した原版WAVの長さ・SHA256とも一致。
- CPU: AMD64 Family 25 Model 33 Stepping 0, AuthenticAMD。6スレッド、AVX512なし。
- WAV参照: `internal/portable/testdata/sign_bracket_wave_original.json`。
- Windows原版比較ログ: `private/sign-bracket-wave-{akane,aoi}.log`。234.99秒 / 236.71秒。
- 最終実装の保存原版再現: Windows 6件全成功、368.20秒。`private/sign-bracket-final-windows-reference.log`。
- Linux amd64 CGO=0でも6件全成功、370.27秒。`private/sign-bracket-linux-reference.log`。原版DLLも保存tensor入力も使わず、文章からWAVを生成した。
- 全フロントエンド回帰 `go test ./internal/phrase -count=1` は63.057秒で成功。portableの全ReadingOriginal/UnknownHiragana回帰、比較ツールの完全性テスト、phrase/portableのvetも成功。
- standalone embedded CLIをソース `ab662f9`、Go 1.26.8、CGO_ENABLED=0、vcs.modified=falseで更新。茜 `漢（－）。` は184844 bytes、SHA256 `01f60462bbea074ec8ac608e45e9475b75b7c8e55606890027b199daf9057eb9` で独立原版と一致。

通常テスト `TestSignBracketReadingOriginal` は全読みの厳密比較。
Windows `TestSignBracketOriginalWAV` は `AOIKANE_TEST_ENGINE` を指定すると通常原版と全WAV比較。
`AOIKANE_SIGN_BRACKET_REFERENCE_AUDIT=1` の `TestSignBracketReferenceWAV` はDLLなしで文章から合成し、保存原版のWAV長・SHA256を比較する。

## 残る差

seed5〜7の768件の読み回帰は一致。seed8は各音声128件中1件の差が残る。
`日1四（6七ａ声。` の `ａ` が原版のエーではなくアールになる。
追加24件の茜原版比較では同じ条件の3件が異なり、単純な `6七ａ。` は一致した。
原版graphでは `1四（6七ａ。` が1つの数詞とchainへ統合され、chainの末尾はナナエー。
参照は `tools/compare_reading/testdata/counter-alpha-seed8-akane-original.json` に保存。
次は括弧で結んだ数字列の末尾候補選択を調べる。既知Intel/別AMDの数値差も未解決。
macOS実機検証はユーザー指定で対象外。GHAは新規実行していない。
