# 大単位・零と助数詞のアクセント

2026-10-13。原版100a90は `2兆零十` の `２兆` を10b010、`零` を1043f0、`十` を10b010で読む。Goは全体のレーを整数アクセント表で再解釈しようとして失敗し、ニとチョーを別の語にしていた。大単位で終わる整数片を先に読み、後続の零の数字対を維持する。

大単位語が4モーラ以下なら短い数字のTk調整を適用し、それを超える通常整数語ではC3助数詞の前で数字語の最後から1つ前に落ちる。`9千万円` はキューセンマ!ンエン、`20億円` はニジューオ!クエン。C1/C2では短い語のTk=1による平板化を適用しない。先頭ゼロからの数字対＋大単位は別の経路であり、その情報を単位の促音化で再読する場合にも伝搬する。初回に先頭ゼロの既存回帰で差が出たため修正し、失敗ログも保存した。

原版追加32文＋桁数境界16文、全48文×両声＝96読みが一致。全キャプチャの正常終了・入力hash・CP932・native report・保存原版WAVの全サイズとSHA256を検証し、`TestLargeZeroCounterReadingOriginal` に全件固定。[全6入力の処理観測](evidence/large-zero-counter-akane-original.json)は数詞呼出し、助数詞引数、全stage、独立SDK formatterとの一致を検証した。フロントエンド観測であり波形証拠は別に扱う。

seed2全128文×両声の整形読み差は1→0文になった。residual32も両声の全32文が一致。入力を削らず、失敗例を置き換えずに[全件前後比較](evidence/large-zero-counter-reading-comparison.json)を保存。`TestRandomNumeralSeed2ReadingOriginal` と `TestResidualNumeralReadingOriginal` は環境変数なしの通常回帰に昇格した。最終の全ReadingOriginal回帰と全件厳密ゲートは106.777秒で通過。数詞・句処理source全回帰も通過（句処理74.961秒）。変更パッケージのvetも通過。

## 波形比較

`2兆零十。`、`9千万円。`、`20万個年。`、`京・9千万円2兆零十、。` の4文×両声＝8WAV、928,692 PCMサンプルが通常SDKと全バイト一致。保存原版/Go WAVを全caseのreportと独立native採取ファイルに照合した。[波形の全証拠](evidence/large-zero-counter-wav.json)。波形テストは最終の数字対伝搬修正前にコンパイルし、その後の最終コードは全96読み・既存全読み・全件厳密ゲートで検証した。

Go1.26.8 / Windows amd64 / CGO=0、AMD Family25 Model33、原版6スレッド、AVX512畳み込みなし。通常SDK比較は `AOIKANE_TEST_ENGINE` / `TestLargeZeroCounterOriginalWAV`。DLL不要の厳密参照再生は `AOIKANE_LARGE_ZERO_COUNTER_REFERENCE_AUDIT=1` / `TestLargeZeroCounterReferenceWAV`。seed2の全128文の通常SDK/Go波形比較は `TestRandomNumeralSeed2OriginalWAV` で両声を指定して実施する。読み一致は全256波形や任意文章・全設定・全CPUの一致を意味しない。プロジェクト全体は未完了。

最終の数字対情報伝搬修正を含むclean 5d44783の分離した純Go CLIでも、seed2 case7を葵で合成し、通常SDKの保存WAV全体に一致（343,918 bytes）。CGO=0、portableタグ、vcs.modified=falseを確認。[最終CLI照合](evidence/large-zero-final-cli-wav.json)。

Linux amd64でも最終コードのportableバイナリで全8件の厳密原版WAV再生が一致して正常終了（504.29s）。Go1.26.8・CGO=0。[Linux照合](evidence/large-zero-counter-linux-wav.json)。
