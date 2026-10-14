# 括弧付き数字列と電話番号の原版再現

## 原因と修正

seed8 case41 `日1四（6七ａ声。` では、原版のエーがGoではアールになった。
数字表の `ａ` はエーで、辞書の `ａ` はアールという別候補を持つ。
Goが数字間の括弧を未対応の接続記号としていたため、前後の数詞統合が止まり、辞書候補を選んでいた。

数字間の `（` と `）` を同じ数詞へ統合し、原版の強い休止を残す。
開閉括弧が交互に現れる `1（2）3ａ。` も1つの数詞となる。
通常の括弧付き数字列はwhole-numberの特殊助数詞読みを使わず、最後の数字に助数詞表の読みを結合する。

電話番号には原版表の別規則を適用する。
先頭ゼロ・各ブロック長が括弧付き電話の構成に一致する場合、または前に電話/FAX/TELの指定文脈がある場合は、2桁ずつ読み、数字ブロック間にノと強い休止を付ける。
前置電話規則が受け入れる末尾はなし・番・号。末尾がａなら、電話番号の構成による自動規則に一致する場合だけ電話読みを使う。
電話読みの末尾の助数詞は最終の2桁グループに付け、単一数字の音変化や特殊読みは適用しない。
文章専用の読み置換や数値の許容差は追加していない。

## 原版の観測と厳密回帰

`capture_reading_batch.py` で通常SDKの合成を正常終了まで実行し、formatterを観測。
原版DLL SHA256、入力順、全件成功、正常detach、実際の保存WAVの長さ/SHA256を確認した。

- 括弧・数字表記・a/R/b/助数詞等を変えた44件×両音声。
- 特殊助数詞・先頭ゼロ・電話の16件×両音声。
- 電話/FAX/TEL/郵便番号の文脈と数字表記の10件×両音声。
- 合計140件。読みの全配列が最終Goと一致し、`TestBracketNumberReadingOriginal` に保存。
- `internal/portable/testdata/bracket_number_reading_original.json` は6つの通常原版capture SHA256を含む。
- seed8の128件×両音声も全読み一致。`TestRandomSeed8ReadingOriginal` を追加。
- 新規seed9の128件×両音声も全読み一致。通常原版参照を `tools/compare_reading/testdata/seed9-{akane,aoi}-original.json` に保存し、`TestRandomSeed9ReadingOriginal` を追加。
- seed5〜9は計1280件の読みが一致。これを1280件のGo WAV一致とは扱わない。

原版graphの数詞候補も確認した。`1四（6七ａ。` は `１四（６七ａ` の数詞とchainに統合され、末尾chainはロクｂナナエー。
`電話01（23）4567。` はゼロイチノニーサンノ（accentにs1）とヨンゴーｂロクナナになる。

## 完全WAV比較

3文章×両音声の6件、計983555 PCM16サンプルで、Windows通常原版APIとGoのWAVが全バイト一致した。
対象は `日1四（6七ａ声。`、`1（2）3ａ。`、`電話01（23）4567ａ。`。
観測時の原版WAVも独立のフックなし通常原版WAVの長さ・SHA256に一致した。
CPUはAMD64 Family 25 Model 33 Stepping 0, AuthenticAMD、6スレッド、AVX512なし。
通常原版比較ログは `private/bracket-number-wave-{akane,aoi}.log`。435.37秒 / 433.57秒。
WAV参照は `internal/portable/testdata/bracket_number_wave_original.json`、独立比較report SHA256を2つ保持する。

`TestBracketNumberOriginalWAV` はWindowsの通常原版と全WAV比較する。
`AOIKANE_BRACKET_NUMBER_REFERENCE_AUDIT=1` の `TestBracketNumberReferenceWAV` は文章から合成し、原版WAVの長さ・SHA256を厳密比較する。
最終コードでの保存原版6件再現も両OSで全成功。
- Windows: 645.71秒。`private/bracket-number-final-windows-reference.log`。
- Linux amd64 CGO=0: 616.75秒。`private/bracket-number-linux-reference.log`。文章から生成し、原版DLLや保存tensor入力は使わない。
- 最終フロントエンド/数字処理の全回帰は75.407秒/0.696秒で成功。全ReadingOriginal/UnknownHiraganaの回帰とphrase/numeral/portableのvetも成功。
- standalone embedded CLIをソース `9bd2262`、Go 1.26.8、CGO_ENABLED=0、vcs.modified=falseで更新。茜 `日1四（6七ａ声。` は290398 bytes、SHA256 `23908d15e92c366225197b4ba7155aee340c412015b44b417ec58275389d73fa` で独立原版と一致。
既知Intel/別AMDの数値差、未検証文章・設定は引き続き残る。
macOS実機検証はユーザー指定で対象外。GHAは新規実行していない。
