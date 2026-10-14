# 数詞seed5の全入力比較

2026-10-13。seed1〜4の既知入力に一致した後、固定生成器のseed5・count=128・max_units=16を追加した。葵・茜の全256原版採取について、正式な終了、全入力順序、DLL hash、入力hash、CP932整形読み、native report hash、保存元WAVのサイズとSHA256を検証した。

clean d3e0d07・Go1.26.8・CGO=0の比較バイナリを固定し、全128文をGoへ渡した。両声ともcase10・13・81・112の4読み差を発見。全件を[変更前の全比較](evidence/numeral-seed5-reading-baseline.json)に保持する。`AOIKANE_NUMERAL_SEED5_READING_AUDIT=1` / `TestRandomNumeralSeed5ReadingOriginal` は全128文×両声を読み、現時点で計8読み差で失敗する。入力や失敗例を削除・置換していない。

[全4入力の原版観測](evidence/numeral-seed5-residual-akane-original.json)は数詞enter/leave、助数詞引数、全stageを検証し、独立SDK整形読みに一致した。観測はフロントエンドのみで、Goの波形一致を証明しない。

- case10 `２兆万1円．５は、Ｃ個．は9。`：原版は２兆万をニチョーマンとして連結し、Goはニチョーとマンに分ける。
- case13 `0兆（個本兆九0・十９1八。`：原版は中点を強い区切りとして読むが、Goはゼロノとノを加える。
- case81 `A日．月人１月声0０秒9声。`：原版は００をレーにし、レービョーの1グループにする。Goはゼロビョーになる。
- case112 `9−千二兆千秒8兆1時。`：原版はセンビョーの表アクセントとハッチョーの結合を維持する。Goは秒のアクセントと8兆のグループが異なる。

この4/128はプロジェクトの完成度ではない。任意入力・全設定・全CPUの完全一致は未達。seed2全128文×両声の波形比較は別プロセスで継続中。

## 一致済みseedの全波形テスト

読み一致済みseed3/4について、`TestRandomNumeralSeed3OriginalWAV` と `TestRandomNumeralSeed4OriginalWAV` を追加した。固定生成器の全128入力が保存原版の全入力と同じことを先に照合し、全入力を通常SDK/Go波形比較へ渡す。読みの一致による入力の選別は行わない。`AOIKANE_TEST_ENGINE`、`AOIKANE_DIFF_VOICE`（葵・茜）、`AOIKANE_DIFF_OUTPUT` を指定して実行する。

今回確認したのはテストのコンパイル・登録・vetであり、SDK環境変数なしの実行は明示的にSKIPした。seed3/4の全256WAVを完走した結果としては扱わない。

## 秒の先頭ゼロ修正

case81は[秒の先頭ゼロ再構築](zero-seconds-original.md)で読み一致した。全128文×両声を再比較し、残る差はcase10・13・112の3文ずつ。上の変更前全件と失敗ゲートを維持する。

## 中点と先頭単位の修正

case13は[混在数詞と先頭単位の再構築](mixed-point-unit-original.md)で読みと両声の全WAVが一致した。全128文×両声を再比較し、残る差はcase10・112の2文ずつ。変更前の全件比較と厳密失敗ゲートを維持する。

## 大単位の表語幹修正

case10は[大単位の表語幹再構築](large-unit-chain-original.md)で読みと両声の全WAVが一致した。全128文×両声の差はcase112の1文ずつ。新たなゼロ2文の未解決差も全件監査へ保持する。

## 区切り列の内部秒の修正

case112は[区切り列の内部秒再構築](connector-seconds-original.md)で読みと両声の全WAVが一致した。全128文×両声の読み差はゼロになり、全件を通常回帰へ昇格した。seed5全256WAVの通常比較テストを追加したが完走は未実施。新たなゼロ2文の未解決差は別途全件監査に保持する。
