# 数字接続と英字後のマイナスの原版再現

2026-10-12、seed6 / generator=1 / max_units=12の各音声128文章から発見した
case45「葉音3ー一。」とcase46「ЖＡ−\t0（あΩ。。」の読み差を修正した。
後者の\tは実際のタブ。タブは文字正規化で削除されるため、原版の無音マイナスは
タブそのものではなく、直前の英字に由来する。

数字間の「ー」「―」を数の接続として扱い、従来のハイフン処理へ接続した。
「3ー1」「三ー一」はサンノイチだが、Jap/Eng表記の混在する「3ー一」は
サンｐイチ/サンイチ/2-0-2,s1,2-2-2になることを原版graphで確認した。
表記の混在するハイフン・長音・ダッシュ・スラッシュ・コロンは、
整数・比率・分数等に置き換えず書かれた休止を保つ。
この接続内の助数詞では特殊読みを適用せず、3ー一人の後半はヒトリではなくイチニン。

マイナスが英字のPOS=0xb040000の直後に接続される場合は無音候補を保持する。
A−3、B−3、α−3、Ω−3で確認した。学校−3、あ−3、Aの後に空白を挟む場合や
プラスは原版に従って発音する。数の直前の符号候補の適用には入力位置の隣接も確認する。

## 読み比較

- seed5とseed6の各128件×葵・茜、計512件で全読み配列一致。
- 長音、複数接続、電話番号、混在数字、助数詞、日付、空白・タブ、
  英字・ギリシャ文字の前後の追加136件も全読み配列一致。
- 追加の期待値は通常SDK採取6回から作成した
  internal/portable/testdata/connector_reading_original.json。採取元のSHA256を保存した。
- TestConnectorReadingOriginalとTestRandomSeed6ReadingOriginalを通常の回帰テストへ追加。
- 既存の読み・数字・小数・電話・時刻・日付・符号の回帰テスト、numeral全テスト、go vetが成功。

原版graphはprivate/reading-seed6-failures-akane-graph.jsonl。
追加採取はprivate/connector-seed6-{akane,aoi}.json、connector-extra-*、connector-mixed-*。
採取した原版WAVは対応する*-probe/に保持している。読みの一致を全WAV一致とは扱わない。

## 完全WAV比較

観測ツールなしの通常原版SDKを独立に呼び出し、以下の3文章×葵・茜=6件が
Windows amd64、CGO=0のGoとWAV全バイト一致した。合計648978 PCM16サンプル。

- 葉音3ー一。
- ЖＡ−\t0（あΩ。。（\tは実際のタブ）
- 3ー一人。

原版はAMD64 Family 25 Model 33 Stepping 0、通常threads=6、AVX512 convolutionなし。
原版DLLのSHA256は711aa449c58754bd89402093e077695e8e6175159623c8cf063b979aa3515c17。
原版WAV長・SHA256と比較reportのハッシュは
internal/portable/testdata/connector_wave_original.json。
独立原版WAVは、読み採取時の原版WAV長・SHA256とも一致した。
ログはprivate/connector-seed6-wave-akane.log（215.98秒）、
private/connector-seed6-wave-aoi.log（213.57秒）。途中tensorの注入は行っていない。

Linux amd64、CGO=0でも6件すべてが保存原版WAV長・SHA256と一致した。
原版DLLや途中tensorは再現入力に使っていない。ログはprivate/connector-wave-linux.log。
Windowsのembedded CLIを修正commit84b0c40から更新し、CGO_ENABLED=0と
vcs.modified=falseを確認した。茜の「葉音3ー一。」は原版の208912 bytes / SHA256
2395b0b75741d75fd41e9b9eb98373927b9b22d1343ea0115a173d5b77f124e3と一致した。
ログはprivate/connector-cli.log。

```powershell
$env:CGO_ENABLED = '0'
$env:AOIKANE_CONNECTOR_REFERENCE_AUDIT = '1'
go test ./internal/portable -run '^TestConnectorOriginalWAV$' -count=1 -timeout 30m -v
```

Intel Vocoder等の既知CPU差は未解決。512件の読みと6件のWAV一致を、
任意文章・全CPU・全設定の保証として扱わない。

## 次のランダム比較

seed7 / generator=1 / max_units=12の128件ずつを追加取得した。
各127件の読みが一致し、case74「\n）！é。é 😀A。」に1件の相違が残る。
先頭の\nは実際の改行。原版は空の読み<S><F>を3つとエーの読みを1つ、
Goは空の読みを2つとエーの読みを1つ返す。
原版graphの先頭に空のSurfaceと空の候補の形態素があることを観測した。
これはまだ修正していない。元graphはprivate/reading-seed7-failure-akane-graph.jsonl。
原版参考はtools/compare_reading/testdata/seed7-{akane,aoi}-original.jsonで、
厳密比較は相違を除外せず非ゼロ終了する。seed7全件のGo WAV一致は未検証。

後の先頭改行修正でseed7の読み差は解消した。追加比較と完全WAV検証は
[leading-newline-original.md](leading-newline-original.md)を参照。
