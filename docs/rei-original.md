# 零と単独数単位の原版再現

2026-10-12、seed5の読み比較で残っていたcase105の長音とcase125の
アクセント／区切りの差を修正した。どちらも「零」が数値読みに入らず、
隣の助数詞と別の辞書語のまま処理されることが原因だった。

bancharcxの数字表は「零」をType=Jap / Read=レイ / Sound=レーとしている。
Goの数字判定と値変換表には「零」がなかった。数字判定に加え、
読みでは「〇」のゼロへ置き換えず「零」の表の発音を保持する。
「零零」「一零」等は桁の読みを2桁ごとのアクセント群にまとめ、
「零十」「零万」等は数と単位の間の群を保持する。
「零零零」の最後の零も空の発音へ落とさない。

追加比較で単独の「万」「億」「兆」にイチを補ってしまう問題も発見した。
原版の単独万はマン、単独億はオク。単独数単位は表の文字通りの読みを使い、
一万等の位取りの値変換には従来の数値処理を使う。

## 読みの検証

- seed5 / generator=1 / max_units=12、128件×葵・茜=256件は全読み配列一致。
- 零、連続零、混在数字、助数詞、単独数単位、前後の辞書語の追加136件も全読み配列一致。
- 追加の期待値は原版6回の正常な通常SDK採取から作成した
  internal/portable/testdata/rei_reading_original.json。採取元のSHA256も保存。
- TestReiReadingOriginalとTestRandomSeed5ReadingOriginalを通常の回帰テストへ追加。
- 従来の読み回帰テストと93件の保存原版参考も一致。numeral全テストとgo vetが成功。

原版graph_inの零零はレイレイ/レーレー/2-3-4、零十は
レイｂジュウ/レージュー/1-1-2,s0,1-1-2、零一日は
レイイチニチ/レーイチニチ/2-4-6だった。
観測元はprivate/rei-extended-graph.jsonl。

## 完全WAVの別比較

通常原版SDKを観測ツールなしで独立に呼び出し、Windows amd64、CGO=0のGoと
以下の5文章×葵・茜=10件をWAV全バイト比較した。すべて一致した。
合計1206043 PCM16サンプル。途中tensorを再現入力に使用していない。

- 開0a土木土零日ー下A三。
- 七金六九B零C校3。
- 万。
- 零零零日。
- 零万人。

原版はAMD64 Family 25 Model 33 Stepping 0、通常threads=6、AVX512 convolutionなし。
原版WAV長・SHA256と独立比較reportのSHA256を
internal/portable/testdata/rei_wave_original.jsonに保存した。
通常SDKの独立原版WAVは、読み採取時に同じ文を原版が合成したWAV長・SHA256とも一致した。
Windows元ログはprivate/rei-seed5-wave-{akane,aoi}.log（275.64秒 / 285.03秒）、
private/rei-extended-wave-{akane,aoi}.log（161.85秒 / 169.92秒）。

Linux amd64、CGO=0でも保存原版のWAV長・SHA256が10件すべて一致した。
長い2文章×両音声の4件はprivate/rei-wave-linux.log（516.81秒）、
追加3文章×両音声の6件はprivate/rei-extended-wave-linux.log（220.23秒）。
途中tensorも原版DLLもLinux再現入力には使用していない。

Windows実行ファイルdist/aoikane-purego.exeも修正commit da91c2eから更新した。
go1.26.8、CGO_ENABLED=0、vcs.modified=falseを確認。
embedded資産で茜の「万。」を読み上げ、独立原版WAVの114764 bytes / SHA256
eb2723eaaccc43e82d31762d67bae11edcffb4bfa6a67d50a6fe9ec4368f379fとも一致した。
CLIログはprivate/rei-cli.log。

```powershell
$env:CGO_ENABLED = '0'
$env:AOIKANE_REI_REFERENCE_AUDIT = '1'
go test ./internal/portable -run '^TestReiOriginalWAV$' -count=1 -timeout 30m -v
```

## 次の相違

seed6の128件×葵・茜も通常原版から取得し、各2件の読み差が新たに見つかった。
case45「葉音3ー一。」では原版は３ー一を1つの数としてまとめ、
サンｐイチ/サンイチ/2-0-2,s1,2-2-2になるがGo側は短い区切りへ落ちる。
case46のタブ前のマイナスは原版では無音候補を選ぶがGo側は発音される。
元graphはprivate/reading-seed6-failures-akane-graph.jsonl。
参照データを除外せず、厳密比較は非ゼロ終了のまま保持した。
既知Intel Vocoder等のCPU差も未解決。392件の読み一致や10件のWAV一致を、
任意文章・全CPUの保証として扱わない。

上記seed6の読み差は後の数字接続・マイナス修正で解消した。
追加の原版比較と完全WAV検証は[connector-original.md](connector-original.md)を参照。
