package evidence

import "sort"

// RegisterTags 受控语域标签词表（语言学语域/语体）。
// 这些标签用于约束义项配对：若两侧语域集合互斥，则该对齐存在语域冲突。
var RegisterTags = []string{
	"formal",     // 正式
	"informal",   // 非正式
	"slang",      // 俚语
	"technical",  // 专业/术语
	"literary",   // 文学
	"colloquial", // 口语
	"archaic",    // 古语
	"dialect",    // 方言
}

// IsValidRegister 校验单个语域标签是否受控。
func IsValidRegister(tag string) bool {
	for _, r := range RegisterTags {
		if r == tag {
			return true
		}
	}
	return false
}

// NormalizeRegisters 过滤非法标签并去重，保持受控词表顺序的稳定输出。
func NormalizeRegisters(tags []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, t := range tags {
		if !IsValidRegister(t) {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool {
		return indexOfTag(out[i]) < indexOfTag(out[j])
	})
	return out
}

// RegisterCompatibility 判断两组语域标签是否相容。
// 返回 (compatible, conflict)：
//   - 任一侧为空：相容且无冲突（缺乏证据不等于冲突）。
//   - 交集非空：相容且无冲突。
//   - 双方均非空且互斥：不相容且存在冲突（应在对齐裁决中提示）。
func RegisterCompatibility(a, b []string) (compatible bool, conflict bool) {
	if len(a) == 0 || len(b) == 0 {
		return true, false
	}
	sa := toSet(a)
	for _, t := range b {
		if _, ok := sa[t]; ok {
			return true, false
		}
	}
	return false, true
}

func toSet(tags []string) map[string]struct{} {
	m := make(map[string]struct{}, len(tags))
	for _, t := range tags {
		m[t] = struct{}{}
	}
	return m
}

func indexOfTag(tag string) int {
	for i, r := range RegisterTags {
		if r == tag {
			return i
		}
	}
	return len(RegisterTags)
}
