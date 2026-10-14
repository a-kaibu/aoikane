# 空白を挟む句点と空文の再現

seed15 case54「。\t。−？0！。」（実タブ）の通常原版は4文、Goは3文だった。
空白を挟んだ句点も連続した終端記号とみなして捨てていたことが原因。
水平空白が挟まった場合は新しい終端記号の列を開始し、次の句点で空文を残すよう修正した。
直接連続する句点・疑問符・感嘆符の抑制と、改行直後の終端記号の抑制は保持する。

通常原版SDKの追加32文章×両音声の64件が全読み一致。
空白/実タブ/全角空白/縦タブ/改ページ/NBSPと、LF/CR/CRLFの対照を含む。
全talk成功・正常終了と保存WAVの長さ・SHA256を採取時に確認。
原版参照はinternal/portable/testdata/tab_sentence_reading_original.json。
seed5〜14の2560件と既存の追加読み比較は31.352秒で成功。
phrase/numeralの全回帰は62.438秒/0.649秒で成功。go vetも成功。

「。\t。−？0！。」「あ。 。い。」「。\t\n。」の両音声6件は、
フック・tensor差し替えのないWindows通常原版APIとGo WAV全バイトが一致。
茜53.33秒、葵55.13秒、計798720 PCM16サンプル。
独立合成の原版WAVのSHA256は先に採取した32件の保存WAVとも一致した。
原版参照はinternal/portable/testdata/tab_sentence_wave_original.json。
AOIKANE_TAB_SENTENCE_REFERENCE_AUDIT=1で文章からDLLなしの全WAV再現を厳密比較する。
Linux amd64 CGO=0の保存原版6件再現も74.02秒で全成功。
ログはprivate/tab-sentence-{akane,aoi}-wave.logとprivate/tab-sentence-linux-reference.log。

seed15の空文差は解消した。一零の読み差は残り、seed15全一致はまだ未達。
一零と文字種混在・助数詞・単位を含む48文章×両音声の通常原版参考を追加保存した。
tools/compare_reading/testdata/mixed-rei-seed15-{akane,aoi}-original.jsonで、
この空文修正時点の実装は各48件中31件不一致、比較ツールは終了コード1となる。
文字種区間に零が含まれる場合の桁読みへ通す試作では各11件まで減ったが、
先頭ゼロと助数詞の変化・単位・小数点に未再現分岐があるため試作はコミットしていない。
全48件を結果によらず保持し、許容差や入力の除外は追加していない。
試作メモはprivate/mixed-rei-prototype-note.txt。前後の厳密比較reportはprivate/mixed-rei-48-*-comparison.json。
これらの残る分岐、既知CPU差と未検証入力・設定の保証も継続課題。


4dbf2f1のclean treeからdist/aoikane-purego.exeを更新した。
Go 1.26.8、CGO_ENABLED=0、GOAMD64=v1、vcs.modified=falseを確認。
--portable embeddedで茜「あ。 。い。」を文章から合成し、原版WAVと一致。
274604 bytes、SHA256 `46dc082bb156ea3c63731907126273d5fdd91f007c0516ad82990452c2632d4f`。

後続79c5b86で混在区間の零と小数前の桁読みを保持する修正を行った。
seed15全256件の読み比較は成功し、通常回帰へ追加。追加48件×両音声は各10件の
不一致が残り、厳密監査は失敗する。独立した通常原版との6件のWAV全バイト一致も
確認した。区間・助数詞の観測と検証範囲の詳細はdocs/mixed-rei-original.md。

さらに小単位と助数詞の区間モードを再現し、追加48件×両音声の全96件の読み差を解消。
十百千・先頭ゼロ・十九を広げた追加64件×両音声も全128件の読みが一致した。
既知CPU差と未検証入力・設定は継続課題。検証範囲はdocs/mixed-rei-original.mdに記録。
