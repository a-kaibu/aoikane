# 原版の設定受理範囲

2026-10-12、ローカルの通常原版DLLの公開configuration setterを直接呼び、
各呼出しで新しいconfigurationを作って96件を記録・厳密再確認した。
setterの戻り値の比較であり、96件の音声生成比較ではない。

| 設定 | 有限値の原版受理範囲（両端を含む） |
| --- | --- |
| Rate | 0.5〜4 |
| Pitch | 0.5〜2 |
| Volume | 0〜2 |
| Emph | 0〜2 |

境界直前・直後の隣接double値も比較した。範囲外は-3、受理は0。
期待値は `internal/engine/testdata/original_config_bounds.json`、
元記録は `private/original-config-bounds.json` と `.log`。
fixtureは元記録のSHA256と実際の原版DLL SHA256を保持する。

共有Request.Validateの一律0.01〜10を原版の各設定範囲へ変更した。
純Goの直接SynthesizeにもDefaults後のValidateを追加し、原版が拒否する
pitch=0.01、rate=4.01、volume/emphasis=2.01などを合成しない。
Volume/Emphの正の0.01未満は原版が受理するため、新しい検証でも受理する。
既存Requestのゼロ指定は「既定値1」の契約を維持する。これは生のsetterへ0を渡す操作とは異なる。
SDK setterはNaNを受理したが、既存Requestの非有限値拒否は維持し、NaNの音声生成は行わない。

原版setter96件の厳密再確認は `TestOriginalConfigurationSetters`。
`AOIKANE_CONFIG_REFERENCE_AUDIT=1` と `AOIKANE_TEST_ENGINE` が必要。
有限の非ゼロ値およびInfの88件については、共有Validateの受理/拒否と原版setter結果を
Windows/Linuxで照合した。ゼロのDefaultsも両OSで確認した。

速度上限Rate=4、茜、文章「あ。」はWindows通常原版APIと純GoでWAV全バイト一致。
85964 bytes、42960 PCM16サンプル、SHA256
`7fdb43556a3f230a955f2c4732accc22b5b27df84af51f6db7db3d9d31938047`。
Linux amd64 CGO=0でも同じ文章と設定から、原版の長さ・SHA256を再現した。
`TestRateBoundaryReferenceWAV` は `AOIKANE_CONTROLLED_REFERENCE_AUDIT=1` で厳密監査する。
元原版/Go WAVは `private/akane-rate-boundary/`。
ログは `private/akane-rate-boundary.log`、`private/config-boundary-linux.log`。

HTTP APIテスト、CLIのビルドテスト、portable/httpapiのgo vetも成功。
engineの通常go vetは既存native_windows.goの5箇所のunsafe.Pointer警告を報告する。
そのファイルは変更していない。engineはunsafeptr解析を除いたgo vetを確認した。
入力検証の修正は任意文章・全CPU・全設定での波形一致を証明するものではない。
Intel Vocoder等の既知差は残る。
