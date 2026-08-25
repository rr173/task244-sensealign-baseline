// Package matcher 依据例句覆盖度、语域相容性与上位/下位关系，
// 为两个跨语言词条生成候选义项对齐，并识别一对多冲突（需要拆分）。
package matcher

import (
	"sort"

	"task244-sensealign/internal/evidence"
	"task244-sensealign/internal/model"
	"task244-sensealign/internal/store"
)

// senseBundle 捆绑义项与其例句，供 matcher 计算覆盖度。
type senseBundle struct {
	Sense    *model.Sense
	Examples []*model.Example
}

// GenerateCandidates 对源词条与目标词条的全部义项两两配对，产出候选对齐。
// 每个候选带覆盖度、上位/下位标记与语域冲突标记，按覆盖度降序排列。
func GenerateCandidates(s *store.Store, sourceEntryID, targetEntryID string) ([]model.Candidate, error) {
	if sourceEntryID == targetEntryID {
		return nil, model.ErrSameEntry
	}
	srcSenses, err := loadSensesWithExamples(s, sourceEntryID)
	if err != nil {
		return nil, err
	}
	tgtSenses, err := loadSensesWithExamples(s, targetEntryID)
	if err != nil {
		return nil, err
	}
	var out []model.Candidate
	for _, ss := range srcSenses {
		for _, ts := range tgtSenses {
			c := buildCandidate(ss, ts)
			out = append(out, c)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].CovScore > out[j].CovScore
	})
	return out, nil
}

// loadSensesWithExamples 读取词条义项并附带其例句。
func loadSensesWithExamples(s *store.Store, entryID string) ([]*senseBundle, error) {
	senses, err := s.ListSenses(entryID)
	if err != nil {
		return nil, err
	}
	var out []*senseBundle
	for _, sn := range senses {
		exs, err := s.ListExamples(sn.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, &senseBundle{Sense: sn, Examples: exs})
	}
	return out, nil
}

// buildCandidate 计算单对义项的候选属性。
func buildCandidate(src, tgt *senseBundle) model.Candidate {
	cov := evidence.Coverage(src.Sense, src.Examples, tgt.Sense, tgt.Examples)
	compatible, conflict := evidence.RegisterCompatibility(src.Sense.RegisterTags, tgt.Sense.RegisterTags)
	_ = compatible
	hyper, hypo := detectHypernym(src.Sense, tgt.Sense)
	return model.Candidate{
		SourceSenseID:    src.Sense.ID,
		TargetSenseID:    tgt.Sense.ID,
		CovScore:         cov,
		Hypernym:         hyper,
		Hyponym:          hypo,
		RegisterConflict: conflict,
	}
}

// DetectOneToMany 识别一对多冲突：同一源义项存在多个覆盖度超过阈值的目标义项，
// 意味着该源义项可能是多义词，需要拆分。返回 sourceSenseID -> 冲突目标义项ID。
func DetectOneToMany(candidates []model.Candidate, threshold float64) map[string][]string {
	hits := make(map[string][]string)
	for _, c := range candidates {
		if c.CovScore >= threshold {
			hits[c.SourceSenseID] = append(hits[c.SourceSenseID], c.TargetSenseID)
		}
	}
	conflicts := make(map[string][]string)
	for src, tgts := range hits {
		if len(tgts) > 1 {
			conflicts[src] = tgts
		}
	}
	return conflicts
}
