# 混在数詞の大単位の未解決差

8数詞×助数詞なし/人/本/Ｃの32文章を両音声の通常原版SDKで全合成して採取した。
元の32件をすべて保持し、全talk成功・正常detach・crash/timeout/エラーなし、
全原版保存WAVの長さ/SHA256とnative report SHA256を再検証した。
corpus SHA256 `0b2115e3a61bf2f279fb0b01b9cc679be48e3e61bfee043b9f2452ff980b311a`。
参考はtools/compare_reading/testdata/mixed-rei-big-unit-32-{akane,aoi}-original.json。

両音声とも32件中12件が不一致で、比較ツールは終了コード1。
一零十万のジューマンがGoではジュー/マンに分かれる。
二零十億49のジューオクもGoではジュー/オクに分かれる。
零一万０３は前段で零一が分かれ、原版はマン/ゼロサンだがGoはイチマン/サンとなる。
上記3数詞×4助数詞の全12件を残し、許容差・結果による除外は追加しない。
この失敗を合格扱いする通常回帰も追加していない。

原版の100a90/107920など数詞区間と助数詞の観測を進めている。
関連traceはprivate/mixed-rei-big-unit-segments.jsonl、ログはmixed-rei-big-unit-segments.log。
段階診断の終了を全WAV一致の証明とは扱わない。大単位の修正・全WAV比較は未達。


## 大単位区間を再現

原版100a90の3入力のdispatchを追加観測し、3件すべてformat到達・errorなし、
100a90/1043f0/10b010の呼出しが全て釣り合うことを確認した。
数詞と助数詞の引数・累積読み/表記はdocs/evidence/mixed-rei-big-unit-segments.jsonへ保存。
一零十万は1043f0へ一零を渡し、10b010へ十万をまとめて渡す。
零一万０３では100a90の入力は万０３で、裸万と先頭ゼロの桁読みを残す。
二零十億49では10b010へ十億、続いて４９を渡し、音はジューオクヨンジューキューへ連結する。

Goの数詞区間に大単位を追加し、 literal零を含む同一文字種の数詞も
単位を分割する旧処理の前に、この区間処理へ渡すよう変更した。
複数桁→小単位のｂは保持するが、万など大単位の前には同じ条件を適用しない。
連結された音のテーブル照合では大単位を直前の数詞へ付け、そのアクセントと助数詞keyを保持。
単位後の先頭ゼロを大単位でも検出し、裸万をイチマンへ、０３をサンへ置き換えない。

初回試作は12件不一致が残り、従来一致していた23万・67兆も区切っていた。
原版の小単位限定の分岐に合わせて調整した最終試作は、両音声32件が全読み一致。
TestMixedReiBigUnitReadingOriginalへ全64件を通常回帰として追加した。
既存を含む全読み回帰58.578秒、numeral/phrase全回帰0.585秒/75.975秒、go vetが成功。
ログはprivate/mixed-rei-big-unit-v3-{reading,source}-regression.log、mixed-rei-big-unit-v3-vet.log。

上記3文章×両音声6件をTestMixedReiBigUnitOriginalWAVで独立通常原版SDKと全WAV比較中。
一零十万人は両音声とも222960 bytesで全WAV一致した。残りの全完了はまだ主張しない。
ログはprivate/mixed-rei-big-unit-{akane,aoi}-wave.log。
今回の64件の読み一致を任意の大単位・全CPU・全設定での音声完全一致とは扱わない。


独立通常原版SDKとのWindows全6件は茜301.75秒、葵298.74秒で全WAV一致した。
両エンジンの保存WAVをレポートと先に採取した32件の原版WAVハッシュへ再照合し、
全6件で756902 PCM16サンプルが一致。参照はtestdata/mixed_rei_big_unit_wave_original.json。
AOIKANE_MIXED_REI_BIG_UNIT_REFERENCE_AUDIT=1で全文からDLLなしで再生成して比較できる。
Linux amd64 CGO=0の全6件を実行中。ログはprivate/mixed-rei-big-unit-linux-reference.log。

さらに大単位後の先頭ゼロ、十億/十京、小単位との重なり、垓、通貨を含む16数詞・語句に
助数詞なし/人/本/Ｃを付けた64文章を、両音声の通常原版SDKから採取中。
結果に応じて入力を除外しない。corpus SHA256は
`7d49ed44c37364b75ba0deaa3ae5efedef0d9aad863c28a65c4a08e0e5b0a4ba`。
原版採取とGo比較の全完了前に、新しい64件の一致は主張しない。


## 追加64文章の差を修正

