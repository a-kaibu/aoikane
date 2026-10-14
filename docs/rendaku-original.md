# BANCHA連濁の再構築

2026-10-12。seed12 case65「日火Ａ三ｃ47。」で、両音声とも原版は火をビ、Goはヒと読んでいた。
原版の11f9c0・11ec50・115110・128670の後もヒ/カ/ヒー候補が残り、
116ab0の後にビ1候補へ置換されることを観測した。

## 原版から得た条件とデータ

Ghidraで116ab0・117720・1176d0・1175f0・1173b0・129590・1294b0を確認。
117720の適用判定は文節の先頭、品詞、読みの先頭と後続濁音、先/服の表層除外、
和語/漢語の規則、単漢字の音読み/訓読み分類を使う。
129590は表層の最初のCP932 2バイトを引き、選択読みと登録読みのprefixを比較する。
1173b0は前語末尾の漢字の音読みを、読み全体の長さと完全一致で照合する。
長さの一致は逆アセンブル11748bでも確認した。音読みが1モーラ、または濁音を含む2モーラなら拒否する。

原版のbantanが有効な状態（+28=1）で、1294b0の読取専用lookupを呼び、
6355字の4分類を取得した。分類0が音読み、1が訓読み。火はカ/コとヒ/ホ、日はジツ/ニチとカ/ヒ。
全取得には既定の60秒では足りず、最初の診断はtimeout。その不完全観測は資産に使っていない。
128字ごとの進捗記録を追加し、900秒上限の完全観測では全6355字・formatter到達・観測エラーなしを確認した。

- 完全観測: `private/rendaku-full-lexicon-v2.jsonl`
- SHA256: `4c06df41f42bfd16c6e3821ce2ca170b0dd701facae76ccfb5b028d15ca9ebb0`
- 単漢字規則行SHA256: `9a5ad3b3e2eb2059063b5c28afbf3b73163098ea82d523e1a8790faa4c6dc88c`（葵・茜同一）
- 連濁規則: 標準は和語222/漢語21、関西は和語218/漢語20。
- `tools/export_rendaku_assets.py` は完全観測と両音声の原版復号済み行から `assets/rules/rendaku.json` を作る。
- この辞書観測はformatter後に中断する。全WAV一致の証拠にはしない。

## 接続と比較

GoのApplyRendakuを文節分割後・BOUND前へ接続した。
D/S文節の先頭で前語をリセットし、適用時は選択候補の読み・発音の先頭を濁らせる。
アクセント・接続規則は保持し、候補を1個にする。

葵で数詞後の声などが余計に濁る回帰を検出。原版は115110で声の手前をS文節に分けていた。
標準語の未接続bancutも原版から取得し、`assets/rules/bancut-standard.json` として接続した。
3文章の同じロード済み表から28グループ・31判定を観測し、一致を確認した。
文章専用の読替えや数値許容差は追加していない。

seed12の128件×両音声が一致し、TestRandomSeed12ReadingOriginalへ追加した。
seed5〜12の計2048件と既存の追加読みが原版と一致。
品詞・語種・数字・句読点・文節境界を変えた48文章×両音声の96件も一致し、
TestRendakuReadingOriginalに通常原版の参考と採取SHA256を保存した。
最終のportable読み回帰は24.870秒で成功。phrase/numeralの全回帰は56.389秒/0.759秒で成功。

WAV比較は「日火Ａ三ｃ47。」「日会社。」「八開三九）9声。」×両音声。
TestRendakuOriginalWAVはWindowsのフックなし通常原版APIと全バイト比較する。
AOIKANE_RENDAKU_REFERENCE_AUDIT=1のTestRendakuReferenceWAVはDLLや原版tensorを使わず、
文章から生成したWAVのサイズ・SHA256を保存原版全体と比較する。
原版採取reportと実WAVのサイズ・SHA256を検証して参照を作った。
Windowsの直接原版比較は6件全成功、763136 PCM16サンプルで全バイト一致。
茜254.22秒、葵259.13秒。新しい独立の通常原版WAVは、96件採取時の原版WAVのSHA256/サイズとも一致した。
ログは `private/rendaku-{akane,aoi}-wave.log`、参考は `internal/portable/testdata/rendaku_wave_original.json`。
Linux amd64 CGO=0の6件再現も485.23秒で全成功（`private/rendaku-linux-reference.log`）。
原版DLLや途中tensorは使わず、文章から合成して全WAVのサイズ・SHA256を比較した。
standalone embedded CLIもbf92215 / Go 1.26.8 / CGO_ENABLED=0 / vcs.modified=falseで更新。
茜「日火Ａ三ｃ47。」の283738 bytes・SHA256
`0f0817952100190064811966d6948761a90c6b42e0790ce82626b403174c1ad7`が原版と一致した。

既知Intel/別AMDの数値差と未検証入力/設定の保証は残る。
