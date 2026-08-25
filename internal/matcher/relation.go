package matcher

import (
	"task244-sensealign/internal/evidence"
	"task244-sensealign/internal/model"
)

// detectHypernym 通过义项定义词元集合的包含关系，启发式识别上位/下位关系。
// 若源义项定义词元是目标义项定义的真子集，则源为下位（hyponym）、目标为上位；
// 反向则源为上位（hypernym）。该标记用于提示“部分重合”而非“完全确认”。
func detectHypernym(src, tgt *model.Sense) (hyper bool, hypo bool) {
	sa := evidence.TokenSetOf(src.Definition)
	sb := evidence.TokenSetOf(tgt.Definition)
	if len(sa) == 0 || len(sb) == 0 {
		return false, false
	}
	if isProperSubset(sa, sb) {
		return false, true // 源是目标的真子集 → 源为下位
	}
	if isProperSubset(sb, sa) {
		return true, false // 目标是源的真子集 → 源为上位
	}
	return false, false
}

// TokenSetOf 暴露 evidence 的分词集合（定义文本聚合），供关系检测复用。
func TokenSetOf(text string) map[string]struct{} {
	return evidence.TokenSetOf(text)
}

func isProperSubset(a, b map[string]struct{}) bool {
	if len(a) >= len(b) {
		return false
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			return false
		}
	}
	return true
}