追加64文章の原版採取は両音声とも正常終了し、全128保存WAVの長さ・SHA256を
採取結果と独立SDKレポートへ再照合した。初回は両音声とも8件不一致だった。
二零十京49の4助数詞形はジュッケイをジューケイと読み、
一零０３万の4助数詞形はゼロサンマンをゼロ/サンマンと分けていた。
全64入力を維持したまま両音声のfixtureと通常回帰
TestMixedReiBigUnitExtendedReadingOriginalへ保存した。

原版4入力（二零十京49人、一零０３万本、十京、０３万）の段階観測は
全formatに到達し、errorなし、100a90×4/1043f0×2/10b010×5のenter/leaveが一致。
docs/evidence/mixed-rei-big-unit-64-segments.jsonに読みと引数を保持する。
f7a10の逆コンパイルでは、兆・京の前の一/八/十で末尾1モーラをッへ変更している。
Goの大単位付与と連結音照合へこの変化を加え、先頭ゼロの桁読みの最後のグループを
大単位へ付けるよう変更した。０３万は原版の2-5-6と一致する。
試作でジュッのテーブルkeyが欠けて数詞変換が失敗したため、促音化した語幹も登録した。
最終比較は追加64文章×両音声で全読み一致（v4比較レポート）。除外・許容差はない。

numeral/phrase全回帰0.611秒/71.487秒、既存を含む全読み回帰61.107秒、go vetが成功。
ログはprivate/mixed-rei-big-unit-64-{source,reading}-regression.logと同-vet.log。
TestMixedReiBigUnitExtendedOriginalWAVで上記4文章×両音声の独立通常SDKとの全WAV比較を開始。
比較中のため、新しい8件の波形一致は未証明である。

前段の大単位3文章×両音声はLinux amd64 CGO=0でも全6WAV一致、437.85秒で正常終了した。
clean main f41f3e8のDLL-free CLIでも、茜の零一万０３本は全243114 bytesが
独立原版WAVと同一（SHA256 5ef517b52ec303e274bd109afefd4a077cfa349fa116d6be4e2bd8607e21b958）。
今回の追加64文章の修正後の波形比較とは区別する。


一/八/十の大単位付与を1の単独桁にも適用した最終ソースの再確認は、
numeral0.595秒・phrase78.787秒・全読み回帰66.166秒で成功、go vetも成功。
ログはprivate/mixed-rei-big-unit-64-final-{source,reading}-regression.logと同-final-vet.log。
追加の促音条件確認として一京/八京/十京/一兆/八兆/十兆/０８京/０１兆を
助数詞なし/人/本/Ｃと組み合わせた32文章の原版採取を開始した。
採取完了・比較前なのでこの32件の一致は未証明。


追加の促音32文章は茜で通常原版採取・全32WAV再hashが完了。corpus SHA256 b9519f3c175b2fe5f1d9844362b48a1160554ac29fc5e5c718824d0df87fca46。
一/八/十の京・兆（先頭ゼロなし）は全読み一致したが、０８京/０１兆の計7助数詞形が不一致。
原版はゼロ/ハッケイ、ゼロ/イッチョーに分けるが、Goはゼロハッケイ等の単一groupへまとめる。
０３万と同じゼロ桁group付与を促音化した数詞にも適用した条件が広すぎる。
全32入力をfixtureへ保持し、許容差や除外は加えない。葵の同32入力の原版採取を開始。
今回の不一致の解消と全WAV確認は未達。ログはprivate/mixed-rei-big-unit-sokuon-32-akane-comparison.log。


## 先頭ゼロと促音化の組み合わせ

促音32文章は葵も採取が完了し、全32保存WAVとSDKレポートを再hashした。
茜と同じ7件の読み不一致を確認した。０８京本・０１兆人・０３万の段階観測は
全3formatに到達し、100a90×3/10b010×3がenter/leave一致、errorなし。
累積音はゼロハッケイ/ゼロイッチョー/ゼロサンマンであり、
最終formatでは前2つだけゼロが独立する。証拠は
 docs/evidence/mixed-rei-zero-sokuon-segments.json。

最初に全先頭ゼロを音テーブルで再照合する試作を行い、促音32文章は両音声一致したが、
既存64文章の０３万関連8件が分裂した。全読み回帰は失敗し、
その試作のLinux extended全8WAVも4件が不一致で終わった。
ログはprivate/mixed-rei-zero-sokuon-reading-regression.logと
private/mixed-rei-big-unit-extended-linux-reference.log。失敗は合格扱いしていない。
促音化しない場合は元の桁groupを保持するv3修正で全読み回帰68.502秒が成功した。

