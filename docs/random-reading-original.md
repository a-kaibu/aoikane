# ランダム文章の原版読み比較

通常原版SDKを1プロセスで順に呼び出す tools/original_reading_probe と、
そのformatter出力を観測する tools/capture_reading_batch.py、
Goの読み全体と厳密比較する tools/compare_reading を追加した。
原版は実際に各文章の完全WAVも生成して保存する。tensor注入やthread数の強制はない。
読みの観測値とWAVの一致検証は別であり、このツールの一致は読みだけを証明する。

## 実行

```powershell
$env:GOTOOLCHAIN = 'go1.26.8'
$env:CGO_ENABLED = '0'
go build -o private/original-reading-probe.exe ./tools/original_reading_probe
go run ./tools/compare_reading -generate -seed 5 -count 128 -max-units 12 -output private/reading-inputs.json
uv run --python 3.12 --with frida==17.23.1 python tools/capture_reading_batch.py --corpus private/reading-inputs.json --output private/new-reading-akane.json --voice KotonohaAkane_ns_48
go run ./tools/compare_reading -reference private/new-reading-akane.json -output private/new-reading-comparison.json
```

葵ではvoiceをKotonohaAoi_ns_48に変更する。出力名は毎回新しいものを使う。
既存の原版WAV保存フォルダがある場合は採取を拒否し、過去のreportの混入を避ける。
生成途中のhelper異常、空formatter、入力順違い、通常talkの失敗、timeout、crash、
正常なprocess終了の欠如、原版WAV/report不一致はいずれも成功にしない。
採取中に例外が起きた場合も生成した子プロセスを終了する。
原版DLLの観測対象ハッシュを固定し、CP932の生bytesと読み、コーパス・実行helper・
原版reportのハッシュを記録する。比較側でも生CP932と読みが一致することを確認する。
相違が1つでもあれば比較コマンドは非ゼロ終了し、全ケースのreportを残す。

## 2026-10-12の検証

既存原版参考85文章とseed3の8文章、計93件を普通のSDK経由で観測した。
85文章は独立に取得していた原版の先頭文の読みと一致し、8文章の原版完全WAVは
過去に観測なしで保存したseed3の原版WAV長・SHA256と一致した。
93件の完全な読み配列もGoと一致した。新しい整合性検査でも93件一致。
整備後の新しい2件の原版採取・読み比較も一致した。

seed5 / generator=1 / max_units=12の128件を葵・茜それぞれで取得した。
保存した原版参考は以下。すべて正常なprocess終了、talk成功、完全WAV保存を確認した。

- tools/compare_reading/testdata/seed5-akane-original.json
- tools/compare_reading/testdata/seed5-aoi-original.json

この採取は通常SDK呼出しとformatterの観測であり、通常原版のCPU間WAV差を解決しない。
初回は各3件の読み相違を発見した。未知ひらがな修正15df94d後は各2件の相違が残る。

| index | 文章 | 発見した相違（後の零修正で解消） |
| ---: | --- | --- |
| 105 | 開0a土木土零日ー下A三。 | 原版は「ー」の無音候補を選ぶがGoは発音候補を残す |
| 125 | 七金六九B零C校3。 | 茜の「レーシーコー」の上昇位置、葵の「ナナキン」後の境界 |

```powershell
# 零と単独数単位の修正後は、どちらも128件の読みがすべて一致する。
go run ./tools/compare_reading -reference tools/compare_reading/testdata/seed5-akane-original.json -output private/seed5-akane-audit.json
go run ./tools/compare_reading -reference tools/compare_reading/testdata/seed5-aoi-original.json -output private/seed5-aoi-audit.json
```

未知ひらがなの修正対象は別途、観測なしの原版と完全WAV比較を行った。
詳細は[unknown-hiragana-original.md](unknown-hiragana-original.md)。
256件すべてのGo WAV一致はまだ検証していない。既存のランダム完全WAV比較を維持し、
今回のツールは読み段階の相違を多くの文章から発見するために追加した。
不完全な原版データの拒否テストと両Goツールのgo vet、Python構文検査も成功。

## 零修正後の追加比較

seed5の読み256件はすべて一致した。零と単独数単位の追加136件も一致した。
完全WAVの別比較は[rei-original.md](rei-original.md)を参照。
seed6 / generator=1 / max_units=12の128件ずつも通常原版から取得したが、
各2件の読み差を発見した。後の数字接続とマイナス修正で、両音声128件ずつの読みがすべて一致した。
原版参考はtools/compare_reading/testdata/seed6-akane-original.jsonとseed6-aoi-original.json。
case45「葉音3ー一。」は数字間の区切り（$1_1と|0）、
case46「ЖＡ−\t0（あΩ。。」はタブ前のマイナスがGoだけ発音される。
これらの読み差は解消した。詳細は[connector-original.md](connector-original.md)。既知CPU差は未解決。原版WAVもprivate/reading-seed6-128-*-probe/に保持したが、
seed6の256件のGo WAV一致はまだ検証していない。

## seed10の追加比較

seed5〜9の計1280件は全読み一致。seed10 / generator=1 / max_units=12の128件を両音声で通常原版から採取した。
全件のtalk成功・正常終了・保存WAVの完全性を確認。参考は `tools/compare_reading/testdata/seed10-{akane,aoi}-original.json`。
初回は葵128件全読み一致、茜2件の休止長が異なった。BANCHA準備処理の文節統合を移植して両音声128件全読み一致となり、厳密回帰へ追加した。
Goのseed10全WAV比較はまだ実施していない。

| index | 文章 | 修正前の原版 / Go |
| ---: | --- | --- |
| 63 | 一人私私読む一人でます。 | 最初の私の後が `$2_2` / `$1_1` |
| 93 | 零木「校九（下1、火私。 | 木の後が `$1_1` / `$2_2` |

原版の137520（関西弁の文節休止判定）のentry/leaveを観測した。
その原版入力グラフをGoの `ApplyKansaiChunkBound` に渡すと、2文章とも全morphの判定状態が原版leaveと一致する。
一方、Goが文章から作る文節構造には以下の差がある。休止判定本体より前の処理を調査する。

- case63: 原版は末尾の一人とでますが同じ主文節（ordinal3）に入る。Goでは一人がordinal3、でますがordinal4の別主文節となり、係り先も変わる。
- case93: 原版は校九（下1、をordinal1とそのS分割へ収める。Goは（からordinal2の別主文節を作る。
- 原版stage: `private/reading-seed10-akane-bound-stages.jsonl`。
- 原版の文節表示: `private/reading-seed10-akane-graph-in.txt`。
- 同一原版入力での比較ログ: `private/reading-seed10-bound-debug.log`。
- Go側の文章からの文節生成ログ: `private/reading-seed10-frontend-debug.log`。

