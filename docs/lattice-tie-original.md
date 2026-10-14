# 形態素解析の同点経路を原版に合わせる

seed14葵case38（実改行を含む）で第2文の０の境界Kindが原版0、Go29になっていた。
原版BOUND前の語グラフをGoへ渡すと予測結果は一致するため、モデル計算より前の差を調べた。
原版は末尾の連続マイナスを－－/－、Goは－/－－へ分割していた。

Ghidraの逆コンパイルで以下を確認した。
- 1d4cd0と1d1fe0は辞書・未知語候補を作成するたびに開始候補リストへ先頭挿入する。
- 1e2f90はそのリストを走査し、末尾位置のリストへも先頭挿入する。
- 1e2600は総コストが同点なら前ノードの累積コストが小さい経路を優先する。
  それも同点なら最初に走査したノードを保持する。

Goの候補走査を作成順の逆順にし、総コスト→前ノード累積コスト→末尾リスト順で経路を選ぶ。
同じ末尾位置・context・コストに圧縮する際も、新しい先頭ノードを保持する。
原版由来の7語の表層・基本形・品詞・選択読み候補はTestLatticeTieMorphologyOriginalで一致。
期待値はtestdata/lattice_tie_morphology_original.jsonに観測SHA256と共に保存。
語グラフを入力として差し替えた診断と、文章から独立合成する最終比較は区別する。

新規32文章×両音声の64件が全読み一致。連続マイナス1〜7個、英字/数詞/漢字の前置を含む。
seed14葵・茜も各128件全読み一致となり、厳密回帰へ追加した。
seed5〜14の2560件と既存の追加読み比較は38.975秒で成功。
frontend全回帰は2.081秒、go vetも成功。差を許容する条件や文章専用置換は追加していない。

対象の実改行を含む文章、短縮文、2個マイナスの対照文の両音声6件は、
Windowsの通常原版SDK APIとGo WAV全バイトが一致した。
茜264.76秒、葵265.30秒、合計691252 PCM16サンプル。
各原版のSHA256は先に採取した32件の保存WAVとも一致。
通常原版参照はtestdata/lattice_tie_wave_original.json。
AOIKANE_LATTICE_TIE_REFERENCE_AUDIT=1でDLLなしの文章→WAV生成を厳密比較する。
Linux amd64 CGO=0の保存原版6件再現も333.83秒で全成功。
ログはprivate/lattice-tie-{akane,aoi}-wave.logとprivate/lattice-tie-linux-reference.log。

次のseed15では両音声ともcase54の空文数とcase89の一零の読みが異なり、比較は終了コード1。
この不一致は残っている。既知CPU差と未検証入力・設定の保証も未達。


4085284のclean treeから単体CLI dist/aoikane-purego.exeを更新した。
Go 1.26.8、CGO_ENABLED=0、GOAMD64=v1、vcs.modified=falseを確認。
--portable embeddedで葵「0ЖＡ～）－－－。」を文章から合成し、原版WAVと一致。
182926 bytes、SHA256 `c87be923f5371928097c7071b8c5e73e2f35b95761b67892ac4c3d21ec98302d`。
原版SDK診断用dist/aoikane-dev.exeは保持した。