追加で００３万/０１８京/０１１兆/０８億/００８京/００１兆/〇八京/〇三万を
助数詞なし/人/本/Ｃと組み合わせた32入力を両音声から採取。
全64WAV/レポート再hash済み、corpus SHA256は
4a52cc0893974a03044e8a8f6c75c3d11f488bfd10b86f525c34ce680168f09a。
茜初回は14件不一致で、18をイチ/ハチと読み、連続ゼロを別groupへ分割していた。
促音化した大単位では先頭ゼロの桁groupを保持し、後続数値を通常の数詞として読むよう変更。
この区間間にｂを追加しない。０３万など変化しない場合のペアgroupは維持する。
最終v4では促音32・追加ゼロ32の両音声すべてが全読み一致した。
TestMixedReiBigUnitSokuonReadingOriginalとTestMixedReiZeroBigUnitReadingOriginalへ
全128件を保存し、除外・許容差はない。

最終v4のnumeral0.735秒/phrase64.131秒・全読み回帰58.579秒、追加128読み回帰1.296秒、
go vetが成功。ログはprivate/mixed-rei-zero-sokuon-v4-{source,reading}-regression.log、
同-v4-new-reading-regression.logと同-v4-vet.log。

独立通常SDKとのextended4文章×両音声8WAVはWindowsで茜306.60秒/葵296.42秒の全一致。
全保存original/Go WAVとレポートを再hashし、841556 PCM16サンプルが一致。
参照testdata/mixed_rei_big_unit_extended_wave_original.jsonと
AOIKANE_MIXED_REI_BIG_UNIT_EXTENDED_REFERENCE_AUDIT=1で全文からのDLL-free再生成を追加。
Linuxの修正済みv3再比較をprivate/mixed-rei-big-unit-extended-v3-linux-reference.logで実行中。

０８京本・０１兆人・０８京人の独立通常SDK比較も全6WAV一致、茜218.35秒/葵216.41秒。
保存WAVを先に採取した促音32文章の原版WAVハッシュにも照合した。
参照testdata/mixed_rei_big_unit_sokuon_wave_original.jsonを追加し、最終v4の
Linux DLL-free CGO=0全文再生成をAOIKANE_MIXED_REI_BIG_UNIT_SOKUON_REFERENCE_AUDIT=1で実行中。
ログprivate/mixed-rei-big-unit-sokuon-v4-linux-reference.log。

最終v4は００８京なども扱う変更を含むため、０１８京本・００８京Ｃ・００１兆人・００３万の
新しい4文章×両音声を通常原版SDKと全WAV比較中。
TestMixedReiZeroBigUnitOriginalWAV、private/mixed-rei-zero-big-unit-{akane,aoi}-wave.log。
これら未完了の比較を全一致とは扱わない。全CPU・任意入力・全設定の完成証明も未達。


最終修正の複数ゼロ4文章×両音声はWindowsの全8WAVが一致し、茜333.41秒/葵344.94秒で正常終了。
保存original/Go WAVをレポートと追加32文章の採取時原版SHA256へ全件再照合した。
計846372 PCM16サンプルが一致。参照testdata/mixed_rei_zero_big_unit_wave_original.jsonを追加し、
TestMixedReiZeroBigUnitReferenceWAV / AOIKANE_MIXED_REI_ZERO_BIG_UNIT_REFERENCE_AUDIT=1で
全文からのDLL-free再生成と全WAV長さ/SHA256比較を行う。Linux版を実行中。
ログprivate/mixed-rei-zero-big-unit-linux-reference.log。参照追加後のgo vetも成功。

clean main 73f872b、Go1.26.8、CGO_ENABLED=0の実行ファイルでも葵０１８京本を
synth --portable embeddedで生成し、独立通常原版WAVと全222608 bytesが一致した。
SHA256 9a6ed61d44c3ea6542efee62287b24a8fa3bb4aa41c3b02a7e2009cf0eda47d3。
ログprivate/mixed-rei-zero-big-unit-cli-{aoi,build-info}.log。

修正済みv3のextended8WAVはLinuxで全一致、482.26秒で正常終了。
最終v4の促音6WAVもLinuxで全一致、435.71秒で正常終了。
ログprivate/mixed-rei-big-unit-{extended-v3,sokuon-v4}-linux-reference.log。
層値・中間特徴の差し替えをせず、全文から生成したWAV全体を比較している。


複数ゼロ8WAVのLinux amd64 CGO=0全文再生成も全体一致、470.33秒で正常終了した。ログprivate/mixed-rei-zero-big-unit-linux-reference.log。