CaboChaの最初の文節構造は一致していた。原版128370の準備処理が、数詞変換より前に人→で、九→（の文節境界を統合していた。
形態素の表層と品詞を分類する128100、および境界分類の組合せをGoへ移植した。詳細は[numeral-chunk-original.md](numeral-chunk-original.md)。
seed5〜10の計1536件が全読み一致。seed10全128件のGo WAV比較とは区別する。
既知Intel/別AMDのWAV数値差も引き続き未解決。GHAは新規実行していない。


## seed11 / seed12の追加比較

seed11は128件×葵・茜が全読み一致。参考は `tools/compare_reading/testdata/seed11-{akane,aoi}-original.json`。
corpus SHA256 `987bbf313c857f4ca2b336dec85b04b6916c142fe52ab459be8457bac7b96bc8`。
`TestRandomSeed11ReadingOriginal` は4.223秒で成功。seed5〜11の計1792件の読みが一致。

seed12も128件×両音声を通常原版から採取し、全件talk正常終了と実保存WAVのサイズ・SHA256を検証。
参考は `tools/compare_reading/testdata/seed12-{akane,aoi}-original.json`。両音声ともcase65だけが異なる。

| 音声 | 文章 | 原版 | Go |
| --- | --- | --- | --- |
| 茜 | 日火Ａ三ｃ47。 | ヒビエー… | ヒヒエー… |
| 葵 | 日火Ａ三ｃ47。 | ヒ、ビエー… | ヒ、ヒエー… |

`private/seed12-akane-voicing-stages.jsonl` の通常原版観測では、11f9c0・11ec50・115110・128670の前後は火の候補がヒ/カ/ヒー。
116ab0の後にビ1候補へ置換される。BOUNDや整形で初めて生じる差ではない。
Ghidraで117720（適用条件）と1176d0（濁音対応表）の関数を逆コンパイルした。
117720は品詞・読み内の濁音・表層の除外・和語/漢語/語種の規則を評価する。
原版復号済み規則は `private/rule-lines-trace/frontend-data/{standard,kansai}/lang/bandakrcx.lines`。
116ab0の移植と追加原版比較が次の作業。文章専用の置換や許容差は追加していない。
seed11/12の全Go WAV比較は未実施。既知CPU差も残る。


seed12の連濁差は116ab0と標準語bancutの接続で解消し、両音声128件全読み一致となった。
通常の厳密回帰へ追加。seed5〜12の計2048件が一致。
追加48文章×両音声の96件の読み、および3文章×両音声の6件のWindows完全WAVも原版と一致。
詳細は[rendaku-original.md](rendaku-original.md)。seed12全128件のGo WAV一致とは区別する。


## 長めのseed13と全WAV監査

seed13 / generator_version=1 / max_units=24 / count=128を採取。これまでの生成上限12単位を2倍へ広げた。
128件×葵・茜が全読み一致。通常原版の全talk成功・正常detach・crash/timeout/観測エラーなしと、
保存した128 WAVすべてのサイズ・SHA256が採取reportに一致することを確認した。
参考は `tools/compare_reading/testdata/seed13-{akane,aoi}-original.json`。

- corpus SHA256: `1626e664afb974622fa0cc6e71214e6175f4de99c8adf802877e5a6ecc96c8b7`
- 茜capture SHA256: `1a4bd1904c58bf1d127fee76e10ce0e269fd0b405ec8acae8a46af72be30259f`
- 葵capture SHA256: `b26978eb6ffc157b2c30dc323c6b012fe84982da6ef136549129e97f7649c53b`
- TestRandomSeed13LongReadingOriginal成功（4.580秒）。seed5〜13の計2304件の読みが一致。

全WAV監査は生成順の先頭4件を固定して両音声で実行する。結果・文章内容で除外/選別しない。
case0は未知ひらがな列、case1は複数文と数字/漢字/英字、case2は実改行・タブ・複数文・非CP932文字、case3は普通の日本語と英字を含む。
TestRandomSeed13LongOriginalWAVはWindowsの通常原版API（フック・tensor注入なし）から独立に合成し、Go WAVと全バイト比較する。
監査ログは `private/random-seed13-{akane,aoi}-wave.log`。Windowsの全8件が成功（茜603.47秒、葵622.56秒）。新規の通常原版WAVも保存参照の長さ・SHA256と一致。

保存原版参照は `internal/portable/testdata/random_seed13_wave_original.json`。
元の通常原版report SHA256と実WAVのサイズ・SHA256を検証して8件を抜き出した。
対象8件には合計2314543 PCM16サンプルがある。Windowsで全サンプルを含むWAV全バイトがGoと一致した。
AOIKANE_RANDOM_SEED13_REFERENCE_AUDIT=1のTestRandomSeed13LongReferenceWAVは、
原版DLLや途中tensorを使わず文章から合成し、全WAVのサイズ・SHA256を厳密比較する。
Linux amd64 CGO=0のテストバイナリをコンパイル済み。8件のLinux再現は1207.79秒で全成功した。文章からDLLなしで生成した全WAVの長さ・SHA256が原版参照と一致。
seed13全128件のGo WAV一致とは区別する。既知CPU差も残る。


## seed14: 新しいアクセント差を保存

seed14 / generator_version=1 / max_units=24 / count=128を両音声の通常原版で採取した。
全talk成功・正常detach・crash/timeout/観測エラーなし、全保存WAVの長さ・SHA256検証済み。
参考は `tools/compare_reading/testdata/seed14-{akane,aoi}-original.json`。
corpus SHA256 `42de873c17a87879dc67ff9bd4adf60b8460fa58578bf687f8761d8d09ee26b6`。
茜capture SHA256 `2ecbe1d0ed827db04096afb4d159b38cada78daff1a98a4de8ca51141ef1c328`、
葵capture SHA256 `f4399e78746b0c044e15db245c7d874f598b4d07f70283403dc5a4beca4a236d`。

読み比較は両音声とも128件中1件不一致で、比較ツールは終了コード1となった。
差を許容するテストや文章専用の置換は追加していない。seed14の全一致は未達。

- 茜case9 `2上Ａa九Ｃ（3八九Ｃ三。`：原版のキュ^ーシ!ーに対しGoは^キューシ!ー。
- 葵case38 ` （あＡ－！0ЖＡ〜）−－－\n。`（実改行）：第2文の原版はゼ^ロジェーエ!ー、Goは^ゼ!ロ$2_2ジェ^ーエ!ー。

