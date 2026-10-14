# 零秒の汎用読み分岐

2026-10-13。`零秒`は原本の数詞汎用処理を通り、`レイビョウ / レービョー / 2-2-4 / *` の1候補になる。Goは表の例外を採用しており、第1候補が `1-1-4`、第2候補がゼロ読みだった。元DLLの `11ec50` の後に保持される候補と照合し、既存の `零分` と同じく例外表を通さない分岐に合わせた。`〇秒` の例外表処理は保持する。

茜の `零秒月。` と `五読む零秒月（本2。` では、この候補の違いが後続助数詞とのアクセント結合に残っていた。単独の `零秒。` は最終の整形読みでは差が消えるため、文脈と内部候補を併せて確認した。

## 原本照合

- 零秒・〇秒・０秒と月/年/個月、動詞・名詞の前置を含む25文×2声＝50件の読みが一致。
- 固定seed1のランダム数詞128文×2声＝256件の読みが一致。従来残った茜case29も解消。
- 継続数詞24文×2声＝48件も一致。計354件を通常回帰へ接続し、入力・失敗ケースは除外していない。
- 原本フロントエンド4入力の全ステージ呼出しと数詞関数の出入を監査。最終整形読みは独立の通常SDK呼出しと一致。[候補と全呼出し](evidence/rei-seconds-akane-original.json)、[354件の比較](evidence/rei-seconds-reading-comparison.json)。
- 通常SDKとGoの4文×2声＝8件、763,752 PCMサンプルのWAV全体が一致。原本/Goの保存ファイルと報告SHA-256を再検証し、独立に採取した通常原本WAVとも同一。厳密基準は `internal/portable/testdata/rei_seconds_wave_original.json`。

WAV対象は `零秒。`、`零秒月。`、`五読む零秒月（本2。`、`〇秒月。`。CPUはAMD Family 25 Model 33、元処理6スレッド、AVX512畳み込みなし。Go 1.26.8 / Windows amd64 / CGO=0。source回帰と全ReadingOriginal回帰、変更パッケージのvetが通過。

`AOIKANE_TEST_ENGINE` を設定して `go test ./internal/portable -run TestReiSecondsOriginalWAV -count=1 -v` を実行すると通常SDKと比較する。DLL不要の厳密再生は `AOIKANE_REI_SECONDS_REFERENCE_AUDIT=1` と `TestReiSecondsReferenceWAV`。ランダム数詞全128文のWAV比較は `TestRandomNumeralSeed1OriginalWAV` に追加した。読みの一致だけで全128文のWAV一致や任意文章・全CPU・全設定の完成とは扱わない。

Linux amd64 / CGO=0で同じ8件の厳密WAV再生も全件通過（416.36秒）。
