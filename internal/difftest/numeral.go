package difftest

import "math/rand/v2"

// NumeralTexts deliberately mixes numeral scripts, units, counters and surrounding
// text. Keep this stream separate from Texts so existing case indices stay stable.
func NumeralTexts(seed uint64, count, maxUnits int) []string {
	r := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	tokens := []string{"0", "1", "2", "5", "8", "9", "０", "１", "２", "５", "８", "９", "〇", "零", "一", "二", "三", "五", "八", "九", "十", "百", "千", "万", "億", "兆", "京", "本", "人", "円", "個", "年", "月", "日", "時", "分", "秒", "％", ".", "．", "・", "、", "（", "）", "−", "－", "A", "Ａ", "Ｃ", "の", "は", "声", "読む", "学校"}
	out := make([]string, count)
	for i := range out {
		length := 1 + r.IntN(maxUnits)
		text := ""
		for j := 0; j < length; j++ {
			text += tokens[r.IntN(len(tokens))]
		}
		out[i] = text + "。"
	}
	return out
}
