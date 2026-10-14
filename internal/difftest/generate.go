// Package difftest generates deterministic text for original-engine comparisons.
package difftest

import "math/rand/v2"

const GeneratorVersion = 1

// Texts mixes unstructured characters with random word sequences. Every fourth
// case uses a different alphabet; generation is independent of prior failures.
func Texts(seed uint64, count, maxRunes int) []string {
	r := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	alphabets := [][]rune{
		[]rune("あいうえおかきくけこさしすせそたちつてとなにぬねのはひふへほまみむめもやゆよらりるれろわをんっゃゅょーがぎぐげござじずぜぞぱぴぷぺぽ"),
		[]rune("漢字日月火水木金土学校人私琴葉音声開閉生上下一二三四五六七八九零0123456789ＡＢＣａｂｃABCabc、。！？（）「」ー"),
		[]rune("あアｱ漢é😀ΩЖＡA０0〜～−－。、！？・（）「」\n\t "),
	}
	words := []string{"私", "猫", "魚", "窓", "口", "学校", "駅", "今日", "昨日", "生", "上手", "一人", "三人", "12", "3.5", "ABC", "hello", "が", "を", "は", "の", "に", "で", "と", "開ける", "読む", "食べる", "行く", "です", "ます", "青い", "大きな"}
	out := make([]string, count)
	for i := range out {
		length := 1 + r.IntN(maxRunes)
		text := ""
		if i%4 == 3 {
			for j := 0; j < length; j++ {
				text += words[r.IntN(len(words))]
			}
		} else {
			alphabet := alphabets[i%4]
			for j := 0; j < length; j++ {
				text += string(alphabet[r.IntN(len(alphabet))])
			}
		}
		out[i] = text + "。"
	}
	return out
}