原版段階観測 `private/seed14-{akane,aoi}-accent-stages.jsonl` を採取。
エラーなしで茜1文、葵2文のformatを確認した。この観測は整形後にプロセスを終了する診断で、WAV一致を示さない。
茜では11f9c0後に九Ｃが1語へ統合され、候補はキューシー/2-3-4/*。
Goの語グラフでは九とＣが別語のままで、後段のアクセント結合で差が生じている。
後続の開括弧と数字によるfindNumeralRunsのblocked判定を次に調べる。
葵では原版graph_inの０はKind=0、ＡはKind=29。Goの境界予測後は０もKind=29になる。
末尾の連続マイナスの解析語順にも差を観測した。BOUNDの入力・形態素解析を引き続き照合する。
Go診断ログは `private/seed14-go-graph.log`。一時診断テストは除去した。

seed5〜13の2304件の読み一致とseed13のWindows全8 WAV一致は、seed14の不一致と区別する。
seed13のLinux全8 WAV比較は1207.79秒で全成功。既知CPU差も未解決。


## seed14の解消とseed15の追加差

数詞の助数詞後接続判定と形態素解析の同点経路選択を修正し、seed14両音声128件の読みが全一致。
TestRandomSeed14LongReadingOriginalを追加し、seed5〜14の計2560件が一致。
詳細は[counter-connector-original.md](counter-connector-original.md)と[lattice-tie-original.md](lattice-tie-original.md)。

seed15 / generator_version=1 / max_units=24 / count=128を通常原版で採取した。
全256件のtalk成功・正常detach・crash/timeout/エラーなしと実保存WAVの長さ・SHA256を検証。
参考はtools/compare_reading/testdata/seed15-{akane,aoi}-original.json。
corpus SHA256 `fc85b1ae1fbbbb6008632bd5b667f0bb5057fca3b37eed655f35dba6eadc6a88`。
茜capture SHA256 `a84f1dffa0722704322edff4028aee2e686b1af1bfc477f0e0ab5b80df0b64bc`、
葵capture SHA256 `6e09f5493cdcfae51cde56e193dcf057585819b027700a51e3c3682e70763b5c`。

両音声とも128件中2件が異なり、比較ツールは終了コード1。差を許容する回帰は追加していない。
case54「。\t。−？0！。」（実タブ）は、原版4文、Go3文で冒頭の空文が1つ足りない。
case89「日五月8Ａ一零67人二生開（0人ａー七木ｂ。」では一零を原版イチレー、Goジューと読む。
通常原版段階観測private/seed15-akane-rei-stages.jsonlで、graph_inの一零６７人の候補が
イチレーロクジュー/2-3-4,s0,2-0-4になり、後続のシチニンを展開語列へ保持することを確認。
この観測は整形後に終了する診断でWAV一致を示さない。読み差と全Go WAV比較は次の課題。
既知CPU差も未解決。


seed15 case54の空文差は水平空白後の終端記号列のリセットで解消した。
追加32文章×両音声64件の読みと対象3文章×両音声6件のWindows全WAVが一致。
case89は未解決で、関連48文章×両音声の通常原版参考も保持した。
現行実装は各31件不一致、比較ツール終了コード1。全件保持し、結果による除外はしない。
詳細は[tab-sentence-original.md](tab-sentence-original.md)。


## seed15の解消

水平空白後の空文処理に加えて、混在数詞の文字種境界・小単位・助数詞の区間モードを再現し、
seed15の128文章×両音声が全読み一致した。seed5〜15は計2816件の通常回帰で一致する。
関連48文章×両音声の96件と追加64文章×両音声の128件も、全件を保持して読みが一致。
修正対象の独立通常原版との全WAV比較とLinux CGO=0での再現は
[mixed-rei-original.md](mixed-rei-original.md)に記録した。
読み一致を全2816件のGo WAV一致とは扱わない。既知CPU差も残る。


## seed16の係り先の差

新規128文章×両音声で、葵は全件一致、茜は数詞結合後の係り先の変更により1件不一致。
原版の128370/18fbe0と段階観測を照合して修正し、両音声全256件が読み一致した。
seed5〜16の計3072件と、今回の関連32文章×両音声64件も通常回帰へ追加。
全読み回帰は成功した。全WAV比較は実行中であり、全3072件のGo WAV一致とは扱わない。
詳細は[absorbed-chunk-original.md](absorbed-chunk-original.md)。既知CPU差も残る。


## seed17の追加読み検証

seed17 / generator_version=1 / max_units=24 / count=128を両音声の通常原版で採取した。
全256件のtalk成功・正常detach・crash/timeout/エラーなしを確認。
全保存原版WAVの長さ/SHA256とnative report SHA256も再検証した。
corpus SHA256 `ee73b8e21904f951431a180325fd92c3f0aed8813133d120d0177a4cd42579cc`。
茜capture SHA256 `5c7cb0bf660439457d28bb5d32087ef051178fbba84a6ae16e9a2be0caf4c14b`、
葵capture SHA256 `ed42620477cf2b5bba3ec82a3833895e475ddda3a8f9d96042a9e98c434c5553`。
参考はtools/compare_reading/testdata/seed17-{akane,aoi}-original.json。

両音声の全128件が読み一致。TestRandomSeed17LongReadingOriginalを通常回帰へ追加し、
3.61秒で成功した。seed5〜17の読みの対象は計3328件になった。
既存の読み回帰はseed16時点で49.250秒で成功している。ログは
private/random-reading-seed17-{akane,aoi}-comparison.log、random-reading-seed17-regression.log。

結果や文章の内容で選別せず、生成器の先頭4文章×両音声の8件について
TestRandomSeed17LongOriginalWAVで独立通常原版SDKとの全WAV比較を開始した。
未検証のひらがな列、連続句点、異種文字とタブ、長い単語列を含む。
ログはprivate/random-seed17-{akane,aoi}-wave.log。全8件の完了はまだ主張しない。
読み一致を全3328件のGo WAV一致とは扱わない。既知CPU差と全設定の再現も残る。


seed17の先頭4文章×両音声の8件について、全保存原版WAVとnative reportを再ハッシュし、
testdata/random_seed17_wave_original.jsonへ参照を追加した。
TestRandomSeed17LongReferenceWAVはseed・生成器版・生成上限・両音声の順序と入力文章を
生成器の全128件の先頭4件へ照合してから、全文からGo合成し全WAVの長さ/SHA256を比較する。
AOIKANE_RANDOM_SEED17_REFERENCE_AUDIT=1で実行する。
Linux amd64 CGO=0の実行を開始し、private/random-seed17-linux-reference.logへ記録中。
通常原版とのWindows比較・Linux再現の全8件完了はまだ主張しない。


seed17の独立通常原版SDKとのWindows全WAV比較は、茜604.62秒、葵635.78秒で全8件成功。
全原版とGoの保存WAVを参照とレポートの長さ/SHA256へ再照合し、計1385449 PCM16サンプルが一致。
report_sha256は独立比較2件へ更新し、元の観測時native report SHA256をcapture_report_sha256へ保持。
Linux全8件の再現は進行中。今回のWindows8件を全3328件のGo WAV一致とは扱わない。


## seed18の末尾空白の差

新規128文章×両音声で、茜は全件一致、葵は改行前の末尾空白を除いていたため1件不一致。
原版の解析語列を観測し、先頭を除き末尾空白を保持するように文分割を修正した。
両音声全256件と関連32文章×両音声64件が読み一致し、通常回帰へ追加。
既存全読み回帰とphrase全回帰も成功。seed5〜18は計3584件。
問題の全文は両音声とも通常原版SDKと全WAV一致し、短縮文・対照文の比較を継続中。
詳細は[space-tail-original.md](space-tail-original.md)。
別途、大単位の数詞32文章×両音声では各12件の読み差を見つけ、全件を保存して調査中。
詳細は[mixed-rei-big-unit-original.md](mixed-rei-big-unit-original.md)。任意文章の完成は未達。


seed17のLinux amd64 CGO=0の全8件再現も1020.97秒で全WAV長さ/SHA256が一致した。
原版の層値を入力へ差し替えず、全文から合成している。
ログはprivate/random-seed17-linux-reference.log。
seed18の末尾空白修正のWindows独立全6件も成功。詳細は[space-tail-original.md](space-tail-original.md)。


## seed19の混在数詞末尾の長音

seed19 / generator_version1 / count128 / max_units24の入力を固定し、通常原版SDKから
茜・葵の全256件の読みとWAVを採取した。status0、全format、complete、errorなし、
crash/timeoutなし、正常detachを確認し、全保存WAVの長さ/SHA256とSDKレポートSHA256を再照合。
corpus SHA256 006e062a26565e8a79235bcf9461a6e13f50dc91bd4abbc5598b78a279b10b40。
採取をtools/compare_reading/testdata/seed19-{akane,aoi}-original.jsonへ保存した。

両音声とも128件中1件（index81）が不一致で、比較ツールの終了コードは1。
入力は「1。b声02二八六三木2音八七学。」、原版のゼロニに対しGoはゼロニー。
前後の数詞・語句と第1文の読みは一致する。失敗入力を除外せず全128件を残す。
比較private/seed19-{akane,aoi}-comparison.json/.log。seed19全件の完成は未達。

短縮した02二八六・02二・02の原版段階観測は全3format、errorなし、呼出しenter/leave一致。
02二八六と02二では最初のgroupがゼロニ、02単独はゼロニー。
数詞変換直後にそれぞれ2-3-3,s0と2-3-4になっており、後段だけの長音除去ではない。
証拠docs/evidence/seed19-zero-ni-segments.json、raw private/seed19-zero-ni-segments.jsonl。
この診断は全WAV比較の証明とは扱わない。修正と全WAV検証は未完了。

TestRandomSeed19LongOriginalWAVで、生成器の先頭4入力を結果で選別せず
両音声の独立通常SDKと全文から比較するテストを追加して実行中。
ログprivate/random-seed19-{akane,aoi}-wave.log。
この代表8件の進行を256件の全WAV一致やindex81の修正完了とは扱わない。


## seed19の混在数詞の変換を修正

原版1043f0と10b010は別の経路で、単独02は前者、02二の02区間は後者へ入る。
Ghidraで1043f0から呼ばれるf7060を逆コンパイルし、二/五の末尾にーを加える処理を確認。
10b010の関連観測では025二がゼロニジューゴｂニ、002二がゼロゼロニｂニ、
05三がゼロゴｂサンになる。短縮3文章の全format、errorなし、呼出し釣り合いを確認。
証拠docs/evidence/seed19-zero-digit-short-segments.json。

関連48文章（12数詞×助数詞なし/人/本/Ｃ）を両音声の通常原版SDKから採取。
全96保存WAVとSDKレポートを再hashし、status0・complete・error/crash/timeoutなし・正常detachを確認。
corpus SHA256 fb53275df4b05239db2f146689c56cad712a8eb5b6a6b279a59b5a73dc9cb29b。
全入力をtools/compare_reading/testdata/seed19-zero-digit-48-{akane,aoi}-original.jsonへ保存。
茜初回は48件中31件不一致で、長音以外に012をゼロイチ/ニ、025をゼロニー/ゴと読んでいた。

数値fragmentの先頭ゼロは、ゼロのペアgroupを保持して後続をValueと通常数詞変換へ渡す。
残りが1桁かつ先頭ゼロが奇数個なら、最後のゼロと短い桁音をpairにする。
単独の桁読み経路は保持し、 literal零を含む一零などはこの数値化をしない。
最初の試作は一零を10としてジュウへ変えたため12件差が残った。
先頭ゼロの条件を加えた最終v3は関連48文章×両音声とseed19全128文章×両音声の全読み一致。
TestSeed19ZeroDigitReadingOriginalとTestRandomSeed19LongReadingOriginalへ全352件を通常回帰として追加。
結果による除外・許容差はない。seed5〜19のランダム読み回帰は計3840件になる。

既存全読み回帰65.323秒、追加352読み回帰5.107秒、numeral0.613秒/phrase81.201秒、go vetが成功。
ログprivate/seed19-zero-digit-v3-{reading,source}-regression.log、同-v3-new-reading-regression.logと同-v3-vet.log。
TestSeed19ZeroDigitOriginalWAVで元の不一致全文と025二・単独02の対照を
両音声の独立通常原版SDKと全WAV比較中。ログprivate/seed19-zero-digit-{akane,aoi}-wave.log。
全6件の比較完了前なので、新しい修正の全WAV一致はまだ主張しない。
seed19の代表先頭4文章の比較も継続中。任意文章・全CPU・全設定の完成は未達。


seed19代表先頭4文章×両音声の独立通常原版SDKとのWindows全8WAV比較も全一致し、茜757.69秒/葵794.27秒で正常終了。
全保存original/Go WAVをレポートと採取時の原版SHA256へ再照合し、計1786464 PCM16サンプルが一致。
参照testdata/random_seed19_wave_original.jsonは生成器先頭4文章の順序を保持し、
TestRandomSeed19LongReferenceWAV / AOIKANE_RANDOM_SEED19_REFERENCE_AUDIT=1で全文から再生成して全WAV比較する。
Linuxで最終修正ソースの全8件を再生成する比較を開始する。代表8件を全3840ランダム件の波形一致とは扱わない。

Linuxのログはprivate/random-seed19-linux-reference.log。参照の生成器/seed/最初の4入力/両音声の順序も厳密に検査する。参照追加後のgo vetが成功。


seed19修正確認の元全文・025二・単独02を使う独立通常SDKとのWindows全6WAV比較は、
茜352.08秒/葵359.14秒で全一致して正常終了した。全保存original/Go WAVをレポートと
seed19・関連48文章の採取時原版SHA256へ再照合し、950049 PCM16サンプルが一致。
参照testdata/seed19_zero_digit_wave_original.jsonを追加した。
TestSeed19ZeroDigitReferenceWAV / AOIKANE_SEED19_ZERO_DIGIT_REFERENCE_AUDIT=1で
全文からDLLなしで再生成し、全WAV長さ/SHA256を比較する。Linux CGO=0の全6件は実行中。
ログprivate/seed19-zero-digit-linux-reference.log。参照追加後のgo vetが成功。
clean main b4188edのCGO=0 CLIでも元全文の茜合成を実行中。完了前の一致は主張しない。

## seed20とランダム全WAV比較の分割

seed20 / generator_version1 / count128 / max_units24を両音声の通常原版SDKで採取。
全256件のstatus0、complete、error/crash/timeoutなし、正常detach、全保存WAVとレポート再hashを確認。
corpus SHA256 1feafd104c5acb073b6d3dc75d4e660238dfa6b7e3b3bc33a70dbc78dbb846f5。
両音声とも全128読みが一致し、TestRandomSeed20LongReadingOriginalへ全256件を保存した。
通常回帰4.888秒とgo vetが成功。seed5〜20のランダム読み回帰は計4096件。
今回の全256件の読み一致を全256WAV一致とは扱わない。

TestRandomOriginalWAVにもprepared corpusと同じmoduloによる分割を追加した。
AOIKANE_DIFF_SHARD/SHARDSを使い、全生成入力を保持したままindex % shards == shardを比較する。
count128・seed19・max_units24・shards8を固定して、両音声のshard0（各16入力）を実行中。
全128入力のcorpus.jsonが両音声で元の生成器出力と同一で、同じ16indexへ割当てられることを確認した。
割当ては結果・読み一致・合成可否で選別しない。すべての8組×両音声が完了するまで全128WAV一致としない。
代表先頭4件の比較だけでランダム128件の波形一致は証明できないため、この全件比較を進める。
不正な組数と別の組に属するreplay indexは、合成前に明示的に拒否することを確認。
従来のデフォルトはshard0/shards1。複数組で出力先を省略した場合は組別の保存先へ分ける。
ログprivate/random-seed19-full-shard0-{akane,aoi}-wave.log。各組は90分のtest timeoutで実行する。
全128入力を任意の文字列・全CPU・全設定の完成証明とは扱わない。


clean main b4188edのGo1.26.8/CGO=0 CLIで元の不一致全文を茜・synth --portable embeddedで合成した実行も正常終了し、
独立通常原版の全603272 bytesと一致。SHA256 897101f25adabb05727076fbd5012b7854834faf51822539a2b792e57394b3ba。
ログprivate/seed19-zero-digit-cli-{akane,build-info}.log。原版WAVそのものとbyte比較し、保存参照にも再照合した。


## 全ランダム組の完了確認

全128入力×両音声の完了を確認するtools/verify_random_wav_shards.pyを追加した。
全8組の正常終了ログ、各組の128入力・seed・生成器・割当て、欠落/重複のない全indexを検査する。
CPU/Go版/6threads/AVX512=falseが各組で一致し、合成errorなし・全row equal・first_difference=-1が必要。
全original/Go保存WAVをbyte比較し、長さ/SHA256をレポートと先に採取した通常原版へ再照合する。
PCM16 mono48kHzのWAV内容とフレーム数も検査する。2音声×128件の全件が満たされる場合だけcomplete=true。
失敗/未完了時は終了コード1でcomplete=falseを書き、古い成功結果を残さない。

正常な全組、終了前のレポート、欠けた音声/組、欠落・重複、CPU変化、
保存後に変えたPCM、両出力とレポートだけを同時に変更した場合の7テストが成功。
ログprivate/random-shard-audit-final-tests.log。
実行中の実データにも照合を行い、未終了のshard0を正しくcomplete=falseとして拒否した。
private/random-seed19-full-audit.json/.logはこの未完了結果であり、全256WAVの成功証明ではない。

全組終了後に実行するコマンド:

```powershell
python tools/verify_random_wav_shards.py --root private/random-seed19-full-wave --corpus private/random-seed19-reading-inputs.json --captures tools/compare_reading/testdata --logs private --log-prefix random-seed19-full --shards 8 --output private/random-seed19-full-audit.json
```

先頭ゼロ修正の全6WAVのLinux DLL-free CGO=0再生成は612.47秒で全体一致して正常終了した。
ログprivate/seed19-zero-digit-linux-reference.log。
seed19代表先頭4文章×両音声の全8WAVもLinux DLL-free CGO=0で1255.88秒で全体一致して正常終了した。
ログprivate/random-seed19-linux-reference.log。固定参照の全WAV長さとSHA256を厳密に検査した。
全128入力を8組に分けたWindows通常原版比較は継続中。未完了組を合格扱いしない。


## seed21の読み比較

seed21 / generator_version1 / count128 / max_units24を固定し、両音声の通常原版を全採取した。
コーパスprivate/random-seed21-reading-inputs.jsonのSHA256は
d6df7fb2bec670dcef249048566efd02b72c38b18bc1441af1304c1b692b8286。
原版の全256WAVの長さ・SHA256とnative reportのSHA256を再確認し、全case status0、
complete=true、通常終了、error/crash/timeoutなしを確認した。
全128文章×茜・葵の256読みはGoと完全一致。
採取tools/compare_reading/testdata/seed21-{akane,aoi}-original.json、
比較private/seed21-{akane,aoi}-comparison.json/.log。
通常回帰TestRandomSeed21LongReadingOriginalを追加した。読みの一致は全256Go WAVの一致証明ではない。

seed19の全WAV比較は両音声のshard0に加え、shard1も開始した。
各組16入力、全128入力のコーパスを保持してindex % 8による割当てを変えない。
ログprivate/random-seed19-full-shard1-{akane,aoi}-wave.log。全8組終了後の厳密再照合が必要。


## seed22の長さを増やした読み比較

seed22 / generator_version1 / count128 / max_units32を固定して通常原版を全採取した。
コーパスprivate/random-seed22-reading-inputs.jsonのSHA256は
e57ca1c6ef7a10b52d5dbd40ff2d27fdfb4766e8a6e030046cff51bc933b9500。
原版全256WAVとnative reportを再hashし、全case status0、通常終了、error/crash/timeoutなし。
128文章×両音声の全256読みがGoと一致し、TestRandomSeed22LongReadingOriginalが5.860秒で成功。
採取tools/compare_reading/testdata/seed22-{akane,aoi}-original.json、
比較private/seed22-{akane,aoi}-comparison.json/.log、回帰private/seed22-reading-regression.log。
読み一致は全256Go WAVの一致証明ではない。

## seed19のshard0完了

全128入力を8組に分けた通常原版/Go WAV比較のshard0は、
茜2309.898秒・葵2312.662秒で正常終了し、各16入力の全32WAVがbyte一致した。
保存した全original/Go WAVの長さ・SHA256を再確認し、両音声の採取済み原版へ再照合した。
計5353322 PCM16サンプル。入力・index割当て・CPU/Go版/6threads/AVX512=falseも確認。
証拠docs/evidence/seed19-shard0-wav.jsonはcomplete_shard=true、complete_corpus=false。
全8組の完了は未達。shard1を継続し、shard2を両音声で開始した。
ログprivate/random-seed19-full-shard2-{akane,aoi}-wave.log。全組の正常終了と全体再照合が必要。


## seed23の混在入力の読み比較

seed23 / generator_version1 / count128 / max_units32の全128文章×両音声256読みが一致。
コーパスSHA256は5d6863ef9b0c8077c509016280fff299a3dcd479ba10e75c2bd661247287126e。
通常原版の全WAVとnative reportを再hashし、status0・通常終了・error/crash/timeoutなし。
採取tools/compare_reading/testdata/seed23-{akane,aoi}-original.json、
比較private/seed23-{akane,aoi}-comparison.json/.log。
TestRandomSeed23LongReadingOriginalは8.258秒で成功。
混在生成器seed5〜23の計4864読みが一致。以下の追加数詞コーパスには未解決の差がある。

## 数詞・単位・助数詞を含む追加ランダムコーパス

従来生成器を維持し、tools/compare_readingの-generate -numeralsを追加した。
NumeralTextsは漢数字/半角全角数字、〇/零、十/百/千/万/億/兆/京、
助数詞、記号と周囲の単語を乱数で連結する。結果による入力選別を行わない。
generator_kind=numeral / generator_version1。seed1/count128/max_units16を固定した。
コーパスprivate/random-numeral-seed1-reading-inputs.jsonのSHA256は
af6b3432ac36eea05c0afe2f737a9ee0a426978ad719bccc3117d9c490416f7c。
入力列の固定hash・件数変更時のindex安定性・異なるseedのテストが成功し、
従来seed22生成結果が全byte同一であることも再確認した。
ログprivate/random-numeral-generator-tests.log、go vet private/seed23-generator-vet.log。

両音声の通常原版を全採取し、原版全256WAVとnative reportを再hash。
全case status0・通常終了・error/crash/timeoutなし。
Goとの読み比較は茜40件、葵41件が不一致で、比較ツールは終了コード1。
全128入力×両音声をtools/compare_reading/testdata/random-numeral-seed1-{akane,aoi}-original.jsonへ保存。
全入力と81件の差はdocs/evidence/random-numeral-seed1-reading.jsonへ保存。
比較private/random-numeral-seed1-{akane,aoi}-comparison.json/.log。成功・完成扱いしない。

学校百十９1、万5百時分百−、９十8・三を通常原版で読み取り専用観測した。
100a90/10b010の12 enter/leaveが対応し、全3formatが全SDK採取と一致。
百十/９１、万/５百、９十/８/三へ分割される。万の出力はマンで、イチマンではない。
証拠docs/evidence/random-numeral-seed1-segments.json、
raw private/random-numeral-seed1-segments.jsonl。これは変換経路の観測であり全WAV比較ではない。
単独大単位・混在数字の分割・小数点・ゼロ助数詞等を原版の順序へ合わせる作業が残る。


## seed19全WAV比較で改行終端の未対応を確認

実行中のshard2でcase50とcase58が両音声ともGoの音声生成errorとなった。
通常原版は正常にWAVを生成し、4保存WAVの長さ・SHA256を採取済み原版へ再照合した。
読みの比較は一致しているが、生成器が返す<N>をParseFormattedKanaが拒否している。
CompilePhones/CompileProsodyの終端表にもNがない。原版の終端phone・特徴・ポーズを観測して移植する必要がある。
証拠docs/evidence/seed19-newline-wave-failures.json。reportのhashは実行途中のsnapshotである。
shard2全体はまだ実行中で、終了ログと全件結果の確認が必要。原版WAVの成功をGo成功へ読み替えない。


## 改行N終端の音声生成を再構築

case58の通常原版を読み取り専用で観測し、compileToImphonemeとモデル入口の特徴を保存した。
終端は--_N、phinfo=CF00、prsinfo=0015、beos=4、punc=2、調音bitmap=40000001。
観測時の全WAVはフックなしのSDK採取と全byte/長さ/SHA256が一致。
証拠docs/evidence/newline-end-n-original.json、raw private/newline-end-n-observation。

ParseFormattedKana/CompileProsody/CompilePhones/ModelArticulationへN終端を追加した。
原版のphn/acc/punc/nima/niab・終端phone/bitmapとの通常回帰が成功。
frontend全回帰2.139秒、synthesisの原版モデル回帰171.399秒、go vet成功。
ログprivate/newline-end-{frontend,synthesis}-regression.log、同-final-vet.log。

元の2文章×茜・葵は独立通常原版SDKと全4WAVが一致し、
茜215.81秒・葵183.52秒で正常終了。全保存WAVを再hashし、
seed19の採取済み通常原版にも再照合した。計647351 PCM16サンプル。
ログprivate/newline-end-{akane,aoi}-wave.log。
参照internal/portable/testdata/newline_end_wave_original.jsonと
TestNewlineEndReferenceWAVを追加し、Linux DLL-free CGO=0の全4WAV比較を開始した。
Linux全4WAVは238.21秒で全体一致して正常終了した。ログprivate/newline-end-linux-reference.log。

修正前のshard2は茜1464.40秒・葵1555.65秒で終了コード1。各16件中N終端の2件だけが失敗した。
失敗を含む出力はprivate/random-seed19-full-wave-before-n/shard-2へ保管。
ログprivate/random-seed19-full-shard2-{akane,aoi}-wave-before-n.log。
修正版で同じ全128入力・index % 8によるshard2の割当てを両音声で再実行している。
入力・許容差・割当てを変更して失敗を除外していない。全8組終了後の全体再照合が必要。


## N終端のLinux・CLI確認とshard1完了

Linux DLL-free CGO=0のN終端参照全4WAVは238.21秒で全体一致して正常終了。
clean 3990142、Go1.26.8、CGO_ENABLED=0、GOAMD64=v1、vcs.modified=falseのCLIでも
--portable embeddedで茜の（0改行。を合成し、全WAVを通常原版そのものとbyte比較した。
117644 bytes、SHA256=3809fc672c7e5946c56a5472e67835e7460a5a3b0f160c347be5727b982dbad4。
保存参照へも再照合。ログprivate/newline-end-cli-{build-info,akane}.log。
出力private/newline-end-cli-akane.wav。dist/aoikane-dev.exeは観測用原版driverとして維持。

seed19のshard1は茜2916.74秒・葵2972.17秒で正常終了し、全32WAVがbyte一致。
全保存original/Go WAVを再hashし、採取済み原版へ再照合した。計7067932 PCM16サンプル。
証拠docs/evidence/seed19-shard1-wav.jsonはcomplete_shard=true、complete_corpus=false。
shard0/1の完了計64WAVに加え、修正後のshard2を両音声で再実行し、shard3も両音声で開始した。
ログprivate/random-seed19-full-shard3-{akane,aoi}-wave.log。shard4〜7と全組の最終再照合は残っている。


## 2026-10-13 単位から始まる数詞と数詞の品詞判定を修正

原版100a90の分割と120780の名詞数詞POS条件を再構築。
百十９１を一つの値へ加算せず、万５百へ暗黙の一を加えず、分割した読みを結合する。
京は選択済み品詞が地名のときキョーを維持し、京五は辞書の人名キョーゴを維持する。
単位先頭56文章×茜・葵の全112読みが通常原版と一致。数詞ランダム128文章×両音声は
茜40→24、葵41→25の読み差へ改善し、各16件を修正、新規退行0件。残る49件は全入力とともに保存した。
証拠docs/evidence/unit-head-{kyou-original,reading-comparison}.json。
既存の全原版読み回帰79.439秒、numeral/phrase全回帰0.705秒/82.376秒で正常終了。
代表3文章×両音声の全6WAVは通常SDKと全体一致（茜252.684秒、葵253.239秒）。
全保存WAVを再hashし、別途採取済み原版へ照合した。Linux CGO=0で同じ6参照の再現を検証中。

seed19の修正後shard2とshard3も両音声とも正常終了し、保存全64WAVを再hashして採取済み原版へ再照合。
shard2は5495221、shard3は7221441 PCM16サンプル。証拠docs/evidence/seed19-shard{2,3}-wav.json。
shard0〜3の計128WAV（各音声64入力）が完了。shard4を両音声で開始した。
残りshard4〜7と全組の最終監査、数詞49読み差、全CPUの一致は未解決。


## 2026-10-13 数詞の整数fragmentと桁繰上を再構築

10b010の右からの桁展開と100a90の分割条件を移植し、十十・千千の足し算や
明示された一の削除を防いだ。一千はイッセンに変化する。桁区切り繰上の+2n#1uを
数字の展開として扱い、10000十は原版と同じ１０００００を読む。
先頭ゼロのpaired accentと零の混在時の分割は既存の原版観測に沿って保持した。
原版の読み取り専用dispatch観測は全formatterを独立SDK採取へ照合。
証拠docs/evidence/integer-fragment-{original,long-original}.json。

関連68文章と長い桁16文章×両音声、計168読みが全一致し、全原版WAV/native reportを再hash。
通常読み回帰とnative sound回帰へ追加した。final原版読み回帰104.368秒で正常終了。
numeral/phrase全回帰1.184秒/87.706秒とgo vet成功。
数詞ランダムseed1の全256入力は各6件改善、新規退行0件。
読み差は茜24→18、葵25→19、計37件残る。差を含む全入力を保存した。
証拠docs/evidence/integer-fragment-reading-comparison.json。

代表3文章×両音声の6WAVは通常SDKと全体一致（茜433.581秒、葵440.039秒）。
長い桁の代表2文章×両音声の4WAVも全体一致（茜116.302秒、葵117.457秒）。
全10保存WAVを再hashし、別途採取した原版WAVへ照合。
internal/portable/testdata/integer_fragment{,_long}_wave_original.jsonへ参照を保存し、
Linux DLL-free CGO=0で最終ソースから全10WAVを再生成する厳密比較を開始した。

前回の単位先頭・POS修正のLinux全6WAVは458.76秒で全体一致し、終了コード0。
ログprivate/unit-head-linux-reference.log。
seed19全WAV比較のshard4は両音声で継続中。全8組の最終監査は未完了。
数詞37読み差、2単位の桁繰上・巨大数のfallbackなど未検証入力、全CPUの数値差は残る。
任意文章・全設定・全CPUでの完成証明は未達。


## 2026-10-13 整数fragment修正のstandalone CLI確認

clean edf0b8e、Go1.26.8、CGO_ENABLED=0、GOAMD64=v1、vcs.modified=falseの
embedded CLIで茜の学校１百5。を合成。213260 bytes、SHA256=7bf58d80b00c1b71eb42d0d9d55f36223a08821ec1d20123a18a96b436d0c99f。
フックなし通常原版WAVそのものと全byte比較し、独立採取WAVと保存参照へも再照合した。
ログprivate/integer-fragment-cli-{build-info,akane}.log、出力private/integer-fragment-cli-akane.wav。
Linuxの全10WAV再現とseed19全WAV比較のshard4はまだ進行中。
数詞37読み差の次の調査として、ゼロ＋助数詞＋後続数字などの関連50文章を両音声で原版採取中。
任意入力・全CPU・全設定の完全一致はまだ未達。


### 2026-10-13 秒を挟む数詞・零分の原版処理を移植

原版11f9c0/100a90の読み取り専用観測11件を独立SDK採取へ照合し、
全formatterとstage/segmentの出入りを検証。時間パターン N'秒n/J'秒j は
同じ数字表記の後半を小数部として読む。最後の助数詞許可は標準ルールの0、
関西ルールの*を読み取り、〇秒零秒の茜レー／葵ゼロの違いも再現した。
混在表記・単位付き後半はgeneric数詞とs0境界で同じmorphへまとめる。
既に消費した助数詞の秒を次の数詞のconnector blockerにしない。
零分は特殊読み候補を使わず、原版1043f0のレーからレーフンを作る。
証拠docs/evidence/zero-counter-context-{akane,aoi}-original.json。

関連50文章と追加時間表記24文章×両音声、全148読みが一致。
全148原版WAVとnative reportを再hashして通常の読み回帰へ保存した。
numeral/phrase回帰0.563秒/75.206秒、既存全原版読み回帰90.837秒、
新規148読み回帰2.530秒、go vetは正常終了。
数詞seed1の全256入力を保持して比較すると茜18→14、葵19→13、
新しい不一致0、計10件改善。残る読み差27件を含め全入力を証拠に保持。
証拠docs/evidence/zero-counter-reading-comparison.json。

代表〇秒２声、〇秒零秒、零秒八億Ａ、零分×両音声の8WAVを通常SDKで比較し、
全byte一致（茜280.52秒、葵281.92秒）。保存した両WAVを再hashし、
独立SDK採取のWAVそのものとも全byte照合した。
参照internal/portable/testdata/zero_counter_wave_original.json。
Go1.26.8・Linux amd64・CGO=0のDLL-free再生で同じ8WAVの厳密比較を継続中。
ログprivate/zero-counter-linux-reference.log。

前回整数fragmentのLinux全10WAV比較は正常終了し、全体一致。
6件628.65秒、長い桁4件129.70秒。証拠docs/evidence/integer-fragment-linux-wav.json。
seed19のshard4も両音声で正常終了し、保存全32WAVを独立原版へ再照合、
4463161 PCM16サンプルが一致。証拠docs/evidence/seed19-shard4-wav.json。
seed19のshard0〜4計160WAVが完了。shard5は両音声で継続中、shard6/7は未実行。
全8組の最終監査、数詞27読み差、巨大数など未検証入力、全CPUの数値差は残る。
任意入力・全設定・全CPUでの完全一致はまだ未達。


### 2026-10-13 秒パターン修正のstandalone CLI確認

clean 85f21a9、Go1.26.8、CGO_ENABLED=0、GOAMD64=v1、vcs.modified=falseの
embedded CLIで〇秒零秒。を茜・葵の両音声で合成し、197062 / 198706 bytes。
両方とも通常原版WAVそのものと全byte比較し、独立SDK採取と保存参照にも一致。
証拠docs/evidence/zero-counter-cli-wav.json、ログprivate/zero-counter-cli-{build-info,akane,aoi}.log。
同じ8原版WAVのLinux厳密比較とseed19 shard5は継続中。
数詞の残り27読み差・全ランダムWAV・全CPUなどの完成条件は未達。
次の小数・単位調査3件のnative dispatch/formatterは独立SDKへ照合済み、
private/decimal-unit-context-akane-validated.jsonへ保存した。Goとの一致証明ではない。


### 2026-10-13 小数の単位fragment・先頭単位を原版へ揃える

千．千２億5など、通常の小数部パターンが受け付けない単位を含む入力を
原版100a90のgeneric connector処理へ送る。単位・零をValueへまとめてしまわず、
元の表記ごとに数詞readerを通し、末尾のテンとs0境界を同じaccent groupへ追加する。
万．５はイチマンテンではなくマンテン、０億．５はゼロオク＋テン。
兆一零．千の一零はpaired groupを保ち、裸の先頭単位に一を足さない。
単位付き小数部のEngパターンが小文字nを要求する範囲も再現した。
原版の読取専用dispatch/候補をSDK formatterへ照合し保存した。
証拠docs/evidence/decimal-unit-context-akane-original.json。
この観測の９十8・三は次の残課題で、今回一致した入力とは区別している。

新規48文章×両音声、全96読みが一致。全96原版WAV/native reportを再hashし、
通常読み回帰に追加した。新規回帰0.973秒、既存全読み回帰88.363秒、
numeral/phrase全回帰0.613秒/75.420秒、go vetは正常終了。
数詞seed1の全256入力を維持し、茜14→7、葵13→6、計14件改善、新しい不一致0。
差の残る13件を含む全入力をdocs/evidence/decimal-unit-reading-comparison.jsonへ保存。

兆一零．千Ａ19、千．千２億5、０億．５、万．５×両音声の代表8WAVも
フックなし通常SDKと全体一致（茜355.07秒、葵352.32秒）。
保存した原版/Go全8WAVを再hashし、独立SDK採取のWAVそのものにも全byte一致。
参照internal/portable/testdata/decimal_unit_wave_original.json。
Go1.26.8・Linux amd64・CGO=0・GOAMD64=v1から同じ8WAVの厳密比較を継続中。
ログprivate/decimal-unit-linux-reference.log。

前回のゼロ＋秒/零分修正はLinux全8WAVでも正常終了、388.45秒で全体一致。
証拠docs/evidence/zero-counter-linux-wav.json。
seed19全WAVのshard0〜4計160WAVは完了、shard5両音声は継続中。
shard6/7と全8組の最終監査は未完了。
残る数詞13読み差の6文章については、原版のchunkごとの繰返しstageをすべて保持して
SDK formatterへ照合、private/numeral-residual-context-akane-validated.jsonへ保存した。
中点・単位とアクセント処理の継続範囲、巨大数などの未検証入力、CPU数値差は残る。
任意入力・全設定・全CPUでの完全一致はまだ未達。


### 2026-10-13 小数単位修正のstandalone CLI確認

clean 65370e2、Go1.26.8、CGO_ENABLED=0、GOAMD64=v1、vcs.modified=falseの
embedded CLIで万．５。を茜・葵で合成、165166 / 163520 bytes。
通常原版WAVそのものと全byte一致し、独立SDK採取と保存参照へも再照合した。
証拠docs/evidence/decimal-unit-cli-wav.json。
次の中点・継続範囲用24文章×両音声も原版採取と全48WAV/reportの再hashを完了。
private/numeric-continuation-24-{akane,aoi}-original.jsonへ保存。Goとの一致は未証明。
Linux小数単位8WAVとseed19 shard5の厳密比較は継続中。
数詞13読み差、全ランダムWAV、全CPUなどの完成条件は未達。


追加の継続範囲24文章×両音声を65370e2のGoへ比較したところ、茜21・葵20の読み差がある。
全48入力と原版の採取情報をフィルタせずdocs/evidence/numeric-continuation-reading-baseline.jsonへ保存。
中点、点、３項のノ区切り、単位と秒、零＋単位、点の後の万万などが次の対象。
上記の「残り13件」は既存seed1の128文章×両音声における件数で、
任意の数詞に残る全件数を意味しない。新しい関連コーパスも完成判定に含める。
