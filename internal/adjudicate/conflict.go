package adjudicate

import (
	"task244-sensealign/internal/store"
)

// ConflictReport 汇总批次内的对齐冲突，供统计与自检使用。
type ConflictReport struct {
	// OneToMany：源义项ID -> 其多个高覆盖目标义项（需拆分）。
	OneToMany map[string][]string
	// RegisterConflicts：存在语域冲突的已裁决/候选对齐ID。
	RegisterConflicts []string
	// HypernymPairs：存在上位/下位关系的对齐ID（提示部分重合）。
	HypernymPairs []string
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
