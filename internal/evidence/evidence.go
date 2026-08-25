// Package evidence 整理义项用法证据：例句分词、跨语言义项覆盖度计算。
// 覆盖度用于 matcher 生成候选对齐时的置信评分，是“例句覆盖”约束的可计算实现。
package evidence

import (
	"strings"
	"unicode"

	"task244-sensealign/internal/model"
)

// Tokenize 将文本切分为可比对的词元集合。
// 拉丁文按字母序列成词（小写化）；中日韩表意文字逐字成词元，
// 使同语言例句可获得稳定的集合重叠。标点与空白均丢弃。
func Tokenize(text string) []string {
	var tokens []string
	var buf strings.Builder
	flush := func() {
		if buf.Len() > 0 {
			tokens = append(tokens, buf.String())
			buf.Reset()
		}
	}
	for _, r := range text {
		switch {
		case unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) ||
			unicode.Is(unicode.Katakana, r) || unicode.Is(unicode.Hangul, r):
			flush()
			tokens = append(tokens, string(r))
		case unicode.IsLetter(r):
			buf.WriteRune(unicode.ToLower(r))
		default:
			flush()
		}
	}
	flush()
	return tokens
}

// tokenSet 将若干文本聚合为去重词元集合（频次不敏感，便于集合重叠）。
func tokenSet(texts ...string) map[string]struct{} {
	set := make(map[string]struct{})
	for _, t := range texts {
		for _, tok := range Tokenize(t) {
			set[tok] = struct{}{}
		}
	}
	return set
}

// TokenSetOf 将单段文本聚合为去重词元集合（供关系检测等复用）。
func TokenSetOf(text string) map[string]struct{} {
	return tokenSet(text)
}

// SenseTokenSet 汇总义项定义与全部例句文本为词元集合。
func SenseTokenSet(sn *model.Sense, examples []*model.Example) map[string]struct{} {
	texts := []string{sn.Definition}
	for _, ex := range examples {
		texts = append(texts, ex.Text, ex.Translation)
	}
	return tokenSet(texts...)
}

// Coverage 计算两个义项（含例句）的语义覆盖度，返回 [0,1]。
// 采用双向包含率的 F1：covSrcInTgt = |S∩T|/|S|，covTgtInSrc = |T∩S|/|T|，
// 二者调和平均。仅当双向均有实质覆盖时得分较高，单方向覆盖判为部分重合。
func Coverage(a *model.Sense, aExs []*model.Example, b *model.Sense, bExs []*model.Example) float64 {
	sa := SenseTokenSet(a, aExs)
	sb := SenseTokenSet(b, bExs)
	if len(sa) == 0 || len(sb) == 0 {
		return 0
	}
	inter := 0
	for t := range sa {
		if _, ok := sb[t]; ok {
			inter++
		}
	}
	covSrcInTgt := float64(inter) / float64(len(sa))
	covTgtInSrc := float64(inter) / float64(len(sb))
	sum := covSrcInTgt + covTgtInSrc
	if sum == 0 {
		return 0
	}
	return 2 * covSrcInTgt * covTgtInSrc / sum
}
