# BANCHA準備処理の文節統合

2026-10-12。seed10の茜case63/case93で、読みとアクセントは一致していたが休止長が異なっていた。
通常原版のCaboCha treeを観測すると、最初の文節境界と係り先はGoと一致。
tree観測時の完全WAVも、それぞれ独立のフックなし通常原版WAVと一致した。
差はCaboCha後のBANCHA準備処理128370で生じていた。

`probe_graph.py` の `PROBE_DELETIONS=1` で18fbe0の文節削除と128100の境界分類を観測。
削除の呼出元は128596で、最初の11f9c0数詞変換より前だった。
case63は人（分類5）→で（分類6）、case93は九（分類4）→（（分類3）の境界を統合する。
分類には基本形ではなく形態素の表層を使う。での基本形は別の文字列のため区別が必要。

| 分類 | 原版の判定（上から優先） |
| ---: | --- |
| 1 | 前置助数詞表、またはPOS 02030000 |
| 2 | 符号表 |
| 3 | 1文字の数字接続詞（125c90の種別4） |
| 4 | POS 01060000 |
| 5 | 助数詞表、またはPOS 01090700 |
| 6 | 接辞表 |
| 0 | その他 |

統合する左右分類は1→1/2/3/4、2→2/3/4、3→4、4→3/4/5/6、5→6。
最初の文節ordinalは保持する。削除した文節への係り先はその先へ追跡する。
準備処理で統合した境界を数詞走査と読み選択へ反映し、数詞による単語削除の文節縮約も維持した。

原版診断は `private/seed10-boundary-classes.jsonl`、SHA256
`95a4035064633c9524a13d41d5dc3c420226a4d4b9f666327c52a33e8cb4ffa9`。
2文章ともformatterの結果は保存した通常原版参考と一致、観測エラーなし。
この診断はformatter後に中断するため、それ自体を全WAV一致の証拠にはしない。

seed10の128件×葵・茜が全読み一致し、通常の厳密回帰へ追加した。
seed5〜10の計1536件および既存の追加読み比較も成功。
文節・数詞パッケージの全回帰は69.756秒/0.557秒、portableの読み回帰は21.441秒で成功。
修正対象2文章×両音声のフックなし通常原版WAV比較は専用テスト
`TestNumeralChunkOriginalWAV` で行う。

Intel/別AMDの既知数値差と未検証入力・設定の完全一致は引き続き未達。

Windows CGO=0の通常原版APIとの完全WAV比較は4件全成功。
茜279.40秒、葵280.41秒。フック・tensor注入なし。

| 文章 | 茜 bytes | 葵 bytes |
| --- | ---: | ---: |
| 一人私私読む一人でます。 | 369394 | 406688 |
| 零木「校九（下1、火私。 | 408652 | 443498 |

計814028 PCM16サンプルで全バイト一致。各WAVのSHA256は通常原版seed10採取時とも一致。
期待値は `internal/portable/testdata/numeral_chunk_wave_original.json` に保存。
採取元の原版report SHA256および保存WAVのサイズ・SHA256を検証して参照を作成した。
`AOIKANE_NUMERAL_CHUNK_REFERENCE_AUDIT=1` の `TestNumeralChunkReferenceWAV` は、
DLLや原版tensorを使わず文章からWAVを生成し、全体サイズ・SHA256を厳密比較する。
Linux amd64 CGO=0の4件監査は564.69秒で全成功（`private/numeral-chunk-linux-reference.log`）。
更新したstandalone embedded CLI（a5c6cb2 / Go 1.26.8 / CGO_ENABLED=0 / vcs.modified=false）でも、
茜「一人私私読む一人でます。」の369394 bytes・SHA256
`faf03c3a4792c5c5043915316ca1ebc58805b05d6944f7fe3a3ef8317a081285`が原版と一致。
