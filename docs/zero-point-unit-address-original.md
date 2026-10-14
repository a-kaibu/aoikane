# 〇・億の住所読みへの誤分岐

2026-10-14。数詞seed7 case103「.〇・億の億分２日〇９の。」で、Goが〇・億を住所の「ゼロノ／オク」へ変換していた。原版は先頭がliteral〇または０の場合に住所パターンへ入らず、書かれた強いポーズを保持する。電話・郵便番号の明示パターンとliteral零は別の分岐のままにする。

[原版内部観測](evidence/zero-point-unit-address-akane-original.json)は3入力で100a90の出力ゼロｐオクを確認し、stageの対応、数詞の呼出し順、戻り値、最終整形と独立SDK採取の一致を検証した。内部観測は読み処理の証拠である。

関連20文と境界8文を両声で通常SDKから採取し、全56件の終了状態、DLL・入力・native reportのSHA256、入力順、CP932読み、実WAVを監査した。[20文とseed7全128文の読み比較](evidence/zero-point-unit-address-reading-comparison.json)と[境界8文の比較](evidence/zero-point-unit-address-boundary-reading.json)を保持する。境界「〇・億兆の。」で茜のアクセント群にも差があり、書かれた大単位列の最初の単位を独立させた。全28文×両声とseed7全128文×両声が一致し、入力を選別せず通常回帰に追加した。

Windowsでは元の長い文章、〇・億の、住所読みを保持する一・億の、の3文×両声を公開Pure GoバイナリのHTTP合成と通常SDKで比較した。全6WAV・735330 PCMサンプルが全バイト一致し、独立SDK採取とも一致した。[証拠](evidence/zero-point-unit-address-wav.json)のバイナリは後続の複数大単位修正より前に作成したもの。Go1.26.8、CGO=0、AMD64 Family25 Model33、threads6、AVX512 convolutions falseの結果。

最終ソースの全ReadingOriginal通常回帰は154.519秒で通過。数詞・文章処理の回帰、追加境界の回帰、vetも通過した。保存済みの全6WAVと境界「〇・億兆の。」両声の計8WAVをLinuxで再合成する参照テストも用意し、比較を継続している。任意入力・全設定・全CPUの完全一致は未達。
