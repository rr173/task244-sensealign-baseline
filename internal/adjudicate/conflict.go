package adjudicate

import (
	"task244-sensealign/internal/model"
	"task244-sensealign/internal/store"
)

// ConflictReport 汇总批次内尚未解决的对齐冲突，供统计与自检使用。
// 仅统计当前仍有效的候选（不含已否决的对齐），以反映尚未解决的冲突。
type ConflictReport struct {
	// OneToMany：源义项ID -> 其多个高覆盖目标义项（需拆分）。
	OneToMany map[string][]string
	// RegisterConflicts：存在语域冲突的当前有效对齐ID。
	RegisterConflicts []string
	// HypernymPairs：存在上位/下位关系的当前有效对齐ID（提示部分重合）。
	HypernymPairs []string
}

// activeAlignment 判断对齐是否仍计入冲突统计。已否决的候选视为冲突已被人工解决，
// 不再是当前有效的候选，因此从冲突报告中排除。
func activeAlignment(a *model.Alignment) bool {
	return a.Relation != model.AlignRejected
}

// BuildConflictReport 扫描批次对齐，产出冲突报告。
func BuildConflictReport(s *store.Store, batchID string) (*ConflictReport, error) {
	aligns, err := s.ListAlignmentsForBatch(batchID)
	if err != nil {
		return nil, err
	}
	rep := &ConflictReport{
		OneToMany:         make(map[string][]string),
		RegisterConflicts: []string{},
		HypernymPairs:     []string{},
	}
	coverBySrc := make(map[string][]string)
	for _, a := range aligns {
		// 已否决的对齐不再计入冲突：其冲突已被人工裁决解决。
		if !activeAlignment(a) {
			continue
		}
		if a.CovScore >= 0.5 {
			coverBySrc[a.SourceSenseID] = append(coverBySrc[a.SourceSenseID], a.TargetSenseID)
		}
		if a.RegisterConflict {
			rep.RegisterConflicts = append(rep.RegisterConflicts, a.ID)
		}
		if a.Hypernym || a.Hyponym {
			rep.HypernymPairs = append(rep.HypernymPairs, a.ID)
		}
	}
	for src, tgts := range coverBySrc {
		if len(tgts) > 1 {
			rep.OneToMany[src] = tgts
		}
	}
	return rep, nil
}

// Summarize 将冲突报告压缩为可读统计。
func (r *ConflictReport) Summarize() map[string]int {
	return map[string]int{
		"one_to_many":       len(r.OneToMany),
		"register_conflict": len(r.RegisterConflicts),
		"hypernym_pairs":    len(r.HypernymPairs),
	}
}
